package main

import (
	"fmt"
	"github.com/moustiqueserver/moustique/clients/go/moustique"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	c := moustique.New("192.168.1.79", "33334", "GoDemo")

	c.Subscribe("/test/topic", func(topic, message, from string) {
		fmt.Printf("[GO] %s: %s (från %s)\n", topic, message, from)
	})

	fmt.Println("Go-klienten startad – publicerar testmeddelande...")
	c.Publish("/test/topic", "Hej från Go-klienten!")
	c.PutVal("/test/value", "go-value-42")

	fmt.Println("Lyssnar på /test/topic – avsluta med Ctrl+C")

	// Fånga Ctrl+C
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.Pickup()
		case <-sigs:
			fmt.Println("\nAvslutar Go-klienten...")
			return
		}
	}
}
