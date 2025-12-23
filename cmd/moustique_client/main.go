// cmd/moustique_client/main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"moustique/clients/go"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const defaultHost = "127.0.0.1"
const defaultPort = "33335"

func main() {
	host := flag.String("h", defaultHost, "Moustique server host")
	port := flag.String("p", defaultPort, "Moustique server port")
	pwd := flag.String("pwd", "", "Password for admin commands (stats, clients, etc.)")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		usage()
		os.Exit(1)
	}

	action := args[0]
	c := moustique.New(*host, *port, "moustique_client")

	switch action {
	case "pub":
		if len(args) < 3 {
			fmt.Println("Usage: moustique_client pub -t <topic> -m <message>")
			os.Exit(1)
		}
		topic := flag.String("t", "", "Topic")
		message := flag.String("m", "", "Message")
		flag.CommandLine.Parse(args[1:])
		if *topic == "" || *message == "" {
			fmt.Println("Error: -t topic and -m message are required for pub")
			os.Exit(1)
		}
		if err := c.Publish(*topic, *message); err != nil {
			fmt.Printf("Publish failed: %v\n", err)
			os.Exit(1)
		}

	case "put":
		if len(args) < 3 {
			fmt.Println("Usage: moustique_client put -t <topic> -m <value>")
			os.Exit(1)
		}
		topic := flag.String("t", "", "Topic")
		value := flag.String("m", "", "Value")
		flag.CommandLine.Parse(args[1:])
		if *topic == "" || *value == "" {
			fmt.Println("Error: -t topic and -m value are required for put")
			os.Exit(1)
		}
		if err := c.PutVal(*topic, *value); err != nil {
			fmt.Printf("Put failed: %v\n", err)
			os.Exit(1)
		}

	case "sub":
		if len(args) < 2 {
			fmt.Println("Usage: moustique_client sub -t <topic>")
			os.Exit(1)
		}
		topic := flag.String("t", "", "Topic")
		flag.CommandLine.Parse(args[1:])
		if *topic == "" {
			fmt.Println("Error: -t topic is required for sub")
			os.Exit(1)
		}
		if err := c.Subscribe(*topic, func(t, m, f string) {
			fmt.Printf("%s [%s] %s\n", time.Now().Format("15:04:05"), f, m)
		}); err != nil {
			fmt.Printf("Subscribe failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Subscribed to %s – listening (Ctrl+C to stop)\n", *topic)
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.Pickup()
			case <-sigs:
				fmt.Println("\nGoodbye!")
				return
			}
		}

	case "stats", "clients", "topics", "posters", "peerhosts", "version":
		if *pwd == "" {
			fmt.Println("Error: -pwd <password> required for admin commands")
			os.Exit(1)
		}
		data, err := adminCommand(c, *host, *port, *pwd, action)
		if err != nil {
			fmt.Printf("Failed: %v\n", err)
			os.Exit(1)
		}
		if *verbose {
			prettyPrint(data)
		} else {
			simplePrint(action, data)
		}

	default:
		fmt.Printf("Unknown action: %s\n", action)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`moustique_client – command line client for Moustique HTTP pub/sub broker

Usage:
  moustique_client pub -t <topic> -m <message>     Publish message
  moustique_client put -t <topic> -m <value>       Set value (PUTVAL)
  moustique_client sub -t <topic>                  Subscribe and listen
  moustique_client stats -pwd <pwd>                Show server statistics
  moustique_client clients -pwd <pwd>              List active clients
  moustique_client topics -pwd <pwd>               List topics
  moustique_client posters -pwd <pwd>              List posters
  moustique_client peerhosts -pwd <pwd>            List peer hosts
  moustique_client version -pwd <pwd>              Show server version

Options:
  -h <host>      Server host (default: 127.0.0.1)
  -p <port>      Server port (default: 33335)
  -pwd <password> Required for admin commands
  -v             Verbose JSON output (for admin commands)
`)
}

func adminCommand(c *moustique.Client, host, port, pwd, cmd string) (map[string]any, error) {
	// We reuse the existing get functions from the library if possible, but since they're not exported, we do a simple GET
	// This is a minimal implementation – can be extended later
	url := fmt.Sprintf("http://%s:%s/%s", host, port, strings.ToUpper(cmd))
	payload := map[string]string{
		"pwd": moustique.Enc(pwd),
	}
	// Simple POST for admin endpoints
	// Note: This is a simplified version – in production, use proper client
	// For now, we just print a placeholder
	fmt.Printf("Admin command '%s' not fully implemented yet – password protected endpoint.\n", cmd)
	return nil, fmt.Errorf("not implemented")
}

func prettyPrint(data any) {
	b, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(b))
}

func simplePrint(cmd string, data any) {
	fmt.Printf("%s:\n", strings.Title(cmd))
	// Simple formatted output – can be extended per command
	prettyPrint(data)
}
