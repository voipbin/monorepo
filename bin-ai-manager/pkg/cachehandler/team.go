package cachehandler

import (
	"context"
	"fmt"

	uuid "github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/team"
)

// TeamGet returns cached team info
func (h *handler) TeamGet(ctx context.Context, id uuid.UUID) (*team.Team, error) {
	key := fmt.Sprintf("ai:team:%s", id)

	var res team.Team
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TeamSet sets the team info into the cache.
func (h *handler) TeamSet(ctx context.Context, data *team.Team) error {
	key := fmt.Sprintf("ai:team:%s", data.ID)

	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	return nil
}
