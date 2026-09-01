package core_config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	TimeZone   *time.Location
	ServerPort string
}

func NewConfig() (*Config, error) {
	tz := os.Getenv("TIME_ZONE")
	if tz == "" {
		tz = "UTC"
	}

	zone, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("load time zone: %s: %w", tz, err)
	}

	port := os.Getenv("TCP_PORT")
	if port == "" {
		port = "8080"
	}
	if port[0] != ':' {
		port = ":" + port
	}

	return &Config{
		TimeZone:   zone,
		ServerPort: port,
	}, nil
}

func NewConfigMust() *Config {
	config, err := NewConfig()
	if err != nil {
		panic(fmt.Errorf("get core config: %w", err))
	}
	return config
}
