package provision

import (
	"strings"
	"sync"
	"time"
)

type TicketReplayGuard struct {
	mu   sync.Mutex
	used map[string]int64
	max  int
}

func NewTicketReplayGuard(maxEntries int) *TicketReplayGuard {
	if maxEntries < 128 {
		maxEntries = 128
	}
	return &TicketReplayGuard{
		used: make(map[string]int64),
		max:  maxEntries,
	}
}

func (g *TicketReplayGuard) Use(nonce string, expiresAtUnix int64, now time.Time) bool {
	if g == nil {
		return false
	}
	nonce = strings.TrimSpace(nonce)
	if nonce == "" || expiresAtUnix <= now.Unix() {
		return false
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	g.cleanupLocked(now.Unix())
	if _, exists := g.used[nonce]; exists {
		return false
	}

	if len(g.used) >= g.max {
		g.evictOldestLocked()
	}
	g.used[nonce] = expiresAtUnix
	return true
}

func (g *TicketReplayGuard) cleanupLocked(nowUnix int64) {
	for nonce, expiry := range g.used {
		if expiry <= nowUnix {
			delete(g.used, nonce)
		}
	}
}

func (g *TicketReplayGuard) evictOldestLocked() {
	var oldestNonce string
	var oldestExpiry int64
	for nonce, expiry := range g.used {
		if oldestNonce == "" || expiry < oldestExpiry {
			oldestNonce = nonce
			oldestExpiry = expiry
		}
	}
	if oldestNonce != "" {
		delete(g.used, oldestNonce)
	}
}
