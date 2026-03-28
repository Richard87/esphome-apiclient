package main

import (
	"fmt"
	"os"
	"path/filepath"

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
		SSID     string `yaml:"ssid"`
		Password string `yaml:"password"`
	} `yaml:"wifi"`
}

func parseYaml(path string) (*ESPHomeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Load secrets if they exist
	secrets, err := loadSecrets(filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("load secrets: %w", err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("unmarshal to node: %w", err)
	}

	// Substitute !secret tags
	if err := substituteSecrets(&root, secrets); err != nil {
		return nil, fmt.Errorf("substitute secrets: %w", err)
	}

	var cfg ESPHomeConfig
	if err := root.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return &cfg, nil
}

func loadSecrets(dir string) (map[string]string, error) {
	secretsPath := filepath.Join(dir, "secrets.yaml")
	data, err := os.ReadFile(secretsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("read secrets file: %w", err)
	}

	var secrets map[string]string
	if err := yaml.Unmarshal(data, &secrets); err != nil {
		return nil, fmt.Errorf("unmarshal secrets: %w", err)
	}
	return secrets, nil
}

func substituteSecrets(node *yaml.Node, secrets map[string]string) error {
	if node.Tag == "!secret" {
		val, ok := secrets[node.Value]
		if !ok {
			return fmt.Errorf("secret %q not found", node.Value)
		}
		node.Value = val
		node.Tag = "!!str" // Standard string tag
	}
	for _, child := range node.Content {
		if err := substituteSecrets(child, secrets); err != nil {
			return err
		}
	}
	return nil
}
