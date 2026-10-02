package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateTestConfigOptionalFields(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	content := `{
		"localPort": 10808,
		"addressPort": "80",
		"userId": "test-uuid",
		"wsHeaderHost": "test.com",
		"wsHeaderPath": "/test",
		"frontingTimeout": 5.0,
		"nTries": 1,
		"writer": "csv",
		"security": "none",
		"xhttpMode": "packet-up"
	}`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	var c Configuration
	c = c.CreateTestConfig(configPath)

	if c.Config.Sni != "test.com" {
		t.Errorf("expected Sni to default to wsHeaderHost 'test.com', got '%s'", c.Config.Sni)
	}
	if c.Config.Security != "none" {
		t.Errorf("expected Security to be 'none', got '%s'", c.Config.Security)
	}
	if c.Config.XhttpMode != "packet-up" {
		t.Errorf("expected XhttpMode to be 'packet-up', got '%s'", c.Config.XhttpMode)
	}
}
