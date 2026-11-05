package carriertrackhandler

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock { return &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)} }
func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}
func (c *fakeClock) add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func Test_lruTTL_expiry(t *testing.T) {
	clk := newFakeClock()
	c := newLRU[string](4, clk.now)

	c.put("a", "1", time.Minute)
	if v, ok := c.get("a"); !ok || v != "1" {
		t.Fatalf("expected a hit, got %q %v", v, ok)
	}

	clk.add(59 * time.Second)
	if _, ok := c.get("a"); !ok {
		t.Errorf("must still be there before the ttl")
	}
	clk.add(2 * time.Second)
	if _, ok := c.get("a"); ok {
		t.Errorf("must be gone after the ttl")
	}
	if c.len() != 0 {
		t.Errorf("an expired entry must be removed when it is found, len = %d", c.len())
	}
}

func Test_lruTTL_nonPositiveTTLStoresNothing(t *testing.T) {
	c := newLRU[string](4, newFakeClock().now)
	c.put("a", "1", 0)
	c.put("b", "1", -time.Second)
	if c.len() != 0 {
		t.Errorf("expected nothing stored, len = %d", c.len())
	}
}

func Test_lruTTL_evictsTheLeastRecentlyUsed(t *testing.T) {
	c := newLRU[int](3, newFakeClock().now)
	c.put("a", 1, time.Hour)
	c.put("b", 2, time.Hour)
	c.put("c", 3, time.Hour)

	c.get("a") // a is now the most recent, b the oldest
	c.put("d", 4, time.Hour)

	if c.len() != 3 {
		t.Fatalf("the cache must hold at most its capacity, len = %d", c.len())
	}
	if _, ok := c.get("b"); ok {
		t.Errorf("b was the least recently used and must be evicted")
	}
	for _, k := range []string{"a", "c", "d"} {
		if _, ok := c.get(k); !ok {
			t.Errorf("%s must still be cached", k)
		}
	}
}

func Test_lruTTL_overwriteKeepsOneEntry(t *testing.T) {
	c := newLRU[int](3, newFakeClock().now)
	c.put("a", 1, time.Hour)
	c.put("a", 2, time.Hour)
	if v, _ := c.get("a"); v != 2 || c.len() != 1 {
		t.Errorf("got value %d with len %d", v, c.len())
	}
}

func Test_lruTTL_delete(t *testing.T) {
	c := newLRU[int](3, newFakeClock().now)
	c.put("a", 1, time.Hour)
	c.del("a")
	c.del("missing")
	if _, ok := c.get("a"); ok {
		t.Errorf("deleted entry still there")
	}
}

func Test_lruTTL_boundedUnderManyKeys(t *testing.T) {
	c := newLRU[int](256, newFakeClock().now)
	for i := 0; i < 10000; i++ {
		c.put(fmt.Sprintf("k%d", i), i, time.Hour)
	}
	if c.len() != 256 {
		t.Errorf("len = %d, expected the capacity 256", c.len())
	}
}

func Test_TokenCache(t *testing.T) {
	clk := newFakeClock()
	tc := newTokenCache(2, clk.now)

	tc.Put("k1", "t1", time.Minute)
	tc.Put("k2", "t2", time.Minute)
	tc.Put("k3", "t3", time.Minute)

	if _, ok := tc.Get("k1"); ok {
		t.Errorf("k1 must be evicted past the capacity")
	}
	if v, ok := tc.Get("k3"); !ok || v != "t3" {
		t.Errorf("k3: %q %v", v, ok)
	}
	tc.Delete("k3")
	if _, ok := tc.Get("k3"); ok {
		t.Errorf("k3 must be deleted")
	}
	clk.add(2 * time.Minute)
	if _, ok := tc.Get("k2"); ok {
		t.Errorf("k2 must be expired")
	}
}
