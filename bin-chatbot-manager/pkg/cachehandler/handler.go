package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	uuid "github.com/gofrs/uuid"

	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-chatbot-manager/models/chatbotcall"
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

// ChatbotGet returns cached chatbot info
func (h *handler) ChatbotGet(ctx context.Context, id uuid.UUID) (*chatbot.Chatbot, error) {
	key := fmt.Sprintf("chatbot:chatbot:%s", id)

	var res chatbot.Chatbot
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ChatbotSet sets the chatbot info into the cache.
func (h *handler) ChatbotSet(ctx context.Context, data *chatbot.Chatbot) error {
	key := fmt.Sprintf("chatbot:chatbot:%s", data.ID)

	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	return nil
}

// ChatbotcallGet returns cached chatbotcall info
func (h *handler) ChatbotcallGet(ctx context.Context, id uuid.UUID) (*chatbotcall.Chatbotcall, error) {
	key := fmt.Sprintf("chatbot:chatbotcall:%s", id)

	var res chatbotcall.Chatbotcall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ChatbotcallSet sets the chatbotcall info into the cache.
func (h *handler) ChatbotcallSet(ctx context.Context, data *chatbotcall.Chatbotcall) error {

	key := fmt.Sprintf("chatbot:chatbotcall:%s", data.ID)
	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	keyWithTranscribeID := fmt.Sprintf("chatbot:chatbotcall:transcribe_id:%s", data.TranscribeID)
	if err := h.setSerialize(ctx, keyWithTranscribeID, data); err != nil {
		return err
	}

	keyWithReferenceID := fmt.Sprintf("chatbot:chatbotcall:transcribe_id:%s", data.TranscribeID)
	if err := h.setSerialize(ctx, keyWithReferenceID, data); err != nil {
		return err
	}

	return nil
}

// ChatbotcallGetByTranscribeID returns cached chatbotcall info of the given transcribe id.
func (h *handler) ChatbotcallGetByTranscribeID(ctx context.Context, transcribeID uuid.UUID) (*chatbotcall.Chatbotcall, error) {
	key := fmt.Sprintf("chatbot:chatbotcall:transcribe_id:%s", transcribeID)

	var res chatbotcall.Chatbotcall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ChatbotcallGetByReferenceID returns cached chatbotcall info of the given reference id.
func (h *handler) ChatbotcallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*chatbotcall.Chatbotcall, error) {
	key := fmt.Sprintf("chatbot:chatbotcall:reference_id:%s", referenceID)

	var res chatbotcall.Chatbotcall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
