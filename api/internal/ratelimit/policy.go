package ratelimit

import "time"

// P0 defaults — tune via env later if needed.
const (
	LoginIPLimit       = 15
	LoginIPWindow      = 15 * time.Minute
	LoginFailLimit     = 5
	LoginFailWindow    = 15 * time.Minute
	RegisterIPLimit    = 5
	RegisterIPWindow   = time.Hour
	PostActorLimit     = 8
	PostActorWindow    = time.Minute
	ThreadActorLimit   = 3
	ThreadActorWindow  = 10 * time.Minute
	AuthEmailIPLimit   = 10
	AuthEmailIPWindow  = time.Hour
	AuthEmailAddrLimit = 3
	AuthEmailAddrWindow = time.Hour
)