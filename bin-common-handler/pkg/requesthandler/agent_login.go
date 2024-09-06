package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	amrequest "monorepo/bin-agent-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
)

// AgentV1Login sends a request to agent-manager
// to login.
// it returns agent if it succeed.
// timeout: milliseconds
func (r *requestHandler) AgentV1Login(ctx context.Context, timeout int, username string, password string) (*amagent.Agent, error) {
	uri := "/v1/login"

	req := &amrequest.V1DataLoginPost{
		Username: username,
		Password: password,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAgent(ctx, uri, sock.RequestMethodPost, "agent/login", timeout, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amagent.Agent
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
