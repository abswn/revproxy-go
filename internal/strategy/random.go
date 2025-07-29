package strategy

import (
	"math/rand"
	"sync"
	"time"

	"github.com/abswn/revproxy-go/internal/ban"
	"github.com/abswn/revproxy-go/internal/config"
)

var (
	randomRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	randomMu   sync.Mutex
)

// Random strategy selects a (non-banned) target URL randomly
func Random(targets []config.URLConfig, bm *ban.BanManager) (config.URLConfig, bool) {

	var validTargets []config.URLConfig

	// Filter out banned URLs and prepare a list with valid weights
	for _, target := range targets {
		if !bm.IsBanned(target.URL) {
			validTargets = append(validTargets, target)
		}
	}

	if len(validTargets) == 0 {
		return config.URLConfig{}, false
	}

	// Choose a random number in [0, len(validTargets)]
	randomMu.Lock()
	targetIndex := randomRand.Intn(len(validTargets))
	randomMu.Unlock()

	return validTargets[targetIndex], true
}
