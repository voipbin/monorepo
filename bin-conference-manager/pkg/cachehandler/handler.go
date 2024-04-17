package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/models/conferencecall"
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

// ConferenceGet returns conference info
func (h *handler) ConferenceGet(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {
	key := fmt.Sprintf("conference:conference:%s", id)

	var res conference.Conference
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConferenceSet sets the conference info into the cache.
func (h *handler) ConferenceSet(ctx context.Context, conference *conference.Conference) error {
	key := fmt.Sprintf("conference:conference:%s", conference.ID)

	if err := h.setSerialize(ctx, key, conference, time.Hour*24); err != nil {
		return err
	}

	return nil
}

// ConferencecallGet returns conferencecall info
func (h *handler) ConferencecallGet(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error) {
	key := fmt.Sprintf("conference:conferencecall:%s", id)

	var res conferencecall.Conferencecall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConferencecallSet sets the conferencecall info into the cache.
func (h *handler) ConferencecallSet(ctx context.Context, data *conferencecall.Conferencecall) error {
	key := fmt.Sprintf("conference:conferencecall:%s", data.ID)
	if err := h.setSerialize(ctx, key, data, time.Hour*24); err != nil {
		return err
	}

	keyWithReferenceID := fmt.Sprintf("conference:conferencecall:reference_id:%s", data.ID)
	if err := h.setSerialize(ctx, keyWithReferenceID, data, time.Hour*24); err != nil {
		return err
	}

	return nil
}

// ConferencecallGetByReferenceID returns cached conferencecall info of the given reference id.
func (h *handler) ConferencecallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*conferencecall.Conferencecall, error) {
	key := fmt.Sprintf("conference:conferencecall:reference_id:%s", referenceID)

	var res conferencecall.Conferencecall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
