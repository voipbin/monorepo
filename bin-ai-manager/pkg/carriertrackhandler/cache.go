package carriertrackhandler

import (
	"container/list"
	"sync"
	"time"
)

// lruTTL is a bounded, expiring least-recently-used map. It is the one store
// behind the result cache, the token cache and the cooldowns, so that nothing
// the adapters keep in memory can grow past a fixed number of entries.
type lruTTL[V any] struct {
	mu    sync.Mutex
	limit int
	order *list.List // front is the most recently used
	items map[string]*list.Element
	now   func() time.Time
}

type lruEntry[V any] struct {
	key     string
	value   V
	expires time.Time
}

func newLRU[V any](limit int, now func() time.Time) *lruTTL[V] {
	return &lruTTL[V]{
		limit: limit,
		order: list.New(),
		items: map[string]*list.Element{},
		now:   now,
	}
}

// get returns the live value of key and marks it as recently used. An expired
// entry is removed when it is found.
func (c *lruTTL[V]) get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var zero V
	el, ok := c.items[key]
	if !ok {
		return zero, false
	}
	e := el.Value.(*lruEntry[V])
	if !c.now().Before(e.expires) {
		c.order.Remove(el)
		delete(c.items, key)
		return zero, false
	}
	c.order.MoveToFront(el)
	return e.value, true
}

// put stores value for ttl. A ttl of zero or less stores nothing. Past the
// limit the least recently used entry is dropped.
func (c *lruTTL[V]) put(key string, value V, ttl time.Duration) {
	if ttl <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	expires := c.now().Add(ttl)
	if el, ok := c.items[key]; ok {
		e := el.Value.(*lruEntry[V])
		e.value, e.expires = value, expires
		c.order.MoveToFront(el)
		return
	}

	c.items[key] = c.order.PushFront(&lruEntry[V]{key: key, value: value, expires: expires})
	for c.order.Len() > c.limit {
		oldest := c.order.Back()
		c.order.Remove(oldest)
		delete(c.items, oldest.Value.(*lruEntry[V]).key)
	}
}

func (c *lruTTL[V]) del(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.order.Remove(el)
		delete(c.items, key)
	}
}

func (c *lruTTL[V]) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

// TokenCache holds carrier access tokens in memory only. Its keys are opaque
// (the handler derives them with a keyed hash), so no key or customer id is
// stored in it.
type TokenCache struct {
	lru *lruTTL[string]
}

func newTokenCache(limit int, now func() time.Time) *TokenCache {
	return &TokenCache{lru: newLRU[string](limit, now)}
}

// Get returns the cached token of key.
func (c *TokenCache) Get(key string) (string, bool) { return c.lru.get(key) }

// Put caches token for ttl.
func (c *TokenCache) Put(key, token string, ttl time.Duration) { c.lru.put(key, token, ttl) }

// Delete drops the token of key.
func (c *TokenCache) Delete(key string) { c.lru.del(key) }
