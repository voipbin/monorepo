package servicehandler

import "testing"

// Test_DirectScopeVersionCurrent_isALever turns the warning at the constant's
// declaration into something enforced rather than merely written down.
//
// The value is 2 rather than 1 because an earlier three-stage rollout would
// have issued 1 first. That plan was dropped and no token was ever minted with
// 1, which makes 2 look like an off-by-one. It is not.
//
// Lowering it is silently safe *today* -- every live token carries 2, so the
// buildJWTIdentity check still passes and nothing visibly breaks. What it
// destroys is the lever: the next time the scope contract genuinely changes,
// bumping to 2 would no-op against every already-minted token instead of
// invalidating them, and the clients would never receive the 401 they treat as
// a reboot signal.
//
// Every other test in this package references the constant symbolically, so
// none of them can notice a change to its value. This one can.
func Test_DirectScopeVersionCurrent_isALever(t *testing.T) {
	const shipped = 2

	if DirectScopeVersionCurrent != shipped {
		t.Fatalf(
			"DirectScopeVersionCurrent is a rollout lever, not a counter.\n"+
				"expect: %d, got: %d\n"+
				"Raising it invalidates every outstanding direct token in one step, which is intended.\n"+
				"Lowering it looks harmless and is not: see the comment at the declaration and design section 9.1.",
			shipped, DirectScopeVersionCurrent,
		)
	}
}
