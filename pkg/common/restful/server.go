package restful

import (
	"context"
	"net/http"
	"strings"
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
	// route panics to zerolog
	g.Use(RecoveryWithLogger())
	g.Use(CORSMiddleware())
	g.Use(RequestLogger())
	// direct gin internal output to zerolog (avoid duplicate default logger middleware)
	gin.DefaultWriter = zerologWriter{}
	gin.DefaultErrorWriter = zerologWriter{}

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

// zerologWriter adapts gin's writer to zerolog
type zerologWriter struct{}

func (zerologWriter) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		logger.GetLogger().Info().Msg(msg)
	}
	return len(p), nil
}

// RecoveryWithLogger logs panic with stack/latency via zerolog (simplified)
func RecoveryWithLogger() gin.HandlerFunc {
	return gin.RecoveryWithWriter(zerologWriter{})
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

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
