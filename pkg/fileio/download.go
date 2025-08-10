package fileio

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"go4pack/pkg/common/fs"
)

// Handlers focused on downloading and metadata listing.

func downloadHandler(c *gin.Context) {
	filename := c.Param("filename")
	fsys, err := fs.New()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "filesystem init failed"})
		return
	}
	db, err := ensureDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db init failed"})
		return
	}
	var fr FileRecord
	if err := db.Where("filename = ?", filename).First(&fr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	data, rErr := fsys.ReadObjectHashed(fr.MD5)
	if rErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read failed"})
		return
	}
	dispType := "attachment"
	if strings.HasPrefix(fr.MIME, "video/") || fr.MIME == "application/pdf" {
		dispType = "inline"
	}
	c.Header("Content-Disposition", dispType+"; filename="+filename)
	c.Header("Content-Length", strconv.FormatInt(fr.Size, 10))
	c.Header("Content-Type", fr.MIME)
	c.Writer.Write(data)
}

func downloadByMD5Handler(c *gin.Context) {
	md5v := c.Param("md5")
	fsys, err := fs.New()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "filesystem init failed"})
		return
	}
	db, err := ensureDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db init failed"})
		return
	}
	var fr FileRecord
	if err := db.Where("md5 = ?", md5v).First(&fr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	data, rErr := fsys.ReadObjectHashed(fr.MD5)
	if rErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read failed"})
		return
	}
	dispType := "attachment"
	if strings.HasPrefix(fr.MIME, "video/") || fr.MIME == "application/pdf" {
		dispType = "inline"
	}
	c.Header("Content-Disposition", dispType+"; filename="+fr.Filename)
	c.Header("Content-Length", strconv.FormatInt(fr.Size, 10))
	c.Header("Content-Type", fr.MIME)
	c.Writer.Write(data)
}
