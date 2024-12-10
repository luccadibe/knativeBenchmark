package config

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Targets []Target `yaml:"targets"`
	Rate    Rate     `yaml:"rate"`
	BaseURL string   `yaml:"baseUrl"`
}

type Target struct {
	URL        string            `yaml:"url"`
	Headers    map[string]string `yaml:"headers,omitempty"`
	Weight     int               `yaml:"weight"`
	HostHeader string            `yaml:"-"` // Used in dev mode
}

// Custom duration type for YAML parsing
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}

	duration, err := time.ParseDuration(str)
	if err != nil {
		return err
	}

	d.Duration = duration
	return nil
}

type Rate struct {
	RequestsPerSecond int      `yaml:"requestsPerSecond"`
	Duration          Duration `yaml:"duration"`
	Rampup            Duration `yaml:"rampup"`
}

func Load(path string, devMode bool) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if devMode {
		cfg.BaseURL = "http://localhost:8080"
		// In dev mode, URLs in targets are used as Host headers
		for i := range cfg.Targets {
			cfg.Targets[i].HostHeader = cfg.Targets[i].URL
			cfg.Targets[i].URL = cfg.BaseURL
		}
		log.Printf("Dev mode enabled: using %s as base URL", cfg.BaseURL)
	}

	log.Printf("Loaded rate config: RPS=%d, Duration=%v, Rampup=%v",
		cfg.Rate.RequestsPerSecond,
		cfg.Rate.Duration.Duration,
		cfg.Rate.Rampup.Duration)

	return &cfg, nil
}