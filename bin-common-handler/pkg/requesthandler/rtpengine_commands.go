package requesthandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
)

// RTPEngineV1CommandsSend sends an RTPEngine NG protocol command to a specific rtpengine-proxy instance.
//
// The command is sent as POST /v1/commands to the rtpengine-proxy identified by rtpengineID.
// The rtpengineID determines the target queue: rtpengine.<rtpengineID>.request
//
// The only required field in command is "call-id". Fields like "from-tag", "to-tag",
// and "via-branch" are optional.
//
// Request examples:
//
//	{"command": "start recording", "call-id": "abc123@sip-server"}
//	{"command": "stop recording", "call-id": "abc123@sip-server"}
//
// Response examples:
//
//	Success: {"result": "ok"}
//	Error:   {"result": "error", "error-reason": "Unknown call-id"}
func (r *requestHandler) RTPEngineV1CommandsSend(ctx context.Context, rtpengineID string, command map[string]interface{}) (map[string]interface{}, error) {
	uri := "/v1/commands"

	m, err := json.Marshal(command)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRTPEngine(ctx, rtpengineID, uri, sock.RequestMethodPost, "rtpengine/commands", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
