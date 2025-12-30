package main

import (
	"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Logging  LoggingConfig  `yaml:"logging"`
	Security SecurityConfig `yaml:"security"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port           int           `yaml:"port"`
	MQTTPort       int           `yaml:"mqtt_port"`        // MQTT listener port, 0 = disabled
	Timeout        time.Duration `yaml:"timeout"`
	AllowPublic    *bool         `yaml:"allow_public"` // Pointer to detect if set
	MaxRequestSize int64         `yaml:"max_request_size"` // in bytes, 0 = unlimited
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level     string `yaml:"level"`
	Directory string `yaml:"directory"` // directory for log files
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	AllowedPeers       []string `yaml:"allowed_peers"`
	BlockedPeers       []string `yaml:"blocked_peers"`
	MaxTopicLength     int      `yaml:"max_topic_length"`     // max length for topic names
	MaxMessageSize     int64    `yaml:"max_message_size"`     // max size for messages in bytes
	DefaultRateLimit   int      `yaml:"default_rate_limit"`   // requests per minute, 0 = unlimited
	Fail2banJail       string   `yaml:"fail2ban_jail"`        // fail2ban jail name, empty = disabled
	Fail2banLevel      string   `yaml:"fail2ban_level"`       // strict, normal, relaxed, minimal
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults
	if config.Server.Port == 0 {
		config.Server.Port = 33334
	}
	// MQTT port defaults to 1883 if not explicitly set to 0
	// To disable MQTT, explicitly set mqtt_port: 0 in config
	if config.Server.Timeout == 0 {
		config.Server.Timeout = 30 * time.Second
	}
	if config.Server.AllowPublic == nil {
		defaultVal := false
		config.Server.AllowPublic = &defaultVal
	}
	if config.Server.MaxRequestSize == 0 {
		config.Server.MaxRequestSize = 10 * 1024 * 1024 // 10MB default
	}
	if config.Database.Path == "" {
		config.Database.Path = "./data"
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Directory == "" {
		config.Logging.Directory = "/var/log/moustique"
	}
	if config.Security.MaxTopicLength == 0 {
		config.Security.MaxTopicLength = 256
	}
	if config.Security.MaxMessageSize == 0 {
		config.Security.MaxMessageSize = 1 * 1024 * 1024 // 1MB default
	}
	if config.Security.DefaultRateLimit == 0 {
		config.Security.DefaultRateLimit = 1000 // 1000 req/min default
	}
	if config.Security.Fail2banJail == "" {
		config.Security.Fail2banJail = "moustique"
	}
	if config.Security.Fail2banLevel == "" {
		config.Security.Fail2banLevel = "normal"
	}

	return &config, nil
}

// GenerateDefaultConfig generates a default configuration file
func GenerateDefaultConfig(path string) error {
	defaultAllowPublic := false
	config := Config{
		Server: ServerConfig{
			Port:           33334,
			Timeout:        30 * time.Second,
			AllowPublic:    &defaultAllowPublic,
			MaxRequestSize: 10 * 1024 * 1024, // 10MB
		},
		Database: DatabaseConfig{
			Path: "./data",
		},
		Logging: LoggingConfig{
			Level:     "info",
			Directory: "/var/log/moustique",
		},
		Security: SecurityConfig{
			AllowedPeers: []string{
				"172.16.0.0/12",
				"192.168.0.0/16",
			},
			BlockedPeers:       []string{},
			MaxTopicLength:     256,
			MaxMessageSize:     1 * 1024 * 1024, // 1MB
			DefaultRateLimit:   1000,            // 1000 requests per minute
			Fail2banJail:       "moustique",
			Fail2banLevel:      "normal", // strict, normal, relaxed, minimal
		},
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
