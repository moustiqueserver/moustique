package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BrokerManager manages per-user broker instances
type BrokerManager struct {
	brokers       map[string]*Broker
	mu            sync.RWMutex
	logger        *log.Logger
	dataDir       string
	defaultBroker *Broker
	ctx           context.Context
	fail2banJail  string
	fail2banLevel string
}

// NewBrokerManager creates a new broker manager
func NewBrokerManager(logger *log.Logger, dataDir string, allowPublic bool, fail2banJail string, fail2banLevel string) *BrokerManager {
	bm := &BrokerManager{
		brokers:       make(map[string]*Broker),
		logger:        logger,
		dataDir:       dataDir,
		ctx:           nil, // Will be set when Start() is called
		fail2banJail:  fail2banJail,
		fail2banLevel: fail2banLevel,
	}

	// Note: Default broker creation is deferred until InitializeDefault() is called with context
	return bm
}

// InitializeDefault creates the default/public broker (called from Start with context)
func (bm *BrokerManager) InitializeDefault(ctx context.Context, allowPublic bool) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.ctx = ctx

	// Create default/public broker if allowed
	if allowPublic {
		defaultDataDir := filepath.Join(bm.dataDir, "public")
		if err := os.MkdirAll(defaultDataDir, 0755); err != nil {
			bm.logger.Printf("Warning: Could not create public data dir: %v", err)
		} else {
			db, err := NewDatabase(filepath.Join(defaultDataDir, "moustique.db"))
			if err != nil {
				bm.logger.Printf("Warning: Could not create public database: %v", err)
			} else {
				if err := db.LoadAll(); err != nil {
					bm.logger.Printf("Warning: Could not load public database: %v", err)
				}

				// Create user log file for public broker
				userLogPath := filepath.Join(defaultDataDir, "user.log")
				userLogFile, err := os.OpenFile(userLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
				if err != nil {
					bm.logger.Printf("Warning: Could not create public user log file: %v", err)
				}
				userLogger := log.New(userLogFile, "[public] ", log.LstdFlags)

				bm.defaultBroker = NewBroker(bm.logger, db, false, bm.fail2banJail, bm.fail2banLevel)
				bm.defaultBroker.SetUserLogger(userLogger, userLogPath)
				bm.defaultBroker.LogUser("Public broker initialized")

				// Publish resubscribe system message for clients to re-register
				bm.defaultBroker.PublishSystemMessage("/server/action/resubscribe", "resubscribe")

				// Start maintenance for public broker
				go bm.defaultBroker.StartMaintenance(ctx)
				bm.logger.Println("Created public/default broker for unauthenticated access")
			}
		}
	}

	return nil
}

// GetOrCreateBroker gets or creates a broker for a specific user
func (bm *BrokerManager) GetOrCreateBroker(username string) (*Broker, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if broker, exists := bm.brokers[username]; exists {
		return broker, nil
	}

	// Ensure context is set
	if bm.ctx == nil {
		return nil, fmt.Errorf("broker manager context not initialized")
	}

	// Create user data directory
	userDataDir := filepath.Join(bm.dataDir, "users", username)
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create user data directory: %w", err)
	}

	// Create database for this user
	dbPath := filepath.Join(userDataDir, "moustique.db")
	db, err := NewDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Load existing data
	if err := db.LoadAll(); err != nil {
		bm.logger.Printf("Warning: Could not load database for user %s: %v", username, err)
	}

	// Create user log file
	userLogPath := filepath.Join(userDataDir, "user.log")
	userLogFile, err := os.OpenFile(userLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		bm.logger.Printf("Warning: Could not create user log file for %s: %v", username, err)
	}

	userLogger := log.New(userLogFile, fmt.Sprintf("[%s] ", username), log.LstdFlags)

	// Create broker
	broker := NewBroker(bm.logger, db, false, bm.fail2banJail, bm.fail2banLevel)
	broker.SetUserLogger(userLogger, userLogPath)
	bm.brokers[username] = broker

	// Publish resubscribe system message for clients to re-register
	broker.PublishSystemMessage("/server/action/resubscribe", "resubscribe")

	// Start maintenance for this user's broker
	go broker.StartMaintenance(bm.ctx)

	bm.logger.Printf("Created broker instance for user: %s", username)
	broker.LogUser("Broker initialized")
	return broker, nil
}

// GetBroker gets an existing broker (returns nil if not found)
func (bm *BrokerManager) GetBroker(username string) *Broker {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.brokers[username]
}

// GetDefaultBroker returns the public/default broker
func (bm *BrokerManager) GetDefaultBroker() *Broker {
	return bm.defaultBroker
}

// GetAllUsers returns list of all active users
func (bm *BrokerManager) GetAllUsers() []string {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	users := make([]string, 0, len(bm.brokers))
	for username := range bm.brokers {
		users = append(users, username)
	}
	return users
}

// SaveAll saves all user databases
func (bm *BrokerManager) SaveAll() error {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	// Save default broker if exists
	if bm.defaultBroker != nil && bm.defaultBroker.db != nil {
		if err := bm.defaultBroker.db.SaveAll(); err != nil {
			return fmt.Errorf("failed to save public database: %w", err)
		}
	}

	// Save all user brokers
	for username, broker := range bm.brokers {
		if broker.db != nil {
			if err := broker.db.SaveAll(); err != nil {
				return fmt.Errorf("failed to save database for user %s: %w", username, err)
			}
		}
	}

	return nil
}

// UserAuth handles user authentication with persistence
type UserAuth struct {
	users    map[string]string // username -> password hash
	mu       sync.RWMutex
	filePath string
}

// UserData for JSON persistence
type UserData struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
}

// NewUserAuth creates a new user authentication handler
func NewUserAuth(dataDir string) (*UserAuth, error) {
	// Create users directory
	usersDir := filepath.Join(dataDir, "users")
	if err := os.MkdirAll(usersDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create users directory: %w", err)
	}

	filePath := filepath.Join(usersDir, "users.json")

	ua := &UserAuth{
		users:    make(map[string]string),
		filePath: filePath,
	}

	// Load existing users
	if err := ua.Load(); err != nil {
		// If file doesn't exist, that's okay
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load users: %w", err)
		}
	}

	return ua, nil
}

// Load loads users from disk
func (ua *UserAuth) Load() error {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	data, err := ioutil.ReadFile(ua.filePath)
	if err != nil {
		return err
	}

	var userData []UserData
	if err := json.Unmarshal(data, &userData); err != nil {
		return fmt.Errorf("failed to parse users file: %w", err)
	}

	ua.users = make(map[string]string)
	for _, user := range userData {
		ua.users[user.Username] = user.PasswordHash
	}

	return nil
}

// Save saves users to disk
func (ua *UserAuth) Save() error {
	ua.mu.RLock()
	defer ua.mu.RUnlock()

	userData := make([]UserData, 0, len(ua.users))
	for username, hash := range ua.users {
		userData = append(userData, UserData{
			Username:     username,
			PasswordHash: hash,
		})
	}

	data, err := json.MarshalIndent(userData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users: %w", err)
	}

	if err := ioutil.WriteFile(ua.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write users file: %w", err)
	}

	return nil
}

// AddUser adds or updates a user
func (ua *UserAuth) AddUser(username, password string) error {
	ua.mu.Lock()
	ua.users[username] = hashPassword(password)
	ua.mu.Unlock()

	return ua.Save()
}

// ValidateUser checks if username/password is valid
func (ua *UserAuth) ValidateUser(username, password string) bool {
	ua.mu.RLock()
	defer ua.mu.RUnlock()

	hash, exists := ua.users[username]
	if !exists {
		return false
	}
	return hash == hashPassword(password)
}

// hashPassword creates a SHA256 hash of the password
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// Server handles HTTP connections
type Server struct {
	port           int
	timeout        time.Duration
	logger         *log.Logger
	accessLogger   *RotatingLogger
	errorLogger    *RotatingLogger
	brokerManager  *BrokerManager
	userAuth       *UserAuth
	security       *SecurityChecker
	rateLimiter    *RateLimiter
	debug          bool
	version        string
	allowPublic    bool
	maxRequestSize int64
	maxTopicLength int
	maxMessageSize int64
}

// NewServer creates a new HTTP server
func NewServer(port int, timeout time.Duration, logger *log.Logger, dataDir string, debug bool, Version string, allowPublic bool, allowedPeers []string, maxRequestSize int64, maxTopicLength int, maxMessageSize int64, defaultRateLimit int, fail2banJail string, fail2banLevel string, logDir string) (*Server, error) {
	// Create data directory
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	userAuth, err := NewUserAuth(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user auth: %w", err)
	}

	// Create access logger
	accessLogger, err := NewRotatingLogger(logDir, "moustique_access.log")
	if err != nil {
		logger.Printf("Warning: Failed to create access logger: %v", err)
		accessLogger = nil // Continue without access logging
	}

	// Create error logger
	errorLogger, errLog := NewRotatingLogger(logDir, "moustique_error.log")
	if errLog != nil {
		logger.Printf("Warning: Failed to create error logger: %v", errLog)
		errorLogger = nil // Continue without error logging
	}

	return &Server{
		port:           port,
		timeout:        timeout,
		logger:         logger,
		accessLogger:   accessLogger,
		errorLogger:    errorLogger,
		brokerManager:  NewBrokerManager(logger, dataDir, allowPublic, fail2banJail, fail2banLevel),
		userAuth:       userAuth,
		security:       NewSecurityChecker(allowedPeers),
		rateLimiter:    NewRateLimiter(defaultRateLimit, dataDir),
		debug:          debug,
		version:        Version,
		allowPublic:    allowPublic,
		maxRequestSize: maxRequestSize,
		maxTopicLength: maxTopicLength,
		maxMessageSize: maxMessageSize,
	}, nil
}

// AddUser adds a user to the system
func (s *Server) AddUser(username, password string) error {
	if err := s.userAuth.AddUser(username, password); err != nil {
		return err
	}
	s.logger.Printf("User added: %s", username)
	return nil
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	// Initialize broker manager with context and create default broker if needed
	if err := s.brokerManager.InitializeDefault(ctx, s.allowPublic); err != nil {
		return fmt.Errorf("failed to initialize broker manager: %w", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer listener.Close()

	s.logger.Printf("Starting Moustique Multi-Tenant Server on port %d", s.port)
	if s.allowPublic {
		s.logger.Printf("Public/unauthenticated access is ENABLED")
	}

	// Start rate limiter cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.rateLimiter.Cleanup()
			}
		}
	}()

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
				if strings.Contains(err.Error(), "too many open files") {
					time.Sleep(100 * time.Millisecond)
				}
				continue
			}
		}

		// Wait for slot in semaphore
		select {
		case semaphore <- struct{}{}:
			go func(c net.Conn) {
				defer func() {
					<-semaphore
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
		s.sendUnauthorized(conn, "Peer not allowed")
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
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Server) handleRequest(conn net.Conn, req *http.Request, peerHost string) {
	start := time.Now()
	statusCode := 200 // default to 200 OK
	var username string

	// Defer access logging
	defer func() {
		if s.accessLogger != nil {
			duration := float64(time.Since(start).Nanoseconds()) / 1e6 // milliseconds
			s.accessLogger.LogAccess(peerHost, req.Method, req.URL.Path, username, statusCode, duration)
		}
	}()

	// Parse form data
	var rawParams url.Values
	if req.Method == "GET" {
		rawParams = req.URL.Query()
	} else {
		// Limit request body size
		limitedReader := io.LimitReader(req.Body, s.maxRequestSize)
		body, err := ioutil.ReadAll(limitedReader)
		req.Body.Close()
		if err != nil {
			s.sendBadRequest(conn)
			return
		}

		// Check if body was truncated (exceeded limit)
		if int64(len(body)) == s.maxRequestSize {
			if s.debug {
				s.logger.Printf("Request body too large from %s", peerHost)
			}
			// Record oversized request
			defaultBroker := s.brokerManager.GetDefaultBroker()
			if defaultBroker != nil {
				defaultBroker.RecordInvalidRequest(peerHost, "oversized_request")
			}
			// Log error
			if s.errorLogger != nil {
				s.errorLogger.LogError(peerHost, "oversized_request", fmt.Sprintf("Request exceeded max size of %d bytes", s.maxRequestSize))
			}
			s.sendErrorWithStatus(conn, 413, "Request Entity Too Large")
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

	// Public endpoints (no auth required)
	switch path {
	case "":
		s.ServeWebAdmin(conn)
		return
	case "admin":
		s.ServeWebAdmin(conn)
		return
	case "VERSION":
		s.handleVersion(conn, "running")
		return
	case "FILEVERSION":
		s.handleVersion(conn, "file")
		return
	case "superadmin":
		s.ServeSuperAdmin(conn)
		return
	case "signup":
		s.ServeSignup(conn)
		return
	case "SIGNUP":
		s.handleSignup(conn, params)
		return
	case "favicon.ico", "favicon.svg":
		s.ServeFavicon(conn)
		return
	}

	// Admin endpoints (require admin password, not user auth)
	if strings.HasPrefix(path, "ADMIN/") {
		adminPwd := params["admin_password"]
		if !s.validateAdminPassword(adminPwd) {
			s.sendUnauthorized(conn, "Invalid admin password")
			return
		}

		switch path {
		case "ADMIN/LIST_USERS":
			s.handleAdminListUsers(conn, params)
		case "ADMIN/ADD_USER":
			s.handleAdminAddUser(conn, params)
		case "ADMIN/DELETE_USER":
			s.handleAdminDeleteUser(conn, params)
		case "ADMIN/SET_RATE_LIMIT":
			s.handleAdminSetRateLimit(conn, params)
		case "ADMIN/GET_RATE_LIMIT":
			s.handleAdminGetRateLimit(conn, params)
		case "ADMIN/SERVER_LOG":
			s.GetRecentLogs(conn, 100)
		default:
			s.sendNotFound(conn)
		}
		return
	}

	// Determine which broker to use
	var broker *Broker
	var err error

	username = params["username"]  // Set in outer scope for access logging
	password := params["password"]

	if username == "" || password == "" {
		// No credentials provided - use default broker if allowed
		if !s.allowPublic {
			s.sendUnauthorized(conn, "Username and password required")
			return
		}
		broker = s.brokerManager.GetDefaultBroker()
		if broker == nil {
			s.sendError(conn, fmt.Errorf("public access not configured"))
			return
		}
		if s.debug {
			s.logger.Printf("Using public broker for unauthenticated request from %s:%s", peerHost, params["from"])
		}
	} else {
		// Credentials provided - validate and get user broker
		if !s.userAuth.ValidateUser(username, password) {
			// Record invalid credentials attempt
			defaultBroker := s.brokerManager.GetDefaultBroker()
			if defaultBroker != nil {
				defaultBroker.RecordInvalidRequest(peerHost, "invalid_credentials")
			}
			// Log error
			if s.errorLogger != nil {
				s.errorLogger.LogError(peerHost, "invalid_credentials", fmt.Sprintf("Failed login attempt for user: %s", username))
			}
			s.sendUnauthorized(conn, "Invalid credentials")
			return
		}

		// Check rate limit for authenticated users
		if !s.rateLimiter.AllowRequest(username) {
			limit := s.rateLimiter.GetUserLimit(username)
			if s.debug {
				s.logger.Printf("Rate limit exceeded for user %s (limit: %d req/min)", username, limit)
			}
			// Get or create broker to record rate limit violation
			broker, _ := s.brokerManager.GetOrCreateBroker(username)
			if broker != nil {
				broker.RecordInvalidRequest(peerHost, "rate_limit_exceeded")
			}
			// Log error
			if s.errorLogger != nil {
				s.errorLogger.LogError(peerHost, "rate_limit_exceeded", fmt.Sprintf("User %s exceeded rate limit of %d req/min", username, limit))
			}
			s.sendErrorWithStatus(conn, 429, "Too Many Requests")
			return
		}

		broker, err = s.brokerManager.GetOrCreateBroker(username)
		if err != nil {
			s.sendError(conn, fmt.Errorf("failed to get broker: %w", err))
			return
		}
	}

	// Update request count
	broker.mu.Lock()
	broker.requestCount++
	if broker.minuteRequestCountTimestamp == 0 || time.Now().Unix()-broker.minuteRequestCountTimestamp > 60 {
		broker.minuteRequestCountTimestamp = time.Now().Unix()
		broker.minuteRequestCount = 0
	}
	broker.minuteRequestCount++
	broker.mu.Unlock()

	// Route to specific handler
	switch path {
	case "PICKUP":
		s.handlePickup(conn, params, peerHost, broker)
	case "POST":
		s.handlePost(conn, params, peerHost, broker)
	case "SUBSCRIBE":
		s.handleSubscribe(conn, params, peerHost, broker)
	case "PUTVAL":
		s.handlePutVal(conn, params, peerHost, broker)
	case "GETVAL":
		s.handleGetVal(conn, params, broker)
	case "GETVALSBYREGEX":
		s.handleGetValsByRegex(conn, params, broker)
	case "STATUS":
		s.handleStatus(conn, params, broker)
	case "STATS":
		s.handleStats(conn, params, broker)
	case "CLIENTS":
		s.handleClients(conn, params, broker)
	case "POSTERS":
		s.handlePosters(conn, params, broker)
	case "LOG":
		s.handleLog(conn, params, broker)
	case "TOPICS":
		s.handleTopics(conn, params, broker)
	case "CROOKS":
		s.handleCrooks(conn, params, broker)
	default:
		// Only ban if it's not a browser resource request (favicon, css, js, etc)
		// Browser requests for resources should just get 404, not banned
		if !strings.Contains(path, ".") && path != "favicon.ico" {
			// Looks like an API endpoint that doesn't exist - record as crook and ban
			if s.debug {
				s.logger.Printf("Invalid API endpoint '%s' from %s - banning", path, peerHost)
			}
			broker.RecordInvalidRequest(peerHost, "invalid_endpoint")
			// Log error
			if s.errorLogger != nil {
				s.errorLogger.LogError(peerHost, "invalid_endpoint", fmt.Sprintf("Invalid API endpoint: %s", path))
			}
		}
		s.sendNotFound(conn)
	}

	// Track serve time for broker stats
	elapsed := float64(time.Since(start).Nanoseconds()) / 1e6
	broker.serveTime += elapsed
}

// Handler methods

func (s *Server) handlePickup(conn net.Conn, params map[string]string, peerHost string, broker *Broker) {
	client := params["client"]
	if client == "" {
		if s.debug {
			s.logger.Printf("PICKUP request missing client parameter")
		}
		s.sendNotFound(conn)
		return
	}

	messages, err := broker.Pickup(client, peerHost)
	if err != nil {
		s.sendError(conn, err)
		return
	}

	s.sendJSON(conn, messages)
}

// validateInput validates topic and message inputs
func (s *Server) validateInput(topic, message string) error {
	// Validate topic length
	if len(topic) > s.maxTopicLength {
		return fmt.Errorf("topic name too long (max %d characters)", s.maxTopicLength)
	}

	// Validate topic characters (alphanumeric, underscore, hyphen, dot, slash)
	for _, c := range topic {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			 (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.' || c == '/') {
			return fmt.Errorf("topic contains invalid characters")
		}
	}

	// Validate message size
	if int64(len(message)) > s.maxMessageSize {
		return fmt.Errorf("message too large (max %d bytes)", s.maxMessageSize)
	}

	return nil
}

// validateSubscribeTopic validates topic for SUBSCRIBE (allows wildcards)
func (s *Server) validateSubscribeTopic(topic string) error {
	// Validate topic length
	if len(topic) > s.maxTopicLength {
		return fmt.Errorf("topic name too long (max %d characters)", s.maxTopicLength)
	}

	// Validate topic characters (allow +, *, and # for wildcards)
	for _, c := range topic {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			 (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.' || c == '/' || c == '+' || c == '*' || c == '#') {
			return fmt.Errorf("topic contains invalid characters")
		}
	}

	return nil
}

func (s *Server) handlePost(conn net.Conn, params map[string]string, peerHost string, broker *Broker) {
	topic := params["topic"]
	message := params["message"]
	from := params["from"]

	if topic == "" || message == "" {
		s.sendNotFound(conn)
		return
	}

	// Validate inputs
	if err := s.validateInput(topic, message); err != nil {
		if s.debug {
			s.logger.Printf("Validation error from %s: %v", peerHost, err)
		}
		broker.RecordInvalidRequest(peerHost, "validation_error")
		// Log error
		if s.errorLogger != nil {
			s.errorLogger.LogError(peerHost, "validation_error", fmt.Sprintf("POST validation failed: %v", err))
		}
		s.sendBadRequest(conn)
		return
	}

	updatedTime := time.Now().Unix()
	if t := params["updated_time"]; t != "" {
		if parsed, err := strconv.ParseInt(t, 10, 64); err == nil {
			updatedTime = parsed
		}
	}

	err := broker.Publish(topic, message, from, peerHost, updatedTime)
	if err != nil {
		s.sendError(conn, err)
		return
	}

	s.sendOK(conn)
}

func (s *Server) handleSubscribe(conn net.Conn, params map[string]string, peerHost string, broker *Broker) {
	topic := params["topic"]
	client := params["client"]

	if topic == "" || client == "" {
		s.sendNotFound(conn)
		return
	}

	// Validate topic for subscribe (allow wildcards + and *)
	if err := s.validateSubscribeTopic(topic); err != nil {
		if s.debug {
			s.logger.Printf("Validation error from %s: %v", peerHost, err)
		}
		broker.RecordInvalidRequest(peerHost, "validation_error")
		// Log error
		if s.errorLogger != nil {
			s.errorLogger.LogError(peerHost, "validation_error", fmt.Sprintf("SUBSCRIBE validation failed: %v", err))
		}
		s.sendBadRequest(conn)
		return
	}

	err := broker.Subscribe(topic, client, peerHost)
	if err != nil {
		s.sendError(conn, err)
		return
	}

	s.sendOK(conn)
}

func (s *Server) handlePutVal(conn net.Conn, params map[string]string, peerHost string, broker *Broker) {
	valname := params["valname"]
	val := params["val"]
	message := params["message"]
	from := params["from"]

	if valname == "" || (val == "" && message == "") {
		s.sendNotFound(conn)
		return
	}

	// Validate valname as topic and message
	if err := s.validateInput(valname, message); err != nil {
		if s.debug {
			s.logger.Printf("Validation error: %v", err)
		}
		broker.RecordInvalidRequest(peerHost, "validation_error")
		// Log error
		if s.errorLogger != nil {
			s.errorLogger.LogError(peerHost, "validation_error", fmt.Sprintf("PUT validation failed: %v", err))
		}
		s.sendBadRequest(conn)
		return
	}

	updatedTime := time.Now().Unix()
	if t := params["updated_time"]; t != "" {
		if parsed, err := strconv.ParseInt(t, 10, 64); err == nil {
			updatedTime = parsed
		}
	}

	err := broker.PutValue(valname, val, message, from, updatedTime)
	if err != nil {
		s.sendError(conn, err)
		return
	}

	s.sendOK(conn)
}

func (s *Server) handleGetVal(conn net.Conn, params map[string]string, broker *Broker) {
	topic := params["topic"]
	if topic == "" {
		s.sendNotFound(conn)
		return
	}

	value, err := broker.GetValue(topic)
	if err != nil {
		s.sendNotFound(conn)
		return
	}

	s.sendJSON(conn, value)
}

func (s *Server) handleGetValsByRegex(conn net.Conn, params map[string]string, broker *Broker) {
	pattern := params["topic"]
	if pattern == "" {
		s.sendNotFound(conn)
		return
	}

	values, err := broker.GetValuesByRegex(pattern)
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
		if err != nil {
			s.sendNotFound(conn)
			return
		}
		s.sendJSON(conn, fileversion)
	}
}

func (s *Server) handleSignup(conn net.Conn, params map[string]string) {
	// Get username and password from params (already ROT13+Base64 encoded)
	username := params["username"]
	password := params["password"]

	if username == "" || password == "" {
		s.sendError(conn, fmt.Errorf("username and password required"))
		return
	}

	// Decode username and password
	decodedUsername := decodeROT13Base64(username)
	decodedPassword := decodeROT13Base64(password)

	// Validate username
	if len(decodedUsername) < 3 || len(decodedUsername) > 32 {
		s.sendError(conn, fmt.Errorf("username must be 3-32 characters"))
		return
	}

	// Validate password
	if len(decodedPassword) < 8 {
		s.sendError(conn, fmt.Errorf("password must be at least 8 characters"))
		return
	}

	// Add user
	if err := s.AddUser(decodedUsername, decodedPassword); err != nil {
		s.sendError(conn, fmt.Errorf("failed to create user: %v", err))
		return
	}

	s.logger.Printf("New user registered: %s", decodedUsername)

	// Send success response
	s.sendJSON(conn, map[string]string{
		"status":  "success",
		"message": "Account created successfully",
	})
}

func (s *Server) handleStatus(conn net.Conn, params map[string]string, broker *Broker) {
	html := s.buildStatusPage(broker)
	s.sendHTML(conn, html)
}

func (s *Server) handleStats(conn net.Conn, params map[string]string, broker *Broker) {
	stats := broker.GetStats()
	s.sendJSON(conn, stats)
}

func (s *Server) handleClients(conn net.Conn, params map[string]string, broker *Broker) {
	clients := broker.GetClients()
	s.sendJSON(conn, clients)
}

func (s *Server) handlePosters(conn net.Conn, params map[string]string, broker *Broker) {
	posters := broker.GetPosters()
	s.sendJSON(conn, posters)
}

func (s *Server) handleTopics(conn net.Conn, params map[string]string, broker *Broker) {
	topics := broker.GetTopics()
	s.sendJSON(conn, topics)
}

func (s *Server) handleCrooks(conn net.Conn, params map[string]string, broker *Broker) {
	crooks := broker.GetCrooks()
	s.sendJSON(conn, crooks)
}

func (s *Server) handleLog(conn net.Conn, params map[string]string, broker *Broker) {
	s.GetUserLogs(conn, broker, 100)
}

func (s *Server) buildStatusPage(broker *Broker) string {
	stats := broker.GetStats()

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

// Admin handler methods

func (s *Server) validateAdminPassword(password string) bool {
	// TODO: Store admin password securely in config or environment variable
	adminPasswordHash := hashPassword("admin123") // Change this in production!
	return hashPassword(password) == adminPasswordHash
}

func (s *Server) handleAdminListUsers(conn net.Conn, params map[string]string) {
	// Get all users with their stats
	type UserInfo struct {
		Username      string `json:"username"`
		Requests      int64  `json:"requests"`
		Messages      int64  `json:"messages"`
		Topics        int    `json:"topics"`
		Clients       int    `json:"clients"`
		RateLimit     int    `json:"rate_limit"`      // requests per minute, 0 = unlimited
		CurrentReqMin int    `json:"current_req_min"` // current requests in last minute
	}

	users := []UserInfo{}
	var totalRequests int64
	var totalMessages int64
	var totalRequestsLastMinute int64
	var totalMessagesLastMinute int64
	activeBrokers := 0

	// Include public broker if it exists
	if s.brokerManager.defaultBroker != nil {
		broker := s.brokerManager.defaultBroker
		broker.mu.RLock()

		publicInfo := UserInfo{
			Username:      "public",
			Requests:      broker.requestCount,
			Messages:      broker.messagesProcessed,
			Topics:        len(broker.subscriptions),
			Clients:       len(broker.clients),
			RateLimit:     0, // public broker has no rate limit
			CurrentReqMin: 0,
		}

		// Aggregate totals
		totalRequests += broker.requestCount
		totalMessages += broker.messagesProcessed
		totalRequestsLastMinute += broker.minuteRequestCount
		totalMessagesLastMinute += broker.minuteMessageCount

		if broker.requestCount > 0 {
			activeBrokers++
		}

		broker.mu.RUnlock()
		users = append(users, publicInfo)
	}

	// Get list of users from UserAuth
	s.userAuth.mu.RLock()
	usernames := make([]string, 0, len(s.userAuth.users))
	for username := range s.userAuth.users {
		usernames = append(usernames, username)
	}
	s.userAuth.mu.RUnlock()

	// Get stats for each user
	for _, username := range usernames {
		broker := s.brokerManager.GetBroker(username)

		userInfo := UserInfo{
			Username:      username,
			RateLimit:     s.rateLimiter.GetUserLimit(username),
			CurrentReqMin: s.rateLimiter.GetUserRequestCount(username),
		}

		if broker != nil {
			broker.mu.RLock()
			userInfo.Requests = broker.requestCount
			userInfo.Messages = broker.messagesProcessed
			userInfo.Topics = len(broker.subscriptions)
			userInfo.Clients = len(broker.clients)

			// Aggregate totals
			totalRequests += broker.requestCount
			totalMessages += broker.messagesProcessed
			totalRequestsLastMinute += broker.minuteRequestCount
			totalMessagesLastMinute += broker.minuteMessageCount

			if broker.requestCount > 0 {
				activeBrokers++
			}

			broker.mu.RUnlock()
		}

		users = append(users, userInfo)
	}

	// Calculate actual per-second rates for the last minute
	requestsPerSecond := float64(totalRequestsLastMinute) / 60.0
	messagesPerSecond := float64(totalMessagesLastMinute) / 60.0

	response := map[string]interface{}{
		"users":               users,
		"total":               len(users),
		"total_requests":      totalRequests,
		"total_messages":      totalMessages,
		"requests_per_minute": totalRequestsLastMinute,
		"messages_per_minute": totalMessagesLastMinute,
		"requests_per_second": requestsPerSecond,
		"messages_per_second": messagesPerSecond,
		"active_brokers":      activeBrokers,
	}

	s.sendJSON(conn, response)
}

func (s *Server) handleAdminAddUser(conn net.Conn, params map[string]string) {
	username := params["username"]
	password := params["password"]

	if username == "" || password == "" {
		s.sendBadRequest(conn)
		return
	}

	// Check if user already exists
	s.userAuth.mu.RLock()
	_, exists := s.userAuth.users[username]
	s.userAuth.mu.RUnlock()

	if exists {
		s.sendJSON(conn, map[string]string{
			"status":  "error",
			"message": fmt.Sprintf("User '%s' already exists", username),
		})
		return
	}

	// Add user
	if err := s.AddUser(username, password); err != nil {
		s.sendError(conn, err)
		return
	}

	s.sendJSON(conn, map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("User '%s' created", username),
	})
}

func (s *Server) handleAdminDeleteUser(conn net.Conn, params map[string]string) {
	username := params["username"]
	if username == "" {
		s.sendBadRequest(conn)
		return
	}

	// Remove user from UserAuth
	s.userAuth.mu.Lock()
	delete(s.userAuth.users, username)
	s.userAuth.mu.Unlock()

	// Save updated users
	if err := s.userAuth.Save(); err != nil {
		s.logger.Printf("Failed to save users after deletion: %v", err)
	}

	s.sendJSON(conn, map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("User '%s' deleted", username),
	})
}

func (s *Server) handleAdminSetRateLimit(conn net.Conn, params map[string]string) {
	username := params["username"]
	limitStr := params["limit"]

	if username == "" || limitStr == "" {
		s.sendBadRequest(conn)
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		s.sendJSON(conn, map[string]string{
			"status":  "error",
			"message": "Invalid limit value (must be >= 0, 0 = unlimited)",
		})
		return
	}

	s.rateLimiter.SetUserLimit(username, limit)

	msg := fmt.Sprintf("Rate limit set to %d req/min for user '%s'", limit, username)
	if limit == 0 {
		msg = fmt.Sprintf("Rate limit removed (unlimited) for user '%s'", username)
	}

	s.sendJSON(conn, map[string]interface{}{
		"status":  "success",
		"message": msg,
		"username": username,
		"limit": limit,
	})
}

func (s *Server) handleAdminGetRateLimit(conn net.Conn, params map[string]string) {
	username := params["username"]

	if username == "" {
		s.sendBadRequest(conn)
		return
	}

	limit := s.rateLimiter.GetUserLimit(username)
	count := s.rateLimiter.GetUserRequestCount(username)

	s.sendJSON(conn, map[string]interface{}{
		"status":   "success",
		"username": username,
		"limit":    limit,
		"current_count": count,
	})
}

// Response helpers

func (s *Server) sendOK(conn net.Conn) {
	fmt.Fprintf(conn, "HTTP/1.0 200 OK\r\n")
	fmt.Fprintf(conn, "Connection: close\r\n")
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

func (s *Server) sendSVG(conn net.Conn, svg string) {
	fmt.Fprintf(conn, "HTTP/1.0 200 OK\r\n")
	fmt.Fprintf(conn, "Content-Type: image/svg+xml\r\n")
	fmt.Fprintf(conn, "Content-Length: %d\r\n", len(svg))
	fmt.Fprintf(conn, "\r\n")
	fmt.Fprintf(conn, "%s", svg)
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

func (s *Server) sendUnauthorized(conn net.Conn, message string) {
	fmt.Fprintf(conn, "HTTP/1.1 401 Unauthorized\r\n")
	fmt.Fprintf(conn, "Content-Type: text/plain\r\n")
	fmt.Fprintf(conn, "\r\n")
	fmt.Fprintf(conn, "Access denied: %s\n", message)
}

func (s *Server) sendError(conn net.Conn, err error) {
	fmt.Fprintf(conn, "HTTP/1.0 500 Internal Server Error\r\n")
	fmt.Fprintf(conn, "\r\n")
	fmt.Fprintf(conn, "Error: %v", err)
}

func (s *Server) sendErrorWithStatus(conn net.Conn, statusCode int, message string) {
	statusText := "Error"
	switch statusCode {
	case 413:
		statusText = "Request Entity Too Large"
	case 429:
		statusText = "Too Many Requests"
	}
	fmt.Fprintf(conn, "HTTP/1.0 %d %s\r\n", statusCode, statusText)
	fmt.Fprintf(conn, "Connection: close\r\n")
	fmt.Fprintf(conn, "\r\n")
	fmt.Fprintf(conn, "%s", message)
}

func formatJSON(data interface{}) string {
	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonData)
}
