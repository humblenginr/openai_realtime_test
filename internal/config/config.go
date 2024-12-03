package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Websocket WebsocketConfig `yaml:"websocket"`
	Audio     AudioConfig     `yaml:"audio"`
}

type ServerConfig struct {
	Port           int    `yaml:"port"`
	ReadTimeout    string `yaml:"read_timeout"`
	WriteTimeout   string `yaml:"write_timeout"`
	MaxMessageSize int    `yaml:"max_message_size"`
}

type WebsocketConfig struct {
	PingInterval    string `yaml:"ping_interval"`
	PongWait        string `yaml:"pong_wait"`
	WriteWait       string `yaml:"write_wait"`
	MaxMessageQueue int    `yaml:"max_message_queue"`
}

type AudioConfig struct {
	SampleRate int `yaml:"sample_rate"`
	Channels   int `yaml:"channels"`
	BitDepth   int `yaml:"bit_depth"`
}

// LoadConfig reads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &config, nil
}
