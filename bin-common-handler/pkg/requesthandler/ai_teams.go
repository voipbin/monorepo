package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	amteam "monorepo/bin-ai-manager/models/team"
	amrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// AIV1TeamList sends a request to ai-manager
// to getting a list of teams info.
// it returns detail list of teams info if it succeed.
func (r *requestHandler) AIV1TeamList(ctx context.Context, pageToken string, pageSize uint64, filters map[amteam.Field]any) ([]amteam.Team, error) {
	uri := fmt.Sprintf("/v1/teams?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/teams", 30000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []amteam.Team
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// AIV1TeamGet returns the team.
func (r *requestHandler) AIV1TeamGet(ctx context.Context, teamID uuid.UUID) (*amteam.Team, error) {
	uri := fmt.Sprintf("/v1/teams/%s", teamID.String())

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/teams/<team-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amteam.Team
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1TeamCreate sends a request to ai-manager
// to creating a team.
// it returns created team if it succeed.
func (r *requestHandler) AIV1TeamCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	startMemberID uuid.UUID,
	members []amteam.Member,
) (*amteam.Team, error) {
	uri := "/v1/teams"

	data := &amrequest.V1DataTeamsPost{
		CustomerID:    customerID,
		Name:          name,
		Detail:        detail,
		StartMemberID: startMemberID,
		Members:       members,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/teams", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amteam.Team
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1TeamDelete sends a request to ai-manager
// to deleting a team.
// it returns deleted team if it succeed.
func (r *requestHandler) AIV1TeamDelete(ctx context.Context, teamID uuid.UUID) (*amteam.Team, error) {
	uri := fmt.Sprintf("/v1/teams/%s", teamID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodDelete, "ai/teams/<team-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amteam.Team
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1TeamUpdate sends a request to ai-manager
// to updating a team.
// it returns updated team if it succeed.
func (r *requestHandler) AIV1TeamUpdate(
	ctx context.Context,
	teamID uuid.UUID,
	name string,
	detail string,
	startMemberID uuid.UUID,
	members []amteam.Member,
) (*amteam.Team, error) {
	uri := fmt.Sprintf("/v1/teams/%s", teamID)

	data := &amrequest.V1DataTeamsIDPut{
		Name:          name,
		Detail:        detail,
		StartMemberID: startMemberID,
		Members:       members,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPut, "ai/teams/<team-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amteam.Team
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
