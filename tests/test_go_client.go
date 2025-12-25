package main

import (
	"fmt"
	"os"
	"time"

	"github.com/moustiqueserver/moustique/clients/go/moustique"
)

func testPublic(host, port string) bool {
	client := moustique.New(host, port, "go-test-public", "", "")

	err := client.Publish("/test/go/public", "Hello from Go public!")
	if err != nil {
		fmt.Printf("✗ Go public publish failed: %v\n", err)
		return false
	}

	fmt.Println("✓ Go public publish successful")
	return true
}

func testAuth(host, port, username, password string) bool {
	client := moustique.New(host, port, "go-test-auth", username, password)

	err := client.Publish("/test/go/auth", "Hello from Go auth!")
	if err != nil {
		fmt.Printf("✗ Go authenticated publish failed: %v\n", err)
		return false
	}

	fmt.Println("✓ Go authenticated publish successful")
	return true
}

func testPutval(host, port, username, password string) bool {
	client := moustique.New(host, port, "go-test-putval", username, password)

	testKey := "/test/go/value"
	testValue := "GoTestValue123"
	err := client.PutVal(testKey, testValue)
	if err != nil {
		fmt.Printf("✗ Go PUTVAL failed: %v\n", err)
		return false
	}

	fmt.Println("✓ Go PUTVAL successful")
	return true
}

func testSubscribe(host, port, username, password string) bool {
	client1 := moustique.New(host, port, "go-test-subscriber", username, password)
	client2 := moustique.New(host, port, "go-test-publisher", username, password)

	testTopic := "/test/go/subscribe"
	receivedMessages := []string{}

	callback := func(topic, message, from string) {
		receivedMessages = append(receivedMessages, message)
	}

	// Subscribe
	err := client1.Subscribe(testTopic, callback)
	if err != nil {
		fmt.Printf("✗ Go SUBSCRIBE failed: %v\n", err)
		return false
	}
	time.Sleep(100 * time.Millisecond)

	// Publish a message
	testMessage := "GoSubscribeTest789"
	err = client2.Publish(testTopic, testMessage)
	if err != nil {
		fmt.Printf("✗ Go SUBSCRIBE publish failed: %v\n", err)
		return false
	}
	time.Sleep(100 * time.Millisecond)

	// Pickup messages
	err = client1.Pickup()
	if err != nil {
		fmt.Printf("✗ Go PICKUP failed: %v\n", err)
		return false
	}

	// Check if message was received
	found := false
	for _, msg := range receivedMessages {
		if msg == testMessage {
			found = true
			break
		}
	}

	if found {
		fmt.Println("✓ Go SUBSCRIBE/PICKUP successful")
		return true
	} else {
		fmt.Println("✗ Go SUBSCRIBE/PICKUP failed: message not received")
		return false
	}
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: test_go_client <public|auth|putval|subscribe> <host> <port> [username] [password]")
		os.Exit(1)
	}

	mode := os.Args[1]
	host := os.Args[2]
	port := os.Args[3]

	var success bool

	switch mode {
	case "public":
		success = testPublic(host, port)
	case "auth":
		if len(os.Args) < 6 {
			fmt.Println("Auth mode requires username and password")
			os.Exit(1)
		}
		username := os.Args[4]
		password := os.Args[5]
		success = testAuth(host, port, username, password)
	case "putval":
		if len(os.Args) < 6 {
			fmt.Println("PUTVAL mode requires username and password")
			os.Exit(1)
		}
		username := os.Args[4]
		password := os.Args[5]
		success = testPutval(host, port, username, password)
	case "subscribe":
		if len(os.Args) < 6 {
			fmt.Println("SUBSCRIBE mode requires username and password")
			os.Exit(1)
		}
		username := os.Args[4]
		password := os.Args[5]
		success = testSubscribe(host, port, username, password)
	default:
		fmt.Printf("Unknown mode: %s\n", mode)
		os.Exit(1)
	}

	if success {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
