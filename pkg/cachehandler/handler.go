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
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
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

// ConferenceGet returns conference call info
func (h *handler) ConferenceGet(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {
	key := fmt.Sprintf("conference:%s", id)

	var res conference.Conference
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConferenceSet sets the conference info into the cache.
func (h *handler) ConferenceSet(ctx context.Context, conference *conference.Conference) error {
	key := fmt.Sprintf("conference:%s", conference.ID)

	if err := h.setSerialize(ctx, key, conference); err != nil {
		return err
	}

	return nil
}

// NumberGetByNumber returns number call info
func (h *handler) NumberGetByNumber(ctx context.Context, num string) (*number.Number, error) {
	key := fmt.Sprintf("number-number:%s", num)

	var res number.Number
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// NumberSetByNumber sets the number info into the cache.
func (h *handler) NumberSetByNumber(ctx context.Context, numb *number.Number) error {
	key := fmt.Sprintf("number-number:%s", numb.Number)

	if err := h.setSerialize(ctx, key, numb); err != nil {
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

// CallDTMFGet returns the given call's dtmf info from the cache
func (h *handler) CallDTMFGet(ctx context.Context, callID uuid.UUID) (string, error) {
	key := fmt.Sprintf("call:%s:dtmf", callID)

	var res string
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return "", err
	}

	return res, nil
}

// CallDTMFSet sets the given call's dtmf info into the cache.
func (h *handler) CallDTMFSet(ctx context.Context, callID uuid.UUID, dtmf string) error {
	key := fmt.Sprintf("call:%s:dtmf", callID)

	if err := h.setSerialize(ctx, key, dtmf); err != nil {
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

// CallExternalMediaGet returns the given call's external media info from the cache
func (h *handler) CallExternalMediaGet(ctx context.Context, callID uuid.UUID) (*externalmedia.ExternalMedia, error) {
	key := fmt.Sprintf("call:%s:external_media", callID)

	var res externalmedia.ExternalMedia
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CallExternalMediaSet sets the given call's external media info into the cache.
func (h *handler) CallExternalMediaSet(ctx context.Context, callID uuid.UUID, data *externalmedia.ExternalMedia) error {
	key := fmt.Sprintf("call:%s:external_media", callID)

	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	return nil
}

// CallExternalMediaDelete deletes the given call's external media info from the cache.
func (h *handler) CallExternalMediaDelete(ctx context.Context, callID uuid.UUID) error {
	key := fmt.Sprintf("call:%s:external_media", callID)

	if _, err := h.Cache.Del(ctx, key).Result(); err != nil {
		return err
	}

	return nil
}
