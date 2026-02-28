package servicehandler

import (
	"context"
	"fmt"

	amteam "monorepo/bin-ai-manager/models/team"

	amagent "monorepo/bin-agent-manager/models/agent"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// teamGet returns the team info.
func (h *serviceHandler) teamGet(ctx context.Context, id uuid.UUID) (*amteam.Team, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "teamGet",
		"team_id": id,
	})

	res, err := h.reqHandler.AIV1TeamGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the resource info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// TeamCreate is a service handler for team creation.
func (h *serviceHandler) TeamCreate(
	ctx context.Context,
	a *amagent.Agent,
	name string,
	detail string,
	startMemberID uuid.UUID,
	members []amteam.Member,
	parameter map[string]any,
) (*amteam.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "TeamCreate",
		"customer_id":     a.CustomerID,
		"name":            name,
		"detail":          detail,
		"start_member_id": startMemberID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.AIV1TeamCreate(
		ctx,
		a.CustomerID,
		name,
		detail,
		startMemberID,
		members,
		parameter,
	)
	if err != nil {
		log.Errorf("Could not create a new team. err: %v", err)
		return nil, err
	}
	log.WithField("team", tmp).Debug("Created a new team.")

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// TeamGetsByCustomerID gets the list of teams of the given customer id.
func (h *serviceHandler) TeamGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*amteam.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TeamGetsByCustomerID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// filters
	filters := map[string]string{
		"deleted":     "false",
		"customer_id": a.CustomerID.String(),
	}

	typedFilters, err := h.convertTeamFilters(filters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return nil, err
	}

	tmps, err := h.reqHandler.AIV1TeamList(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get teams info from the ai manager. err: %v", err)
		return nil, fmt.Errorf("could not find teams info. err: %v", err)
	}

	res := []*amteam.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// convertTeamFilters converts map[string]string to map[amteam.Field]any
func (h *serviceHandler) convertTeamFilters(filters map[string]string) (map[amteam.Field]any, error) {
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, amteam.FieldStruct{})
	if err != nil {
		return nil, err
	}

	result := make(map[amteam.Field]any, len(typed))
	for k, v := range typed {
		result[amteam.Field(k)] = v
	}

	return result, nil
}

// TeamGet gets the team of the given id.
func (h *serviceHandler) TeamGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amteam.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TeamGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"team_id":     id,
	})

	tmp, err := h.teamGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get team info from the ai manager. err: %v", err)
		return nil, fmt.Errorf("could not find team info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// TeamDelete deletes the team.
func (h *serviceHandler) TeamDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amteam.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TeamDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"team_id":     id,
	})
	log.Debug("Deleting a team.")

	c, err := h.teamGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get team info from the ai-manager. err: %v", err)
		return nil, fmt.Errorf("could not find team info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.AIV1TeamDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the team. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// TeamUpdate is a service handler for team update.
func (h *serviceHandler) TeamUpdate(
	ctx context.Context,
	a *amagent.Agent,
	id uuid.UUID,
	name string,
	detail string,
	startMemberID uuid.UUID,
	members []amteam.Member,
	parameter map[string]any,
) (*amteam.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "TeamUpdate",
		"customer_id":     a.CustomerID,
		"id":              id,
		"name":            name,
		"detail":          detail,
		"start_member_id": startMemberID,
	})

	c, err := h.teamGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get team info from the ai-manager. err: %v", err)
		return nil, fmt.Errorf("could not find team info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.AIV1TeamUpdate(
		ctx,
		id,
		name,
		detail,
		startMemberID,
		members,
		parameter,
	)
	if err != nil {
		log.Errorf("Could not update the team. err: %v", err)
		return nil, err
	}
	log.WithField("team", tmp).Debugf("Updated team info. team_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
