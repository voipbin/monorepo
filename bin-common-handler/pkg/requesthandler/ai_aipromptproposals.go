package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	amaipromptproposal "monorepo/bin-ai-manager/models/aipromptproposal"
	amrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// AIV1AIPromptProposalCreate sends a request to ai-manager to create a prompt proposal.
// It returns the created AIPromptProposal record if it succeeds.
func (r *requestHandler) AIV1AIPromptProposalCreate(ctx context.Context, customerID uuid.UUID, aiID uuid.UUID, auditIDs []uuid.UUID, language string) (*amaipromptproposal.AIPromptProposal, error) {
	uri := "/v1/aipromptproposals"

	data := &amrequest.V1DataAIPromptProposalsPost{
		CustomerID: customerID,
		AIID:       aiID,
		AuditIDs:   auditIDs,
		Language:   language,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aipromptproposals", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amaipromptproposal.AIPromptProposal
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIPromptProposalList sends a request to ai-manager to get a paginated list of prompt proposals.
// It returns a list of AIPromptProposal records if it succeeds.
func (r *requestHandler) AIV1AIPromptProposalList(ctx context.Context, pageToken string, pageSize uint64, filters map[amaipromptproposal.Field]any) ([]*amaipromptproposal.AIPromptProposal, error) {
	uri := fmt.Sprintf("/v1/aipromptproposals?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/aipromptproposals", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []*amaipromptproposal.AIPromptProposal
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// AIV1AIPromptProposalGet sends a request to ai-manager to get a single prompt proposal.
// It returns the AIPromptProposal record if it succeeds.
func (r *requestHandler) AIV1AIPromptProposalGet(ctx context.Context, id uuid.UUID) (*amaipromptproposal.AIPromptProposal, error) {
	uri := fmt.Sprintf("/v1/aipromptproposals/%s", id.String())

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/aipromptproposals/<id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amaipromptproposal.AIPromptProposal
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIPromptProposalAccept sends a request to ai-manager to accept a prompt proposal.
// It returns the updated AIPromptProposal record if it succeeds.
func (r *requestHandler) AIV1AIPromptProposalAccept(ctx context.Context, customerID uuid.UUID, id uuid.UUID) (*amaipromptproposal.AIPromptProposal, error) {
	uri := fmt.Sprintf("/v1/aipromptproposals/%s/accept", id.String())

	data := &amrequest.V1DataAIPromptProposalsAcceptPost{
		CustomerID: customerID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aipromptproposals/<id>/accept", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amaipromptproposal.AIPromptProposal
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIPromptProposalReject sends a request to ai-manager to reject a prompt proposal.
// It returns the updated AIPromptProposal record if it succeeds.
func (r *requestHandler) AIV1AIPromptProposalReject(ctx context.Context, customerID uuid.UUID, id uuid.UUID) (*amaipromptproposal.AIPromptProposal, error) {
	uri := fmt.Sprintf("/v1/aipromptproposals/%s/reject", id.String())

	data := &amrequest.V1DataAIPromptProposalsAcceptPost{
		CustomerID: customerID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aipromptproposals/<id>/reject", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amaipromptproposal.AIPromptProposal
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIPromptProposalDelete sends a request to ai-manager to delete a prompt proposal.
// It returns the deleted AIPromptProposal record if it succeeds.
func (r *requestHandler) AIV1AIPromptProposalDelete(ctx context.Context, id uuid.UUID) (*amaipromptproposal.AIPromptProposal, error) {
	uri := fmt.Sprintf("/v1/aipromptproposals/%s", id.String())

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodDelete, "ai/aipromptproposals/<id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amaipromptproposal.AIPromptProposal
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
