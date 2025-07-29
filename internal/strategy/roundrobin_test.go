package strategy_test

import (
	"sync"
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
	for i := range 6 {
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
	bm.BanURL("http://a.com", 1*time.Second)
	bm.BanURL("http://b.com", 1*time.Second)

	// Expect c.com to be selected repeatedly
	for range 3 {
		urlCfg, ok := strategy.RoundRobin(urls, &counter, bm)
		if !ok || urlCfg.URL != "http://c.com" {
			t.Fatalf("Expected http://c.com, got %s", urlCfg.URL)
		}
	}

	// Wait for bans to expire
	time.Sleep(1 * time.Second)

	seen := make(map[string]bool)
	for i := range 6 {
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
	bm.BanURL("http://x.com", 1*time.Second)

	urlCfg, ok := strategy.RoundRobin(urls, &counter, bm)
	if ok {
		t.Errorf("Expected no available URLs, got %s", urlCfg.URL)
	}
}

func TestRoundRobin_EmptyTargets(t *testing.T) {
	var counter uint32
	bm := ban.NewManager()

	urlCfg, ok := strategy.RoundRobin([]config.URLConfig{}, &counter, bm)
	if ok {
		t.Errorf("Expected no available URLs, got %s", urlCfg.URL)
	}
}

func TestRoundRobin_LargeCounter(t *testing.T) {
	urls := []config.URLConfig{
		{URL: "http://a.com"},
		{URL: "http://b.com"},
	}
	var counter uint32 = 4_294_967_290 // Close to math.MaxUint32
	bm := ban.NewManager()

	urlCfg, ok := strategy.RoundRobin(urls, &counter, bm)
	if !ok {
		t.Fatalf("Expected valid URL, got none")
	}

	if urlCfg.URL != "http://a.com" && urlCfg.URL != "http://b.com" {
		t.Errorf("Unexpected URL selected: %s", urlCfg.URL)
	}
}

func TestRoundRobin_Concurrency(t *testing.T) {
	urls := []config.URLConfig{
		{URL: "http://a.com"},
		{URL: "http://b.com"},
		{URL: "http://c.com"},
	}
	var counter uint32
	bm := ban.NewManager()

	seen := make(map[string]int)
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			urlCfg, ok := strategy.RoundRobin(urls, &counter, bm)
			if ok {
				mu.Lock()
				seen[urlCfg.URL]++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if len(seen) != 3 {
		t.Errorf("Expected all 3 URLs to be selected, saw %d", len(seen))
	}
}
