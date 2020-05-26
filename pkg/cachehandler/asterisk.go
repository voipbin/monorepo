package cachehandler

import (
	"context"
	"fmt"
)

// AsteriskAddressInternerGet returns Asterisk's internal ip address
func (h *handler) AsteriskAddressInternerGet(ctx context.Context, id string) (string, error) {
	key := fmt.Sprintf("asterisk.%s.address-internal", id)

	res, err := h.Cache.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return res, nil
}
