package main

import (
	"embed"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

//go:embed static/admin.html
var adminHTML embed.FS

// ServeWebAdmin serves the web admin interface
func (s *Server) ServeWebAdmin(conn net.Conn) {
	// Read the embedded HTML file
	htmlBytes, err := adminHTML.ReadFile("static/admin.html")
	if err != nil {
		s.logger.Printf("Failed to read admin.html: %v", err)
		s.sendError(conn, err)
		return
	}

	html := string(htmlBytes)

	// Send response
	fmt.Fprintf(conn, "HTTP/1.0 200 OK\r\n")
	fmt.Fprintf(conn, "Content-Type: text/html; charset=utf-8\r\n")
	fmt.Fprintf(conn, "Content-Length: %d\r\n", len(html))
	fmt.Fprintf(conn, "\r\n")
	fmt.Fprintf(conn, "%s", html)
}

// GetRecentLogs returns the last N lines from the log
func (s *Server) GetRecentLogs(conn net.Conn, lines int) {
	// Read log file (if it exists)
	logPath, err := filepath.Abs(s.logger.Writer().(*os.File).Name())
	if err != nil {
		s.logger.Printf("Failed to get absolute log path: %v", err)
		s.sendJSON(conn, map[string]string{"error": "Failed to determine absolute log path"})
		return
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		s.sendJSON(conn, map[string]string{"error": "Log file not found"})
		return
	}

	// Get last N lines
	logLines := strings.Split(string(content), "\n")
	start := len(logLines) - lines
	if start < 0 {
		start = 0
	}

	recentLines := logLines[start:]
	s.sendJSON(conn, map[string]interface{}{
		"lines": recentLines,
		"total": len(logLines),
	})
}
