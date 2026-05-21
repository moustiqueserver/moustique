// main.go - Multi-tenant version with fixes
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	DefaultPort    = 33335
	DefaultTimeout = 5 * time.Second
)

var version = "1.0.1"

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	generateConfig := flag.Bool("generate-config", false, "Generate default config file")
	debug := flag.Bool("debug", false, "Enable debug logging")
	addUser := flag.String("add-user", "", "Add user (format: username:password)")
	listUsers := flag.Bool("list-users", false, "List all users")
	flag.Parse()

	// Generate config if requested
	if *generateConfig {
		if err := GenerateDefaultConfig(*configPath); err != nil {
			log.Fatalf("Failed to generate config: %v", err)
		}
		log.Printf("Generated default config at %s", *configPath)
		return
	}

	if _, err := os.Stat("/etc/moustique/config.yaml"); err == nil {
		*configPath = "/etc/moustique/config.yaml"
	}

	// Load config
	config, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override debug from flag
	if *debug {
		config.Logging.Level = "debug"
	} else {
		*debug = config.Logging.Level == "debug"
	}

	// Setup logger
	var logOutput io.Writer = os.Stderr

	if config.Logging.Directory != "" {
		// Ensure log directory exists
		if err := os.MkdirAll(config.Logging.Directory, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Could not create log directory %s: %v\n", config.Logging.Directory, err)
		} else {
			logPath := filepath.Join(config.Logging.Directory, "moustique.log")
			file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not open log file %s: %v\n", logPath, err)
			} else {
				logOutput = file
			}
		}
	} else if *debug {
		logOutput = os.Stdout
	}

	logger := log.New(logOutput, "[moustique] ", log.LstdFlags)

	// Setup error logger for moustique_err.log
	var errorLogger *log.Logger
	if config.Logging.Directory != "" {
		errLogPath := filepath.Join(config.Logging.Directory, "moustique_err.log")
		errFile, err := os.OpenFile(errLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			logger.Printf("Warning: Could not open error log %s: %v", errLogPath, err)
		} else {
			errorLogger = log.New(errFile, "[moustique-err] ", log.LstdFlags)
		}
	}

	fileVersion, err := GetFileVersion()
	if err != nil {
		logger.Printf("Warning: Could not calculate file version: %v", err)
		fileVersion = version
	}

	// Initialize data directory
	dataDir := config.Database.Path
	if dataDir == "" {
		dataDir = "./data"
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Fatalf("Failed to create data directory: %v", err)
	}

	// Check if public access is allowed
	allowPublic := false
	if config.Server.AllowPublic != nil {
		allowPublic = *config.Server.AllowPublic
	}

	// Initialize server with multi-tenant support
	server, err := NewServer(
		config.Server.Port,
		config.Server.Timeout,
		logger,
		dataDir,
		*debug,
		fileVersion,
		allowPublic,
		config.Security.AllowedPeers,
		config.Server.MaxRequestSize,
		config.Security.MaxTopicLength,
		config.Security.MaxMessageSize,
		config.Security.DefaultRateLimit,
		config.Security.Fail2banJail,
		config.Security.Fail2banLevel,
		config.Logging.Directory,
		&config.Lightning,
		&config.SystemEvents,
	)
	if err != nil {
		logger.Fatalf("Failed to create server: %v", err)
	}

	// Handle list users
	if *listUsers {
		// Load user auth to list users
		logger.Println("Registered users:")
		// This would require exposing the user list from UserAuth
		logger.Println("(User listing feature - implement if needed)")
		return
	}

	// Handle user management
	if *addUser != "" {
		// Parse username:password
		parts := strings.SplitN(*addUser, ":", 2)
		if len(parts) != 2 {
			logger.Fatalf("Invalid format. Use: username:password")
		}
		if err := server.AddUser(parts[0], parts[1]); err != nil {
			logger.Fatalf("Failed to add user: %v", err)
		}
		logger.Printf("User added successfully: %s", parts[0])
		return
	}

	// Add demo users in debug mode
	if *debug {
		server.AddUser("demo", "demo123")
		server.AddUser("alice", "alice123")
		server.AddUser("bob", "bob123")
		logger.Println("Debug mode: Added demo users (demo/demo123, alice/alice123, bob/bob123)")
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)

	// Start MQTT server if configured
	if config.Server.MQTTPort > 0 {
		go func() {
			if err := server.StartMQTT(config.Server.MQTTPort); err != nil {
				logger.Printf("MQTT server error: %v", err)
			}
		}()
	}

	// Start landing page server if configured (port > 0)
	if config.Server.LandingPort != nil && *config.Server.LandingPort > 0 {
		landingCfg := &LandingServerConfig{
			Port:               *config.Server.LandingPort,
			Logger:             logger,
			ErrorLogger:        errorLogger,
			Fail2banJail:       config.Security.Fail2banJail,
			SystemEventsBroker: nil, // Not available yet, initialized in server.Start()
		}
		go func() {
			if err := StartLandingServer(ctx, landingCfg); err != nil {
				logger.Printf("Landing server error: %v", err)
			}
		}()
	} else {
		logger.Println("Landing page server disabled (landing_port: 0)")
	}

	// Start main server in goroutine
	go func() {
		if err := server.Start(ctx); err != nil {
			logger.Printf("Server error: %v", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Printf("Received signal %v - shutting down gracefully...", sig)

	// Cancel context to stop server
	cancel()

	// Give server time to finish current requests
	logger.Println("Waiting for active requests to complete...")
	time.Sleep(time.Second)

	// Save all user databases
	logger.Println("Saving all databases...")
	startSave := time.Now()

	if err := server.brokerManager.SaveAll(); err != nil {
		logger.Printf("ERROR: Failed to save databases: %v", err)
		logger.Println("Some data may be lost!")
		os.Exit(1)
	}

	logger.Printf("All databases saved successfully in %v", time.Since(startSave))
	logger.Println("Shutdown complete")
}
