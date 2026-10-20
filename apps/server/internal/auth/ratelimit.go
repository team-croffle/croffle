package auth

import (
	"sync"
	"time"
)

// Rate limiting is deliberately small: an in-memory token bucket per
// (ip, username). It makes password guessing expensive without pulling in a
// store, and a restart forgetting the counters is acceptable for 0.1.
const (
	loginBurst       = 5
	loginRefill      = 30 * time.Second
	limiterEvictIdle = 10 * time.Minute
	limiterMaxKeys   = 10_000
)

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// Limiter allows a burst of attempts and then one attempt per refill interval.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	now     func() time.Time
}

func NewLimiter(now func() time.Time) *Limiter {
	if now == nil {
		now = time.Now
	}
	return &Limiter{buckets: map[string]*bucket{}, now: now}
}

// Allow consumes one attempt for key and reports whether it was permitted.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	b, ok := l.buckets[key]
	if !ok {
		// A full map means something is hammering with varying keys; dropping
		// the idle entries is cheaper than growing without bound.
		if len(l.buckets) >= limiterMaxKeys {
			l.evictLocked(now)
		}
		b = &bucket{tokens: loginBurst, lastSeen: now}
		l.buckets[key] = b
	}

	b.tokens += now.Sub(b.lastSeen).Seconds() / loginRefill.Seconds()
	if b.tokens > loginBurst {
		b.tokens = loginBurst
	}
	b.lastSeen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Reset clears the bucket for key; called after a successful login so a user
// who simply mistyped is not punished afterwards.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
}

func (l *Limiter) evictLocked(now time.Time) {
	for key, b := range l.buckets {
		if now.Sub(b.lastSeen) > limiterEvictIdle {
			delete(l.buckets, key)
		}
	}
}
