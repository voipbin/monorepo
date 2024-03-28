package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	callapplication "gitlab.com/voipbin/bin-manager/call-manager.git/models/callapplication"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
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

// AsteriskAddressInternalGet returns Asterisk's internal ip address
func (h *handler) AsteriskAddressInternalGet(ctx context.Context, id string) (string, error) {
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

// CallGet returns cached call info
func (h *handler) CallGet(ctx context.Context, id uuid.UUID) (*call.Call, error) {
	key := fmt.Sprintf("call:%s", id)

	var res call.Call
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallSet sets the bridge info into the cache.
func (h *handler) CallSet(ctx context.Context, call *call.Call) error {
	key := fmt.Sprintf("call:%s", call.ID)

	if err := h.setSerialize(ctx, key, call); err != nil {
		return err
	}

	return nil
}

// ConfbridgeGet returns confbridge info
func (h *handler) ConfbridgeGet(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {
	key := fmt.Sprintf("confbridge:%s", id)

	var res confbridge.Confbridge
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConfbridgeSet sets the confbridge info into the cache.
func (h *handler) ConfbridgeSet(ctx context.Context, data *confbridge.Confbridge) error {
	key := fmt.Sprintf("confbridge:%s", data.ID)

	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	return nil
}

// RecordingGet returns record info from the cache
func (h *handler) RecordingGet(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	key := fmt.Sprintf("recording:%s", id)

	var res recording.Recording
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RecordingSet sets the record info into the cache.
func (h *handler) RecordingSet(ctx context.Context, record *recording.Recording) error {
	key := fmt.Sprintf("recording:%s", record.ID)

	if err := h.setSerialize(ctx, key, record); err != nil {
		return err
	}

	return nil
}

// CallAppAMDGet gets the given callapplication amd info from the cache.
func (h *handler) CallAppAMDGet(ctx context.Context, channelID string) (*callapplication.AMD, error) {

	key := fmt.Sprintf("callapplication:amd:%s", channelID)

	var res callapplication.AMD
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallAppAMDSet sets the given callapplication amd info into the cache.
func (h *handler) CallAppAMDSet(ctx context.Context, channelID string, app *callapplication.AMD) error {

	key := fmt.Sprintf("callapplication:amd:%s", channelID)

	if err := h.setSerialize(ctx, key, app); err != nil {
		return err
	}

	return nil
}

// ExternalMediaGet returns the given external media info from the cache
func (h *handler) ExternalMediaGet(ctx context.Context, externalMediaID uuid.UUID) (*externalmedia.ExternalMedia, error) {
	key := fmt.Sprintf("external_media:%s", externalMediaID)

	var res externalmedia.ExternalMedia
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ExternalMediaGetByReferenceID returns the given external media info of the given reference id from the cache
func (h *handler) ExternalMediaGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*externalmedia.ExternalMedia, error) {
	key := fmt.Sprintf("external_media:reference_id:%s", referenceID)

	var res externalmedia.ExternalMedia
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ExternalMediaSet sets the given external media info into the cache.
func (h *handler) ExternalMediaSet(ctx context.Context, data *externalmedia.ExternalMedia) error {

	key := fmt.Sprintf("external_media:%s", data.ID)
	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	keyReferenceID := fmt.Sprintf("external_media:reference_id:%s", data.ReferenceID)
	if err := h.setSerialize(ctx, keyReferenceID, data); err != nil {
		return err
	}

	return nil
}

// ExternalMediaDelete deletes the given external media info from the cache.
func (h *handler) ExternalMediaDelete(ctx context.Context, externalMediaID uuid.UUID) error {

	tmp, err := h.ExternalMediaGet(ctx, externalMediaID)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("external_media:%s", tmp.ID)
	if _, err := h.Cache.Del(ctx, key).Result(); err != nil {
		return err
	}

	keyReferenceID := fmt.Sprintf("external_media:reference_id:%s", tmp.ReferenceID)
	if _, err := h.Cache.Del(ctx, keyReferenceID).Result(); err != nil {
		return err
	}

	return nil
}

// GroupcallGet returns cached groupcall info
func (h *handler) GroupcallGet(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	key := fmt.Sprintf("call:groupcall:%s", id)

	var res groupcall.Groupcall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// GroupcallSet sets the groupcall info into the cache.
func (h *handler) GroupcallSet(ctx context.Context, data *groupcall.Groupcall) error {
	key := fmt.Sprintf("call:groupcall:%s", data.ID)

	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	return nil
}
