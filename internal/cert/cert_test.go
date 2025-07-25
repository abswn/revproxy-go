package cert

import (
	"os"
	"testing"
)

func TestEnsureCert(t *testing.T) {
	certPath := "certs/test.crt"
	keyPath := "certs/test.key"

	// Cleanup before and after
	os.Remove(certPath)
	os.Remove(keyPath)
	// defer os.Remove(certPath)
	// defer os.Remove(keyPath)

	// Run
	c, k, err := EnsureCert(certPath, keyPath)
	if err != nil {
		t.Fatalf("EnsureCert failed: %v", err)
	}

	// Check files
	if _, err := os.Stat(c); err != nil {
		t.Errorf("Cert file not created: %s", c)
	}
	if _, err := os.Stat(k); err != nil {
		t.Errorf("Key file not created: %s", k)
	}
}
