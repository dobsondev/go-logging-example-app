package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dobsondev/go-logging-example-app/handlers"
	"github.com/dobsondev/go-logging-example-app/middleware"
	"github.com/dobsondev/go-logging-example-app/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const PORT string = ":8080"

func main() {
	// Create a logger that writes JSON to standard output
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// Set it as the default logger for the whole app
	slog.SetDefault(logger)

	ctx := context.Background()

	shutdown, err := telemetry.InitTracer(ctx)
	if err != nil {
		slog.Error("failed to init tracer", "err", err)
		os.Exit(1)
	}

	router := http.NewServeMux()

	router.HandleFunc("GET /health", handlers.HealthCheck)
	router.HandleFunc("GET /errors/{count}", handlers.LogErrors)

	middlewareStack := middleware.CreateStack(
		middleware.LogRequest,
	)

	srv := http.Server{
		Addr:    PORT,
		Handler: otelhttp.NewHandler(middlewareStack(router), "go-logging-example-app"),
	}

	// Channel to listen for OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine so it doesn't block
	go func() {
		fmt.Printf("Listening on port %s\n", PORT)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Block until we receive a signal
	<-quit
	slog.Info("shutting down server...")

	// Give in-flight requests 30 seconds to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown the HTTP server gracefully
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "err", err)
	}

	// Flush any remaining traces to Tempo/Jaeger
	if err := shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown tracer", "err", err)
	}

	slog.Info("server exited cleanly")
}
