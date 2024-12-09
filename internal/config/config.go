package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Websocket WebsocketConfig `mapstructure:"websocket"`
	Audio     AudioConfig     `mapstructure:"audio"`
	Azure     AzureConfig     `mapstructure:"azure"`
	AIConfig  AIConfig        `mapstructure:"ai"`
}

type AIConfig struct {
	SystemPromptFilePath string `mapstructure:"system_prompt_filepath"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type WebsocketConfig struct {
	PingInterval    string `mapstructure:"ping_interval"`
	PongWait        string `mapstructure:"pong_wait"`
	WriteWait       string `mapstructure:"write_wait"`
	MaxMessageQueue int    `mapstructure:"max_message_queue"`
}

type AudioFormat string

const (
	PCM16 AudioFormat = "pcm_16"
	WAV   AudioFormat = "wav"
	MP3   AudioFormat = "mp3"
)

// this is the configuration of the audio the hardware will be sending
// this is also the configuration of the audio the harware expects to receive
type AudioConfig struct {
	SampleRate  int         `mapstructure:"sample_rate"`
	Channels    int         `mapstructure:"channels"`
	AudioFormat AudioFormat `mapstructure:"format"`
}

type AzureConfig struct {
	OpenAIKey  string `mapstructure:"openai_key"`
	ServiceURL string `mapstructure:"service_url"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Set default values
	v.SetDefault("server.port", 8080)
	v.SetDefault("websocket.ping_interval", "30s")
	v.SetDefault("websocket.pong_wait", "60s")
	v.SetDefault("websocket.write_wait", "10s")
	v.SetDefault("websocket.max_message_queue", 256)
	v.SetDefault("audio.sample_rate", 16000)
	v.SetDefault("audio.channels", 2)
	v.SetDefault("audio.format", "pcm_16")

	// Config file support
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/pixa/")

	// Environment variables support
	v.SetEnvPrefix("PIXA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Check for Azure OpenAI environment variables
	if azureKey := os.Getenv("AZURE_OPENAI_KEY"); azureKey != "" {
		v.Set("azure.openai_key", azureKey)
	}
	if azureURL := os.Getenv("AZURE_OPENAI_URL"); azureURL != "" {
		v.Set("azure.service_url", azureURL)
	}

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate required configurations
	if config.Azure.OpenAIKey == "" {
		return nil, fmt.Errorf("AZURE_OPENAI_KEY environment variable is required")
	}
	if config.Azure.ServiceURL == "" {
		return nil, fmt.Errorf("AZURE_OPENAI_URL environment variable or azure.service_url config is required")
	}

	return &config, nil
}

// ValidateConfig validates the configuration values
func ValidateConfig(cfg *Config) error {
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", cfg.Server.Port)
	}

	if cfg.Audio.SampleRate <= 0 {
		return fmt.Errorf("invalid sample rate: %d", cfg.Audio.SampleRate)
	}

	if cfg.Audio.Channels <= 0 {
		return fmt.Errorf("invalid number of channels: %d", cfg.Audio.Channels)
	}

	if cfg.Audio.AudioFormat != PCM16 && cfg.Audio.AudioFormat != WAV && cfg.Audio.AudioFormat != MP3 {
		return fmt.Errorf("invalid audio format: %s", cfg.Audio.AudioFormat)
	}

	return nil
}
