package main

import (
	"fmt"
	"log"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
)

// MQTTServer wraps the Mochi MQTT server
type MQTTServer struct {
	server        *mqtt.Server
	brokerManager *BrokerManager
	userAuth      *UserAuth
	logger        *log.Logger
}

// NewMQTTServer creates a new MQTT server instance
func NewMQTTServer(brokerManager *BrokerManager, userAuth *UserAuth, logger *log.Logger) *MQTTServer {
	return &MQTTServer{
		brokerManager: brokerManager,
		userAuth:      userAuth,
		logger:        logger,
	}
}

// Start starts the MQTT server on the specified port
func (m *MQTTServer) Start(port int) error {
	// Create MQTT server
	m.server = mqtt.New(&mqtt.Options{
		InlineClient: false,
	})

	// Add custom hook for handling everything (auth + subscriptions + messages)
	hook := &MoustiqueMQTTHook{
		brokerManager: m.brokerManager,
		userAuth:      m.userAuth,
		logger:        m.logger,
	}
	if err := m.server.AddHook(hook, nil); err != nil {
		return fmt.Errorf("failed to add moustique hook: %w", err)
	}

	// Create TCP listener
	tcp := listeners.NewTCP(listeners.Config{
		ID:      "mqtt",
		Address: fmt.Sprintf(":%d", port),
	})
	if err := m.server.AddListener(tcp); err != nil {
		return fmt.Errorf("failed to add MQTT listener: %w", err)
	}

	// Start server in background
	go func() {
		m.logger.Printf("Starting MQTT server on port %d", port)
		if err := m.server.Serve(); err != nil {
			m.logger.Printf("MQTT server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the MQTT server
func (m *MQTTServer) Stop() error {
	if m.server != nil {
		return m.server.Close()
	}
	return nil
}

// MoustiqueMQTTHook integrates MQTT with Moustique broker
type MoustiqueMQTTHook struct {
	mqtt.HookBase
	brokerManager *BrokerManager
	userAuth      *UserAuth
	logger        *log.Logger
}

// ID returns the hook ID
func (h *MoustiqueMQTTHook) ID() string {
	return "moustique-mqtt-hook"
}

// Provides indicates which hook methods this hook provides
func (h *MoustiqueMQTTHook) Provides(b byte) bool {
	return b == mqtt.OnConnectAuthenticate ||
		b == mqtt.OnACLCheck ||
		b == mqtt.OnConnect ||
		b == mqtt.OnDisconnect ||
		b == mqtt.OnSubscribe ||
		b == mqtt.OnUnsubscribe ||
		b == mqtt.OnPublish
}

// OnConnectAuthenticate checks username and password
func (h *MoustiqueMQTTHook) OnConnectAuthenticate(cl *mqtt.Client, pk packets.Packet) bool {
	username := string(pk.Connect.Username)
	password := string(pk.Connect.Password)

	if username == "" || password == "" {
		h.logger.Printf("MQTT authentication failed: empty credentials")
		return false
	}

	valid := h.userAuth.ValidateUser(username, password)
	if valid {
		h.logger.Printf("MQTT authentication successful for user: %s", username)
	} else {
		h.logger.Printf("MQTT authentication failed for user: %s", username)
	}
	return valid
}

// OnACLCheck checks if user has access to topic
func (h *MoustiqueMQTTHook) OnACLCheck(cl *mqtt.Client, topic string, write bool) bool {
	// All authenticated users can publish and subscribe to any topic
	return true
}

// OnConnect is called when a client successfully connects
func (h *MoustiqueMQTTHook) OnConnect(cl *mqtt.Client, pk packets.Packet) error {
	username := string(pk.Connect.Username)
	clientID := cl.ID

	h.logger.Printf("MQTT client connected: %s (username: %s)", clientID, username)

	// Get user's broker
	broker := h.brokerManager.GetBroker(username)
	if broker == nil {
		h.logger.Printf("ERROR: No broker found for user %s", username)
		return fmt.Errorf("no broker for user")
	}

	// Register MQTT client in broker (clientID is the unique channel key)
	msgChan := broker.RegisterMQTTClient(clientID)

	// Start goroutine to push messages from channel to MQTT client
	go h.pushMessagesToClient(cl, msgChan)

	return nil
}

// OnDisconnect is called when a client disconnects
func (h *MoustiqueMQTTHook) OnDisconnect(cl *mqtt.Client, err error, expire bool) {
	username := string(cl.Properties.Username)
	clientID := cl.ID
	h.logger.Printf("MQTT client disconnected: %s (username: %s)", clientID, username)

	// Get user's broker and unregister client
	broker := h.brokerManager.GetBroker(username)
	if broker != nil {
		// Use clientID as clientName (same as in Subscribe)
		broker.UnregisterMQTTClient(clientID, clientID)
	}
}

// OnSubscribe is called when a client subscribes to topics
func (h *MoustiqueMQTTHook) OnSubscribe(cl *mqtt.Client, pk packets.Packet) packets.Packet {
	username := string(cl.Properties.Username)
	clientID := cl.ID

	// Get user's broker
	broker := h.brokerManager.GetBroker(username)
	if broker == nil {
		h.logger.Printf("ERROR: No broker found for user %s", username)
		return pk
	}

	for _, filter := range pk.Filters {
		topic := filter.Filter
		h.logger.Printf("MQTT client %s (user: %s) subscribed to: %s", clientID, username, topic)

		// Use MQTT clientID as the Moustique client name (each MQTT session = separate client)
		if err := broker.SubscribeWithType(topic, clientID, cl.Net.Remote, "mqtt", clientID); err != nil {
			h.logger.Printf("Failed to subscribe MQTT client %s to %s: %v", clientID, topic, err)
		}
	}

	return pk
}

// OnUnsubscribe is called when a client unsubscribes from a topic
func (h *MoustiqueMQTTHook) OnUnsubscribe(cl *mqtt.Client, pk packets.Packet) packets.Packet {
	username := string(cl.Properties.Username)
	h.logger.Printf("MQTT client %s unsubscribed from: %v", username, pk.Filters)
	return pk
}

// OnPublish is called when a client publishes a message
func (h *MoustiqueMQTTHook) OnPublish(cl *mqtt.Client, pk packets.Packet) (packets.Packet, error) {
	username := string(cl.Properties.Username)
	clientID := cl.ID
	topic := pk.TopicName
	message := string(pk.Payload)

	h.logger.Printf("MQTT publish from %s (user: %s) to %s", clientID, username, topic)

	// Get user's broker
	broker := h.brokerManager.GetBroker(username)
	if broker == nil {
		h.logger.Printf("ERROR: No broker found for user %s", username)
		return pk, fmt.Errorf("no broker for user")
	}

	// Publish to Moustique broker (use clientID as 'from' to identify which program sent it)
	if err := broker.Publish(topic, message, clientID, cl.Net.Remote, pk.Created); err != nil {
		h.logger.Printf("Failed to publish MQTT message: %v", err)
		return pk, err
	}

	return pk, nil
}

// pushMessagesToClient sends messages from channel to MQTT client
func (h *MoustiqueMQTTHook) pushMessagesToClient(cl *mqtt.Client, msgChan chan *Message) {
	for msg := range msgChan {
		// MQTT standard: send only the message payload as plaintext
		// This makes it compatible with mosquitto and other MQTT clients
		// Clients that need metadata can subscribe to the HTTP API instead
		payload := []byte(msg.Message)

		// Publish message to this specific client
		topic := msg.Topic

		// Create publish packet to send to client
		pk := packets.Packet{
			FixedHeader: packets.FixedHeader{
				Type:   packets.Publish,
				Qos:    0, // QoS 0 for now
				Retain: false,
			},
			TopicName: topic,
			Payload:   payload,
		}

		// Send packet to client
		if err := cl.WritePacket(pk); err != nil {
			h.logger.Printf("Failed to push message to MQTT client %s: %v", cl.ID, err)
		}
	}
}
