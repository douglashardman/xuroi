package ratelimit

import (
	"testing"
	"time"
)

func TestAllowWithinWindow(t *testing.T) {
	l := New()
	key := "test"
	window := time.Minute

	for i := 0; i < 3; i++ {
		if !l.Allow(key, 3, window) {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
	if l.Allow(key, 3, window) {
		t.Fatal("fourth attempt should be denied")
	}
	if l.RetryAfter(key) < 1 {
		t.Fatal("retry-after should be positive")
	}
}

func TestBlockedDoesNotIncrement(t *testing.T) {
	l := New()
	key := "blocked"
	window := time.Minute

	if l.Blocked(key, 2, window) {
		t.Fatal("empty key should not be blocked")
	}
	l.Hit(key, window)
	if l.Blocked(key, 2, window) {
		t.Fatal("one hit should not block")
	}
	l.Hit(key, window)
	if !l.Blocked(key, 2, window) {
		t.Fatal("two hits should block")
	}
	if l.Allow(key, 2, window) {
		t.Fatal("allow should also deny when blocked")
	}
}

func TestAllowResetsAfterWindow(t *testing.T) {
	l := New()
	key := "reset"
	window := 20 * time.Millisecond

	if !l.Allow(key, 1, window) {
		t.Fatal("first attempt should be allowed")
	}
	if l.Allow(key, 1, window) {
		t.Fatal("second attempt should be denied")
	}
	time.Sleep(25 * time.Millisecond)
	if !l.Allow(key, 1, window) {
		t.Fatal("attempt after window should be allowed")
	}
}