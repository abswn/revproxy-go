package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMainConfig(t *testing.T) {
	tmp := `port: 1234
https_cert_path: ""
https_key_path: ""
log:
  level: "debug"
  output: "stdout"
  format: "json"
`

	path := "test_main_config.yaml"
	os.WriteFile(path, []byte(tmp), 0644)
	defer os.Remove(path)

	cfg, err := LoadMainConfig(path)
	if err != nil {
		t.Fatalf("Failed to load main config: %v", err)
	}

	if cfg.Port != 1234 || cfg.Log.Level != "debug" {
		t.Errorf("Unexpected config values: %+v", cfg)
	}
}

func TestLoadEnabledEndpointConfigs(t *testing.T) {
	dir := "testconfigs"
	os.Mkdir(dir, 0755)
	defer os.RemoveAll(dir)

	valid := `enable: true
endpoints:
  "/x":
    strategy: round-robin
    urls:
      - url: "http://a.com"
exclusions: []
`
	invalid := `enable: false
endpoints: {}
exclusions: []
`

	os.WriteFile(filepath.Join(dir, "site1.yaml"), []byte(valid), 0644)
	os.WriteFile(filepath.Join(dir, "site2.yaml"), []byte(invalid), 0644)
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("dummy"), 0644)

	cfgs, err := LoadEnabledEndpointConfigs(dir)
	if err != nil {
		t.Fatalf("Failed to load endpoint configs: %v", err)
	}

	if len(cfgs) != 1 {
		t.Errorf("Expected 1 enabled config, got %d", len(cfgs))
	}
}
