package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferenceconfbridge"
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
func (h *handler) setSerialize(ctx context.Context, key string, data interface{}, duration time.Duration) error {
	tmp, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := h.Cache.Set(ctx, key, tmp, duration).Err(); err != nil {
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

	if err := h.setSerialize(ctx, key, conference, time.Hour*24); err != nil {
		return err
	}

	return nil
}

// ConferenceConfbridgeGet returns conference-confbridge
func (h *handler) ConferenceConfbridgeGet(ctx context.Context, confbridgeID uuid.UUID) (*conferenceconfbridge.ConferenceConfbridge, error) {
	key := fmt.Sprintf("conference-confbridge:%s", confbridgeID)

	var res conferenceconfbridge.ConferenceConfbridge
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConferenceConfbridgeSet sets the conference-confbridge info into the cache.
func (h *handler) ConferenceConfbridgeSet(ctx context.Context, data *conferenceconfbridge.ConferenceConfbridge) error {
	key := fmt.Sprintf("conference-confbridge:%s", data.ConfbridgeID)

	if err := h.setSerialize(ctx, key, data, time.Hour*24*7); err != nil {
		return err
	}

	return nil
}
