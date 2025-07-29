package strategy

import (
	"math/rand"
	"sync"
	"time"

	"github.com/abswn/revproxy-go/internal/ban"
	"github.com/abswn/revproxy-go/internal/config"
)

var (
	weightedRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	weightedMu   sync.Mutex
)

// Weighted selects a backend URL from the given targets based on a weighted random distribution.
// It filters out banned URLs and targets with zero weight, then selects a target proportionally
// to its weight.
//
// Parameters:
// - targets: A slice of URLConfig objects representing the available targets.
// - bm: A BanManager instance used to check if a URL is banned.
//
// Returns:
// - A URLConfig object representing the selected target.
// - A boolean indicating whether a valid target was found.
func Weighted(targets []config.URLConfig, bm *ban.BanManager) (config.URLConfig, bool) {

	var validTargets []config.URLConfig
	totalWeight := 0

	// Filter out banned URLs and prepare a list with valid weights
	for _, target := range targets {
		// 0 or negative weight is skipped
		if target.Weight <= 0 || bm.IsBanned(target.URL) {
			continue
		}
		validTargets = append(validTargets, target)
		totalWeight += target.Weight
	}

	if totalWeight == 0 || len(validTargets) == 0 {
		return config.URLConfig{}, false // No usable backend
	}

	// Choose a random number in [1, totalWeight)
	weightedMu.Lock()
	targetWeight := weightedRand.Intn(totalWeight) + 1
	weightedMu.Unlock()

	// Iterate and select based on cumulative weight
	cumulative := 0
	for _, target := range validTargets {
		cumulative += target.Weight
		if cumulative >= targetWeight {
			return target, true
		}
	}

	// Should never reach here
	return config.URLConfig{}, false
}
