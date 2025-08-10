package fileio

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"go4pack/pkg/common/compress"
	"go4pack/pkg/common/file"
	"go4pack/pkg/common/fs"
)

// streamUploadHandler handles large file uploads with streaming (reduces memory usage)
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

	if _, _, err = fsys.CommitTempAsHashed(finalTempPath, md5sum); err != nil {
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
		rec = FileRecord{
			Filename:        header.Filename,
			Size:            written,
			CompressedSize:  compressedSize,
			CompressionType: compressionType,
			MD5:             md5sum,
			MIME:            mimeType,
			AnalysisStatus:  "none",
		}
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

	resp := gin.H{
		"filename":         header.Filename,
		"original_size":    written,
		"compressed_size":  compressedSize,
		"compression_type": compressionType,
		"md5":              md5sum,
		"mime":             mimeType,
		"analysis_status":  rec.AnalysisStatus,
		"id":               rec.ID,
	}
	c.JSON(http.StatusOK, resp)
}
