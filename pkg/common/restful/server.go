package restful

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go4pack/pkg/common/logger"
)

// Server wraps gin.Engine with graceful shutdown support
type Server struct {
	Engine      *gin.Engine
	httpServer  *http.Server
	addr        string
	shutdownDur time.Duration
}

// Option pattern for server configuration
type Option func(*Server)

func WithAddress(addr string) Option             { return func(s *Server) { s.addr = addr } }
func WithShutdownTimeout(d time.Duration) Option { return func(s *Server) { s.shutdownDur = d } }

// NewServer creates a new RESTful server instance
func NewServer(opts ...Option) *Server {
	g := gin.New()
	g.Use(gin.Recovery())
	g.Use(RequestLogger())
	g.Use(gin.Logger())

	s := &Server{
		Engine:      g,
		addr:        ":8080",
		shutdownDur: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(s)
	}

	s.httpServer = &http.Server{Addr: s.addr, Handler: s.Engine}
	return s
}

// Start runs the server asynchronously
func (s *Server) Start() error {
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.GetLogger().Error().Err(err).Msg("server error")
		}
	}()
	logger.GetLogger().Info().Str("addr", s.addr).Msg("REST server started")
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, s.shutdownDur)
	defer cancel()
	return s.httpServer.Shutdown(ctxTimeout)
}

// RequestLogger logs basic request info
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		logger.GetLogger().Info().Int("status", status).Str("method", c.Request.Method).Str("path", c.Request.URL.Path).Dur("latency", latency).Msg("request")
	}
}
