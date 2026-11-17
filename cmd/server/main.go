package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DeanXu2357/taskbe/internal/httpapi"
	"github.com/DeanXu2357/taskbe/internal/logx"
	"github.com/DeanXu2357/taskbe/internal/task/service"
	"github.com/DeanXu2357/taskbe/internal/task/store/memory"
)

const shutdownTimeout = 10 * time.Second

func main() {
	logger := logx.New(log.Default())

	store := memory.New()
	svc := service.New(store, time.Now)
	mux := http.NewServeMux()
	httpapi.Register(mux, svc, logger)
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		stop()
		logger.Error("server failed", "error", err)
		os.Exit(1)
		return
	case <-ctx.Done():
	}
	// Restore default signal handling so a second signal terminates the process immediately.
	stop()

	logger.Info("shutting down", "timeout", shutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	err := srv.Shutdown(shutdownCtx)
	cancel()
	if err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}
