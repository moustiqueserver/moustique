// main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	DefaultPort    = 33335
	DefaultTimeout = 5 * time.Second
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	generateConfig := flag.Bool("generate-config", false, "Generate default config file")
	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	// Generate config if requested
	if *generateConfig {
		if err := GenerateDefaultConfig(*configPath); err != nil {
			log.Fatalf("Failed to generate config: %v", err)
		}
		log.Printf("Generated default config at %s", *configPath)
		return
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

	// Öka file descriptor limit (Linux)
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err == nil {
		rLimit.Cur = rLimit.Max
		syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		log.Printf("Set file descriptor limit to %d", rLimit.Cur)
	}

	// Setup logger
	var logOutput io.Writer = os.Stderr // default

	if config.Logging.File != "" {
		file, err := os.OpenFile(config.Logging.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Kunde inte öppna loggfil %s: %v\n", config.Logging.File, err)
		} else {
			logOutput = file
		}
	} else if *debug {
		logOutput = os.Stdout
	}

	logger := log.New(logOutput, "[moustique] ", log.LstdFlags)

	version, err := GetFileVersion()
	if err != nil {
		logger.Fatalf("Could not calculate file version: %v", err)
	}

	// Initialize database
	//db, err := NewDatabase("/opt/data/mousqlite.db")
	db, err := NewDatabase(config.Database.Path)
	if err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Load values from database
	logger.Println("Loading data from database...")
	if err := db.LoadAll(); err != nil {
		logger.Printf("Warning: Failed to load values: %v", err)
	}

	// Initialize broker
	broker := NewBroker(logger, db, *debug)

	// Initialize server
	//server := NewServer(*port, DefaultTimeout, logger, broker, *debug, version)
	server := NewServer(config.Server.Port, config.Server.Timeout, logger, broker, *debug, version)

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

	// Start maintenance routines
	go broker.StartMaintenance(ctx)

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Printf("Received signal %v - shutting down gracefully...", sig)

	// Cancel context to stop server
	cancel()

	// Give server time to finish current requests
	logger.Println("Waiting for active requests to complete...")
	time.Sleep(time.Second)

	// Save database - KRITISKT STEG
	logger.Println("Saving database to disk...")
	startSave := time.Now()
	if err := db.SaveAll(); err != nil {
		logger.Printf("CRITICAL ERROR: Failed to save database: %v", err)
		logger.Println("Data may be lost!")
		os.Exit(1)
	}
	logger.Printf("Database saved successfully in %v", time.Since(startSave))

	logger.Println("Shutdown complete")
}
