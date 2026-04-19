package requesthandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
)

// KamailioProviderHealthResult is the result of a provider health check via kamailio-proxy.
type KamailioProviderHealthResult struct {
	Status     string `json:"status"`      // "healthy" | "unhealthy"
	ResultCode string `json:"result_code"` // SIP response code e.g. "200", "404", or "timeout"
}

// KamailioV1ProviderHealthCheck sends a SIP OPTIONS health check request to voip-kamailio-proxy
// via RabbitMQ RPC and returns the result.
//
// The kamailio-proxy sends a SIP OPTIONS packet via Go UDP to <hostname>:5060 and returns:
//   - Status "healthy" if any SIP response is received
//   - Status "unhealthy" if no response within the 5s SIP timeout
//   - ResultCode: the SIP response code (e.g. "200", "404") or "timeout"
//
// RPC timeout is 10s — longer than the 5s SIP read deadline in voip-kamailio-proxy,
// accounting for network overhead between GKE and the Kamailio VM.
func (r *requestHandler) KamailioV1ProviderHealthCheck(ctx context.Context, hostname string) (*KamailioProviderHealthResult, error) {
	uri := "/v1/providers/health"

	type Data struct {
		Hostname string `json:"hostname"`
	}

	m, err := json.Marshal(Data{Hostname: hostname})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestKamailio(ctx, uri, sock.RequestMethodPost, "kamailio/providers/health", requestTimeoutKamailio, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res KamailioProviderHealthResult
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
