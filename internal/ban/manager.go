package ban

import (
	"sync"
	"time"
)

// BanManager keeps track of banned endpoint indices and their expiry times.
type BanManager struct {
	mu         sync.RWMutex
	bannedURLs map[string]time.Time
}

// NewManager initializes a new BanManager.
func NewManager() *BanManager {
	return &BanManager{
		bannedURLs: make(map[string]time.Time),
	}
}

// BanURL bans the endpoint for the specified duration.
func (m *BanManager) BanURL(url string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bannedURLs[url] = time.Now().Add(duration)
}

// IsBanned checks if the endpoint is currently banned.
func (m *BanManager) IsBanned(url string) bool {
	m.mu.RLock()
	expiry, ok := m.bannedURLs[url]
	m.mu.RUnlock()
	return ok && time.Now().Before(expiry)
}

// StartEvictionLoop starts a background goroutine that removes expired bans periodically.
func (m *BanManager) StartEvictionLoop(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			m.evictExpired()
		}
	}()
}

// evictExpired removes expired entries from the banned list.
func (m *BanManager) evictExpired() {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	for url, expiry := range m.bannedURLs {
		if now.After(expiry) {
			delete(m.bannedURLs, url)
		}
	}
}
