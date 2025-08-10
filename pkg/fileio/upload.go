package fileio

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"

	"go4pack/pkg/common/compress"
	elfutil "go4pack/pkg/common/elf"
	"go4pack/pkg/common/file"
	"go4pack/pkg/common/fs"
	"go4pack/pkg/common/logger"
	"go4pack/pkg/common/worker"
)

// scheduleELFAnalysis submits an async job to analyze ELF and update DB record.
func scheduleELFAnalysis(recID uint, data []byte) {
	_ = worker.Submit(func() {
		logger.GetLogger().Debug().Uint("record_id", recID).Msg("starting async ELF analysis")
		db, err := ensureDB()
		if err != nil {
			return
		}
		analysis, aerr := elfutil.AnalyzeBytes(data)
		if aerr != nil {
			msg := aerr.Error()
			db.Model(&FileRecord{}).Where("id = ?", recID).Updates(map[string]any{"analysis_status": "error", "analysis_error": msg})
			logger.GetLogger().Error().Uint("record_id", recID).Err(aerr).Msg("elf analysis failed")
			return
		}
		b, _ := json.Marshal(analysis)
		js := string(b)
		cache := &ElfAnalyzeCached{FileID: recID, Data: js}
		_ = db.Where("file_id = ?", recID).Assign(map[string]any{"data": js}).FirstOrCreate(cache).Error
		db.Model(&FileRecord{}).Where("id = ?", recID).Update("analysis_status", "done")
		logger.GetLogger().Info().Uint("record_id", recID).Int("size", len(data)).Msg("elf analysis completed")
	})
}

func uploadHandler(c *gin.Context) {
	fileHdr, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer fileHdr.Close()

	fsys, err := fs.New()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "filesystem init failed"})
		return
	}

	data, err := io.ReadAll(fileHdr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read file failed"})
		return
	}

	originalSize := int64(len(data))
	md5sum := file.MD5Sum(data)
	mimeType := file.DetectMIME(data, header.Filename)
	preCT := compress.IsCompressedOrMIME(data, mimeType)

	if err := fsys.WriteObjectHashedWithMIME(md5sum, data, mimeType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "store file failed"})
		return
	}
	if vErr := fsys.VerifyHashedRegular(md5sum); vErr != nil {
		_ = fsys.DeleteObject(md5sum)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stored object"})
		return
	}
	compressedSize, err := fsys.GetHashedObjectSize(md5sum)
	if err != nil {
		logger.GetLogger().Warn().Err(err).Str("hash", md5sum).Msg("failed to get compressed size")
		compressedSize = originalSize
	}
	compressionType := fsys.GetCompressor().Type().String()
	if preCT != compress.None {
		compressionType = preCT.String()
	}
	// create DB record (analysis pending if ELF)
	db, dbErr := ensureDB()
	var rec FileRecord
	if dbErr == nil {
		rec = FileRecord{Filename: header.Filename, Size: originalSize, CompressedSize: compressedSize, CompressionType: compressionType, MD5: md5sum, MIME: mimeType, AnalysisStatus: "none"}
		if len(data) >= 4 && data[0] == 0x7f && data[1] == 'E' && data[2] == 'L' && data[3] == 'F' {
			rec.AnalysisStatus = "pending"
		}
		_ = db.Create(&rec).Error
	}
	if rec.AnalysisStatus == "pending" {
		scheduleELFAnalysis(rec.ID, data)
	}
	logger.GetLogger().Info().Str("filename", header.Filename).Str("hash", md5sum).Int64("original_size", originalSize).Int64("compressed_size", compressedSize).Str("compression", compressionType).Str("mime", mimeType).Msg("file uploaded")
	resp := gin.H{"filename": header.Filename, "original_size": originalSize, "compressed_size": compressedSize, "compression_type": compressionType, "compression_ratio": float64(compressedSize) / float64(originalSize), "md5": md5sum, "mime": mimeType, "analysis_status": rec.AnalysisStatus, "id": rec.ID}
	c.JSON(http.StatusOK, resp)
}

func uploadMultiHandler(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		return
	}
	form := c.Request.MultipartForm
	files := form.File["files"]
	if len(files) == 0 {
		files = form.File["file"]
	}
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files provided"})
		return
	}
	fsys, err := fs.New()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "filesystem init failed"})
		return
	}
	db, dbErr := ensureDB()
	type result struct {
		ID               uint    `json:"id"`
		Filename         string  `json:"filename"`
		OriginalSize     int64   `json:"original_size"`
		CompressedSize   int64   `json:"compressed_size"`
		CompressionType  string  `json:"compression_type"`
		CompressionRatio float64 `json:"compression_ratio"`
		MD5              string  `json:"md5"`
		MIME             string  `json:"mime"`
		AnalysisStatus   string  `json:"analysis_status"`
		Error            string  `json:"error,omitempty"`
	}
	results := make([]result, len(files))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i, fh := range files {
		wg.Add(1)
		idx := i
		fheader := fh
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			res := &results[idx]
			res.Filename = fheader.Filename
			f, err := fheader.Open()
			if err != nil {
				res.Error = "open failed"
				return
			}
			data, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				res.Error = "read failed"
				return
			}
			res.OriginalSize = int64(len(data))
			res.MD5 = file.MD5Sum(data)
			res.MIME = file.DetectMIME(data, fheader.Filename)
			preCT := compress.IsCompressedOrMIME(data, res.MIME)
			if err := fsys.WriteObjectHashedWithMIME(res.MD5, data, res.MIME); err != nil {
				res.Error = "store failed"
				return
			}
			if vErr := fsys.VerifyHashedRegular(res.MD5); vErr != nil {
				res.Error = "invalid stored object"
				return
			}
			cs, err := fsys.GetHashedObjectSize(res.MD5)
			if err != nil {
				cs = res.OriginalSize
			}
			res.CompressedSize = cs
			if preCT != compress.None {
				res.CompressionType = preCT.String()
			} else {
				res.CompressionType = fsys.GetCompressor().Type().String()
			}
			if res.OriginalSize > 0 {
				res.CompressionRatio = float64(res.CompressedSize) / float64(res.OriginalSize)
			}
			if dbErr == nil && db != nil {
				rec := &FileRecord{Filename: res.Filename, Size: res.OriginalSize, CompressedSize: res.CompressedSize, CompressionType: res.CompressionType, MD5: res.MD5, MIME: res.MIME, AnalysisStatus: "none"}
				if len(data) >= 4 && data[0] == 0x7f && data[1] == 'E' && data[2] == 'L' && data[3] == 'F' {
					rec.AnalysisStatus = "pending"
				}
				_ = db.Create(rec).Error
				res.ID = rec.ID
				res.AnalysisStatus = rec.AnalysisStatus
				if rec.AnalysisStatus == "pending" {
					scheduleELFAnalysis(rec.ID, data)
				}
			}
			logger.GetLogger().Info().Str("filename", res.Filename).Str("hash", res.MD5).Int64("original_size", res.OriginalSize).Int64("compressed_size", res.CompressedSize).Str("compression", res.CompressionType).Str("mime", res.MIME).Msg("file uploaded (multi)")
		}()
	}
	wg.Wait()
	c.JSON(http.StatusOK, gin.H{"results": results, "count": len(results)})
}

func streamUploadHandler(c *gin.Context) {
	fileHdr, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer fileHdr.Close()
	fsys, err := fs.New()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "filesystem init failed"})
		return
	}
	temp, err := os.CreateTemp(fsys.GetObjectsPath(), "up-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "temp create failed"})
		return
	}
	defer temp.Close()
	h := md5.New()
	var written int64
	buf := make([]byte, 32*1024)
	for {
		n, rerr := fileHdr.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if _, err := h.Write(chunk); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
				return
			}
			if _, err := temp.Write(chunk); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "write failed"})
				return
			}
			written += int64(n)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "read failed"})
			return
		}
	}
	md5sum := hex.EncodeToString(h.Sum(nil))
	if _, err := temp.Seek(0, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seek failed"})
		return
	}
	head := make([]byte, 512)
	nHead, _ := io.ReadFull(temp, head)
	mimeType := file.DetectMIME(head[:nHead], header.Filename)
	if _, err := temp.Seek(0, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seek failed"})
		return
	}
	firstBytes := head[:nHead]
	preCT := compress.IsCompressedOrMIME(firstBytes, mimeType)
	finalTempPath := temp.Name()
	if preCT == compress.None {
		if _, err := temp.Seek(0, 0); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "seek failed"})
			return
		}
		compTemp, err := os.CreateTemp(fsys.GetObjectsPath(), "upc-*")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "temp comp failed"})
			return
		}
		cWriter := fsys.GetCompressor()
		data, err := io.ReadAll(temp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "read temp failed"})
			return
		}
		compressedData, err := cWriter.Compress(data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "compress failed"})
			return
		}
		if _, err := compTemp.Write(compressedData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "write comp failed"})
			return
		}
		compTemp.Close()
		_ = os.Remove(finalTempPath)
		finalTempPath = compTemp.Name()
		written = int64(len(data))
	}
	_, _, err = fsys.CommitTempAsHashed(finalTempPath, md5sum)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}
	if vErr := fsys.VerifyHashedRegular(md5sum); vErr != nil {
		_ = fsys.DeleteObject(md5sum)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stored object"})
		return
	}
	compressedSize, _ := fsys.GetHashedObjectSize(md5sum)
	compressionType := preCT.String()
	if preCT == compress.None {
		compressionType = fsys.GetCompressor().Type().String()
	}
	// schedule async if elf magic
	if _, err := temp.Seek(0, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seek failed"})
		return
	}
	magic := make([]byte, 4)
	n, _ := temp.Read(magic)
	isELF := n == 4 && magic[0] == 0x7f && magic[1] == 'E' && magic[2] == 'L' && magic[3] == 'F'
	if _, err := temp.Seek(0, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seek failed"})
		return
	}
	var rec FileRecord
	if db, err := ensureDB(); err == nil {
		rec = FileRecord{Filename: header.Filename, Size: written, CompressedSize: compressedSize, CompressionType: compressionType, MD5: md5sum, MIME: mimeType, AnalysisStatus: "none"}
		if isELF {
			rec.AnalysisStatus = "pending"
		}
		_ = db.Create(&rec).Error
		if isELF {
			if dataAll, rErr := io.ReadAll(temp); rErr == nil {
				scheduleELFAnalysis(rec.ID, dataAll)
			}
		}
	}
	resp := gin.H{"filename": header.Filename, "original_size": written, "compressed_size": compressedSize, "compression_type": compressionType, "md5": md5sum, "mime": mimeType, "analysis_status": rec.AnalysisStatus, "id": rec.ID}
	c.JSON(http.StatusOK, resp)
}
