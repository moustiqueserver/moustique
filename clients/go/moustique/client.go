package moustique

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

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
}

type message struct {
	Topic   string `json:"topic"`
	Message string `json:"message"`
	From    string `json:"from"`
}

// New creates a new Moustique client
// Usage: New(ip, port, clientName, username, password)
// username and password are optional - omit them to use public broker
func New(ip, port string, args ...string) *Client {
	clientName := "go-client"
	username := ""
	password := ""

	if len(args) > 0 && args[0] != "" {
		clientName = args[0]
	}
	if len(args) > 1 {
		username = args[1]
	}
	if len(args) > 2 {
		password = args[2]
	}

	clientName += "-" + uuid.New().String()[:8]

	return &Client{
		BaseURL:    fmt.Sprintf("http://%s:%s", ip, port),
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		ClientName: clientName,
		Username:   username,
		Password:   password,
		callbacks:  make(map[string][]func(topic, message, from string)),
	}
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

	c.mu.Lock()
	c.callbacks[topic] = append(c.callbacks[topic], callback)
	c.mu.Unlock()

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

	c.mu.Lock()
	defer c.mu.Unlock()

	for topic, msgs := range data {
		for _, msg := range msgs {
			callbacks := c.callbacks[topic]
			for _, cb := range callbacks {
				cb(msg.Topic, msg.Message, msg.From)
			}
		}
	}
	return nil
}

func (c *Client) GetClientName() string {
	return c.ClientName
}
