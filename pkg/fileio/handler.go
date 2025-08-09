package fileio

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"

	"go4pack/pkg/common/fs"
	"go4pack/pkg/common/logger"
)

// RegisterRoutes registers file upload/download routes under given router group
func RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/upload", uploadHandler)
	rg.GET("/download/:filename", downloadHandler)
}

func uploadHandler(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	fsys, err := fs.New()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "filesystem init failed"})
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read file failed"})
		return
	}

	if err := fsys.WriteObject(header.Filename, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save file failed"})
		return
	}

	logger.GetLogger().Info().Str("filename", header.Filename).Int("size", len(data)).Msg("file uploaded")
	c.JSON(http.StatusOK, gin.H{"filename": header.Filename, "size": len(data)})
}

func downloadHandler(c *gin.Context) {
	filename := c.Param("filename")
	fsys, err := fs.New()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "filesystem init failed"})
		return
	}

	objectPath := filepath.Join(fsys.GetObjectsPath(), filename)
	f, err := os.Open(objectPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	defer f.Close()

	stat, _ := f.Stat()
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Length", strconv.FormatInt(stat.Size(), 10))
	c.File(objectPath)

	logger.GetLogger().Info().Str("filename", filename).Int64("size", stat.Size()).Msg("file downloaded")
}
