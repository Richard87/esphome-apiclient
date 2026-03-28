package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ESPHomeConfig represents the subset of an ESPHome YAML config we need.
type ESPHomeConfig struct {
	ESPHome struct {
		Name string `yaml:"name"`
	} `yaml:"esphome"`
	API struct {
		Encryption struct {
			Key string `yaml:"key"`
		} `yaml:"encryption"`
	} `yaml:"api"`
	WiFi struct {
		SSID string `yaml:"ssid"`
	} `yaml:"wifi"`
}

func parseYAML(path string) (*ESPHomeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	var cfg ESPHomeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	return &cfg, nil
}
