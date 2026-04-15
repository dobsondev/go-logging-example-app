// Package handlers contains all HTTP handlers for the API.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func LogErrors(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Randomly (33% chance) simulate some extra work to show how spans work...
	randomNum := rand.Intn(100)
	if randomNum < 33 {
		extraWork(ctx)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	errorCount := r.PathValue("count")
	errorsToProduce, err := strconv.Atoi(errorCount)

	for i := 1; i <= errorsToProduce; i++ {
		slog.Error("This is an error", "count", i)
	}

	msg := fmt.Sprintf("%d errors written to logs", errorsToProduce)
	response := APIResponse{
		Status:  "ok",
		Message: msg,
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func extraWork(ctx context.Context) {
	tracer := otel.Tracer("go-logging-example-app")
	ctx, span := tracer.Start(ctx, "extraWork")
	defer span.End()

	span.SetAttributes(attribute.String("work.type", "simulated"))

	slog.Info("Simulating extra work in the handler... sleeping for 2 seconds")
	time.Sleep(2 * time.Second)
}
