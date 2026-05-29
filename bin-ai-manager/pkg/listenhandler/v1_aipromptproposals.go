package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// processV1AIPromptProposalsPost handles POST /v1/aipromptproposals.
func (h *listenHandler) processV1AIPromptProposalsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1AIPromptProposalsPost", "request": m})

	var req request.V1DataAIPromptProposalsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(400), nil
	}

	rec, err := h.aipromptproposalHandler.Create(ctx, req.CustomerID, req.AIID, req.AuditIDs, req.Language)
	if err != nil {
		log.Errorf("Could not create proposal. err: %v", err)
		s := err.Error()
		switch {
		case strings.Contains(s, "rate limit exceeded"):
			return simpleResponse(429), nil
		case strings.Contains(s, "audit prompt version mismatch"),
			strings.Contains(s, "invalid audit set"):
			return simpleResponse(400), nil
		case strings.Contains(s, "ai not found"):
			return simpleResponse(404), nil
		default:
			return errorResponse(err), nil
		}
	}

	data, mErr := json.Marshal(rec)
	if mErr != nil {
		log.Errorf("Could not marshal response. err: %v", mErr)
		return simpleResponse(500), nil
	}
	return &sock.Response{StatusCode: 202, DataType: "application/json", Data: data}, nil
}

// processV1AIPromptProposalsGet handles GET /v1/aipromptproposals.
func (h *listenHandler) processV1AIPromptProposalsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1AIPromptProposalsGet", "request": m})

	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse URI. err: %v", err)
		return simpleResponse(400), nil
	}

	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}
	typedFilters, err := utilhandler.ConvertFilters[aipromptproposal.FieldStruct, aipromptproposal.Field](aipromptproposal.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	list, err := h.aipromptproposalHandler.List(ctx, pageSize, pageToken, typedFilters)
	if err != nil {
		log.Errorf("Could not list proposals. err: %v", err)
		return errorResponse(err), nil
	}

	data, mErr := json.Marshal(list)
	if mErr != nil {
		log.Errorf("Could not marshal response. err: %v", mErr)
		return simpleResponse(500), nil
	}
	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1AIPromptProposalsIDGet handles GET /v1/aipromptproposals/<id>.
func (h *listenHandler) processV1AIPromptProposalsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1AIPromptProposalsIDGet", "request": m})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid proposal ID.")
		return simpleResponse(400), nil
	}

	rec, err := h.aipromptproposalHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get proposal. err: %v", err)
		return errorResponse(err), nil
	}

	data, mErr := json.Marshal(rec)
	if mErr != nil {
		log.Errorf("Could not marshal response. err: %v", mErr)
		return simpleResponse(500), nil
	}
	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1AIPromptProposalsIDAcceptPost handles POST /v1/aipromptproposals/<id>/accept.
func (h *listenHandler) processV1AIPromptProposalsIDAcceptPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1AIPromptProposalsIDAcceptPost", "request": m})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid proposal ID.")
		return simpleResponse(400), nil
	}

	var req request.V1DataAIPromptProposalsAcceptPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(400), nil
	}

	rec, err := h.aipromptproposalHandler.Accept(ctx, req.CustomerID, id)
	if err != nil {
		log.Errorf("Could not accept proposal. err: %v", err)
		s := err.Error()
		switch {
		case strings.Contains(s, "proposal not found"):
			return simpleResponse(404), nil
		case strings.Contains(s, "proposal not completed"),
			strings.Contains(s, "prompt version drifted"),
			strings.Contains(s, "audit set invalidated"):
			return simpleResponse(409), nil
		default:
			return errorResponse(err), nil
		}
	}

	data, mErr := json.Marshal(rec)
	if mErr != nil {
		log.Errorf("Could not marshal response. err: %v", mErr)
		return simpleResponse(500), nil
	}
	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1AIPromptProposalsIDRejectPost handles POST /v1/aipromptproposals/<id>/reject.
func (h *listenHandler) processV1AIPromptProposalsIDRejectPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1AIPromptProposalsIDRejectPost", "request": m})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid proposal ID.")
		return simpleResponse(400), nil
	}

	var req request.V1DataAIPromptProposalsAcceptPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(400), nil
	}

	rec, err := h.aipromptproposalHandler.Reject(ctx, req.CustomerID, id)
	if err != nil {
		log.Errorf("Could not reject proposal. err: %v", err)
		s := err.Error()
		switch {
		case strings.Contains(s, "proposal not found"):
			return simpleResponse(404), nil
		case strings.Contains(s, "proposal not completed"):
			return simpleResponse(409), nil
		default:
			return errorResponse(err), nil
		}
	}

	data, mErr := json.Marshal(rec)
	if mErr != nil {
		log.Errorf("Could not marshal response. err: %v", mErr)
		return simpleResponse(500), nil
	}
	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1AIPromptProposalsIDDelete handles DELETE /v1/aipromptproposals/<id>.
func (h *listenHandler) processV1AIPromptProposalsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "processV1AIPromptProposalsIDDelete", "request": m})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid proposal ID.")
		return simpleResponse(400), nil
	}

	rec, err := h.aipromptproposalHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete proposal. err: %v", err)
		return errorResponse(err), nil
	}

	data, mErr := json.Marshal(rec)
	if mErr != nil {
		log.Errorf("Could not marshal response. err: %v", mErr)
		return simpleResponse(500), nil
	}
	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}
