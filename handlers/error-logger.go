// Package handlers contains all HTTP handlers for the API.
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
)

func LogErrors(w http.ResponseWriter, r *http.Request) {
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
