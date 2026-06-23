package listenhandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-timeline-manager/models/analysis"
	"monorepo/bin-timeline-manager/pkg/analysishandler"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/request"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/response"
)

// analysisErrorResponse maps analysishandler sentinels to HTTP status codes
// (design §7.2). Cooldown -> 429, not-ended -> 409, masked not-found -> 404,
// else -> 500.
func analysisErrorResponse(err error) *sock.Response {
	switch {
	case errors.Is(err, analysishandler.ErrReanalyzeCooldown):
		return simpleResponse(http.StatusTooManyRequests)
	case errors.Is(err, analysishandler.ErrConcurrencyLimit):
		return simpleResponse(http.StatusTooManyRequests)
	case errors.Is(err, analysishandler.ErrActiveflowNotEnded):
		return simpleResponse(http.StatusConflict)
	case errors.Is(err, analysishandler.ErrNotFound):
		return simpleResponse(http.StatusNotFound)
	default:
		return simpleResponse(http.StatusInternalServerError)
	}
}

func marshalAnalysis(v any) (*sock.Response, error) {
	data, err := json.Marshal(v)
	if err != nil {
		logrus.WithField("func", "marshalAnalysis").Errorf("could not marshal response. err: %v", err)
		return simpleResponse(http.StatusInternalServerError), nil
	}
	return &sock.Response{
		StatusCode: http.StatusOK,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

// v1AnalysesPost handles POST /v1/analyses — trigger an analysis.
func (h *listenHandler) v1AnalysesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "v1AnalysesPost")

	var req request.V1DataAnalysesPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(http.StatusBadRequest), nil
	}
	if req.CustomerID == uuid.Nil || req.ActiveflowID == uuid.Nil {
		return simpleResponse(http.StatusBadRequest), nil
	}

	res, err := h.analysisHandler.Start(ctx, req.CustomerID, req.ActiveflowID, req.Reanalyze)
	if err != nil {
		log.Errorf("Could not start analysis. err: %v", err)
		return analysisErrorResponse(err), nil
	}

	return marshalAnalysis(res)
}

// v1AnalysesGet handles GET /v1/analyses — list the customer's analyses.
// customer_id / page_token / page_size are carried as query params (the
// requesthandler authority), while additional filters (activeflow_id, status)
// arrive as a JSON-marshaled filter map in the body.
func (h *listenHandler) v1AnalysesGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "v1AnalysesGet")

	q := queryValues(m.URI)
	customerID := uuid.FromStringOrNil(q.Get("customer_id"))
	if customerID == uuid.Nil {
		return simpleResponse(http.StatusBadRequest), nil
	}

	// customer_id is injected by the requesthandler authority (query param); any
	// additional filters arrive in the body. There is no soft-delete predicate
	// (delete is a hard delete, so every stored row is live).
	filters := map[analysis.Field]any{}

	// merge body-supplied filters (activeflow_id, status, ...). Never let a
	// body value override the customer_id authority.
	if len(m.Data) > 0 {
		var raw map[string]any
		if err := json.Unmarshal(m.Data, &raw); err != nil {
			log.Errorf("Could not unmarshal filters. err: %v", err)
			return simpleResponse(http.StatusBadRequest), nil
		}
		for k, v := range raw {
			if k == string(analysis.FieldCustomerID) {
				continue // authority comes from the query param only.
			}
			filters[analysis.Field(k)] = normalizeFilterValue(analysis.Field(k), v)
		}
	}

	pageToken := q.Get("page_token")
	pageSize := parsePageSize(q.Get("page_size"))

	res, err := h.analysisHandler.List(ctx, customerID, pageToken, pageSize, filters)
	if err != nil {
		log.Errorf("Could not list analyses. err: %v", err)
		return analysisErrorResponse(err), nil
	}

	out := &response.V1DataAnalysesList{Result: res}
	if uint64(len(res)) == pageSize && pageSize > 0 {
		last := res[len(res)-1]
		if last.TMCreate != nil {
			out.NextPageToken = last.TMCreate.Format("2006-01-02 15:04:05.000000")
		}
	}

	return marshalAnalysis(out)
}

// normalizeFilterValue converts JSON-decoded filter values into the concrete
// types ApplyFields expects (uuid.UUID for *_id fields).
func normalizeFilterValue(field analysis.Field, v any) any {
	switch field {
	case analysis.FieldID, analysis.FieldActiveflowID, analysis.FieldCustomerID:
		if s, ok := v.(string); ok {
			return uuid.FromStringOrNil(s)
		}
	}
	return v
}

// v1AnalysesIDGet handles GET /v1/analyses/<uuid> — get one (ownership-checked).
func (h *listenHandler) v1AnalysesIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "v1AnalysesIDGet")

	id, ok := analysisIDFromURI(m.URI)
	if !ok {
		return simpleResponse(http.StatusBadRequest), nil
	}
	customerID := uuid.FromStringOrNil(queryValues(m.URI).Get("customer_id"))
	if customerID == uuid.Nil {
		return simpleResponse(http.StatusBadRequest), nil
	}

	res, err := h.analysisHandler.Get(ctx, customerID, id)
	if err != nil {
		log.Errorf("Could not get analysis. err: %v", err)
		return analysisErrorResponse(err), nil
	}

	return marshalAnalysis(res)
}

// v1AnalysesIDDelete handles DELETE /v1/analyses/<uuid> — hard-delete (ownership-checked).
func (h *listenHandler) v1AnalysesIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "v1AnalysesIDDelete")

	id, ok := analysisIDFromURI(m.URI)
	if !ok {
		return simpleResponse(http.StatusBadRequest), nil
	}
	customerID := uuid.FromStringOrNil(queryValues(m.URI).Get("customer_id"))
	if customerID == uuid.Nil {
		return simpleResponse(http.StatusBadRequest), nil
	}

	res, err := h.analysisHandler.Delete(ctx, customerID, id)
	if err != nil {
		log.Errorf("Could not delete analysis. err: %v", err)
		return analysisErrorResponse(err), nil
	}

	return marshalAnalysis(res)
}

// --- URI helpers ---

// queryValues returns the parsed query string of a request URI.
func queryValues(uri string) url.Values {
	idx := strings.Index(uri, "?")
	if idx < 0 {
		return url.Values{}
	}
	v, err := url.ParseQuery(uri[idx+1:])
	if err != nil {
		return url.Values{}
	}
	return v
}

// analysisIDFromURI extracts the {id} from /v1/analyses/<uuid>[?...].
func analysisIDFromURI(uri string) (uuid.UUID, bool) {
	path := uri
	if idx := strings.Index(uri, "?"); idx >= 0 {
		path = uri[:idx]
	}
	items := strings.Split(strings.TrimRight(path, "/"), "/")
	if len(items) < 4 {
		return uuid.Nil, false
	}
	id := uuid.FromStringOrNil(items[3])
	if id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}

func parsePageSize(s string) uint64 {
	if s == "" {
		return 0
	}
	var n uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + uint64(c-'0')
	}
	return n
}
