package cachehandler

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"

	"monorepo/bin-sentinel-manager/models/asteriskaddress"
)

// newTestHandler wires the handler against an in-process Redis (miniredis) so the SCAN cursor
// loop, the GET, and the TTL read are exercised against real Redis semantics rather than a mock's
// idea of them.
func newTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	server := miniredis.RunT(t)

	return &handler{
		Addr:  server.Addr(),
		Cache: redis.NewClient(&redis.Options{Addr: server.Addr()}),
	}, server
}

func Test_NewHandler(t *testing.T) {
	h := NewHandler("localhost:6379", "password", 3)

	if h == nil {
		t.Fatalf("Wrong match. expect: non-nil handler, got: nil")
	}

	res, ok := h.(*handler)
	if !ok {
		t.Fatalf("Wrong match. expect: *handler, got: %T", h)
	}
	if res.Addr != "localhost:6379" {
		t.Errorf("Wrong match. expect: localhost:6379, got: %s", res.Addr)
	}
	if res.Password != "password" {
		t.Errorf("Wrong match. expect: password, got: %s", res.Password)
	}
	if res.DB != 3 {
		t.Errorf("Wrong match. expect: 3, got: %d", res.DB)
	}
	if res.Cache == nil {
		t.Errorf("Wrong match. expect: an initialized redis client, got: nil")
	}
}

func Test_Connect(t *testing.T) {
	h, server := newTestHandler(t)

	if err := h.Connect(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	server.Close()

	if err := h.Connect(); err == nil {
		t.Errorf("Wrong match. expect: error against a closed server, got: nil")
	}
}

func Test_AsteriskAddressInternalScan(t *testing.T) {
	tests := []struct {
		name string

		keys map[string]struct {
			value string
			ttl   time.Duration
		}

		expect []*asteriskaddress.AsteriskAddress
	}{
		{
			name: "single_key",

			keys: map[string]struct {
				value string
				ttl   time.Duration
			}{
				asteriskaddress.Key("3e:50:6b:43:bb:32"): {"172.24.0.101", asteriskaddress.TTL},
			},

			expect: []*asteriskaddress.AsteriskAddress{
				{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
			},
		},
		{
			name: "several_keys",

			keys: map[string]struct {
				value string
				ttl   time.Duration
			}{
				asteriskaddress.Key("3e:50:6b:43:bb:32"): {"172.24.0.101", asteriskaddress.TTL},
				asteriskaddress.Key("72:ce:24:e6:51:2f"): {"172.24.0.102", asteriskaddress.TTL},
			},

			expect: []*asteriskaddress.AsteriskAddress{
				{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
				{ID: "72:ce:24:e6:51:2f", Address: "172.24.0.102", TTL: asteriskaddress.TTL},
			},
		},
		{
			name: "two_generations_share_one_ip",

			keys: map[string]struct {
				value string
				ttl   time.Duration
			}{
				asteriskaddress.Key("de:ad:de:ad:de:ad"): {"172.24.0.101", 12 * time.Hour},
				asteriskaddress.Key("3e:50:6b:43:bb:32"): {"172.24.0.101", asteriskaddress.TTL},
			},

			// BOTH are returned: this layer reports what Redis holds, ttl included. Deciding which
			// one is the current occupant is the refresh loop's job, not the cache handler's.
			expect: []*asteriskaddress.AsteriskAddress{
				{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: asteriskaddress.TTL},
				{ID: "de:ad:de:ad:de:ad", Address: "172.24.0.101", TTL: 12 * time.Hour},
			},
		},
		{
			name: "no_keys",

			keys: map[string]struct {
				value string
				ttl   time.Duration
			}{},

			expect: []*asteriskaddress.AsteriskAddress{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler(t)
			ctx := context.Background()

			for key, entry := range tt.keys {
				if err := h.Cache.Set(ctx, key, entry.value, entry.ttl).Err(); err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.AsteriskAddressInternalScan(ctx)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })

			if len(res) != len(tt.expect) {
				t.Fatalf("Wrong match. expect: %d entries, got: %d (%+v)", len(tt.expect), len(res), res)
			}
			for i := range tt.expect {
				if res[i].ID != tt.expect[i].ID {
					t.Errorf("Wrong match at %d. expect: %s, got: %s", i, tt.expect[i].ID, res[i].ID)
				}
				if res[i].Address != tt.expect[i].Address {
					t.Errorf("Wrong match at %d. expect: %s, got: %s", i, tt.expect[i].Address, res[i].Address)
				}
				if res[i].TTL != tt.expect[i].TTL {
					t.Errorf("Wrong match at %d. expect: %v, got: %v", i, tt.expect[i].TTL, res[i].TTL)
				}
			}
		})
	}
}

// Test_AsteriskAddressInternalScan_ignoresForeignKeys pins the SCAN glob: unrelated keys living in
// the same Redis database must not be reported as asterisk addresses.
func Test_AsteriskAddressInternalScan_ignoresForeignKeys(t *testing.T) {
	h, _ := newTestHandler(t)
	ctx := context.Background()

	foreign := []string{
		"asterisk.3e:50:6b:43:bb:32.address-external",
		"kamailio.3e:50:6b:43:bb:32.address-internal",
		"call.abc-123",
		"asterisk..address-internal",
	}
	for _, key := range foreign {
		if err := h.Cache.Set(ctx, key, "x", asteriskaddress.TTL).Err(); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	if err := h.Cache.Set(ctx, asteriskaddress.Key("3e:50:6b:43:bb:32"), "172.24.0.101", asteriskaddress.TTL).Err(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res, err := h.AsteriskAddressInternalScan(ctx)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if len(res) != 1 {
		t.Fatalf("Wrong match. expect: 1 entry, got: %d (%+v)", len(res), res)
	}
	if res[0].ID != "3e:50:6b:43:bb:32" {
		t.Errorf("Wrong match. expect: 3e:50:6b:43:bb:32, got: %s", res[0].ID)
	}
}

// Test_AsteriskAddressInternalScan_skipsKeysWithoutExpiry pins the ttl<0 guard: a key with no
// expiry cannot be proxy-managed (the proxy always Sets a 24h TTL), so it must not be treated as
// evidence about the current occupant of an IP.
func Test_AsteriskAddressInternalScan_skipsKeysWithoutExpiry(t *testing.T) {
	h, _ := newTestHandler(t)
	ctx := context.Background()

	if err := h.Cache.Set(ctx, asteriskaddress.Key("no-expiry"), "172.24.0.101", 0).Err(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res, err := h.AsteriskAddressInternalScan(ctx)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if len(res) != 0 {
		t.Errorf("Wrong match. expect: 0 entries, got: %d (%+v)", len(res), res)
	}
}

// Test_AsteriskAddressInternalScan_reportsDecayingTTL pins that the REMAINING ttl (not the
// configured one) is what reaches the caller -- the whole freshness filter depends on it.
func Test_AsteriskAddressInternalScan_reportsDecayingTTL(t *testing.T) {
	h, server := newTestHandler(t)
	ctx := context.Background()

	if err := h.Cache.Set(ctx, asteriskaddress.Key("3e:50:6b:43:bb:32"), "172.24.0.101", asteriskaddress.TTL).Err(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	server.FastForward(time.Hour)

	res, err := h.AsteriskAddressInternalScan(ctx)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if len(res) != 1 {
		t.Fatalf("Wrong match. expect: 1 entry, got: %d", len(res))
	}
	if res[0].TTL != asteriskaddress.TTL-time.Hour {
		t.Errorf("Wrong match. expect: %v, got: %v", asteriskaddress.TTL-time.Hour, res[0].TTL)
	}
	if res[0].IsFresh() {
		t.Errorf("Wrong match. expect: a one-hour-old key to be stale, got: fresh")
	}
}

// Test_AsteriskAddressInternalScan_expiredKeysDisappear pins the interaction with Redis's own
// expiry: a key past its TTL is simply gone, not returned with a negative ttl.
func Test_AsteriskAddressInternalScan_expiredKeysDisappear(t *testing.T) {
	h, server := newTestHandler(t)
	ctx := context.Background()

	if err := h.Cache.Set(ctx, asteriskaddress.Key("3e:50:6b:43:bb:32"), "172.24.0.101", asteriskaddress.TTL).Err(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	server.FastForward(asteriskaddress.TTL + time.Minute)

	res, err := h.AsteriskAddressInternalScan(ctx)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Wrong match. expect: 0 entries, got: %d (%+v)", len(res), res)
	}
}

// Test_AsteriskAddressInternalScan_multipleCursorRounds pins the SCAN cursor loop: with more keys
// than one COUNT batch, every key must still be returned exactly once.
func Test_AsteriskAddressInternalScan_multipleCursorRounds(t *testing.T) {
	h, _ := newTestHandler(t)
	ctx := context.Background()

	const total = scanBatchSize * 3
	for i := 0; i < total; i++ {
		id := fmt.Sprintf("id-%04d", i)
		if err := h.Cache.Set(ctx, asteriskaddress.Key(id), fmt.Sprintf("172.24.%d.%d", i/256, i%256), asteriskaddress.TTL).Err(); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	res, err := h.AsteriskAddressInternalScan(ctx)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if len(res) != total {
		t.Fatalf("Wrong match. expect: %d entries, got: %d", total, len(res))
	}

	seen := map[string]bool{}
	for _, entry := range res {
		if seen[entry.ID] {
			t.Errorf("Wrong match. expect: each key once, got a duplicate: %s", entry.ID)
		}
		seen[entry.ID] = true
	}
}

// Test_AsteriskAddressInternalScan_serverFailure pins that a real Redis failure surfaces as an
// error rather than as an empty result -- the refresh loop MUST be able to tell "redis is down"
// apart from "there are no fresh candidates", or it would treat a blip as evidence.
func Test_AsteriskAddressInternalScan_serverFailure(t *testing.T) {
	h, server := newTestHandler(t)
	ctx := context.Background()

	if err := h.Cache.Set(ctx, asteriskaddress.Key("3e:50:6b:43:bb:32"), "172.24.0.101", asteriskaddress.TTL).Err(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	server.Close()

	res, err := h.AsteriskAddressInternalScan(ctx)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil (%+v)", res)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil result on error, got: %+v", res)
	}
}
