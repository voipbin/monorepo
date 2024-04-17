package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
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

// delSerialize deletes cached serialized info.
//
//nolint:unused // this is ok
func (h *handler) delSerialize(ctx context.Context, key string) error {
	_, err := h.Cache.Del(ctx, key).Result()
	if err != nil {
		return err
	}

	return nil
}

// TranscribeGet returns cached transcribe info
func (h *handler) TranscribeGet(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	key := fmt.Sprintf("transcribe:transcribe:%s", id)

	var res transcribe.Transcribe
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TranscribeSet sets the transcribe info into the cache.
func (h *handler) TranscribeSet(ctx context.Context, trans *transcribe.Transcribe) error {
	key := fmt.Sprintf("transcribe:transcribe:%s", trans.ID)

	if err := h.setSerialize(ctx, key, trans); err != nil {
		return err
	}

	return nil
}

// TranscriptGet returns cached transcript info
func (h *handler) TranscriptGet(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error) {
	key := fmt.Sprintf("transcribe:transcript:%s", id)

	var res transcript.Transcript
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TranscriptSet sets the transcript info into the cache.
func (h *handler) TranscriptSet(ctx context.Context, trans *transcript.Transcript) error {
	key := fmt.Sprintf("transcribe:transcript:%s", trans.ID)

	if err := h.setSerialize(ctx, key, trans); err != nil {
		return err
	}

	return nil
}

// StreamingGet returns cached streaming info
func (h *handler) StreamingGet(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error) {
	key := fmt.Sprintf("transcribe:streaming:%s", id)

	var res streaming.Streaming
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// streaming.Streaming sets the streaming info into the cache.
func (h *handler) StreamingSet(ctx context.Context, stream *streaming.Streaming) error {
	key := fmt.Sprintf("transcribe:streaming:%s", stream.ID)

	if err := h.setSerialize(ctx, key, stream); err != nil {
		return err
	}

	return nil
}
