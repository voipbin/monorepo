package teamhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/identity"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"monorepo/bin-ai-manager/models/team"
)

// Create creates a new team record.
func (h *teamHandler) Create(ctx context.Context, customerID uuid.UUID, name string, detail string, startMemberID uuid.UUID, members []team.Member, parameter map[string]any) (*team.Team, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Create",
	})

	// Validate team structure (rules 1-5, 7-11)
	if err := validateTeam(startMemberID, members); err != nil {
		return nil, errors.Wrap(err, "validation failed")
	}

	// Rule 6: Verify each member's AIID references an existing AI
	for _, m := range members {
		ai, err := h.db.AIGet(ctx, m.AIID)
		if err != nil {
			return nil, errors.Wrapf(err, "member %s references non-existent ai %s", m.ID, m.AIID)
		}
		log.WithField("ai", ai).Debugf("Retrieved ai info. ai_id: %s", ai.ID)
	}

	id := h.utilHandler.UUIDCreate()

	// create direct hash via direct-manager
	d, err := h.reqHandler.DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeAITeam, id)
	if err != nil {
		log.Errorf("Could not create direct hash. err: %v", err)
		return nil, fmt.Errorf("could not create direct hash: %w", err)
	}
	log.WithField("direct", d).Debugf("Created direct hash. direct_id: %s", d.ID)

	t := &team.Team{
		Identity: identity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Name:          name,
		Detail:        detail,
		StartMemberID: startMemberID,
		Members:       members,
		Parameter:     parameter,
		DirectID:      d.ID,
		DirectHash:    d.Hash,
	}

	if err := h.db.TeamCreate(ctx, t); err != nil {
		// cleanup orphaned direct
		if _, errDelete := h.reqHandler.DirectV1DirectDelete(ctx, d.ID); errDelete != nil {
			log.Errorf("Could not cleanup orphaned direct. direct_id: %s, err: %v", d.ID, errDelete)
		}
		return nil, errors.Wrapf(err, "could not create team")
	}

	res, err := h.db.TeamGet(ctx, t.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get created team")
	}
	log.WithField("team", res).Debugf("Created team. team_id: %s", res.ID)
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, team.EventTypeCreated, res)

	return res, nil
}

// Get returns team.
func (h *teamHandler) Get(ctx context.Context, id uuid.UUID) (*team.Team, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Get",
	})

	res, err := h.db.TeamGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get team")
	}
	log.WithField("team", res).Debugf("Retrieved team info. team_id: %s", res.ID)

	return res, nil
}

// List returns list of teams.
func (h *teamHandler) List(ctx context.Context, size uint64, token string, filters map[team.Field]any) ([]*team.Team, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "List",
	})

	res, err := h.db.TeamList(ctx, size, token, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not list teams")
	}
	log.Debugf("Retrieved teams list. count: %d", len(res))

	return res, nil
}

// Delete deletes the team.
func (h *teamHandler) Delete(ctx context.Context, id uuid.UUID) (*team.Team, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Delete",
	})

	// get the team to retrieve the direct_id before deletion
	t, err := h.db.TeamGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get team for delete")
	}
	log.WithField("team", t).Debugf("Retrieved team info. team_id: %s", t.ID)

	// delete direct hash via direct-manager (best-effort, don't block team deletion)
	if t.DirectID != uuid.Nil {
		if _, errDirect := h.reqHandler.DirectV1DirectDelete(ctx, t.DirectID); errDirect != nil {
			log.Errorf("Could not delete direct hash. direct_id: %s, err: %v", t.DirectID, errDirect)
		}
	}

	if err := h.db.TeamDelete(ctx, id); err != nil {
		return nil, errors.Wrapf(err, "could not delete team")
	}

	res, err := h.db.TeamGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get deleted team")
	}
	log.WithField("team", res).Debugf("Deleted team. team_id: %s", res.ID)
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, team.EventTypeDeleted, res)

	return res, nil
}

// Update updates the team.
func (h *teamHandler) Update(ctx context.Context, id uuid.UUID, name string, detail string, startMemberID uuid.UUID, members []team.Member, parameter map[string]any) (*team.Team, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Update",
	})

	// Validate team structure (rules 1-5, 7-11)
	if err := validateTeam(startMemberID, members); err != nil {
		return nil, errors.Wrap(err, "validation failed")
	}

	// Rule 6: Verify each member's AIID references an existing AI
	for _, m := range members {
		ai, err := h.db.AIGet(ctx, m.AIID)
		if err != nil {
			return nil, errors.Wrapf(err, "member %s references non-existent ai %s", m.ID, m.AIID)
		}
		log.WithField("ai", ai).Debugf("Retrieved ai info. ai_id: %s", ai.ID)
	}

	fields := map[team.Field]any{
		team.FieldName:          name,
		team.FieldDetail:        detail,
		team.FieldStartMemberID: startMemberID,
		team.FieldMembers:       members,
		team.FieldParameter:     parameter,
	}

	if err := h.db.TeamUpdate(ctx, id, fields); err != nil {
		return nil, errors.Wrapf(err, "could not update team")
	}

	res, err := h.db.TeamGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated team")
	}
	log.WithField("team", res).Debugf("Updated team. team_id: %s", res.ID)
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, team.EventTypeUpdated, res)

	return res, nil
}
