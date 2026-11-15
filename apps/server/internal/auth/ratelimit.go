package auth

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Rate limiting is per (ip, username): a burst of attempts, then one per
// refill interval. The bucket itself is golang.org/x/time/rate; this file only
// keeps one limiter per key and forgets the idle ones. Counters live in memory,
// so a restart forgets them, which is acceptable for a single self-hosted box.
const (
	loginBurst       = 5
	loginRefill      = 30 * time.Second
	limiterEvictIdle = 10 * time.Minute
	limiterMaxKeys   = 10_000
)

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Limiter allows a burst of attempts and then one attempt per refill interval.
type Limiter struct {
	mu      sync.Mutex
	entries map[string]*entry
	now     func() time.Time
}

func NewLimiter(now func() time.Time) *Limiter {
	if now == nil {
		now = time.Now
	}
	return &Limiter{entries: map[string]*entry{}, now: now}
}

// Allow consumes one attempt for key and reports whether it was permitted.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	e, ok := l.entries[key]
	if !ok {
		// A full map means something is hammering with varying keys; dropping
		// the idle entries is cheaper than growing without bound.
		if len(l.entries) >= limiterMaxKeys {
			l.evictLocked(now)
		}
		e = &entry{limiter: rate.NewLimiter(rate.Every(loginRefill), loginBurst)}
		l.entries[key] = e
	}
	e.lastSeen = now
	// AllowN with an explicit time, so tests drive the clock instead of
	// sleeping.
	return e.limiter.AllowN(now, 1)
}

// Reset clears the bucket for key; called after a successful login so a user
// who simply mistyped is not punished afterwards.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

func (l *Limiter) evictLocked(now time.Time) {
	for key, e := range l.entries {
		if now.Sub(e.lastSeen) > limiterEvictIdle {
			delete(l.entries, key)
		}
	}
}

// size reports how many keys are tracked; tests use it to check eviction.
func (l *Limiter) size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}
