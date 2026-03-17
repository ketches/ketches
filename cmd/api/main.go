package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/routes"
	"github.com/ketches/ketches/internal/services"
)

const serverShutdownTimeout = 10 * time.Second

func init() {
	app.PrintVersionBanner()
}

func main() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("failed to load .env file: %v", err)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app.InitConfig()

	if err := db.InitDB(); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	if err := services.EnsureBootstrapAdmin(); err != nil {
		log.Fatalf("failed to ensure bootstrap admin: %v", err)
	}

	if err := services.EnsureBuiltinExtensions(); err != nil {
		log.Fatalf("failed to ensure builtin extensions: %v", err)
	}

	if err := services.InitClusters(); err != nil {
		log.Fatalf("failed to initialize clusters: %v", err)
	}
	services.StartClusterNodeTerminalCleanupLoop()

	// Recover active build watchers
	core.GlobalBuildWatcher.RecoverActiveBuilds()
	go func() {
		recoveryCtx, cancel := context.WithTimeout(rootCtx, 30*time.Second)
		defer cancel()

		if err := core.RecoverTerminalBuildLogArchives(recoveryCtx); err != nil && recoveryCtx.Err() == nil {
			log.Printf("failed to recover terminal build log archives: %v", err)
		}
	}()
	go core.StartBuildLogMaintenance(rootCtx)

	r := gin.Default()
	routes.SetupRoutes(r)

	log.Printf("server starting on :%s", app.Config.Port)
	listener, err := net.Listen("tcp", ":"+app.Config.Port)
	if err != nil {
		log.Fatalf("failed to listen on :%s: %v", app.Config.Port, err)
	}

	srv := &http.Server{
		Addr:    ":" + app.Config.Port,
		Handler: r,
	}

	if err := runServer(rootCtx, srv, listener, serverShutdownTimeout); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func runServer(ctx context.Context, srv *http.Server, listener net.Listener, shutdownTimeout time.Duration) error {
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- srv.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}

		err := <-serverErrCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case err := <-serverErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
