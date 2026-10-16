package ai

import (
	"context"
	"net/netip"
)

// SetEngineResolverForTest replaces the resolver used by ValidateEngineBaseURL and returns a
// function that restores the previous one.
//
// FOR TESTS ONLY. NEVER call this from production code: it exists because tests of other
// packages cannot reach the package level variable. A replaced resolver defeats the
// DNS-based address check.
func SetEngineResolverForTest(f func(ctx context.Context, host string) ([]netip.Addr, error)) (restore func()) {
	engineResolverMu.Lock()
	prev := engineResolver
	engineResolver = f
	engineResolverMu.Unlock()

	return func() {
		engineResolverMu.Lock()
		engineResolver = prev
		engineResolverMu.Unlock()
	}
}
