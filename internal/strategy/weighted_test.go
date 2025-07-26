package strategy_test

import (
	"testing"
	"time"

	"github.com/abswn/revproxy-go/internal/ban"
	"github.com/abswn/revproxy-go/internal/config"
	"github.com/abswn/revproxy-go/internal/strategy"
)

func TestWeighted_AllValidTargets(t *testing.T) {
	bm := ban.NewManager()
	targets := []config.URLConfig{
		{URL: "http://a.com", Weight: 1},
		{URL: "http://b.com", Weight: 3},
		{URL: "http://c.com", Weight: 6},
	}

	// Count selections over many iterations to verify weight distribution
	counts := make(map[string]int)
	for i := 0; i < 10000; i++ {
		selected, ok := strategy.Weighted(targets, bm)
		if !ok {
			t.Fatal("Expected a valid selection but got false")
		}
		counts[selected.URL]++
	}

	if counts["http://a.com"] < 800 || counts["http://a.com"] > 1200 {
		t.Errorf("http://a.com selected %d times, expected around 1000", counts["http://a.com"])
	}
	if counts["http://b.com"] < 2800 || counts["http://b.com"] > 3200 {
		t.Errorf("http://b.com selected %d times, expected around 3000", counts["http://b.com"])
	}
	if counts["http://c.com"] < 5800 || counts["http://c.com"] > 6200 {
		t.Errorf("http://c.com selected %d times, expected around 6000", counts["http://c.com"])
	}
}

func TestWeighted_SomeTargetsBanned(t *testing.T) {
	bm := ban.NewManager()
	bm.BanURL("http://a.com", time.Minute)

	targets := []config.URLConfig{
		{URL: "http://a.com", Weight: 5},
		{URL: "http://b.com", Weight: 5},
	}

	counts := make(map[string]int)
	for i := 0; i < 1000; i++ {
		selected, ok := strategy.Weighted(targets, bm)
		if !ok {
			t.Fatal("Expected a valid selection but got false")
		}
		counts[selected.URL]++
	}

	if counts["http://a.com"] != 0 {
		t.Errorf("Banned URL http://a.com should not be selected")
	}
	if counts["http://b.com"] != 1000 {
		t.Errorf("Expected http://b.com to be selected 1000 times, got %d", counts["http://b.com"])
	}
}

func TestWeighted_AllBanned(t *testing.T) {
	bm := ban.NewManager()
	bm.BanURL("http://a.com", time.Minute)
	bm.BanURL("http://b.com", time.Minute)

	targets := []config.URLConfig{
		{URL: "http://a.com", Weight: 3},
		{URL: "http://b.com", Weight: 2},
	}

	_, ok := strategy.Weighted(targets, bm)
	if ok {
		t.Error("Expected false when all URLs are banned")
	}
}

func TestWeighted_ZeroWeights(t *testing.T) {
	bm := ban.NewManager()

	targets := []config.URLConfig{
		{URL: "http://a.com", Weight: 0},
		{URL: "http://b.com", Weight: 0},
	}

	_, ok := strategy.Weighted(targets, bm)
	if ok {
		t.Error("Expected false when all weights are zero")
	}
}

func TestWeighted_MixedZeroWeightAndBanned(t *testing.T) {
	bm := ban.NewManager()
	bm.BanURL("http://b.com", time.Minute)

	targets := []config.URLConfig{
		{URL: "http://a.com", Weight: 0},
		{URL: "http://b.com", Weight: 10},
	}

	_, ok := strategy.Weighted(targets, bm)
	if ok {
		t.Error("Expected false when all targets are banned or zero-weight")
	}
}

func TestWeighted_SingleValidTarget(t *testing.T) {
	bm := ban.NewManager()

	targets := []config.URLConfig{
		{URL: "http://only.com", Weight: 10},
	}

	for i := 0; i < 100; i++ {
		selected, ok := strategy.Weighted(targets, bm)
		if !ok || selected.URL != "http://only.com" {
			t.Error("Expected single valid target to always be selected")
		}
	}
}
