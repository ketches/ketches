package main

import (
	"context"
	"errors"
	"fmt"
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
	nodeTerminalCleanupDone := services.StartClusterNodeTerminalCleanupLoop(rootCtx)

	// Recover active build watchers
	core.GlobalBuildWatcher.SetParentContext(rootCtx)
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

	// Restore default signal handling so a second interrupt can force-exit if shutdown stalls.
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancel()

	core.GlobalBuildWatcher.StopAll()
	if err := waitForGracefulShutdown(shutdownCtx, nodeTerminalCleanupDone, core.GlobalBuildWatcher.Wait); err != nil {
		log.Printf("graceful shutdown incomplete: %v", err)
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

func waitForGracefulShutdown(
	ctx context.Context,
	nodeTerminalCleanupDone <-chan struct{},
	waitForBuildWatchers func(),
) error {
	if nodeTerminalCleanupDone != nil {
		select {
		case <-nodeTerminalCleanupDone:
		case <-ctx.Done():
			return fmt.Errorf("node terminal cleanup loop did not stop before timeout: %w", ctx.Err())
		}
	}

	if waitForBuildWatchers != nil {
		done := make(chan struct{})
		go func() {
			defer close(done)
			waitForBuildWatchers()
		}()

		select {
		case <-done:
		case <-ctx.Done():
			return fmt.Errorf("build watchers did not stop before timeout: %w", ctx.Err())
		}
	}

	return nil
}
