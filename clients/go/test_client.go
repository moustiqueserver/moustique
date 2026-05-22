package main

// Moustique Go Client - HTTP + MQTT Integration Test

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/moustiqueserver/moustique/clients/go/moustique"
)

func runTest(protocol, ip, port, username, password, mqttPort string) bool {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("Testing with %s protocol\n", strings.ToUpper(protocol))
	fmt.Println(strings.Repeat("=", 60) + "\n")

	useMqtt := (protocol == "mqtt")

	fmt.Println("=== Moustique Go Client – Multi-tenant Test ===")
	fmt.Printf("Protocol: %s\n", strings.ToUpper(protocol))
	fmt.Printf("Server: %s:%s\n", ip, port)
	if useMqtt {
		fmt.Printf("MQTT Port: %s\n", mqttPort)
	}
	fmt.Printf("Username: %s\n\n", username)

	// Create client with authentication
	var client *moustique.Client
	if useMqtt {
		client = moustique.New(ip, port, fmt.Sprintf("GoTestClient-%s", protocol), username, password, "true", mqttPort)
	} else {
		client = moustique.New(ip, port, fmt.Sprintf("GoTestClient-%s", protocol), username, password)
	}

	fmt.Printf("Client ID: %s\n\n", client.GetClientName())

	// Message callback
	messageCallback := func(topic, message, from string) {
		timestamp := time.Now().Format("15:04:05")
		fmt.Printf("[%s] MESSAGE → '%s': %s (from %s)\n", timestamp, topic, message, from)
	}

	// 1. Publish
	fmt.Println("1. Publishing message...")
	if err := client.Publish("/test/topic/go", fmt.Sprintf("Hello from %s test!", strings.ToUpper(protocol))); err != nil {
		fmt.Printf("   Error: %v\n", err)
		return false
	}
	time.Sleep(500 * time.Millisecond)

	// 2. Set value
	fmt.Println("2. Setting value...")
	if err := client.PutVal("/test/value/go", fmt.Sprintf("go-%s-v1", protocol)); err != nil {
		fmt.Printf("   Error: %v\n", err)
		return false
	}
	time.Sleep(500 * time.Millisecond)

	// 3. Get value (full struct)
	fmt.Println("3. Getting value (full)...")
	result, err := client.GetVal("/test/value/go")
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		return false
	}
	expected := fmt.Sprintf("go-%s-v1", protocol)
	if result.Message != expected {
		fmt.Printf("   GetVal mismatch: got %q, want %q\n", result.Message, expected)
		return false
	}
	fmt.Printf("   GetVal OK: message=%q from=%q updated=%s\n", result.Message, result.From, result.UpdatedNiceDateTime)

	// GetValString convenience method
	fmt.Println("   Getting value (string only)...")
	strVal, err := client.GetValString("/test/value/go")
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		return false
	}
	fmt.Printf("   GetValString OK: %q\n", strVal)
	time.Sleep(500 * time.Millisecond)

	// 4. Subscribe and receive
	fmt.Printf("4. Subscribing to /test/topic/go...\n")
	if err := client.Subscribe("/test/topic/go", messageCallback); err != nil {
		fmt.Printf("   Error: %v\n", err)
		return false
	}

	fmt.Println("   Sending message to trigger callback...")
	time.Sleep(1 * time.Second) // Give subscription time to register
	if err := client.Publish("/test/topic/go", fmt.Sprintf("This message should appear in callback! (%s)", strings.ToUpper(protocol))); err != nil {
		fmt.Printf("   Error: %v\n", err)
	}

	if useMqtt {
		fmt.Println("   Waiting for MQTT messages (10 seconds)...")
		time.Sleep(10 * time.Second)
	} else {
		fmt.Println("   Polling for HTTP messages (10 seconds)...")
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		done := time.After(10 * time.Second)

		for {
			select {
			case <-ticker.C:
				client.Pickup()
			case <-done:
				goto testComplete
			}
		}
	}

testComplete:
	fmt.Printf("\n=== %s test complete! ===\n", strings.ToUpper(protocol))

	if useMqtt {
		client.Disconnect()
	}

	return true
}

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: go run test_client.go <ip> <port> <username> <password> [mqtt_port]")
		fmt.Println("Example: go run test_client.go localhost 33334 demo demo123")
		fmt.Println("         go run test_client.go localhost 33334 demo demo123 1883")
		os.Exit(1)
	}

	ip := os.Args[1]
	port := os.Args[2]
	username := os.Args[3]
	password := os.Args[4]
	mqttPort := "1883"
	if len(os.Args) > 5 {
		mqttPort = os.Args[5]
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Moustique Go Client - HTTP + MQTT Integration Test")
	fmt.Println(strings.Repeat("=", 60))

	// Test HTTP
	httpSuccess := runTest("http", ip, port, username, password, mqttPort)

	// Test MQTT
	mqttSuccess := runTest("mqtt", ip, port, username, password, mqttPort)

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	if httpSuccess {
		fmt.Println("HTTP test: ✓ PASSED")
	} else {
		fmt.Println("HTTP test: ✗ FAILED")
	}
	if mqttSuccess {
		fmt.Println("MQTT test: ✓ PASSED")
	} else {
		fmt.Println("MQTT test: ✗ FAILED")
	}
	fmt.Println(strings.Repeat("=", 60) + "\n")

	if httpSuccess && mqttSuccess {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
