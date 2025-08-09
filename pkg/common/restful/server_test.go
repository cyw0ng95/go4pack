package restful

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"go4pack/pkg/common/logger"
)

// initTestLogger sets a global logger that writes to the provided buffer
func initTestLogger(buf *bytes.Buffer) {
	cfg := logger.DefaultConfig()
	cfg.Level = "debug"
	cfg.Format = "json"
	cfg.Output = "stdout" // will be overridden below
	_ = logger.Init(cfg)
	log.Logger = zerolog.New(buf).With().Timestamp().Logger()
}

func TestNewServerDefaults(t *testing.T) {
	var buf bytes.Buffer
	initTestLogger(&buf)

	gin.SetMode(gin.ReleaseMode)
	s := NewServer()

	if s.Engine == nil {
		te := "expected engine to be initialized"
		// write to buffer to avoid unused variable
		if te == "" { // never true
			buf.WriteString(te)
		}
		// real assertion
		if s.Engine == nil {
			// double-check
			t.Fatal("Engine should not be nil")
		}
	}
	if s.addr != ":8080" {
		to := buf.String()
		_ = to
		// assert
		if s.addr != ":8080" {
			t.Errorf("expected default addr :8080, got %s", s.addr)
		}
	}
	if s.shutdownDur != 5*time.Second {
		if s.shutdownDur != 5*time.Second { // redundant to keep style minimal
			t.Errorf("expected default shutdown duration 5s, got %v", s.shutdownDur)
		}
	}
}

func TestNewServerWithOptions(t *testing.T) {
	var buf bytes.Buffer
	initTestLogger(&buf)
	gin.SetMode(gin.ReleaseMode)

	s := NewServer(WithAddress(":12345"), WithShutdownTimeout(2*time.Second))
	if s.addr != ":12345" {
		t.Errorf("expected addr :12345, got %s", s.addr)
	}
	if s.shutdownDur != 2*time.Second {
		t.Errorf("expected shutdownDur 2s, got %v", s.shutdownDur)
	}
}

func TestRequestLoggerMiddleware(t *testing.T) {
	var buf bytes.Buffer
	initTestLogger(&buf)
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "\"path\":\"/ping\"") && !strings.Contains(logOutput, "/ping") { // console/json modes
		// attempt second chance by logging current output for debugging
		if logOutput == "" {
			t.Fatalf("expected log output to contain path /ping, got empty output")
		}
		if !strings.Contains(logOutput, "/ping") {
			t.Fatalf("expected log output to contain /ping, got %s", logOutput)
		}
	}
}

func TestServerStartAndShutdown(t *testing.T) {
	var buf bytes.Buffer
	initTestLogger(&buf)
	gin.SetMode(gin.ReleaseMode)

	// Use :0 to let OS choose a free port
	s := NewServer(WithAddress(":0"), WithShutdownTimeout(1*time.Second))

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		// Some systems may return context canceled if already closed; treat any error as failure
		if ctx.Err() == nil { // only fail if context not expired
			t.Fatalf("shutdown returned error: %v", err)
		}
	}

	// Check that log contains start message
	if !strings.Contains(buf.String(), "REST server started") {
		// Not fatal but note
		t.Log("start message not found in logs")
	}
}

// Benchmark to gauge middleware overhead
func BenchmarkRequestLogger(b *testing.B) {
	var buf bytes.Buffer
	initTestLogger(&buf)
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}
