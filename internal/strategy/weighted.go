package strategy

import (
	"math/rand"
	"time"

	"github.com/abswn/revproxy-go/internal/ban"
	"github.com/abswn/revproxy-go/internal/config"
)

var weightedRand = rand.New(rand.NewSource(time.Now().UnixNano()))

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
	for _, u := range targets {
		if u.Weight == 0 || bm.IsBanned(u.URL) {
			continue
		}
		validTargets = append(validTargets, u)
		totalWeight += u.Weight
	}

	if totalWeight == 0 || len(validTargets) == 0 {
		return config.URLConfig{}, false // No usable backend
	}

	// Choose a random number in [1, totalWeight)
	targetWeight := weightedRand.Intn(totalWeight) + 1

	// Iterate and select based on cumulative weight
	cumulative := 0
	for _, u := range validTargets {
		cumulative += u.Weight
		if cumulative >= targetWeight {
			return u, true
		}
	}

	// Should never reach here
	return config.URLConfig{}, false
}
