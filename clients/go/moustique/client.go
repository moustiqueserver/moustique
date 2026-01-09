package moustique

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	ClientName string
	Username   string
	Password   string

	mu        sync.Mutex
	callbacks map[string][]func(topic, message, from string)

	// MQTT support
	IP            string
	UseMQTT       bool
	MQTTPort      int
	mqttClient    mqtt.Client
	mqttConnected bool
}

type message struct {
	Topic   string `json:"topic"`
	Message string `json:"message"`
	From    string `json:"from"`
}

// New creates a new Moustique client
// Usage: New(ip, port, clientName, username, password, useMqtt, mqttPort, useTLS)
// All parameters after port are optional
// useMqtt should be "true" to enable MQTT, mqttPort defaults to "1883"
// useTLS should be "true" to use HTTPS instead of HTTP
func New(ip, port string, args ...string) *Client {
	clientName := "go-client"
	username := ""
	password := ""
	useMqtt := false
	mqttPort := 1883
	useTLS := false

	if len(args) > 0 && args[0] != "" {
		clientName = args[0]
	}
	if len(args) > 1 {
		username = args[1]
	}
	if len(args) > 2 {
		password = args[2]
	}
	if len(args) > 3 && args[3] == "true" {
		useMqtt = true
	}
	if len(args) > 4 && args[4] != "" {
		fmt.Sscanf(args[4], "%d", &mqttPort)
	}
	if len(args) > 5 && args[5] == "true" {
		useTLS = true
	}

	clientName += "-" + uuid.New().String()[:8]

	scheme := "http"
	if useTLS {
		scheme = "https"
	}

	client := &Client{
		BaseURL:    fmt.Sprintf("%s://%s:%s", scheme, ip, port),
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		ClientName: clientName,
		Username:   username,
		Password:   password,
		callbacks:  make(map[string][]func(topic, message, from string)),
		IP:         ip,
		UseMQTT:    useMqtt,
		MQTTPort:   mqttPort,
	}

	if useMqtt {
		client.initMQTT()
	}

	return client
}

func (c *Client) addAuth(payload url.Values) url.Values {
	if c.Username != "" && c.Password != "" {
		payload.Set("username", Enc(c.Username))
		payload.Set("password", Enc(c.Password))
	}
	return payload
}

func (c *Client) Publish(topic, message string) error {
	payload := c.addAuth(url.Values{
		"topic":                {Enc(topic)},
		"message":              {Enc(message)},
		"updated_time":         {Enc(fmt.Sprintf("%d", time.Now().Unix()))},
		"updated_nicedatetime": {Enc(NiceDateTime())},
		"from":                 {Enc(c.ClientName)},
	})

	resp, err := c.HTTPClient.PostForm(c.BaseURL+"/POST", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("publish failed: %d %s", resp.StatusCode, string(body))
	}
	fmt.Printf("Published to %s\n", topic)
	return nil
}

func (c *Client) PutVal(topic, value string) error {
	payload := c.addAuth(url.Values{
		"valname":              {Enc(topic)},
		"val":                  {Enc(value)},
		"updated_time":         {Enc(fmt.Sprintf("%d", time.Now().Unix()))},
		"updated_nicedatetime": {Enc(NiceDateTime())},
		"from":                 {Enc(c.ClientName)},
	})

	req, _ := http.NewRequest("PUT", c.BaseURL+"/PUTVAL", bytes.NewBufferString(payload.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != 308 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("putval failed: %d %s", resp.StatusCode, string(body))
	}
	fmt.Printf("PutVal %s = %s\n", topic, value)
	return nil
}

func (c *Client) Subscribe(topic string, callback func(topic, message, from string)) error {
	c.mu.Lock()
	c.callbacks[topic] = append(c.callbacks[topic], callback)
	c.mu.Unlock()

	if c.UseMQTT && c.mqttConnected {
		// MQTT subscription
		token := c.mqttClient.Subscribe(topic, 0, nil)
		token.Wait()
		if token.Error() != nil {
			fmt.Printf("MQTT subscribe failed, falling back to HTTP: %v\n", token.Error())
		} else {
			fmt.Printf("✓ Subscribed to %s via MQTT\n", topic)
			return nil
		}
	}

	// HTTP subscription
	payload := c.addAuth(url.Values{
		"topic":  {Enc(topic)},
		"client": {Enc(c.ClientName)},
	})

	resp, err := c.HTTPClient.PostForm(c.BaseURL+"/SUBSCRIBE", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("subscribe failed: %d %s", resp.StatusCode, string(body))
	}

	fmt.Printf("%s subscribed to %s\n", c.ClientName, topic)
	return nil
}

func (c *Client) Pickup() error {
	payload := c.addAuth(url.Values{
		"client": {Enc(c.ClientName)},
	})

	resp, err := c.HTTPClient.PostForm(c.BaseURL+"/PICKUP", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	decrypted := Dec(string(body))
	if decrypted == "" {
		return nil
	}

	// Parse JSON: map[string][]message
	var data map[string][]message
	if err := json.Unmarshal([]byte(decrypted), &data); err != nil {
		fmt.Println("Raw pickup data:", decrypted) // fallback
		return nil
	}

	// Handle system message: server restart
	if _, hasResubscribe := data["/server/action/resubscribe"]; hasResubscribe {
		fmt.Println("⚠️  Server restarted - re-subscribing to all topics...")
		c.Resubscribe()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Deliver regular messages to callbacks
	for topic, msgs := range data {
		// Skip system messages
		if topic == "/server/action/resubscribe" {
			continue
		}

		for _, msg := range msgs {
			callbacks := c.callbacks[topic]
			for _, cb := range callbacks {
				cb(msg.Topic, msg.Message, msg.From)
			}
		}
	}
	return nil
}

func (c *Client) Resubscribe() {
	c.mu.Lock()
	topics := make([]string, 0, len(c.callbacks))
	for topic := range c.callbacks {
		topics = append(topics, topic)
	}
	c.mu.Unlock()

	for _, topic := range topics {
		if c.UseMQTT && c.mqttConnected {
			// MQTT resubscribe
			token := c.mqttClient.Subscribe(topic, 0, nil)
			token.Wait()
			if token.Error() != nil {
				fmt.Printf("Failed to resubscribe to '%s' via MQTT: %v\n", topic, token.Error())
			} else {
				fmt.Printf("✓ Re-subscribed to %s via MQTT\n", topic)
			}
		} else {
			// HTTP resubscribe
			payload := c.addAuth(url.Values{
				"topic":  {Enc(topic)},
				"client": {Enc(c.ClientName)},
			})

			resp, err := c.HTTPClient.PostForm(c.BaseURL+"/SUBSCRIBE", payload)
			if err != nil {
				fmt.Printf("Failed to resubscribe to '%s': %v\n", topic, err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Printf("✓ Re-subscribed to %s\n", topic)
			} else {
				fmt.Printf("Failed to resubscribe to '%s': status %d\n", topic, resp.StatusCode)
			}
		}
	}
}

func (c *Client) GetClientName() string {
	return c.ClientName
}

// MQTT Support Methods

func (c *Client) initMQTT() {
	broker := fmt.Sprintf("tcp://%s:%d", c.IP, c.MQTTPort)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(c.ClientName)
	opts.SetCleanSession(true)

	if c.Username != "" && c.Password != "" {
		opts.SetUsername(c.Username)
		opts.SetPassword(c.Password)
	}

	opts.SetOnConnectHandler(func(client mqtt.Client) {
		c.mqttConnected = true
		fmt.Printf("✓ Connected to MQTT broker at %s\n", broker)

		// Resubscribe to all topics
		c.mu.Lock()
		topics := make([]string, 0, len(c.callbacks))
		for topic := range c.callbacks {
			topics = append(topics, topic)
		}
		c.mu.Unlock()

		for _, topic := range topics {
			token := client.Subscribe(topic, 0, nil)
			token.Wait()
			if token.Error() == nil {
				fmt.Printf("✓ Subscribed to %s via MQTT\n", topic)
			}
		}
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		c.mqttConnected = false
		fmt.Printf("⚠️  MQTT connection lost: %v\n", err)
	})

	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		payloadStr := string(msg.Payload())
		var msgTopic, msgText, msgFrom string

		// Try to parse as JSON first (for compatibility)
		// If it fails, treat as plaintext (standard MQTT)
		var msgObj message
		if err := json.Unmarshal(msg.Payload(), &msgObj); err == nil && msgObj.Message != "" {
			// JSON format
			msgTopic = msgObj.Topic
			if msgTopic == "" {
				msgTopic = msg.Topic()
			}
			msgText = msgObj.Message
			msgFrom = msgObj.From
		} else {
			// Standard MQTT: plaintext payload
			msgTopic = msg.Topic()
			msgText = payloadStr
			msgFrom = ""
		}

		// Find matching callbacks
		c.mu.Lock()
		defer c.mu.Unlock()

		for subscribedTopic, callbacks := range c.callbacks {
			if c.topicMatches(subscribedTopic, msgTopic) {
				for _, callback := range callbacks {
					callback(msgTopic, msgText, msgFrom)
				}
			}
		}
	})

	c.mqttClient = mqtt.NewClient(opts)
	token := c.mqttClient.Connect()
	token.Wait()

	if token.Error() != nil {
		fmt.Printf("MQTT connection failed: %v\n", token.Error())
		fmt.Println("Falling back to HTTP mode")
		c.UseMQTT = false
	}
}

func (c *Client) topicMatches(pattern, topic string) bool {
	// Simple MQTT wildcard matching (+ for single level, # for multi-level)
	patternParts := strings.Split(pattern, "/")
	topicParts := strings.Split(topic, "/")

	if len(patternParts) > len(topicParts) && patternParts[len(patternParts)-1] != "#" {
		return false
	}

	for i, patternPart := range patternParts {
		if patternPart == "#" {
			return true // Match everything after
		}
		if i >= len(topicParts) {
			return false
		}
		if patternPart == "+" {
			continue // Match single level
		}
		if patternPart != topicParts[i] {
			return false
		}
	}

	return len(patternParts) == len(topicParts) || patternParts[len(patternParts)-1] == "#"
}

func (c *Client) Disconnect() {
	if c.mqttClient != nil && c.mqttConnected {
		c.mqttClient.Disconnect(250)
		c.mqttConnected = false
		fmt.Println("MQTT client disconnected")
	}
}
