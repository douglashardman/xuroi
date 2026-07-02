package ratelimit

import (
	"sync"
	"time"
)

// Limiter is an in-memory fixed-window counter (single-process v1).
type Limiter struct {
	mu      sync.Mutex
	entries map[string]entry
}

type entry struct {
	count     int
	windowEnd time.Time
}

func New() *Limiter {
	return &Limiter{entries: make(map[string]entry)}
}

// Allow reports whether key is within limit for the window. When allowed, the
// attempt is counted; when denied, the counter is unchanged.
func (l *Limiter) Allow(key string, limit int, window time.Duration) bool {
	if limit <= 0 || window <= 0 {
		return true
	}
	return !l.blocked(key, limit, window, true)
}

// Blocked reports whether key has reached limit without counting a new attempt.
func (l *Limiter) Blocked(key string, limit int, window time.Duration) bool {
	if limit <= 0 || window <= 0 {
		return false
	}
	return l.blocked(key, limit, window, false)
}

// Hit records one attempt against key (e.g. failed login) even when over limit.
func (l *Limiter) Hit(key string, window time.Duration) {
	if window <= 0 {
		return
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	e, ok := l.entries[key]
	if !ok || now.After(e.windowEnd) {
		l.entries[key] = entry{count: 1, windowEnd: now.Add(window)}
		return
	}
	e.count++
	l.entries[key] = e
}

func (l *Limiter) blocked(key string, limit int, window time.Duration, count bool) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	e, ok := l.entries[key]
	if !ok || now.After(e.windowEnd) {
		if count {
			l.entries[key] = entry{count: 1, windowEnd: now.Add(window)}
		}
		return false
	}
	if e.count >= limit {
		return true
	}
	if count {
		e.count++
		l.entries[key] = e
	}
	return false
}

// RetryAfter returns whole seconds until the current window resets (minimum 1).
func (l *Limiter) RetryAfter(key string) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	e, ok := l.entries[key]
	if !ok {
		return 1
	}
	sec := int(time.Until(e.windowEnd).Seconds())
	if sec < 1 {
		return 1
	}
	return sec
}