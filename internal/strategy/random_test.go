package strategy_test

import (
	"testing"
	"time"

	"github.com/abswn/revproxy-go/internal/ban"
	"github.com/abswn/revproxy-go/internal/config"
	"github.com/abswn/revproxy-go/internal/strategy"
)

func TestRandom_AllValid(t *testing.T) {
	bm := ban.NewManager()
	targets := []config.URLConfig{
		{URL: "http://backend1.com"},
		{URL: "http://backend2.com"},
		{URL: "http://backend3.com"},
	}

	// Call multiple times to verify all URLs are eventually picked
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		chosen, ok := strategy.Random(targets, bm)
		if !ok {
			t.Fatal("Expected valid backend but got none")
		}
		seen[chosen.URL] = true
	}

	if len(seen) != len(targets) {
		t.Errorf("Expected all targets to be selected over time, but got: %v", seen)
	}
}

func TestRandom_SomeBanned(t *testing.T) {
	bm := ban.NewManager()
	targets := []config.URLConfig{
		{URL: "http://backend1.com"},
		{URL: "http://backend2.com"},
		{URL: "http://backend3.com"},
	}

	// Ban one of the URLs
	bm.BanURL("http://backend2.com", 5*time.Second)

	// Ensure we never get the banned one
	for i := 0; i < 50; i++ {
		chosen, ok := strategy.Random(targets, bm)
		if !ok {
			t.Fatal("Expected valid backend but got none")
		}
		if chosen.URL == "http://backend2.com" {
			t.Errorf("Got banned URL: %s", chosen.URL)
		}
	}
}

func TestRandom_AllBanned(t *testing.T) {
	bm := ban.NewManager()
	targets := []config.URLConfig{
		{URL: "http://backend1.com"},
		{URL: "http://backend2.com"},
	}

	bm.BanURL("http://backend1.com", 5*time.Second)
	bm.BanURL("http://backend2.com", 5*time.Second)

	_, ok := strategy.Random(targets, bm)
	if ok {
		t.Error("Expected no valid backends, but got one")
	}
}
