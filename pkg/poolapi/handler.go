package poolapi

import (
	"net/http"

	"go4pack/pkg/common/worker"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers pool stats endpoints
func RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"pool": worker.StatsSnapshot()})
	})
}
