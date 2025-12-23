package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Server handles HTTP connections
type Server struct {
	port     int
	timeout  time.Duration
	logger   *log.Logger
	broker   *Broker
	auth     *Auth
	security *SecurityChecker
	debug    bool
	version  string
}

// NewServer creates a new HTTP server
func NewServer(port int, timeout time.Duration, logger *log.Logger, broker *Broker, debug bool, Version string) *Server {
	return &Server{
		port:     port,
		timeout:  timeout,
		logger:   logger,
		broker:   broker,
		auth:     NewAuth(broker.db),
		security: NewSecurityChecker(),
		debug:    debug,
		version:  Version,
	}
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer listener.Close()

	s.logger.Printf("Starting Moustique on port %d", s.port)

	// Start system checks
	s.startChecks()

	// Set max file descriptors
	//syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{Cur: 1024, Max: 1024})

	// Set max open files
	//syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{Cur: 1024, Max: 1024})

	// Set max memory
	//syscall.Setrlimit(syscall.RLIMIT_DATA, &syscall.Rlimit{Cur: 1024, Max: 1024})

	// Set max processes
	//syscall.Setrlimit(syscall.RLIMIT_NPROC, &syscall.Rlimit{Cur: 1024, Max: 1024})

	maxConnections := 1000
	semaphore := make(chan struct{}, maxConnections)

	// Accept connections
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				s.logger.Printf("Accept error: %v", err)
				// Om vi får "too many open files", vänta lite
				if strings.Contains(err.Error(), "too many open files") {
					time.Sleep(100 * time.Millisecond)
				}
				continue
			}
		}

		// Vänta på plats i semaphore
		select {
		case semaphore <- struct{}{}:
			// Got slot, handle connection
			go func(c net.Conn) {
				defer func() {
					<-semaphore // Release slot when done
					if r := recover(); r != nil {
						s.logger.Printf("Panic in connection handler: %v", r)
					}
				}()
				s.handleConnection(c)
			}(conn)
		case <-ctx.Done():
			conn.Close()
			return nil
		default:
			// No slots available, reject
			if s.debug {
				s.logger.Printf("Connection limit reached, rejecting connection")
			}
			conn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n"))
			conn.Close()
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		if r := recover(); r != nil {
			s.logger.Printf("Recovered from panic in handleConnection: %v", r)
		}
	}()

	// Set deadline
	if err := conn.SetDeadline(time.Now().Add(s.timeout)); err != nil {
		if s.debug {
			s.logger.Printf("Failed to set deadline: %v", err)
		}
		return
	}

	// Check peer authorization
	remoteAddr := conn.RemoteAddr().String()
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		if s.debug {
			s.logger.Printf("Failed to parse remote address: %v", err)
		}
		return
	}

	if !s.security.IsPeerAllowed(host) {
		s.sendUnauthorized(conn)
		if s.debug {
			s.logger.Printf("Unauthorized request from %s", host)
		}
		return
	}

	// Read request
	req, err := s.readRequest(conn)
	if err != nil {
		if s.debug {
			s.logger.Printf("Failed to read request: %v", err)
		}
		s.sendBadRequest(conn)
		return
	}

	// Handle request
	s.handleRequest(conn, req, host)
}

func (s *Server) readRequest(conn net.Conn) (*http.Request, error) {
	// Set en kortare deadline för att läsa requesten
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (s *Server) handleRequest(conn net.Conn, req *http.Request, peerHost string) {
	start := time.Now().UnixNano()

	s.broker.mu.Lock()
	s.broker.requestCount++
	if s.broker.minuteRequestCountTimestamp == 0 || time.Now().Unix()-s.broker.minuteRequestCountTimestamp > 60 {
		s.broker.minuteRequestCountTimestamp = time.Now().Unix()
		s.broker.minuteRequestCount = 0
	}
	s.broker.minuteRequestCount++
	s.broker.mu.Unlock()

	// Parse form data
	var rawParams url.Values
	if req.Method == "GET" {
		rawParams = req.URL.Query()
	} else {
		//body, err := io.ReadAll(req.Body)
		body, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			s.sendBadRequest(conn)
			return
		}

		rawParams, err = url.ParseQuery(string(body))
		if err != nil {
			if s.debug {
				s.logger.Printf("Failed to parse query: %v", err)
			}
			s.sendBadRequest(conn)
			return
		}
	}

	// Decode all parameters (ROT13+Base64)
	params := decodeParams(rawParams)

	// Route to handler
	path := strings.Trim(req.URL.Path, "/")

	switch path {
	case "":
		// Serve web admin (no auth required - HTML handles it)
		s.ServeWebAdmin(conn)
	case "PICKUP":
		s.handlePickup(conn, params)
	case "POST":
		s.handlePost(conn, params, peerHost)
	case "SUBSCRIBE":
		s.handleSubscribe(conn, params)
	case "PUTVAL":
		s.handlePutVal(conn, params)
	case "GETVAL":
		s.handleGetVal(conn, params)
	case "GETVALSBYREGEX":
		s.handleGetValsByRegex(conn, params)
	case "VERSION":
		s.handleVersion(conn, "running")
	case "FILEVERSION":
		s.handleVersion(conn, "file")
	case "STATUS":
		s.handleStatus(conn, params)
	case "STATS":
		s.handleStats(conn, params)
	case "CLIENTS":
		s.handleClients(conn, params)
	case "POSTERS":
		s.handlePosters(conn, params)
	case "LOG":
		s.handleLog(conn, params)
	case "TOPICS":
		s.handleTopics(conn, params)
	default:
		s.sendNotFound(conn)
	}
	end := time.Now().UnixNano()
	elapsed := float64(end-start) / 1e6 // in milliseconds
	s.broker.serveTime += elapsed
}

func (s *Server) handlePickup(conn net.Conn, params map[string]string) {
	client := params["client"]
	if client == "" {
		if s.debug {
			s.logger.Printf("PICKUP request missing client parameter")
		}
		s.sendNotFound(conn)
		return
	}

	messages, err := s.broker.Pickup(client)
	if err != nil {
		s.sendError(conn, err)
		return
	}

	s.sendJSON(conn, messages)
}

func (s *Server) handlePost(conn net.Conn, params map[string]string, peerHost string) {
	topic := params["topic"]
	message := params["message"]
	from := params["from"]

	if topic == "" || message == "" {
		s.sendNotFound(conn)
		return
	}

	updatedTime := time.Now().Unix()
	if t := params["updated_time"]; t != "" {
		if parsed, err := strconv.ParseInt(t, 10, 64); err == nil {
			updatedTime = parsed
		}
	}

	err := s.broker.Publish(topic, message, from, peerHost, updatedTime)
	if err != nil {
		s.sendError(conn, err)
		return
	}

	s.sendOK(conn)
}

func (s *Server) handleSubscribe(conn net.Conn, params map[string]string) {
	topic := params["topic"]
	client := params["client"]

	if topic == "" || client == "" {
		s.sendNotFound(conn)
		return
	}

	err := s.broker.Subscribe(topic, client)
	if err != nil {
		s.sendError(conn, err)
		return
	}

	s.sendOK(conn)
}

func (s *Server) handlePutVal(conn net.Conn, params map[string]string) {
	valname := params["valname"]
	val := params["val"]
	message := params["message"]
	from := params["from"]

	if valname == "" || (val == "" && message == "") {
		s.sendNotFound(conn)
		return
	}

	updatedTime := time.Now().Unix()
	if t := params["updated_time"]; t != "" {
		if parsed, err := strconv.ParseInt(t, 10, 64); err == nil {
			updatedTime = parsed
		}
	}

	err := s.broker.PutValue(valname, val, message, from, updatedTime)
	if err != nil {
		s.sendError(conn, err)
		return
	}

	s.sendOK(conn)
}

func (s *Server) handleGetVal(conn net.Conn, params map[string]string) {
	topic := params["topic"]
	if topic == "" {
		s.sendNotFound(conn)
		return
	}

	value, err := s.broker.GetValue(topic)
	if err != nil {
		s.sendNotFound(conn)
		return
	}

	s.sendJSON(conn, value)
}

func (s *Server) handleGetValsByRegex(conn net.Conn, params map[string]string) {
	pattern := params["topic"]
	if pattern == "" {
		s.sendNotFound(conn)
		return
	}

	values, err := s.broker.GetValuesByRegex(pattern)
	if err != nil {
		s.sendError(conn, err)
		return
	}

	s.sendJSON(conn, values)
}

func (s *Server) handleVersion(conn net.Conn, versionType string) {
	switch versionType {
	case "running":
		s.sendJSON(conn, s.version)
	case "file":
		fileversion, err := GetFileVersion()
		s.sendJSON(conn, fileversion)
		if err != nil {
			s.sendNotFound(conn)
			return
		}
	}
}

func (s *Server) handleStatus(conn net.Conn, params map[string]string) {
	pwd := params["pwd"]
	if !s.auth.CheckPassword(pwd) {
		s.sendUnauthorized(conn)
		return
	}

	html := s.buildStatusPage()
	s.sendHTML(conn, html)
}

func (s *Server) handleStats(conn net.Conn, params map[string]string) {
	pwd := params["pwd"]
	if !s.auth.CheckPassword(pwd) {
		s.sendUnauthorized(conn)
		return
	}

	stats := s.broker.GetStats()
	s.sendJSON(conn, stats)
}

func (s *Server) handleClients(conn net.Conn, params map[string]string) {
	pwd := params["pwd"]
	if !s.auth.CheckPassword(pwd) {
		s.sendUnauthorized(conn)
		return
	}

	clients := s.broker.GetClients()
	s.sendJSON(conn, clients)
}

func (s *Server) handlePosters(conn net.Conn, params map[string]string) {
	pwd := params["pwd"]
	if !s.auth.CheckPassword(pwd) {
		s.sendUnauthorized(conn)
		return
	}

	posters := s.broker.GetPosters()
	s.sendJSON(conn, posters)
}

func (s *Server) handleTopics(conn net.Conn, params map[string]string) {
	pwd := params["pwd"]
	if !s.auth.CheckPassword(pwd) {
		s.sendUnauthorized(conn)
		return
	}

	topics := s.broker.GetTopics()
	s.sendJSON(conn, topics)
}

func (s *Server) handleLog(conn net.Conn, params map[string]string) {
	pwd := params["pwd"]
	if !s.auth.CheckPassword(pwd) {
		s.sendUnauthorized(conn)
		return
	}

	s.GetRecentLogs(conn, 100) // Last 100 lines
}

func (s *Server) buildStatusPage() string {
	stats := s.broker.GetStats()

	html := `<html>
<head><title>Moustique Status</title></head>
<body>
<h1>Moustique Status</h1>
<p>Version: ` + s.version + `</p>
<p>Started: ` + stats["started"].(string) + `</p>
<h2>Statistics</h2>
<pre>` + formatJSON(stats) + `</pre>
</body>
</html>`

	return html
}

func (s *Server) startChecks() {
	restartMsg := &Message{
		UpdatedTime: time.Now().Unix(),
		Message:     "restart",
		Topic:       "/server/action/resubscribe",
	}

	s.broker.mu.Lock()
	s.broker.systemMessageQueue["/server/action/resubscribe"] = []*Message{restartMsg}
	s.broker.mu.Unlock()
}

// Response helpers

func (s *Server) sendOK(conn net.Conn) {
	fmt.Fprintf(conn, "HTTP/1.0 200 OK\r\n")
	fmt.Fprintf(conn, "Connection: close\r\n")
	//fmt.Fprintf(conn, "Connection: keep-alive\r\n")
	//fmt.Fprintf(conn, "Keep-Alive: timeout=15, max=100\r\n")
	fmt.Fprintf(conn, "Content-Length: 0\r\n")
	fmt.Fprintf(conn, "\r\n")
}

func (s *Server) sendJSON(conn net.Conn, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		s.sendError(conn, err)
		return
	}

	encoded := encodeROT13Base64(string(jsonData))

	fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\n")
	fmt.Fprintf(conn, "Connection: close\r\n")
	//fmt.Fprintf(conn, "Connection: keep-alive\r\n")
	fmt.Fprintf(conn, "Keep-Alive: timeout=15, max=500\r\n")
	fmt.Fprintf(conn, "Content-Type: text/plain; charset=utf-8\r\n")
	fmt.Fprintf(conn, "Content-Length: %d\r\n", len(encoded))
	fmt.Fprintf(conn, "\r\n")
	fmt.Fprintf(conn, "%s", encoded)
}

func (s *Server) sendHTML(conn net.Conn, html string) {
	fmt.Fprintf(conn, "HTTP/1.0 200 OK\r\n")
	fmt.Fprintf(conn, "Content-Type: text/html\r\n")
	fmt.Fprintf(conn, "\r\n")
	fmt.Fprintf(conn, "%s", html)
}

func (s *Server) sendNotFound(conn net.Conn) {
	fmt.Fprintf(conn, "HTTP/1.0 404 Not Found\r\n")
	fmt.Fprintf(conn, "\r\n")
	fmt.Fprintf(conn, "<html><body>404 Not Found</body></html>")
}

func (s *Server) sendBadRequest(conn net.Conn) {
	fmt.Fprintf(conn, "HTTP/1.1 400 Bad Request\r\n")
	fmt.Fprintf(conn, "Content-Type: text/plain\r\n")
	fmt.Fprintf(conn, "\r\n")
	fmt.Fprintf(conn, "Invalid request\n")
}

func (s *Server) sendUnauthorized(conn net.Conn) {
	fmt.Fprintf(conn, "HTTP/1.1 401 Unauthorized\r\n")
	fmt.Fprintf(conn, "Content-Type: text/plain\r\n")
	fmt.Fprintf(conn, "\r\n")
	fmt.Fprintf(conn, "Access denied.\n")
}

func (s *Server) sendError(conn net.Conn, err error) {
	fmt.Fprintf(conn, "HTTP/1.0 500 Internal Server Error\r\n")
	fmt.Fprintf(conn, "\r\n")
	fmt.Fprintf(conn, "Error: %v", err)
}

func formatJSON(data interface{}) string {
	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonData)
}
