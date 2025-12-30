package engine

import (
	"log"
)

// DefaultHooks returns default hooks for an agent (logger + response).
func DefaultHooks() Hooks {
	return Hooks{
		LoggerHook{L: log.Default()},
		NewResponseHook(),
	}
}
