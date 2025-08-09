package fileio

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"go4pack/pkg/common/database"
)

// helper to setup router with routes
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rg := r.Group("/files")
	RegisterRoutes(rg)
	return r
}

// reset database singleton and runtime dir for clean state
func resetState(t *testing.T) string {
	database.ResetForTest()
	tempDir := t.TempDir()
	cwd, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(cwd) })
	return tempDir
}

func createMultipartFile(t *testing.T, fieldName, filename, content string) (*bytes.Buffer, string) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewBufferString(content)); err != nil {
		t.Fatalf("write content: %v", err)
	}
	w.Close()
	return &buf, w.FormDataContentType()
}

func TestUploadAndList(t *testing.T) {
	resetState(t)
	r := setupRouter()

	body, contentType := createMultipartFile(t, "file", "test.txt", "hello world test content to compress")
	req := httptest.NewRequest(http.MethodPost, "/files/upload", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload failed code=%d body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("compressed_size")) {
		t.Errorf("expected response to contain compressed metadata")
	}

	// list
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/files/list", nil))
	if w2.Code != http.StatusOK {
		t.Fatalf("list failed code=%d body=%s", w2.Code, w2.Body.String())
	}
	if !bytes.Contains(w2.Body.Bytes(), []byte("test.txt")) {
		t.Errorf("expected list to contain filename, got %s", w2.Body.String())
	}
}

func TestDownload(t *testing.T) {
	resetState(t)
	r := setupRouter()
	// upload first
	body, ct := createMultipartFile(t, "file", "d.txt", strings.Repeat("ABCD", 50))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/files/upload", body)
	req.Header.Set("Content-Type", ct)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload failed: %s", w.Body.String())
	}

	// download
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/files/download/d.txt", nil))
	if w2.Code != http.StatusOK {
		t.Fatalf("download failed code=%d", w2.Code)
	}
	if disp := w2.Header().Get("Content-Disposition"); disp == "" {
		t.Errorf("missing content disposition header")
	}
}

func TestStats(t *testing.T) {
	resetState(t)
	r := setupRouter()
	// upload multiple
	for i := 0; i < 3; i++ {
		content := strings.Repeat("data", i+1)
		body, ct := createMultipartFile(t, "file", filepath.Join("f", "file"+strconv.Itoa(i)+".txt"), content)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/files/upload", body)
		req.Header.Set("Content-Type", ct)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("upload %d failed: %s", i, w.Body.String())
		}
	}
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/files/stats", nil))
	if w2.Code != http.StatusOK {
		t.Fatalf("stats failed: %d", w2.Code)
	}
	if !bytes.Contains(w2.Body.Bytes(), []byte("file_count")) {
		t.Errorf("expected stats body, got %s", w2.Body.String())
	}
}

func TestDownloadNotFound(t *testing.T) {
	resetState(t)
	r := setupRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/files/download/missing.bin", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestConcurrentUploads(t *testing.T) {
	resetState(t)
	r := setupRouter()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			body, ct := createMultipartFile(t, "file", "c"+strconv.Itoa(idx)+".bin", strings.Repeat("X", 1024+idx))
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/files/upload", body)
			req.Header.Set("Content-Type", ct)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("concurrent upload %d failed: %d", idx, w.Code)
			}
		}(i)
	}
	wg.Wait()
}

func TestLargeUpload(t *testing.T) {
	resetState(t)
	r := setupRouter()
	large := strings.Repeat("LARGE", 5000)
	body, ct := createMultipartFile(t, "file", "large.txt", large)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/files/upload", body)
	req.Header.Set("Content-Type", ct)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("large upload failed: %d", w.Code)
	}
}

func TestUploadMissingFile(t *testing.T) {
	resetState(t)
	r := setupRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/files/upload", nil))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUploadTimestampMetadata(t *testing.T) {
	resetState(t)
	r := setupRouter()
	body, ct := createMultipartFile(t, "file", "meta.txt", "metadata test")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/files/upload", body)
	req.Header.Set("Content-Type", ct)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload failed: %d", w.Code)
	}
	// allow db flush
	time.Sleep(10 * time.Millisecond)
	// list and ensure timestamps present
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/files/list", nil))
	if w2.Code != http.StatusOK {
		t.Fatalf("list failed: %d", w2.Code)
	}
	if !bytes.Contains(w2.Body.Bytes(), []byte("created_at")) {
		t.Errorf("expected created_at in list")
	}
}
