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

//go:embed static/landing/admin.png
var adminPng []byte

// StartLandingServer starts the HTTP server for the landing page on the specified port
func StartLandingServer(ctx context.Context, port int, logger *log.Logger) error {
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

	// Serve admin screenshot
	mux.HandleFunc("/admin.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(adminPng)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	// Start server in background
	go func() {
		logger.Printf("Landing page server starting on http://0.0.0.0:%d", port)
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
