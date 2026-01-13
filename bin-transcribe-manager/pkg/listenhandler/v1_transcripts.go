package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/transcript"
)

// processV1TranscriptsGet handles GET /v1/transcripts request
func (h *listenHandler) processV1TranscriptsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1TranscriptsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// parse the filters and convert to typed filters
	stringFilters := h.utilHandler.URLParseFilters(u)
	filters := convertTranscriptFilters(stringFilters)

	tmp, err := h.transcriptHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get transcripts. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// convertTranscriptFilters converts string filters to typed transcript filters
func convertTranscriptFilters(stringFilters map[string]string) map[transcript.Field]any {
	filters := make(map[transcript.Field]any)

	for k, v := range stringFilters {
		switch k {
		case "customer_id":
			filters[transcript.FieldCustomerID] = uuid.FromStringOrNil(v)
		case "transcribe_id":
			filters[transcript.FieldTranscribeID] = uuid.FromStringOrNil(v)
		case "deleted":
			filters[transcript.FieldDeleted] = (v == "true")
		case "direction":
			filters[transcript.FieldDirection] = transcript.Direction(v)
		}
	}

	return filters
}
