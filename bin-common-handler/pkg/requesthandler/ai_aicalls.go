package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	ammessage "monorepo/bin-ai-manager/models/message"
	cbrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (r *requestHandler) AIV1AIcallStart(ctx context.Context, assistanceType amaicall.AssistanceType, assistanceID uuid.UUID, activeflowID uuid.UUID, referenceType amaicall.ReferenceType, referenceID uuid.UUID) (*amaicall.AIcall, error) {
	uri := "/v1/aicalls"

	data := &cbrequest.V1DataAIcallsPost{
		AssistanceType: assistanceType,
		AssistanceID:   assistanceID,

		ActiveflowID: activeflowID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aicalls", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amaicall.AIcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIcallList sends a request to ai-manager
// to getting a list of aicall info of the given customer id.
// it returns detail list of aicall info if it succeed.
func (r *requestHandler) AIV1AIcallList(ctx context.Context, pageToken string, pageSize uint64, filters map[amaicall.Field]any) ([]amaicall.AIcall, error) {
	uri := fmt.Sprintf("/v1/aicalls?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/aicalls", 30000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []amaicall.AIcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// AIV1AIcallGet returns the aicall.
func (r *requestHandler) AIV1AIcallGet(ctx context.Context, aicallID uuid.UUID) (*amaicall.AIcall, error) {

	uri := fmt.Sprintf("/v1/aicalls/%s", aicallID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/aicalls/<aicall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amaicall.AIcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIcallGetSkipCache sends a request to ai-manager to get the aicall,
// bypassing ai-manager's own Redis snapshot cache and reading the row from the
// database.
//
// Use it ONLY where a stale read would produce a wrong, irreversible decision.
// The one such site today is bin-ai-manager's messagehandler stale-reply guard
// (docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.4(b)): the
// guard drops a bot-LLM message whose pipecatcall id does not match the
// AIcall's bound one, and AIcallUpdate's cache refresh discards its own error,
// so a transiently-stale cached PipecatcallID would make the guard drop the
// agent's genuine answer. The guard confirms against the database before
// dropping. Everywhere else, prefer AIV1AIcallGet -- this variant defeats the
// cache on purpose and costs a real query.
func (r *requestHandler) AIV1AIcallGetSkipCache(ctx context.Context, aicallID uuid.UUID) (*amaicall.AIcall, error) {

	uri := fmt.Sprintf("/v1/aicalls/%s?skip_cache=true", aicallID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/aicalls/<aicall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amaicall.AIcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIcallDelete sends a request to ai-manager
// to deleting a aicall.
// it returns deleted aicall if it succeed.
func (r *requestHandler) AIV1AIcallDelete(ctx context.Context, aicallID uuid.UUID) (*amaicall.AIcall, error) {
	uri := fmt.Sprintf("/v1/aicalls/%s", aicallID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodDelete, "ai/aicalls/<aicall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amaicall.AIcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIcallTerminate sends a request to ai-manager
// to terminate an aicall.
// it returns aicall if it succeed.
func (r *requestHandler) AIV1AIcallTerminate(ctx context.Context, aicallID uuid.UUID) (*amaicall.AIcall, error) {
	uri := fmt.Sprintf("/v1/aicalls/%s/terminate", aicallID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aicalls/<aicall-id>/terminate", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amaicall.AIcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIcallListen sends a request to ai-manager to start Insight AI realtime
// call listening on the given aicall.
//
// Mirrors AIV1AIcallTerminate's shape -- POST, no request body -- but with an
// EXPLICIT 10s timeout rather than requestTimeoutDefault (3000ms). ai-manager's
// ProcessListen runs up to three SEQUENTIAL cross-service RPCs synchronously
// (TranscribeV1TranscribeGet, ContactV1CaseGet, CallV1CallGet), and each hop can
// independently take up to its own default timeout -- so three hops can add up
// to roughly 3x a single hop's timeout worst-case, failing the CLIENT's request
// even when ai-manager's own precheck later succeeds.
//
// (The earlier justification, "none of the three is cache-first," was withdrawn:
// CallV1CallGet IS cache-first. Do not reintroduce it if this value is ever
// revisited.)
//
// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.1.
func (r *requestHandler) AIV1AIcallListen(ctx context.Context, aicallID uuid.UUID) (*amaicall.AIcall, error) {
	uri := fmt.Sprintf("/v1/aicalls/%s/listen", aicallID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aicalls/<aicall-id>/listen", 10000, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amaicall.AIcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIcallTerminateWithDelay sends a request to ai-manager
// to terminate an aicall after delayed time.
// it returns null if it succeed.
func (r *requestHandler) AIV1AIcallTerminateWithDelay(ctx context.Context, aicallID uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/aicalls/%s/terminate", aicallID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aicalls/<aicall-id>/terminate", requestTimeoutDefault, delay, ContentTypeNone, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// AIV1AIcallToolExecute sends a request to ai-manager
// to execute the tool on the given aicall.
// it returns response message if it succeed.
func (r *requestHandler) AIV1AIcallToolExecute(
	ctx context.Context,
	aicallID uuid.UUID,
	toolID string,
	toolType ammessage.ToolType,
	function *ammessage.FunctionCall,
	pipecatcallID uuid.UUID,
) (map[string]any, error) {
	uri := fmt.Sprintf("/v1/aicalls/%s/tool_execute", aicallID)

	data := &cbrequest.V1DataAIcallsIDToolExecutePost{
		ID: toolID,

		Type:     toolType,
		Function: *function,

		// The session this tool call arrived on. ai-manager uses it to tell a
		// listen evaluation turn from the agent's own Q&A turn. See
		// docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.3a.
		PipecatcallID: pipecatcallID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aicalls/<aicall-id>/tool_execute", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res map[string]any
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
