package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RotatingLogger handles logging with rotation
type RotatingLogger struct {
	mu           sync.Mutex
	file         *os.File
	logPath      string
	maxSize      int64 // in bytes
	currentSize  int64
	rotateCount  int
}

// NewRotatingLogger creates a new rotating logger
func NewRotatingLogger(logDir, filename string) (*RotatingLogger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logPath := filepath.Join(logDir, filename)

	// Open log file
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open access log: %w", err)
	}

	// Get current file size
	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat access log: %w", err)
	}

	rl := &RotatingLogger{
		file:        file,
		logPath:     logPath,
		maxSize:     3 * 1024 * 1024, // 3MB max per file
		currentSize: fileInfo.Size(),
		rotateCount: 2, // Keep 2 old files (3 total: current + 2 old)
	}

	return rl, nil
}

// Write writes a log line with timestamp
func (rl *RotatingLogger) Write(format string, args ...interface{}) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("%s | %s\n", timestamp, fmt.Sprintf(format, args...))

	// Write to file
	n, err := rl.file.WriteString(logLine)
	if err != nil {
		log.Printf("Failed to write to log: %v", err)
		return
	}

	rl.currentSize += int64(n)

	// Check if rotation is needed
	if rl.currentSize >= rl.maxSize {
		if err := rl.rotate(); err != nil {
			log.Printf("Failed to rotate log: %v", err)
		}
	}
}

// LogAccess writes an access log entry
func (rl *RotatingLogger) LogAccess(clientIP, method, path, username string, statusCode int, duration float64) {
	if username == "" {
		username = "unauthenticated"
	}
	rl.Write("%s | %s | %s | %s | %d | %.2fms", clientIP, method, path, username, statusCode, duration)
}

// LogError writes an error log entry
func (rl *RotatingLogger) LogError(clientIP, errorType, message string) {
	rl.Write("%s | %s | %s", clientIP, errorType, message)
}

// rotate rotates the log files
func (rl *RotatingLogger) rotate() error {
	// Close current file
	if err := rl.file.Close(); err != nil {
		return err
	}

	// Rotate old files: logfile.4 -> logfile.5, etc.
	for i := rl.rotateCount - 1; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", rl.logPath, i)
		newPath := fmt.Sprintf("%s.%d", rl.logPath, i+1)

		// Remove oldest if it exists
		if i == rl.rotateCount-1 {
			os.Remove(newPath)
		}

		// Rename if old file exists
		if _, err := os.Stat(oldPath); err == nil {
			os.Rename(oldPath, newPath)
		}
	}

	// Move current log to .1
	if err := os.Rename(rl.logPath, rl.logPath+".1"); err != nil {
		return err
	}

	// Create new log file
	file, err := os.OpenFile(rl.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	rl.file = file
	rl.currentSize = 0

	return nil
}

// Close closes the logger
func (rl *RotatingLogger) Close() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.file != nil {
		return rl.file.Close()
	}
	return nil
}
