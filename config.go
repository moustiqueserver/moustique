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
	Port        int           `yaml:"port"`
	Timeout     time.Duration `yaml:"timeout"`
	AllowPublic *bool         `yaml:"allow_public"` // Pointer to detect if set
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	AllowedPeers []string `yaml:"allowed_peers"`
	BlockedPeers []string `yaml:"blocked_peers"`
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
	if config.Server.Timeout == 0 {
		config.Server.Timeout = 30 * time.Second
	}
	if config.Server.AllowPublic == nil {
		defaultVal := false
		config.Server.AllowPublic = &defaultVal
	}
	if config.Database.Path == "" {
		config.Database.Path = "./data"
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}

	return &config, nil
}

// GenerateDefaultConfig generates a default configuration file
func GenerateDefaultConfig(path string) error {
	defaultAllowPublic := false
	config := Config{
		Server: ServerConfig{
			Port:        33334,
			Timeout:     30 * time.Second,
			AllowPublic: &defaultAllowPublic,
		},
		Database: DatabaseConfig{
			Path: "./data",
		},
		Logging: LoggingConfig{
			Level: "info",
			File:  "",
		},
		Security: SecurityConfig{
			AllowedPeers: []string{
				"172.16.0.0/12",
				"192.168.0.0/16",
			},
			BlockedPeers: []string{},
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
