package main

import (
	_ "embed"
	"net"
	"os"
	"path/filepath"
	"strings"
)

//go:embed static/admin.html
var adminHTML string

//go:embed static/superadmin.html
var superadminHTML string

func (s *Server) ServeWebAdmin(conn net.Conn) {
	s.sendHTML(conn, adminHTML)
}

func (s *Server) ServeSuperAdmin(conn net.Conn) {
	s.sendHTML(conn, superadminHTML)
}

/*o:embed static/admin.html
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

*/

// GetRecentLogs returns the last N lines from the server log
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

// GetUserLogs returns the last N lines from a user's log
func (s *Server) GetUserLogs(conn net.Conn, broker *Broker, lines int) {
	logPath := broker.GetUserLogPath()
	if logPath == "" {
		s.sendJSON(conn, map[string]interface{}{
			"lines": []string{"User logging not configured"},
			"total": 0,
		})
		return
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		s.sendJSON(conn, map[string]interface{}{
			"lines": []string{"User log file not found or empty"},
			"total": 0,
		})
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
