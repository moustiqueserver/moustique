package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
)

const version = "1.0.0"

// Encoding functions (ROT13 + Base64)
func rot13(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			result[i] = 'a' + (c-'a'+13)%26
		} else if c >= 'A' && c <= 'Z' {
			result[i] = 'A' + (c-'A'+13)%26
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func encode(plaintext string) string {
	// Must match server encoding: ROT13 first, then Base64
	rot13Text := rot13(plaintext)
	return base64.StdEncoding.EncodeToString([]byte(rot13Text))
}

func decode(encoded string) string {
	// Reverse of encode: Base64 decode first, then ROT13
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return encoded
	}
	return rot13(string(decoded))
}

func getNiceDateTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// Client represents a Moustique client
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	ClientName string
	Username   string
	Password   string
	callbacks  map[string][]func(topic, message, from string)
}

type message struct {
	Topic   string `json:"topic"`
	Message string `json:"message"`
	From    string `json:"from"`
}

// NewClient creates a new Moustique client
func NewClient(ip, port, clientName, username, password string) *Client {
	if clientName == "" {
		hostname, _ := os.Hostname()
		clientName = hostname + "-cli"
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
		payload.Set("username", encode(c.Username))
		payload.Set("password", encode(c.Password))
	}
	return payload
}

func (c *Client) Publish(topic, message string) error {
	payload := c.addAuth(url.Values{
		"topic":                {encode(topic)},
		"message":              {encode(message)},
		"updated_time":         {encode(fmt.Sprintf("%d", time.Now().Unix()))},
		"updated_nicedatetime": {encode(getNiceDateTime())},
		"from":                 {encode(c.ClientName)},
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
		"valname":              {encode(topic)},
		"val":                  {encode(value)},
		"updated_time":         {encode(fmt.Sprintf("%d", time.Now().Unix()))},
		"updated_nicedatetime": {encode(getNiceDateTime())},
		"from":                 {encode(c.ClientName)},
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
		"topic":  {encode(topic)},
		"client": {encode(c.ClientName)},
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

	c.callbacks[topic] = append(c.callbacks[topic], callback)
	fmt.Printf("%s subscribed to %s\n", c.ClientName, topic)
	return nil
}

func (c *Client) Pickup() error {
	payload := c.addAuth(url.Values{
		"client": {encode(c.ClientName)},
	})

	resp, err := c.HTTPClient.PostForm(c.BaseURL+"/PICKUP", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	decrypted := decode(string(body))
	if decrypted == "" {
		return nil
	}

	// Parse JSON: map[string][]message
	var data map[string][]message
	if err := json.Unmarshal([]byte(decrypted), &data); err != nil {
		return nil
	}

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

func main() {
	// Define flags
	action := flag.String("a", "", "Action: pub, sub, get, put, stats, clients, topics, version")
	host := flag.String("h", "localhost", "Moustique server host")
	port := flag.String("p", "33334", "Moustique server port")
	topic := flag.String("t", "", "Topic")
	message := flag.String("m", "", "Message")
	clientName := flag.String("n", "", "Client name (auto-generated if not provided)")
	username := flag.String("u", "", "Username for authentication (optional)")
	password := flag.String("pwd", "", "Password for authentication (optional)")
	verbose := flag.Bool("v", false, "Verbose output")
	help := flag.Bool("help", false, "Show help")

	flag.Parse()

	if *help || *action == "" {
		printHelp()
		return
	}

	// Create client
	client := NewClient(*host, *port, *clientName, *username, *password)

	// Execute action
	switch *action {
	case "pub", "publish":
		if *topic == "" || *message == "" {
			fmt.Println("Error: -t (topic) and -m (message) are required for publish")
			os.Exit(1)
		}
		if err := client.Publish(*topic, *message); err != nil {
			fmt.Printf("Error publishing: %v\n", err)
			os.Exit(1)
		}

	case "put", "putval":
		if *topic == "" || *message == "" {
			fmt.Println("Error: -t (topic) and -m (message) are required for putval")
			os.Exit(1)
		}
		if err := client.PutVal(*topic, *message); err != nil {
			fmt.Printf("Error putting value: %v\n", err)
			os.Exit(1)
		}

	case "sub", "subscribe":
		if *topic == "" {
			fmt.Println("Error: -t (topic) is required for subscribe")
			os.Exit(1)
		}
		if *verbose {
			fmt.Printf("Client: %s\n", client.ClientName)
		}
		fmt.Printf("Subscribing to: %s\n", *topic)

		err := client.Subscribe(*topic, func(topic, message, from string) {
			fmt.Printf("%s | %s | %s | %s\n", time.Now().Format("2006-01-02 15:04:05"), topic, from, message)
		})
		if err != nil {
			fmt.Printf("Error subscribing: %v\n", err)
			os.Exit(1)
		}

		// Poll for messages
		fmt.Println("Listening for messages... (Ctrl+C to exit)")
		for {
			if err := client.Pickup(); err != nil {
				if *verbose {
					fmt.Printf("Pickup error: %v\n", err)
				}
			}
			time.Sleep(1 * time.Second)
		}

	case "version":
		fmt.Printf("moustique-cli version %s\n", version)

	default:
		fmt.Printf("Unknown action: %s\n", *action)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Moustique CLI - Command line client for Moustique message broker")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  moustique-cli -a <action> [options]")
	fmt.Println()
	fmt.Println("Actions:")
	fmt.Println("  pub, publish   Publish a message to a topic")
	fmt.Println("  put, putval    Store a key-value pair")
	fmt.Println("  sub, subscribe Subscribe to a topic and listen for messages")
	fmt.Println("  version        Show version information")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -a string      Action to perform (required)")
	fmt.Println("  -h string      Moustique server host (default: localhost)")
	fmt.Println("  -p string      Moustique server port (default: 33334)")
	fmt.Println("  -t string      Topic")
	fmt.Println("  -m string      Message")
	fmt.Println("  -n string      Client name (auto-generated if not provided)")
	fmt.Println("  -u string      Username for authentication (optional)")
	fmt.Println("  -pwd string    Password for authentication (optional)")
	fmt.Println("  -v             Verbose output")
	fmt.Println("  -help          Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Publish to public broker")
	fmt.Println("  moustique-cli -a pub -t /test/topic -m \"Hello World\"")
	fmt.Println()
	fmt.Println("  # Publish with authentication")
	fmt.Println("  moustique-cli -a pub -u alice -pwd secret123 -t /test/topic -m \"Hello\"")
	fmt.Println()
	fmt.Println("  # Subscribe to topic")
	fmt.Println("  moustique-cli -a sub -t /test/topic")
	fmt.Println()
	fmt.Println("  # Subscribe with authentication")
	fmt.Println("  moustique-cli -a sub -u alice -pwd secret123 -t /private/topic")
	fmt.Println()
	fmt.Println("  # Put a value")
	fmt.Println("  moustique-cli -a put -t /config/setting -m \"value123\"")
	fmt.Println()
	fmt.Println("  # Connect to remote server")
	fmt.Println("  moustique-cli -h moustique.host -p 33334 -a pub -t /remote/topic -m \"Hi\"")
}
