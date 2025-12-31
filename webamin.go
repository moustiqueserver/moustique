package main

import (
	_ "embed"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
)

//go:embed static/admin.html
var adminHTML string

//go:embed static/superadmin.html
var superadminHTML string

//go:embed static/favicon.svg
var faviconSVG string

//go:embed static/landing/index.html
var landingHTML string

//go:embed static/signup.html
var signupHTML string

//go:embed static/moustique_logo.png
var logoPNG []byte

//go:embed static/robots.txt
var robotsTxt string

func (s *Server) ServeWebAdmin(conn net.Conn) {
	s.sendHTML(conn, adminHTML)
}

func (s *Server) ServeSignup(conn net.Conn) {
	s.sendHTML(conn, signupHTML)
}

func (s *Server) ServeSuperAdmin(conn net.Conn) {
	s.sendHTML(conn, superadminHTML)
}

func (s *Server) ServeFavicon(conn net.Conn) {
	s.sendSVG(conn, faviconSVG)
}

func (s *Server) ServeLanding(conn net.Conn) {
	s.sendHTML(conn, landingHTML)
}

func (s *Server) ServeLogo(conn net.Conn) {
	s.sendPNG(conn, logoPNG)
}

func (s *Server) ServeRobotsTxt(conn net.Conn) {
	s.sendText(conn, robotsTxt)
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

	// Get last N lines efficiently using tail-like approach
	recentLines, totalLines, err := readLastLines(logPath, lines)
	if err != nil {
		s.sendJSON(conn, map[string]interface{}{
			"lines": []string{fmt.Sprintf("Error reading log: %v", err)},
			"total": 0,
		})
		return
	}

	s.sendJSON(conn, map[string]interface{}{
		"lines": recentLines,
		"total": totalLines,
	})
}

// readLastLines reads the last N lines from a file efficiently
func readLastLines(filepath string, n int) ([]string, int, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	fileSize := stat.Size()

	// If file is small (< 100KB), just read it all
	if fileSize < 100000 {
		content, err := io.ReadAll(file)
		if err != nil {
			return nil, 0, err
		}
		allLines := strings.Split(string(content), "\n")
		totalLines := len(allLines)
		start := totalLines - n
		if start < 0 {
			start = 0
		}
		return allLines[start:], totalLines, nil
	}

	// For large files, read backwards from end
	// Estimate: average line is ~100 bytes, so read n*150 bytes to be safe
	bufSize := int64(n * 150)
	if bufSize > fileSize {
		bufSize = fileSize
	}

	// Seek to position
	offset := fileSize - bufSize
	if offset < 0 {
		offset = 0
	}

	_, err = file.Seek(offset, 0)
	if err != nil {
		return nil, 0, err
	}

	// Read the chunk
	buf := make([]byte, bufSize)
	bytesRead, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return nil, 0, err
	}

	// Split into lines
	lines := strings.Split(string(buf[:bytesRead]), "\n")

	// Remove first partial line if we didn't start at beginning
	if offset > 0 && len(lines) > 0 {
		lines = lines[1:]
	}

	// Get last N lines
	start := len(lines) - n
	if start < 0 {
		start = 0
	}

	// Estimate total lines (this is approximate for large files)
	totalLines := len(lines)
	if offset > 0 {
		// Rough estimate based on bytes per line
		avgLineLen := bufSize / int64(len(lines))
		if avgLineLen > 0 {
			totalLines = int(fileSize / avgLineLen)
		}
	}

	return lines[start:], totalLines, nil
}
