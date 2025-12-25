package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"moustique/clients/go/moustique"
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

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go_test <public|auth> <host> <port> [username] [password]")
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
