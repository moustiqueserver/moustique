package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Message represents an MQTT-style message
type Message struct {
	From                string          `json:"from"`
	Topic               string          `json:"topic"`
	Message             string          `json:"message"`
	UpdatedTime         int64           `json:"updated_time"`
	UpdatedNiceDatetime string          `json:"updated_nicedatetime"`
	Subscribers         map[string]bool `json:"subscribers"`
	IP                  string          `json:"ip"`
}

// Client represents a connected subscriber
type Client struct {
	Name                     string `json:"Name"`
	FirstSeen                int64  `json:"FirstSeen"`
	FirstSeenNiceDatetime    string `json:"FirstSeenNiceDatetime"`
	LatestPickup             int64  `json:"LatestPickup"`
	LatestPickupNiceDatetime string `json:"LatestPickupNiceDatetime"`
	LatestSystemPickup       int64  `json:"LatestSystemPickup"`
	RequestCounter           int    `json:"RequestCounter"`
	IP                       string `json:"IP"`
}

// Provider tracks message posters
type Provider struct {
	Name                     string              `json:"Name"`
	LatestPostsByTopic       map[string]*Message `json:"latest_posts_by_topic"`
	LatestPost               *Message            `json:"latest_post"`
	IP                       string              `json:"IP"`
	FirstSeen                int64               `json:"FirstSeen"`
	FirstSeenNiceDatetime    string              `json:"FirstSeenNiceDatetime"`
	LatestPostTime           int64               `json:"LatestPostTime"`
	LatestPostNiceDatetime   string              `json:"LatestPostNiceDatetime"`
	MessageCount             int                 `json:"MessageCount"`
}

// Broker manages message routing and subscriptions
type Broker struct {
	mu                          sync.RWMutex
	logger                      *log.Logger
	userLogger                  *log.Logger
	userLogPath                 string
	db                          *Database
	debug                       bool
	messageQueue                map[string]map[string][]*Message
	systemMessageQueue          map[string][]*Message
	subscriptions               map[string][]string
	clients                     map[string]*Client
	providers                   map[string]*Provider
	topicExplosionCache         map[string][]string
	messageCount                int64
	minuteMessageCount          int64
	pickupCount                 int64
	getvalCount                 int64
	requestCount                int64
	minuteRequestCount          int64
	serveTime                   float64
	startedTime                 int64
	messageQueueTimeout         time.Duration
	posterStatsTimeout          time.Duration
	minuteRequestCountTimestamp int64
	minuteMessageCountTimestamp int64
	minutePickupCount           int64
	minuteGetvalCount           int64
	minutePickupCountTimestamp  int64
	minuteGetvalCountTimestamp  int64
	messagesProcessed           int64
}

// NewBroker creates a new message broker
func NewBroker(logger *log.Logger, db *Database, debug bool) *Broker {
	fmt.Printf("Creating new Broker instance\n")
	return &Broker{
		logger:              logger,
		userLogger:          nil,
		userLogPath:         "",
		db:                  db,
		debug:               debug,
		messageQueue:        make(map[string]map[string][]*Message),
		systemMessageQueue:  make(map[string][]*Message),
		subscriptions:       make(map[string][]string),
		clients:             make(map[string]*Client),
		providers:           make(map[string]*Provider),
		topicExplosionCache: make(map[string][]string),
		messageQueueTimeout: 5 * time.Minute,
		posterStatsTimeout:  1 * time.Hour,
		startedTime:         time.Now().Unix(),
	}
}

// SetUserLogger sets up user-specific logging
func (b *Broker) SetUserLogger(userLogger *log.Logger, logPath string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.userLogger = userLogger
	b.userLogPath = logPath
}

// LogUser logs to the user-specific log
func (b *Broker) LogUser(format string, v ...interface{}) {
	if b.userLogger != nil {
		b.userLogger.Printf(format, v...)
	}
}

// GetUserLogPath returns the path to the user log file
func (b *Broker) GetUserLogPath() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.userLogPath
}

// Subscribe adds a client subscription to a topic
func (b *Broker) Subscribe(topic, clientName, ip string) error {
	if clientName == "" {
		return fmt.Errorf("client name cannot be empty")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now().Unix()

	if _, exists := b.clients[clientName]; !exists {
		b.clients[clientName] = &Client{
			Name:                     clientName,
			FirstSeen:                now,
			FirstSeenNiceDatetime:    formatNiceDateTime(now),
			LatestPickup:             now,
			LatestPickupNiceDatetime: formatNiceDateTime(now),
			LatestSystemPickup:       now,
			RequestCounter:           0,
			IP:                       ip,
		}
		if b.debug {
			b.logger.Printf("New client: %s from IP: %s", clientName, ip)
		}
		b.LogUser("New client: %s from IP: %s", clientName, ip)
	}

	if !contains(b.subscriptions[topic], clientName) {
		b.subscriptions[topic] = append(b.subscriptions[topic], clientName)
		b.LogUser("Client %s subscribed to topic: %s", clientName, topic)
	}

	if b.messageQueue[clientName] == nil {
		b.messageQueue[clientName] = make(map[string][]*Message)
	}

	client := b.clients[clientName]
	client.LatestPickup = now
	client.LatestPickupNiceDatetime = formatNiceDateTime(now)
	client.RequestCounter++

	if b.debug {
		b.logger.Printf("Added subscription %s for %s", topic, clientName)
	}

	return nil
}

// Publish publishes a message to a topic
func (b *Broker) Publish(topic, message, from, ip string, updatedTime int64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.messagesProcessed++
	b.messageCount++
	if b.messageCount%1000 == 0 {
		b.logger.Printf("%s Processed %d messages", formatNiceDateTime(time.Now().Unix()), b.messageCount)
	}
	if b.minuteMessageCountTimestamp == 0 || time.Now().Unix()-b.minuteMessageCountTimestamp > 60 {
		b.minuteMessageCountTimestamp = time.Now().Unix()
		b.minuteMessageCount = 0
	}
	b.minuteMessageCount++

	b.LogUser("Published message to %s from %s (IP: %s)", topic, from, ip)

	if from == "" {
		from = "UNKNOWN"
	}

	msg := &Message{
		From:                from,
		Topic:               topic,
		Message:             message,
		UpdatedTime:         updatedTime,
		UpdatedNiceDatetime: formatNiceDateTime(updatedTime),
		Subscribers:         make(map[string]bool),
		IP:                  ip,
	}

	provider, exists := b.providers[from]
	if !exists {
		provider = &Provider{
			Name:                   from,
			LatestPostsByTopic:     make(map[string]*Message),
			FirstSeen:              updatedTime,
			FirstSeenNiceDatetime:  formatNiceDateTime(updatedTime),
			MessageCount:           0,
		}
		b.providers[from] = provider
	}

	provider.LatestPostsByTopic[topic] = msg
	provider.LatestPost = msg
	provider.IP = ip
	provider.LatestPostTime = updatedTime
	provider.LatestPostNiceDatetime = formatNiceDateTime(updatedTime)
	provider.MessageCount++

	topics := b.explodeTopic(topic)
	topics = append(topics, "#")

	for _, wildcardTopic := range topics {

		if clients, ok := b.subscriptions[wildcardTopic]; ok {

			for _, clientName := range clients {
				if b.messageQueue[clientName] == nil {
					b.messageQueue[clientName] = make(map[string][]*Message)
				}

				b.messageQueue[clientName][wildcardTopic] = append(
					b.messageQueue[clientName][wildcardTopic], msg)
				msg.Subscribers[clientName] = true
			}
		}
	}

	if err := b.db.SaveValue(topic, msg); err != nil {
		return fmt.Errorf("failed to save value: %w", err)
	}

	return nil
}

// PublishSystemMessage publishes a system message that will be delivered to all clients
func (b *Broker) PublishSystemMessage(topic, message string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now().Unix()
	msg := &Message{
		From:                "SERVER",
		Topic:               topic,
		Message:             message,
		UpdatedTime:         now,
		UpdatedNiceDatetime: formatNiceDateTime(now),
		Subscribers:         make(map[string]bool),
		IP:                  "127.0.0.1",
	}

	b.systemMessageQueue[topic] = append(b.systemMessageQueue[topic], msg)

	if b.debug {
		b.logger.Printf("Published system message to topic: %s", topic)
	}
	b.LogUser("Published system message to topic: %s", topic)
}

// Pickup retrieves messages for a client
func (b *Broker) Pickup(clientName, ip string) (map[string][]*Message, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.pickupCount++
	if b.minutePickupCountTimestamp == 0 || time.Now().Unix()-b.minutePickupCountTimestamp > 60 {
		b.minutePickupCountTimestamp = time.Now().Unix()
		b.minutePickupCount = 0
	}
	b.minutePickupCount++

	normalMessages := b.messageQueue[clientName]

	b.messageQueue[clientName] = make(map[string][]*Message)

	systemMessages := b.getSystemMessages(clientName)

	result := make(map[string][]*Message)
	for topic, msgs := range normalMessages {
		msgsCopy := make([]*Message, len(msgs))
		copy(msgsCopy, msgs)
		result[topic] = msgsCopy
	}
	for topic, msgs := range systemMessages {
		msgsCopy := make([]*Message, len(msgs))
		copy(msgsCopy, msgs)
		//result[topic] = append(result[topic], msgsCopy...)
		result[topic] = msgsCopy
	}

	if client, exists := b.clients[clientName]; exists {
		now := time.Now().Unix()
		client.LatestPickup = now
		client.LatestPickupNiceDatetime = formatNiceDateTime(now)
		client.LatestSystemPickup = now
		client.RequestCounter++
		client.IP = ip  // Update IP in case it changed
	} else {
		if b.debug {
			b.logger.Printf("Pickup request for unknown client: %s", clientName)
		}
	}

	return result, nil
}

// GetValue retrieves a stored value by key
func (b *Broker) GetValue(key string) (*Message, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	b.getvalCount++
	if b.minuteGetvalCountTimestamp == 0 || time.Now().Unix()-b.minuteGetvalCountTimestamp > 60 {
		b.minuteGetvalCountTimestamp = time.Now().Unix()
		b.minuteGetvalCount = 0
	}
	b.minuteGetvalCount++

	value, err := b.db.GetValue(key)
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal([]byte(value), &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return &msg, nil
}

// GetValuesByRegex retrieves values matching a regex pattern
func (b *Broker) GetValuesByRegex(pattern string) (map[string]*Message, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}

	keys := b.db.GetKeysByRegex(re)
	result := make(map[string]*Message)

	for _, key := range keys {
		value, err := b.db.GetValue(key)
		if err != nil {
			continue
		}

		var msg Message
		if err := json.Unmarshal([]byte(value), &msg); err != nil {
			continue
		}

		result[key] = &msg
	}

	return result, nil
}

// PutValue stores a value
func (b *Broker) PutValue(valname, val, message, from string, updatedTime int64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if val == "" {
		val = message
	}

	msg := &Message{
		Message:             val,
		UpdatedTime:         updatedTime,
		UpdatedNiceDatetime: formatNiceDateTime(updatedTime),
		From:                from,
	}

	return b.db.SaveValue(valname, msg)
}

// GetStats returns broker statistics
func (b *Broker) GetStats() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	now := time.Now().Unix()
	secsRunning := now - b.startedTime
	if secsRunning == 0 {
		secsRunning = 1
	}
	if b.requestCount == 0 {
		b.requestCount = 1
	}
	if b.minuteRequestCountTimestamp == 0 {
		b.minuteRequestCountTimestamp = now
	}
	if b.minuteMessageCountTimestamp == 0 {
		b.minuteMessageCountTimestamp = now
	}
	if b.minutePickupCountTimestamp == 0 {
		b.minutePickupCountTimestamp = now
	}
	if b.minuteGetvalCountTimestamp == 0 {
		b.minuteGetvalCountTimestamp = now
	}

	numGoroutines := runtime.NumGoroutine()

	// Count stored values
	valuesCount := 0
	if b.db != nil {
		valuesCount = b.db.CountValues()
	}

	return map[string]interface{}{
		"started":              formatNiceDateTime(b.startedTime),
		"memory_usage":         "N/A",
		"subscription_count":   len(b.subscriptions),
		"goroutines":           numGoroutines,
		"average_request_time": b.serveTime / float64(b.requestCount),
		"values":               valuesCount,
		"clients": map[string]interface{}{
			"subscribers": len(b.messageQueue),
			"posters":     len(b.providers),
		},
		"requests": map[string]interface{}{
			"per_second":             float64(b.requestCount) / float64(secsRunning),
			"per_second_last_minute": float64(b.minuteRequestCount) / float64(time.Now().Unix()-b.minuteRequestCountTimestamp),
			"total":                  b.requestCount,
			"pickups": map[string]interface{}{
				"per_second":             float64(b.pickupCount) / float64(secsRunning),
				"per_second_last_minute": float64(b.minutePickupCount) / float64(time.Now().Unix()-b.minutePickupCountTimestamp),
				"total":                  b.pickupCount,
			},
			"processed": map[string]interface{}{
				"per_second":             float64(b.messageCount) / float64(secsRunning),
				"per_second_last_minute": float64(b.minuteMessageCount) / float64(time.Now().Unix()-b.minuteMessageCountTimestamp),
				"total":                  b.messageCount,
			},
			"getvals": map[string]interface{}{
				"per_second":             float64(b.getvalCount) / float64(secsRunning),
				"per_second_last_minute": float64(b.minuteGetvalCount) / float64(time.Now().Unix()-b.minuteGetvalCountTimestamp),
				"total":                  b.getvalCount,
			},
		},
	}
}

func (b *Broker) explodeTopic(topic string) []string {
	if cached, exists := b.topicExplosionCache[topic]; exists {
		return cached
	}

	var patterns []string
	sections := strings.Split(topic, "/")

	// Loopa från sista elementet ner till index 1
	for i := len(sections) - 1; i >= 1; i-- {
		// Huvudmönster: map { $_ eq $sections[$i] ? $_ : '+' } @sections[$i..$#sections]
		// Jämför varje element från i till slutet med sections[i] själv
		beforeI := sections[:i]
		fromI := sections[i:]
		targetValue := sections[i] // Detta är nyckeln!

		// Ersätt alla element i fromI som INTE är lika med targetValue till +
		mappedFromI := make([]string, len(fromI))
		for j, sec := range fromI {
			if sec == targetValue {
				mappedFromI[j] = sec
			} else {
				mappedFromI[j] = "+"
			}
		}

		patternParts := make([]string, 0, len(sections))
		patternParts = append(patternParts, beforeI...)
		patternParts = append(patternParts, mappedFromI...)
		pattern := strings.Join(patternParts, "/")
		patterns = append(patterns, pattern)

		// "Insprängt wildcard": sätt in + FÖRE position i
		// if($i>2 && $i <= $#sections)
		if i > 2 && i <= len(sections)-1 {
			beforeIMinus1 := sections[:i-1]
			fromI := sections[i:]

			insprangtParts := make([]string, 0, len(sections)+1)
			insprangtParts = append(insprangtParts, beforeIMinus1...)
			insprangtParts = append(insprangtParts, "+")
			insprangtParts = append(insprangtParts, fromI...)
			insprangt := strings.Join(insprangtParts, "/")
			patterns = append(patterns, insprangt)
		}
	}

	// Lägg INTE till original topic här - den ska vara först!
	// Perl pushar in i början, så vi måste prependa
	result := []string{}
	result = append(result, patterns...)

	b.topicExplosionCache[topic] = result
	return result
}

func (b *Broker) getSystemMessages(clientName string) map[string][]*Message {
	result := make(map[string][]*Message)

	client, exists := b.clients[clientName]
	if !exists {
		// Okänd klient - ge alla systemmeddelanden
		if b.debug {
			b.logger.Printf("getSystemMessages: unknown client %s, returning all system messages", clientName)
		}
		return b.systemMessageQueue
	}

	getThoseNewerThan := client.LatestSystemPickup
	client.LatestSystemPickup = time.Now().Unix()

	for topic, messages := range b.systemMessageQueue {
		var deliver []*Message
		for _, msg := range messages {
			if msg.UpdatedTime > getThoseNewerThan {
				deliver = append(deliver, msg)
			}
		}
		if len(deliver) > 0 {
			result[topic] = deliver
		}
	}

	return result
}

// StartMaintenance starts background maintenance tasks
func (b *Broker) StartMaintenance(ctx context.Context) {
	if b.debug {
		b.logger.Printf("StartMaintenance: Starting background maintenance tasks")
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	counter := 0
	for {
		select {
		case <-ctx.Done():
			if b.debug {
				b.logger.Printf("StartMaintenance: Context cancelled, stopping maintenance")
			}
			return
		case <-ticker.C:
			counter++
			if counter%4 == 0 {
				if b.debug {
					b.logger.Printf("Running maintenance cycle %d", counter)
				}
				b.kickInactiveClients()
				b.clearOldPosters()
			}
		}
	}
}

func (b *Broker) kickInactiveClients() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now().Unix()
	toKick := []string{} // Samla först, kicka sedan

	for clientName, client := range b.clients {
		if now-client.LatestPickup > int64(b.messageQueueTimeout.Seconds()) {
			toKick = append(toKick, clientName)
		}
	}

	// Kicka alla inaktiva klienter
	for _, clientName := range toKick {
		if b.debug {
			b.logger.Printf("Kicking %s due to inactivity, last seen: %s", clientName, b.clients[clientName].LatestPickupNiceDatetime)
		}

		// Ta bort från subscriptions
		for topic := range b.subscriptions {
			b.subscriptions[topic] = removeString(b.subscriptions[topic], clientName)
			if len(b.subscriptions[topic]) == 0 {
				delete(b.subscriptions, topic)
			}
		}

		delete(b.messageQueue, clientName)
		delete(b.clients, clientName)
	}

	if b.debug && len(toKick) > 0 {
		b.logger.Printf("Kicked %d inactive clients", len(toKick))
	}
}

func (b *Broker) clearOldPosters() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now().Unix()
	for posterName, provider := range b.providers {
		if provider.LatestPost != nil &&
			now-provider.LatestPost.UpdatedTime > int64(b.posterStatsTimeout.Seconds()) {
			delete(b.providers, posterName)
		}
	}
}

// GetClients returns list of client information
func (b *Broker) GetClients() []*Client {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Use messageQueue to find active clients (those with queues)
	// and return their Client info if available
	clients := make([]*Client, 0, len(b.messageQueue))
	for clientName := range b.messageQueue {
		if client, exists := b.clients[clientName]; exists {
			clients = append(clients, client)
		} else {
			// Fallback: create minimal client info if not in clients map
			clients = append(clients, &Client{
				Name: clientName,
			})
		}
	}
	return clients
}

// GetPosters returns list of poster information
func (b *Broker) GetPosters() []*Provider {
	b.mu.RLock()
	defer b.mu.RUnlock()

	posters := make([]*Provider, 0, len(b.providers))
	for _, provider := range b.providers {
		posters = append(posters, provider)
	}
	return posters
}

// GetTopics returns all topics
func (b *Broker) GetTopics() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.db.GetKeys()
}

func formatNiceDateTime(timestamp int64) string {
	return time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeString(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

func replaceAt(slice []string, index int, value string) []string {
	result := make([]string, len(slice))
	copy(result, slice)
	if index < len(result) {
		result[index] = value
	}
	return result
}
