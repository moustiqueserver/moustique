package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
)

//go:embed static/landing/moustique_logo.png
var moustiqueLogo []byte

// StartLandingServer starts the HTTP server on port 80 for the landing page
func StartLandingServer(ctx context.Context, logger *log.Logger) error {
	mux := http.NewServeMux()

	// Serve landing page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, landingHTML)
	})

	// Serve favicon
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, faviconSVG)
	})

	mux.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, faviconSVG)
	})

	// Serve logo image
	mux.HandleFunc("/moustique_logo.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(moustiqueLogo)
	})

	server := &http.Server{
		Addr:    ":80",
		Handler: mux,
	}

	// Start server in background
	go func() {
		logger.Println("Landing page server starting on http://0.0.0.0:80")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Printf("Landing server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Shutdown gracefully
	logger.Println("Shutting down landing page server...")
	return server.Shutdown(context.Background())
}
