package ban

import (
	"testing"
	"time"
)

func TestBanAndIsBanned(t *testing.T) {
	manager := NewManager()
	url := "http://example.com"
	duration := 100 * time.Millisecond

	manager.BanURL(url, duration)
	if !manager.IsBanned(url) {
		t.Errorf("Expected URL %s to be banned", url)
	}

	time.Sleep(duration + 10*time.Millisecond)

	if manager.IsBanned(url) {
		t.Errorf("Expected URL %s to be unbanned after expiry", url)
	}
}

func TestEvictionLoopRemovesExpiredBan(t *testing.T) {
	manager := NewManager()
	manager.StartEvictionLoop(50 * time.Millisecond)

	url := "http://example.com"
	manager.BanURL(url, 30*time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	if manager.IsBanned(url) {
		t.Errorf("Expected eviction loop to remove expired ban for %s", url)
	}
}
