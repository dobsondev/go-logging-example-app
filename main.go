package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

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
	defer shutdown(ctx)

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

	fmt.Printf("Listening on port %s\n", PORT)
	err = srv.ListenAndServe()
	log.Fatal(err)
}
