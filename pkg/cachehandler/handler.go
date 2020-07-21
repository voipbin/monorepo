package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/channel"
)

// getSerialize returns cached serialized info.
func (h *handler) getSerialize(ctx context.Context, key string, data interface{}) error {
	tmp, err := h.Cache.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(tmp), &data); err != nil {
		return err
	}
	return nil
}

// setSerialize sets the info into the cache after serialization.
func (h *handler) setSerialize(ctx context.Context, key string, data interface{}) error {
	tmp, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := h.Cache.Set(ctx, key, tmp, time.Hour*24).Err(); err != nil {
		return err
	}
	return nil
}

// AsteriskAddressInternerGet returns Asterisk's internal ip address
func (h *handler) AsteriskAddressInternerGet(ctx context.Context, id string) (string, error) {
	key := fmt.Sprintf("asterisk.%s.address-internal", id)

	res, err := h.Cache.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return res, nil
}

// ChannelGet returns cached channel info
func (h *handler) ChannelGet(ctx context.Context, id string) (*channel.Channel, error) {
	key := fmt.Sprintf("channel:%s", id)

	var res channel.Channel
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ChannelSet sets the channel info into the cache.
func (h *handler) ChannelSet(ctx context.Context, channel *channel.Channel) error {
	key := fmt.Sprintf("channel:%s", channel.ID)

	if err := h.setSerialize(ctx, key, channel); err != nil {
		return err
	}

	return nil
}

// BridgeGet returns cached bridge info
func (h *handler) BridgeGet(ctx context.Context, id string) (*bridge.Bridge, error) {
	key := fmt.Sprintf("bridge:%s", id)

	var res bridge.Bridge
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// BridgeSet sets the bridge info into the cache.
func (h *handler) BridgeSet(ctx context.Context, bridge *bridge.Bridge) error {
	key := fmt.Sprintf("bridge:%s", bridge.ID)

	if err := h.setSerialize(ctx, key, bridge); err != nil {
		return err
	}

	return nil
}
