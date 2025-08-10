package fileio

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers file upload/download routes under given router group
func RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/upload", uploadHandler)
	rg.POST("/upload/multi", uploadMultiHandler)
	rg.POST("/upload/stream", streamUploadHandler)

	rg.GET("/download/:filename", downloadHandler)
	rg.GET("/download/by-md5/:md5", downloadByMD5Handler)

	rg.GET("/list", listHandler)
	rg.GET("/stats", statsHandler)
	rg.GET("/meta/:id", metaHandler)
}
