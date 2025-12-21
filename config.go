package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// Config holds all configuration for Moustique
type Config struct {
	Server struct {
		Port           int           `yaml:"port"`
		Host           string        `yaml:"host"`
		Timeout        time.Duration `yaml:"timeout"`
		MaxConnections int           `yaml:"max_connections"`
	} `yaml:"server"`

	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`

	Security struct {
		AllowedIPs       []string `yaml:"allowed_ips"`
		TailscaleEnabled bool     `yaml:"tailscale_enabled"`
		PasswordFile     string   `yaml:"password_file"`
	} `yaml:"security"`

	Logging struct {
		Level string `yaml:"level"`
		File  string `yaml:"file"`
	} `yaml:"logging"`

	Performance struct {
		MessageQueueTimeout time.Duration `yaml:"message_queue_timeout"`
		PosterStatsTimeout  time.Duration `yaml:"poster_stats_timeout"`
		MaintenanceInterval time.Duration `yaml:"maintenance_interval"`
	} `yaml:"performance"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	// Set defaults
	config := &Config{}
	config.Server.Port = 33334
	config.Server.Host = "0.0.0.0"
	config.Server.Timeout = 5 * time.Second
	config.Server.MaxConnections = 1000
	config.Database.Path = "./data/moustique.db"
	config.Security.TailscaleEnabled = true
	config.Security.PasswordFile = "./data/.moustique_pwd"
	config.Logging.Level = "info"
	config.Logging.File = "./logs/moustique.log"
	config.Performance.MessageQueueTimeout = 5 * time.Minute
	config.Performance.PosterStatsTimeout = 1 * time.Hour
	config.Performance.MaintenanceInterval = 30 * time.Second

	// If config file doesn't exist, return defaults
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return config, nil
	}

	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves the current configuration to a YAML file
func SaveConfig(path string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GenerateDefaultConfig creates a default config.yaml file
func GenerateDefaultConfig(path string) error {
	config := &Config{}
	config.Server.Port = 33334
	config.Server.Host = "0.0.0.0"
	config.Server.Timeout = 5 * time.Second
	config.Server.MaxConnections = 1000
	config.Database.Path = "./data/moustique.db"
	config.Security.AllowedIPs = []string{"192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12"}
	config.Security.TailscaleEnabled = true
	config.Security.PasswordFile = "./data/.moustique_pwd"
	config.Logging.Level = "info"
	config.Logging.File = "./logs/moustique.log"
	config.Performance.MessageQueueTimeout = 5 * time.Minute
	config.Performance.PosterStatsTimeout = 1 * time.Hour
	config.Performance.MaintenanceInterval = 30 * time.Second

	return SaveConfig(path, config)
}
