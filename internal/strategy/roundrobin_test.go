package strategy_test

import (
	"testing"
	"time"

	"github.com/abswn/revproxy-go/internal/ban"
	"github.com/abswn/revproxy-go/internal/config"
	"github.com/abswn/revproxy-go/internal/strategy"
)

func TestRoundRobin_CircularSelection(t *testing.T) {
	urls := []config.URLConfig{
		{URL: "http://a.com"},
		{URL: "http://b.com"},
		{URL: "http://c.com"},
	}
	var counter uint32
	bm := ban.NewManager()
	bm.StartEvictionLoop(500 * time.Millisecond)

	seen := make(map[string]bool)
	for i := 0; i < 6; i++ {
		urlCfg, ok := strategy.RoundRobin(urls, &counter, bm)
		if !ok {
			t.Fatalf("Expected valid URL, got none at iteration %d", i)
		}
		seen[urlCfg.URL] = true
	}

	if len(seen) != 3 {
		t.Errorf("Expected to see 3 unique URLs, saw %d", len(seen))
	}
}

func TestRoundRobin_BanLogic(t *testing.T) {
	urls := []config.URLConfig{
		{URL: "http://a.com"},
		{URL: "http://b.com"},
		{URL: "http://c.com"},
	}
	var counter uint32
	bm := ban.NewManager()
	bm.StartEvictionLoop(200 * time.Millisecond)

	// Ban a.com and b.com
	bm.BanURL("http://a.com", 1)
	bm.BanURL("http://b.com", 1)

	// Expect c.com to be selected repeatedly
	for i := 0; i < 3; i++ {
		urlCfg, ok := strategy.RoundRobin(urls, &counter, bm)
		if !ok || urlCfg.URL != "http://c.com" {
			t.Fatalf("Expected http://c.com, got %s", urlCfg.URL)
		}
	}

	// Wait for bans to expire
	time.Sleep(1 * time.Second)

	seen := make(map[string]bool)
	for i := 0; i < 6; i++ {
		urlCfg, ok := strategy.RoundRobin(urls, &counter, bm)
		if !ok {
			t.Fatalf("Expected valid URL, got none at iteration %d", i)
		}
		seen[urlCfg.URL] = true
	}

	if len(seen) != 3 {
		t.Errorf("Expected to see all 3 URLs after bans expired, saw %d", len(seen))
	}
}

func TestRoundRobin_AllBanned(t *testing.T) {
	urls := []config.URLConfig{
		{URL: "http://x.com"},
	}
	var counter uint32
	bm := ban.NewManager()
	bm.BanURL("http://x.com", 1)

	urlCfg, ok := strategy.RoundRobin(urls, &counter, bm)
	if ok {
		t.Errorf("Expected no available URLs, got %s", urlCfg.URL)
	}
}
