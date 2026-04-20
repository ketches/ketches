package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
		fatal("failed to load .env file", err)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app.InitConfig()
	if err := app.ValidateRuntimeConfig(); err != nil {
		fatal("invalid runtime configuration", err)
	}

	if err := db.InitDB(); err != nil {
		fatal("failed to initialize database", err)
	}

	if err := services.EnsureBootstrapAdmin(); err != nil {
		fatal("failed to ensure bootstrap admin", err)
	}

	if err := services.EnsureBuiltinExtensions(); err != nil {
		fatal("failed to ensure builtin extensions", err)
	}

	if err := services.InitClusters(); err != nil {
		fatal("failed to initialize clusters", err)
	}
	if err := services.ReconcilePublicGatewayResources(rootCtx); err != nil {
		fatal("failed to reconcile public gateway resources", err)
	}
	nodeTerminalCleanupDone := services.StartClusterNodeTerminalCleanupLoop(rootCtx)

	// Recover active build watchers
	core.GlobalBuildWatcher.SetParentContext(rootCtx)
	core.GlobalBuildWatcher.RecoverActiveBuilds()
	services.GlobalBuilderWorker.SetParentContext(rootCtx)
	if err := services.GlobalBuilderWorker.RecoverActiveRuns(rootCtx); err != nil {
		fatal("failed to recover builder worker state", err)
	}
	services.GlobalBuilderWorker.Start()
	go func() {
		recoveryCtx, cancel := context.WithTimeout(rootCtx, 30*time.Second)
		defer cancel()

		if err := core.RecoverTerminalBuildLogArchives(recoveryCtx); err != nil && recoveryCtx.Err() == nil {
			slog.Error("failed to recover terminal build log archives", "error", err)
		}
	}()
	go core.StartBuildLogMaintenance(rootCtx)

	r := gin.Default()
	routes.SetupRoutes(r)

	slog.Info("server starting", "port", app.Config.Port)
	listener, err := net.Listen("tcp", ":"+app.Config.Port)
	if err != nil {
		fatal(fmt.Sprintf("failed to listen on :%s", app.Config.Port), err)
	}

	srv := &http.Server{
		Addr:    ":" + app.Config.Port,
		Handler: r,
	}

	if err := runServer(rootCtx, srv, listener, serverShutdownTimeout); err != nil {
		fatal("failed to start server", err)
	}

	// Restore default signal handling so a second interrupt can force-exit if shutdown stalls.
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancel()

	core.GlobalBuildWatcher.StopAll()
	services.GlobalBuilderWorker.Stop()
	if err := waitForGracefulShutdown(shutdownCtx, nodeTerminalCleanupDone, core.GlobalBuildWatcher.Wait, services.GlobalBuilderWorker.Wait); err != nil {
		slog.Error("graceful shutdown incomplete", "error", err)
	}
}

func fatal(message string, err error) {
	slog.Error(message, "error", err)
	os.Exit(1)
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
	waitForBuilderWorker func(),
) error {
	if nodeTerminalCleanupDone != nil {
		select {
		case <-nodeTerminalCleanupDone:
		case <-ctx.Done():
			return app.WrapError("node terminal cleanup loop did not stop before timeout", ctx.Err())
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
			return app.WrapError("build watchers did not stop before timeout", ctx.Err())
		}
	}

	if waitForBuilderWorker != nil {
		done := make(chan struct{})
		go func() {
			defer close(done)
			waitForBuilderWorker()
		}()

		select {
		case <-done:
		case <-ctx.Done():
			return app.WrapError("builder worker did not stop before timeout", ctx.Err())
		}
	}

	return nil
}
