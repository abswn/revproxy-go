package strategy_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/abswn/revproxy-go/internal/ban"
	"github.com/abswn/revproxy-go/internal/config"
	"github.com/abswn/revproxy-go/internal/strategy"
)

func TestRandom_AllValid(t *testing.T) {
	bm := ban.NewManager()
	targets := []config.URLConfig{
		{URL: "http://example.com"},
		{URL: "http://example.org"},
		{URL: "http://example.net"},
	}

	// Call multiple times to verify all URLs are eventually picked
	seen := make(map[string]bool)
	for range 100 {
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
		{URL: "http://example.com"},
		{URL: "http://example.org"},
		{URL: "http://example.net"},
	}

	// Ban one of the URLs
	bm.BanURL("http://example.org", 5*time.Second)

	// Ensure we never get the banned one
	for range 50 {
		chosen, ok := strategy.Random(targets, bm)
		if !ok {
			t.Fatal("Expected valid backend but got none")
		}
		if chosen.URL == "http://example.org" {
			t.Errorf("Got banned URL: %s", chosen.URL)
		}
	}
}

func TestRandom_AllBanned(t *testing.T) {
	bm := ban.NewManager()
	targets := []config.URLConfig{
		{URL: "http://example.com"},
		{URL: "http://example.org"},
	}

	bm.BanURL("http://example.com", 5*time.Second)
	bm.BanURL("http://example.org", 5*time.Second)

	_, ok := strategy.Random(targets, bm)
	if ok {
		t.Error("Expected no valid backends, but got one")
	}
}

func TestRandom_EmptyTargets(t *testing.T) {
	bm := ban.NewManager()
	targets := []config.URLConfig{}

	_, ok := strategy.Random(targets, bm)
	if ok {
		t.Error("Expected no valid backends, but got one")
	}
}

func TestRandom_SingleTarget(t *testing.T) {
	bm := ban.NewManager()
	targets := []config.URLConfig{
		{URL: "http://example.com"},
	}

	chosen, ok := strategy.Random(targets, bm)
	if !ok {
		t.Fatal("Expected valid backend but got none")
	}
	if chosen.URL != "http://example.com" {
		t.Errorf("Expected the only target, but got: %s", chosen.URL)
	}
}

func TestRandom_StatisticalDistribution(t *testing.T) {
	bm := ban.NewManager()
	targets := []config.URLConfig{
		{URL: "http://example.com"},
		{URL: "http://example.org"},
		{URL: "http://example.net"},
	}

	// Count occurrences of each target
	counts := make(map[string]int)
	totalCalls := 10000 // Increase sample size for better statistical reliability

	for range totalCalls {
		chosen, ok := strategy.Random(targets, bm)
		if !ok {
			t.Fatal("Expected valid backend but got none")
		}
		counts[chosen.URL]++
	}

	// Check that all targets are selected and distribution is roughly uniform
	for _, target := range targets {
		if counts[target.URL] == 0 {
			t.Errorf("Target %s was never selected", target.URL)
		}
	}

	// Optional: Check uniformity (e.g., within 10% of expected frequency)
	expectedFrequency := totalCalls / len(targets)
	for url, count := range counts {
		if count < expectedFrequency*90/100 || count > expectedFrequency*110/100 {
			t.Errorf("Target %s selected %d times, expected around %d", url, count, expectedFrequency)
		}
	}
}

func TestRandom_Concurrency(t *testing.T) {
	bm := ban.NewManager()
	targets := []config.URLConfig{
		{URL: "http://example.com"},
		{URL: "http://example.org"},
		{URL: "http://example.net"},
	}

	const numGoroutines = 10
	const iterations = 100
	var wg sync.WaitGroup

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, ok := strategy.Random(targets, bm)
				if !ok {
					t.Error("Expected valid backend but got none")
				}
			}
		}()
	}

	wg.Wait()
}

func TestRandom_ExpiredBan(t *testing.T) {
	bm := ban.NewManager()
	targets := []config.URLConfig{
		{URL: "http://example.com"},
		{URL: "http://example.org"},
	}

	bm.BanURL("http://example.com", 1*time.Second)

	// Initially, the banned URL should not be selected
	for range 10 {
		chosen, ok := strategy.Random(targets, bm)
		if !ok {
			t.Fatal("Expected valid backend but got none")
		}
		if chosen.URL == "http://example.com" {
			t.Errorf("Got banned URL: %s", chosen.URL)
		}
	}

	// Wait for the ban to expire
	time.Sleep(2 * time.Second)

	// Now, the previously banned URL should be selectable
	seen := make(map[string]bool)
	for range 10 {
		chosen, ok := strategy.Random(targets, bm)
		if !ok {
			t.Fatal("Expected valid backend but got none")
		}
		seen[chosen.URL] = true
	}

	if len(seen) != len(targets) {
		t.Errorf("Expected all targets to be selectable after ban expiration, but got: %v", seen)
	}
}

func TestRandom_LargeInput(t *testing.T) {
	bm := ban.NewManager()
	var targets []config.URLConfig
	for i := range 10000 {
		targets = append(targets, config.URLConfig{URL: fmt.Sprintf("http://example%d.com", i)})
	}

	chosen, ok := strategy.Random(targets, bm)
	if !ok {
		t.Fatal("Expected valid backend but got none")
	}

	if chosen.URL == "" {
		t.Error("Expected a valid URL, but got an empty string")
	}
}

func TestRandom_RepeatedBans(t *testing.T) {
	bm := ban.NewManager()
	targets := []config.URLConfig{
		{URL: "http://example.com"},
		{URL: "http://example.org"},
	}

	bm.BanURL("http://example.com", 1*time.Second)

	// Initially, the banned URL should not be selected
	for i := 0; i < 10; i++ {
		chosen, ok := strategy.Random(targets, bm)
		if !ok {
			t.Fatal("Expected valid backend but got none")
		}
		if chosen.URL == "http://example.com" {
			t.Errorf("Got banned URL: %s", chosen.URL)
		}
	}

	// Wait for the ban to expire
	time.Sleep(2 * time.Second)

	// Verify that http://example.com is now selectable
	if bm.IsBanned("http://example.com") {
		t.Errorf("Expected http://example.com to be unbanned, but it is still banned")
	}

	// Ban the URL again
	bm.BanURL("http://example.com", 1*time.Second)

	// Ensure it is banned again
	for i := 0; i < 10; i++ {
		chosen, ok := strategy.Random(targets, bm)
		if !ok {
			t.Fatal("Expected valid backend but got none")
		}
		if chosen.URL == "http://example.com" {
			t.Errorf("Got banned URL: %s", chosen.URL)
		}
	}
}

func TestRandom_NoValidTargets(t *testing.T) {
	bm := ban.NewManager()
	targets := []config.URLConfig{
		{URL: "http://example.com"},
		{URL: "http://example.org"},
	}

	// Ban all targets
	bm.BanURL("http://example.com", 5*time.Second)
	bm.BanURL("http://example.org", 5*time.Second)

	_, ok := strategy.Random(targets, bm)
	if ok {
		t.Error("Expected no valid backends, but got one")
	}
}
