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

type Message struct {
	Topic               string          `json:"topic"`
	Message             string          `json:"message"`
	From                string          `json:"from"`
	UpdatedTime         int64           `json:"updated_time"`
	UpdatedNiceDatetime string          `json:"updated_nicedatetime"`
	Subscribers         map[string]bool `json:"subscribers"`
	IP                  string          `json:"ip"`
}

// NewClient creates a new Moustique client
func NewClient(ip, port, clientName, username, password string, useTLS bool) *Client {
	if clientName == "" {
		hostname, _ := os.Hostname()
		clientName = hostname + "-cli"
	}
	clientName += "-" + uuid.New().String()[:8]

	scheme := "http"
	if useTLS {
		scheme = "https"
	}

	return &Client{
		BaseURL:    fmt.Sprintf("%s://%s:%s", scheme, ip, port),
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

func (c *Client) GetVal(topic string) *Message {
	payload := c.addAuth(url.Values{
		"topic":  {encode(topic)},
		"client": {encode(c.ClientName)},
	})

	req, _ := http.NewRequest("POST", c.BaseURL+"/GETVAL", bytes.NewBufferString(payload.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil //, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != 308 {
		//body, _ := io.ReadAll(resp.Body)
		return nil //, fmt.Errorf("getval failed: %d %s", resp.StatusCode, string(body))
	}
	body, _ := io.ReadAll(resp.Body)

	decrypted := decode(string(body))
	if decrypted == "" {
		return nil //, nil
	}

	// Parse JSON: map[string]Message
	//var data map[string]Message
	var msg *Message
	if err := json.Unmarshal([]byte(decrypted), &msg); err != nil {
		return nil //, nil
	}
	//fmt.Printf("Message: %s : %s", msg.Topic, msg.Message)
	return msg //, err
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

func (c *Client) GetKeys(verbose bool) ([]string, error) {
	payload := c.addAuth(url.Values{
		"client": {encode(c.ClientName)},
	})

	if verbose {
		fmt.Printf("Fetching topics from %s/TOPICS\n", c.BaseURL)
	}

	resp, err := c.HTTPClient.PostForm(c.BaseURL+"/TOPICS", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if verbose {
		fmt.Printf("Response status: %s\n", resp.Status)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %s: %s", resp.Status, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	if verbose {
		fmt.Printf("Response body length: %d bytes\n", len(body))
	}

	decrypted := decode(string(body))
	if verbose {
		fmt.Printf("Decrypted length: %d bytes\n", len(decrypted))
	}

	if decrypted == "" {
		return nil, nil
	}
	// Parse JSON: []string (topic names)
	var data []string
	if err := json.Unmarshal([]byte(decrypted), &data); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}
	if verbose {
		fmt.Printf("Found %d topics\n", len(data))
	}
	return data, nil
}
func (c *Client) GetValsByRegex(pattern string, verbose bool) (map[string]Message, error) {
	payload := c.addAuth(url.Values{
		"topic":  {encode(pattern)},
		"client": {encode(c.ClientName)},
	})

	if verbose {
		fmt.Printf("Sending pattern: %s (encoded: %s)\n", pattern, encode(pattern))
	}

	resp, err := c.HTTPClient.PostForm(c.BaseURL+"/GETVALSBYREGEX", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if verbose {
		fmt.Printf("Response status: %s\n", resp.Status)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %s: %s", resp.Status, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	if verbose {
		fmt.Printf("Response body length: %d bytes\n", len(body))
		if len(body) < 500 {
			fmt.Printf("Raw body: %s\n", string(body))
		}
	}

	decrypted := decode(string(body))
	if verbose {
		fmt.Printf("Decrypted length: %d bytes\n", len(decrypted))
		if len(decrypted) < 500 {
			fmt.Printf("Decrypted: %s\n", decrypted)
		}
	}

	if decrypted == "" {
		return nil, nil
	}

	// Parse JSON: map[string]Message
	var data map[string]Message
	if err := json.Unmarshal([]byte(decrypted), &data); err != nil {
		preview := decrypted
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("failed to parse response (first 200 chars: %s): %v", preview, err)
	}
	return data, nil
}

// isPrintable checks if a string contains only printable ASCII characters
func isPrintable(s string) bool {
	for _, c := range s {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}

// sqlLikeToRegex converts SQL LIKE pattern to regex
// % -> .* (any characters)
// _ -> .  (single character)
func sqlLikeToRegex(pattern string) string {
	// Escape regex special characters except % and _
	result := ""
	for _, c := range pattern {
		switch c {
		case '%':
			result += ".*"
		case '_':
			result += "."
		case '.', '+', '*', '?', '^', '$', '(', ')', '[', ']', '{', '}', '|', '\\':
			result += "\\" + string(c)
		default:
			result += string(c)
		}
	}
	return result
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

	// Parse JSON: map[string][]Message
	var data map[string][]Message
	if err := json.Unmarshal([]byte(decrypted), &data); err != nil {
		return nil
	}

	// Handle system messages first
	if systemMsgs, ok := data["/server/action/resubscribe"]; ok && len(systemMsgs) > 0 {
		fmt.Println("⚠️  Server restarted - re-subscribing to all topics...")
		// Re-subscribe to all topics
		for topic := range c.callbacks {
			payload := c.addAuth(url.Values{
				"topic":  {encode(topic)},
				"client": {encode(c.ClientName)},
			})
			if resp, err := c.HTTPClient.PostForm(c.BaseURL+"/SUBSCRIBE", payload); err == nil {
				resp.Body.Close()
				fmt.Printf("✓ Re-subscribed to %s\n", topic)
			}
		}
	}

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

func main() {
	// Define flags
	action := flag.String("a", "", "Action: pub, sub, get, put, like, version")
	host := flag.String("h", "localhost", "Moustique server host")
	port := flag.String("p", "33334", "Moustique server port")
	topic := flag.String("t", "", "Topic")
	message := flag.String("m", "", "Message")
	clientName := flag.String("n", "", "Client name (auto-generated if not provided)")
	username := flag.String("u", "", "Username for authentication (optional)")
	password := flag.String("pwd", "", "Password for authentication (optional)")
	useTLS := flag.Bool("s", false, "Use HTTPS/TLS (secure mode)")
	verbose := flag.Bool("v", false, "Verbose output")
	help := flag.Bool("help", false, "Show help")

	flag.Parse()

	if *help || *action == "" {
		printHelp()
		return
	}

	// Create client
	client := NewClient(*host, *port, *clientName, *username, *password, *useTLS)

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

	case "get", "getval":
		if *topic == "" {
			fmt.Println("Error: -t (topic) is required for get")
			os.Exit(1)
		}
		//if
		mess := client.GetVal(*topic)
		/*; err != nil {
			fmt.Printf("Error getting: %v\n", err)
			os.Exit(1)
		}*/

		if *verbose {
			jsonBytes, _ := json.MarshalIndent(mess, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			fmt.Printf("%s: %s\n", mess.Topic, mess.Message)
		}

	case "like", "getvalsbyregex":
		if *topic == "" {
			fmt.Println("Error: -t (pattern) is required for like")
			fmt.Println("Use SQL LIKE syntax: % for any characters, _ for single character")
			fmt.Println("Example: moustique-cli -a like -t %outside/temper%")
			os.Exit(1)
		}

		// Convert SQL LIKE pattern to regex
		regexPattern := sqlLikeToRegex(*topic)
		if *verbose {
			fmt.Printf("Pattern: %s -> Regex: %s\n", *topic, regexPattern)
		}

		msgs, err := client.GetValsByRegex(regexPattern, *verbose)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// Filter out corrupted entries
		validMsgs := make(map[string]Message)
		for topic, msg := range msgs {
			if len(topic) > 0 && topic[0] == '/' && isPrintable(topic) {
				validMsgs[topic] = msg
			}
		}

		if *verbose {
			fmt.Printf("Filtered: %d valid entries out of %d total\n", len(validMsgs), len(msgs))
		}

		if len(validMsgs) == 0 {
			fmt.Println("No matches found")
			os.Exit(0)
		}

		if *verbose {
			jsonBytes, _ := json.MarshalIndent(validMsgs, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			// Simple output format
			for topic, msg := range validMsgs {
				fmt.Printf("%s: %s\n", topic, msg.Message)
			}
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

	case "topics", "keys":
		keys, err := client.GetKeys(*verbose)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		// Filter out corrupted keys (those not starting with / or containing non-printable chars)
		validKeys := make([]string, 0)
		for _, key := range keys {
			if len(key) > 0 && key[0] == '/' && isPrintable(key) {
				validKeys = append(validKeys, key)
			}
		}
		if *verbose {
			fmt.Printf("Filtered: %d valid topics out of %d total\n", len(validKeys), len(keys))
		}
		jsonBytes, _ := json.MarshalIndent(validKeys, "", "  ")
		fmt.Println(string(jsonBytes))

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
	fmt.Println("  get, getval    Get a stored value")
	fmt.Println("  like           Search values using SQL LIKE syntax (% = any, _ = single char)")
	fmt.Println("  sub, subscribe Subscribe to a topic and listen for messages")
	fmt.Println("  version        Show version information")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -a string      Action to perform (required)")
	fmt.Println("  -h string      Moustique server host (default: localhost)")
	fmt.Println("  -p string      Moustique server port (default: 33334)")
	fmt.Println("  -t string      Topic or pattern (for like)")
	fmt.Println("  -m string      Message")
	fmt.Println("  -n string      Client name (auto-generated if not provided)")
	fmt.Println("  -u string      Username for authentication (optional)")
	fmt.Println("  -pwd string    Password for authentication (optional)")
	fmt.Println("  -s             Use HTTPS/TLS (secure mode)")
	fmt.Println("  -v             Verbose output (shows full JSON for like)")
	fmt.Println("  -help          Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Publish a message")
	fmt.Println("  moustique-cli -a pub -t /test/topic -m \"Hello World\"")
	fmt.Println()
	fmt.Println("  # Get a stored value")
	fmt.Println("  moustique-cli -a get -t /mushroom/sensors/temperature")
	fmt.Println()
	fmt.Println("  # Search values with SQL LIKE pattern")
	fmt.Println("  moustique-cli -a like -t %btc_price%")
	fmt.Println("  moustique-cli -a like -t /mushroom/sensors/%")
	fmt.Println("  moustique-cli -a like -t %temperature% -v    # verbose output")
	fmt.Println()
	fmt.Println("  # Subscribe to topic")
	fmt.Println("  moustique-cli -a sub -t /test/topic")
	fmt.Println()
	fmt.Println("  # Put a value")
	fmt.Println("  moustique-cli -a put -t /config/setting -m \"value123\"")
	fmt.Println()
	fmt.Println("  # Connect to remote server")
	fmt.Println("  moustique-cli -h 192.168.1.79 -p 33334 -a like -t %sensor%")
}
