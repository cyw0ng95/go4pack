package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go4pack/pkg/common"
	"go4pack/pkg/common/restful"
	"go4pack/pkg/common/worker"
	"go4pack/pkg/fileio"
	"go4pack/pkg/poolapi"
	"go4pack/pkg/process"
	"go4pack/pkg/rpc"
)

// RunAPI starts the API server (former main) so we can have multiple commands.
func RunAPI() {
	if err := common.Init(); err != nil {
		panic(err)
	}
	logger := common.GetLogger()
	logger.Info().Msg("Starting go4pack API service")

	if common.IsDebug() {
		logger.Debug().Msg("Debug mode enabled")
	}

	fsys, err := common.GetFileSystem()
	if err != nil {
		logger.Fatal().Err(err).Msg("Filesystem init failed")
	}
	logger.Info().Str("runtime_path", fsys.GetRuntimePath()).Str("objects_path", fsys.GetObjectsPath()).Msg("Runtime paths ready")

	if err := worker.Init(8); err != nil {
		logger.Error().Err(err).Msg("Worker pool init failed")
	}

	// Initialize process model
	proc := process.New()
	proc.Start()
	// Example RPC handlers
	_ = proc.Register("system.ping", func(ctx context.Context, m rpc.Message) (any, error) {
		return map[string]any{"pong": true, "ts": time.Now().UnixMilli()}, nil
	})
	_ = proc.Register("system.uptime", func(ctx context.Context, m rpc.Message) (any, error) {
		return map[string]any{"uptime_ms": proc.Uptime().Milliseconds()}, nil
	})

	srv := restful.NewServer(restful.WithAddress(":8080"))
	api := srv.Engine.Group("/api")
	fileGroup := api.Group("/fileio")
	fileio.RegisterRoutes(fileGroup)
	poolGroup := api.Group("/pool")
	poolapi.RegisterRoutes(poolGroup)

	if err := srv.Start(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start server")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("Shutting down server...")
	proc.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Server shutdown error")
	}
	logger.Info().Msg("Server exited cleanly")
}
