package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/dobsondev/go-logging-example-app/handlers"
	"github.com/dobsondev/go-logging-example-app/middleware"
)

const PORT string = ":8080"

func main() {
	// Create a logger that writes JSON to standard output
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// Set it as the default logger for the whole app
	slog.SetDefault(logger)

	router := http.NewServeMux()

	router.HandleFunc("GET /errors/{count}", handlers.LogErrors)

	middlewareStack := middleware.CreateStack(
		middleware.LogRequest,
	)

	srv := http.Server{
		Addr:    PORT,
		Handler: middlewareStack(router),
	}

	fmt.Printf("Listening on port %s\n", PORT)
	err := srv.ListenAndServe()
	log.Fatal(err)
}
