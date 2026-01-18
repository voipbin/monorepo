package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/sirupsen/logrus"

	commonsock "monorepo/bin-common-handler/models/sock"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/models/participant"
)

func (h *listenHandler) v1ParticipantsGet(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	u, _ := url.Parse(m.URI)

	// Parse pagination
	tmpSize, _ := strconv.Atoi(u.Query().Get("page_size"))
	pageSize := uint64(tmpSize)
	if pageSize == 0 {
		pageSize = 50
	}
	pageToken := u.Query().Get("page_token")

	// Parse filters from request body using utilhandler pattern
	tmpFilters, err := h.utilHandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		logrus.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// Convert to typed filters
	typedFilters, err := commonutil.ConvertFilters[participant.FieldStruct, participant.Field](
		participant.FieldStruct{},
		tmpFilters,
	)
	if err != nil {
		logrus.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	participants, err := h.participantHandler.ParticipantListWithFilters(ctx, typedFilters, pageToken, pageSize)
	if err != nil {
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(participants)
	return &commonsock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
