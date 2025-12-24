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
	"strings"
	"syscall"
	"time"
)

const (
	DefaultPort    = 33335
	DefaultTimeout = 5 * time.Second
)

var version = "1.0.0-multitenant"

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

	if config.Logging.File != "" {
		file, err := os.OpenFile(config.Logging.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not open log file %s: %v\n", config.Logging.File, err)
		} else {
			logOutput = file
		}
	} else if *debug {
		logOutput = os.Stdout
	}

	logger := log.New(logOutput, "[moustique] ", log.LstdFlags)

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

	// Start server in goroutine
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
