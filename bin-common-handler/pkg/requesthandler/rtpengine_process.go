package requesthandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
)

// RTPEngineV1ProcessSend sends a process management message (exec/kill) to rtpengine-proxy.
// The message is sent as POST /v1/process to the rtpengine-proxy identified by rtpengineID.
// This is fire-and-forget: errors are returned but the caller should not block on them.
//
// Exec example:
//
//	{"type": "exec", "id": "call-uuid", "command": "tcpdump", "parameters": ["udp port 30000 or udp port 30002"]}
//
// Kill example:
//
//	{"type": "kill", "id": "call-uuid"}
func (r *requestHandler) RTPEngineV1ProcessSend(ctx context.Context, rtpengineID string, data interface{}) error {
	uri := "/v1/process"

	m, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = r.sendRequestRTPEngine(ctx, rtpengineID, uri, sock.RequestMethodPost, "rtpengine/process", requestTimeoutDefault, 0, ContentTypeJSON, m)
	return err
}
