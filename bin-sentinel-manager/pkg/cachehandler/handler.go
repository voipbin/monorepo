package cachehandler

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"

	"monorepo/bin-sentinel-manager/models/asteriskaddress"
)

// Connect verifies the Redis connection.
func (h *handler) Connect() error {
	if _, err := h.Cache.Ping(context.Background()).Result(); err != nil {
		return errors.Wrapf(err, "could not connect to the redis. addr: %s", h.Addr)
	}

	return nil
}

// AsteriskAddressInternalScan returns every `asterisk.<id>.address-internal` entry with its value
// and remaining ttl.
//
// A key that disappears between SCAN and GET/TTL (its 24h window elapsed mid-pass) is skipped
// rather than treated as an error: a missing key is simply one fewer candidate for this pass, and
// the caller's sticky-last-known rule already handles "this pass learned nothing" correctly. A
// real Redis failure, by contrast, IS returned as an error, because it must not be mistaken for
// "there are no fresh candidates".
func (h *handler) AsteriskAddressInternalScan(ctx context.Context) ([]*asteriskaddress.AsteriskAddress, error) {
	res := []*asteriskaddress.AsteriskAddress{}

	var cursor uint64
	for {
		keys, next, err := h.Cache.Scan(ctx, cursor, asteriskaddress.KeyPattern, scanBatchSize).Result()
		if err != nil {
			return nil, errors.Wrapf(err, "could not scan the asterisk address keys. cursor: %d", cursor)
		}

		for _, key := range keys {
			id, ok := asteriskaddress.ParseKey(key)
			if !ok {
				// the SCAN glob matched something that is not an addressable key. Not an error,
				// just not ours.
				continue
			}

			address, errGet := h.Cache.Get(ctx, key).Result()
			if errors.Is(errGet, redis.Nil) {
				continue
			}
			if errGet != nil {
				return nil, errors.Wrapf(errGet, "could not get the asterisk address. key: %s", key)
			}

			ttl, errTTL := h.Cache.TTL(ctx, key).Result()
			if errors.Is(errTTL, redis.Nil) {
				continue
			}
			if errTTL != nil {
				return nil, errors.Wrapf(errTTL, "could not get the asterisk address ttl. key: %s", key)
			}

			// a negative ttl means either "no expiry set" (-1) or "key does not exist" (-2).
			// Neither can be a proxy-managed key, which is always Set with a 24h expiry, so
			// neither may be treated as evidence about the current occupant of an IP.
			if ttl < 0 {
				continue
			}

			res = append(res, &asteriskaddress.AsteriskAddress{
				ID:      id,
				Address: address,
				TTL:     ttl,
			})
		}

		cursor = next
		if cursor == 0 {
			break
		}
	}

	return res, nil
}
