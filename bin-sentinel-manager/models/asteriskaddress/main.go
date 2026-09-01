// Package asteriskaddress models the `asterisk.<asterisk-id>.address-internal` Redis keys that
// voip-asterisk-proxy publishes, and the freshness rule sentinel-manager applies when it reads
// them backwards (IP -> asterisk-id).
//
// The reverse direction is inherently ambiguous without a freshness rule: the key carries a 24h
// TTL that the proxy refreshes every 5 minutes, so a dead generation's key for a given IP can
// coexist with the live generation's key for the SAME IP for up to 24h. Accepting whichever the
// SCAN happens to yield last would bind a live container to a dead generation's id, or the
// reverse (design §3.3 step 2).
package asteriskaddress

import (
	"strings"
	"time"
)

const (
	// keyPrefix / keySuffix bracket the asterisk-id inside the Redis key. Kept in sync with
	// voip-asterisk-proxy/cmd/asterisk-proxy/main.go's `asterisk.%s.address-internal` and with
	// bin-call-manager/pkg/cachehandler's forward lookup of the same key.
	keyPrefix = "asterisk."
	keySuffix = ".address-internal"

	// KeyPattern is the Redis SCAN glob matching every key of this family.
	KeyPattern = keyPrefix + "*" + keySuffix

	// TTL is the full time-to-live voip-asterisk-proxy sets on every Set.
	TTL = 24 * time.Hour

	// RefreshInterval is the cadence at which voip-asterisk-proxy re-Sets the key, restoring the
	// TTL to its full value. NOTE: a stale doc comment next to that code says "every 3 min"; the
	// code is the source of truth and sleeps 5 minutes.
	RefreshInterval = 5 * time.Minute

	// FreshnessMargin is how far below a full TTL a key may sit and still count as belonging to
	// the CURRENT occupant of its IP.
	//
	// It is keyed to the proxy's own 5-minute refresh cadence -- NOT to sentinel's unrelated 10s
	// background-loop cadence -- and deliberately spans more than two refresh intervals: a single
	// missed Set from a Redis blip, a GC pause, or the proxy's own non-ticker Sleep-based drift
	// must not misclassify a genuinely healthy container as stale. Design review widened this
	// from an initially proposed 6 minutes for exactly that reason.
	FreshnessMargin = 12 * time.Minute
)

// AsteriskAddress is one `asterisk.<id>.address-internal` key as read back by a reverse scan.
type AsteriskAddress struct {
	// ID is the asterisk-id embedded in the key (voip-asterisk-proxy derives it from the
	// container's MAC address at runtime).
	ID string
	// Address is the key's value: the instance's internal IP address.
	Address string
	// TTL is the key's REMAINING time-to-live at scan time.
	TTL time.Duration
}

// IsFresh reports whether this key was refreshed recently enough to be treated as evidence about
// the CURRENT occupant of its IP address.
//
// A non-fresh key is not proof of anything: it neither identifies the current occupant nor
// invalidates an id already resolved for that container. Callers must treat a "no fresh
// candidate" result as "this pass learned nothing", never as "forget what you knew"
// (design §3.3's sticky-last-known rule).
func (h *AsteriskAddress) IsFresh() bool {
	return h.TTL >= TTL-FreshnessMargin
}

// Key builds the Redis key for the given asterisk-id.
func Key(id string) string {
	return keyPrefix + id + keySuffix
}

// ParseKey extracts the asterisk-id from a Redis key of this family. It reports false for any key
// that does not match the exact `asterisk.<id>.address-internal` shape, including one with an
// empty id -- an empty id is not an address and must never reach a routing key or a recovery
// call.
func ParseKey(key string) (string, bool) {
	// the length guard is not redundant with the prefix/suffix checks: they can OVERLAP, as in
	// "asterisk.address-internal", where both match but there is no id between them.
	if len(key) < len(keyPrefix)+len(keySuffix) {
		return "", false
	}
	if !strings.HasPrefix(key, keyPrefix) || !strings.HasSuffix(key, keySuffix) {
		return "", false
	}

	id := key[len(keyPrefix) : len(key)-len(keySuffix)]
	if id == "" {
		return "", false
	}

	return id, true
}
