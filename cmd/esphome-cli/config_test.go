package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseYAMLWithSecrets(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `
esphome:
  name: test-bt
api:
  encryption:
    key: !secret api_encryption_key
wifi:
  ssid: !secret wifi_ssid
  password: !secret wifi_password
`
	secretsContent := `
wifi_ssid: "test-ssid"
wifi_password: 1234
api_encryption_key: "test-key"
`

	configPath := filepath.Join(tmpDir, "config.yaml")
	secretsPath := filepath.Join(tmpDir, "secrets.yaml")

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	if err := os.WriteFile(secretsPath, []byte(secretsContent), 0644); err != nil {
		t.Fatalf("failed to write secrets: %v", err)
	}

	cfg, err := parseYaml(configPath)
	if err != nil {
		t.Fatalf("parseYaml failed: %v", err)
	}

	if cfg.ESPHome.Name != "test-bt" {
		t.Errorf("expected name 'test-bt', got %q", cfg.ESPHome.Name)
	}
	if cfg.API.Encryption.Key != "test-key" {
		t.Errorf("expected key 'test-key', got %q", cfg.API.Encryption.Key)
	}
	if cfg.WiFi.SSID != "test-ssid" {
		t.Errorf("expected ssid 'test-ssid', got %q", cfg.WiFi.SSID)
	}
	if cfg.WiFi.Password != "1234" {
		t.Errorf("expected password '1234', got %q", cfg.WiFi.Password)
	}
}

func TestParseYAMLWithoutSecrets(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `
esphome:
  name: test-no-secrets
wifi:
  ssid: my-ssid
`
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := parseYaml(configPath)
	if err != nil {
		t.Fatalf("parseYaml failed: %v", err)
	}

	if cfg.ESPHome.Name != "test-no-secrets" {
		t.Errorf("expected name 'test-no-secrets', got %q", cfg.ESPHome.Name)
	}
	if cfg.WiFi.SSID != "my-ssid" {
		t.Errorf("expected ssid 'my-ssid', got %q", cfg.WiFi.SSID)
	}
}
