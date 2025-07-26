package strategy

import (
	"sync/atomic"

	"github.com/abswn/revproxy-go/internal/ban"
	"github.com/abswn/revproxy-go/internal/config"
)

// RoundRobin selects the next available (non-banned) URL using round-robin logic.
func RoundRobin(targets []config.URLConfig, counter *uint32, bm *ban.BanManager) (config.URLConfig, bool) {
	maxAttempts := len(targets)
	for range maxAttempts {
		index := atomic.AddUint32(counter, 1) - 1
		target := targets[int(index)%len(targets)]
		if !bm.IsBanned(target.URL) {
			return target, true
		}
	}
	return config.URLConfig{}, false // All URLs are banned
}
