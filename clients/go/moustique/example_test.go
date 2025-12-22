package moustique

import (
	"time"

	"moustique/clients/go/moustique"
)

func Example() {
	c := moustique.New("192.168.1.79", "33334", "GoDemo")

	c.Subscribe("/test/topic", func(topic, message, from string) {
		println("[GO] ", topic, ":", message, "(from", from, ")")
	})

	_ = c.Publish("/test/topic", "Hej från Go-klienten!")
	_ = c.PutVal("/test/value", "go-value-42")

	println("Go-klienten lyssnar på /test/topic i 20 sekunder...")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	done := make(chan bool)
	go func() {
		time.Sleep(20 * time.Second)
		done <- true
	}()

	for {
		select {
		case <-ticker.C:
			c.Pickup()
		case <-done:
			println("Demo avslutad")
			return
		}
	}
}
