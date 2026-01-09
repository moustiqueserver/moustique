package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

//go:embed static/landing/moustique_logo.png
var moustiqueLogo []byte

//go:embed static/landing/admin.png
var adminPng []byte

// LandingCrook tracks invalid requests to landing server
type LandingCrook struct {
	IP        string
	Attempts  int
	FirstSeen time.Time
	LastSeen  time.Time
	IsBanned  bool
}

// LandingServerConfig holds configuration for the landing server
type LandingServerConfig struct {
	Port               int
	Logger             *log.Logger
	ErrorLogger        *log.Logger
	Fail2banJail       string
	SystemEventsBroker *Broker
}

var (
	landingCrooks   = make(map[string]*LandingCrook)
	landingCrooksMu sync.Mutex
)

// recordLandingInvalidRequest records an invalid request and bans if needed
func recordLandingInvalidRequest(cfg *LandingServerConfig, ip string, path string) {
	if ip == "" {
		return
	}

	landingCrooksMu.Lock()
	defer landingCrooksMu.Unlock()

	now := time.Now()
	crook, exists := landingCrooks[ip]

	if !exists {
		crook = &LandingCrook{
			IP:        ip,
			Attempts:  1,
			FirstSeen: now,
			LastSeen:  now,
			IsBanned:  false,
		}
		landingCrooks[ip] = crook
		cfg.Logger.Printf("Landing: New invalid request from %s: %s", ip, path)
	} else {
		crook.Attempts++
		crook.LastSeen = now
	}

	// Ban on first invalid request (these are clearly scanners/attackers)
	if !crook.IsBanned && cfg.Fail2banJail != "" {
		crook.IsBanned = true

		// Log to error log
		if cfg.ErrorLogger != nil {
			cfg.ErrorLogger.Printf("BANNED: Landing server banning IP %s after %d invalid requests (path: %s)", ip, crook.Attempts, path)
		}
		cfg.Logger.Printf("Landing: Banning IP %s (attempts: %d, path: %s)", ip, crook.Attempts, path)

		// Execute fail2ban command in background
		go func(jail, banIP string) {
			cmd := exec.Command("sudo", "fail2ban-client", "set", jail, "banip", banIP)
			output, err := cmd.CombinedOutput()
			if err != nil {
				cfg.Logger.Printf("Landing: Warning - Failed to ban IP %s via fail2ban jail %s: %v (output: %s)", banIP, jail, err, string(output))
			} else {
				cfg.Logger.Printf("Landing: Successfully banned IP %s via fail2ban jail %s", banIP, jail)
			}
		}(cfg.Fail2banJail, ip)

		// Publish system event if broker available
		if cfg.SystemEventsBroker != nil {
			cfg.SystemEventsBroker.PublishEvent("/system/event/ip/banned", formatJSON(map[string]interface{}{
				"broker":    "landing",
				"ip":        ip,
				"reason":    "invalid_landing_request",
				"path":      path,
				"attempts":  crook.Attempts,
				"banned_at": now.Format("2006-01-02 15:04:05"),
				"timestamp": now.Unix(),
			}))
		}
	}
}

// StartLandingServer starts the HTTP server for the landing page on the specified port
func StartLandingServer(ctx context.Context, cfg *LandingServerConfig) error {
	mux := http.NewServeMux()

	// Serve landing page - ONLY for exact "/" path
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr
		// Extract IP from "ip:port" format
		if host, _, err := splitHostPort(clientIP); err == nil {
			clientIP = host
		}

		// Check X-Forwarded-For header (from nginx)
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			clientIP = xff
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			clientIP = xri
		}

		// Return 404 and record invalid request for any path other than exactly "/"
		if r.URL.Path != "/" {
			cfg.Logger.Printf("Landing 404: %s %s from %s", r.Method, r.URL.Path, clientIP)
			recordLandingInvalidRequest(cfg, clientIP, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		cfg.Logger.Printf("Landing 200: %s %s from %s", r.Method, r.URL.Path, clientIP)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, landingHTML)
	})

	// Serve robots.txt
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, robots)
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
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	// Start server in background
	go func() {
		cfg.Logger.Printf("Landing page server starting on http://0.0.0.0:%d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cfg.Logger.Printf("Landing server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Shutdown gracefully
	cfg.Logger.Println("Shutting down landing page server...")
	return server.Shutdown(context.Background())
}

// splitHostPort splits "host:port" into host and port
func splitHostPort(hostport string) (host, port string, err error) {
	for i := len(hostport) - 1; i >= 0; i-- {
		if hostport[i] == ':' {
			return hostport[:i], hostport[i+1:], nil
		}
	}
	return hostport, "", fmt.Errorf("no port found")
}
