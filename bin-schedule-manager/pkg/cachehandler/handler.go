package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-schedule-manager/models/schedule"
)

// scheduleKey returns the cache key for the given schedule id.
func scheduleKey(id uuid.UUID) string {
	return fmt.Sprintf("schedule:%s", id)
}

// scheduleLockKey returns the dispatch lock key for the given schedule id.
func scheduleLockKey(id uuid.UUID) string {
	return fmt.Sprintf("schedule:lock:%s", id)
}

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

// delSerialize deletes the cached serialized data.
func (h *handler) delSerialize(ctx context.Context, key string) error {
	if err := h.Cache.Del(ctx, key).Err(); err != nil {
		return err
	}
	return nil
}

// ScheduleGet returns cached schedule info
func (h *handler) ScheduleGet(ctx context.Context, id uuid.UUID) (*schedule.Schedule, error) {
	key := scheduleKey(id)

	var res schedule.Schedule
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ScheduleSet sets the schedule info into the cache.
func (h *handler) ScheduleSet(ctx context.Context, s *schedule.Schedule) error {
	key := scheduleKey(s.ID)

	if err := h.setSerialize(ctx, key, s); err != nil {
		return err
	}

	return nil
}

// ScheduleDelete deletes the cached schedule info.
func (h *handler) ScheduleDelete(ctx context.Context, id uuid.UUID) error {
	key := scheduleKey(id)

	if err := h.delSerialize(ctx, key); err != nil {
		return err
	}

	return nil
}
