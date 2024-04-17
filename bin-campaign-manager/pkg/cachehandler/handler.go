package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/models/outplan"
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

// OutplanSet sets the outplan info into the cache.
func (h *handler) OutplanSet(ctx context.Context, t *outplan.Outplan) error {
	key := fmt.Sprintf("outplan:%s", t.ID)

	if err := h.setSerialize(ctx, key, t); err != nil {
		return err
	}

	return nil
}

// OutplanGet returns cached outplan info
func (h *handler) OutplanGet(ctx context.Context, id uuid.UUID) (*outplan.Outplan, error) {
	key := fmt.Sprintf("outplan:%s", id)

	var res outplan.Outplan
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CampaignSet sets the campaign info into the cache.
func (h *handler) CampaignSet(ctx context.Context, t *campaign.Campaign) error {
	key := fmt.Sprintf("campaign:%s", t.ID)

	if err := h.setSerialize(ctx, key, t); err != nil {
		return err
	}

	return nil
}

// CampaignGet returns cached campaign info
func (h *handler) CampaignGet(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	key := fmt.Sprintf("campaign:%s", id)

	var res campaign.Campaign
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CampaigncallSet sets the campaigncall info into the cache.
func (h *handler) CampaigncallSet(ctx context.Context, t *campaigncall.Campaigncall) error {
	key := fmt.Sprintf("campaigncall:%s", t.ID)

	if err := h.setSerialize(ctx, key, t); err != nil {
		return err
	}

	return nil
}

// CampaigncallGet returns cached campaigncall info
func (h *handler) CampaigncallGet(ctx context.Context, id uuid.UUID) (*campaigncall.Campaigncall, error) {
	key := fmt.Sprintf("campaigncall:%s", id)

	var res campaigncall.Campaigncall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
