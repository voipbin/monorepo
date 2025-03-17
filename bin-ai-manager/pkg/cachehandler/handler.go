package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	uuid "github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
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

// AIGet returns cached ai info
func (h *handler) AIGet(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	key := fmt.Sprintf("ai:ai:%s", id)

	var res ai.AI
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AISet sets the ai info into the cache.
func (h *handler) AISet(ctx context.Context, data *ai.AI) error {
	key := fmt.Sprintf("ai:ai:%s", data.ID)

	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	return nil
}

// AIcallGet returns cached aicall info
func (h *handler) AIcallGet(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
	key := fmt.Sprintf("ai:aicall:%s", id)

	var res aicall.AIcall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AIcallSet sets the aicall info into the cache.
func (h *handler) AIcallSet(ctx context.Context, data *aicall.AIcall) error {

	key := fmt.Sprintf("ai:aicall:%s", data.ID)
	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	keyWithTranscribeID := fmt.Sprintf("ai:aicall:transcribe_id:%s", data.TranscribeID)
	if err := h.setSerialize(ctx, keyWithTranscribeID, data); err != nil {
		return err
	}

	keyWithReferenceID := fmt.Sprintf("ai:aicall:transcribe_id:%s", data.TranscribeID)
	if err := h.setSerialize(ctx, keyWithReferenceID, data); err != nil {
		return err
	}

	return nil
}

// AIcallGetByTranscribeID returns cached aicall info of the given transcribe id.
func (h *handler) AIcallGetByTranscribeID(ctx context.Context, transcribeID uuid.UUID) (*aicall.AIcall, error) {
	key := fmt.Sprintf("ai:aicall:transcribe_id:%s", transcribeID)

	var res aicall.AIcall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AIcallGetByReferenceID returns cached aicall info of the given reference id.
func (h *handler) AIcallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*aicall.AIcall, error) {
	key := fmt.Sprintf("ai:aicall:reference_id:%s", referenceID)

	var res aicall.AIcall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// MessageGet returns cached message info
func (h *handler) MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	key := fmt.Sprintf("ai:message:%s", id)

	var res message.Message
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// MessageSet sets the ai info into the cache.
func (h *handler) MessageSet(ctx context.Context, data *message.Message) error {
	key := fmt.Sprintf("ai:message:%s", data.ID)

	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	return nil
}
