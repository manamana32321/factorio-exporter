package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	RCON     RCONConfig     `yaml:"rcon"`
	Factorio FactorioConfig `yaml:"factorio"`
	OTel     OTelConfig     `yaml:"otel"`
	Metrics  MetricsConfig  `yaml:"metrics"`
	Events   EventsConfig   `yaml:"events"`
	Discord  DiscordConfig  `yaml:"discord"`
	Loki     LokiConfig     `yaml:"loki"`
}

type RCONConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"-"` // from env only
}

type FactorioConfig struct {
	Namespace string `yaml:"namespace"`
	PodLabel  string `yaml:"pod_label"`
}

type OTelConfig struct {
	Endpoint    string `yaml:"endpoint"`
	ServiceName string `yaml:"service_name"`
}

type MetricsConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
}

type EventsConfig struct {
	Enabled      bool          `yaml:"enabled"`
	PollInterval time.Duration `yaml:"poll_interval"`
	Types        []string      `yaml:"types"` // list of event types, or ["all"]
}

type DiscordConfig struct {
	Enabled   bool     `yaml:"enabled"`
	BotToken  string   `yaml:"-"` // from env only
	ChannelID string   `yaml:"-"` // from env only
	Events    []string `yaml:"events"`
}

type LokiConfig struct {
	Enabled bool        `yaml:"enabled"`
	Events  interface{} `yaml:"events"` // "all" or []string
}

func defaultConfig() Config {
	return Config{
		RCON: RCONConfig{
			Host: "localhost",
			Port: "27015",
		},
		Factorio: FactorioConfig{
			Namespace: "factorio",
			PodLabel:  "app=factorio-factorio-server-charts",
		},
		OTel: OTelConfig{
			ServiceName: "factorio-exporter",
		},
		Metrics: MetricsConfig{
			Enabled:  true,
			Interval: 15 * time.Second,
		},
		Events: EventsConfig{
			Enabled:      true,
			PollInterval: 2 * time.Second,
			Types:        []string{"all"},
		},
		Discord: DiscordConfig{
			Enabled: true,
			Events:  []string{"all"},
		},
		Loki: LokiConfig{
			Enabled: true,
			Events:  "all",
		},
	}
}

func loadConfig() (Config, error) {
	cfg := defaultConfig()

	configPath := envOr("CONFIG_PATH", "/etc/factorio-exporter/config.yaml")
	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parse config %s: %w", configPath, err)
		}
	}
	// config file is optional — missing file is not an error

	// Env overrides (secrets + runtime values)
	cfg.RCON.Password = os.Getenv("RCON_PASSWORD")
	if v := os.Getenv("RCON_HOST"); v != "" {
		cfg.RCON.Host = v
	}
	if v := os.Getenv("RCON_PORT"); v != "" {
		cfg.RCON.Port = v
	}
	cfg.Discord.BotToken = os.Getenv("DISCORD_BOT_TOKEN")
	cfg.Discord.ChannelID = os.Getenv("DISCORD_CHANNEL_ID")

	if cfg.RCON.Password == "" {
		return cfg, fmt.Errorf("RCON_PASSWORD env is required")
	}

	if cfg.Discord.BotToken != "" && cfg.Discord.ChannelID == "" {
		return cfg, fmt.Errorf("DISCORD_CHANNEL_ID is required when DISCORD_BOT_TOKEN is set")
	}

	if cfg.Discord.BotToken == "" {
		cfg.Discord.Enabled = false
	}

	return cfg, nil
}

// lokiEventsAllowed returns whether a given event type should be sent to Loki.
func (c *Config) lokiEventAllowed(eventType string) bool {
	if !c.Loki.Enabled {
		return false
	}
	if s, ok := c.Loki.Events.(string); ok && s == "all" {
		return true
	}
	if list, ok := c.Loki.Events.([]interface{}); ok {
		for _, v := range list {
			if s, ok := v.(string); ok && s == eventType {
				return true
			}
		}
	}
	return false
}

// discordEventAllowed returns whether a given event type should be sent to Discord.
func (c *Config) discordEventAllowed(eventType string) bool {
	if !c.Discord.Enabled {
		return false
	}
	for _, e := range c.Discord.Events {
		if e == "all" || e == eventType {
			return true
		}
	}
	return false
}

// rconEventEnabled returns whether a given RCON event type should be registered.
func (c *Config) rconEventEnabled(eventType string) bool {
	if !c.Events.Enabled {
		return false
	}
	for _, e := range c.Events.Types {
		if e == "all" || e == eventType {
			return true
		}
	}
	return false
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
