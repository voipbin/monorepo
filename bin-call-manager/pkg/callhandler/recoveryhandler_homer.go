package callhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jart/gosip/sip"
	"github.com/pkg/errors"
)

type TimeRange struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

type homerRequestParam struct {
	Limit       int      `json:"limit,omitempty"`
	OrLogic     bool     `json:"orlogic,omitempty"`
	Search      any      `json:"search"`
	Transaction any      `json:"transaction"`
	Whitelist   []string `json:"whitelist,omitempty"`
}

type homerRequestPayload struct {
	Timestamp TimeRange         `json:"timestamp"`
	Param     homerRequestParam `json:"param"`
	ID        string            `json:"id"`
}

type HomerSIPMessageDetail struct {
	CallID        string `json:"callid"`
	CorrelationID string `json:"correlation_id"`
	Raw           string `json:"raw"`
	MicroTS       int64  `json:"micro_ts"`
	Method        string `json:"method"`
	SrcIP         string `json:"srcIp"`
	DstIP         string `json:"dstIp"`
	SrcPort       int    `json:"srcPort"`
	DstPort       int    `json:"dstPort"`
	ID            int64  `json:"id"`
}

type homerResponseData struct {
	Alias    map[string]string       `json:"alias,omitempty"`
	CallData []any                   `json:"calldata,omitempty"`
	Hosts    map[string]any          `json:"hosts,omitempty"`
	Messages []HomerSIPMessageDetail `json:"messages"`
}

type homerAPIResponse struct {
	Data  homerResponseData `json:"data"`
	Keys  []string          `json:"keys,omitempty"`
	Total int               `json:"total,omitempty"`
}

func (h *recoveryHandler) getSIPMessages(ctx context.Context, callID string) ([]*sip.Msg, error) {
	if h.homerAPIAddress == "" || h.homerAuthToken == "" {
		return nil, fmt.Errorf("missing Homer API address or auth token")
	}

	if callID == "" {
		return nil, fmt.Errorf("call ID cannot be empty")
	}

	homerAPIEndpoint := fmt.Sprintf("%s/api/v3/call/transaction", h.homerAPIAddress)

	now := time.Now()
	fromTimestamp := now.Add(defaultHomerSearchTimeRange).UnixMilli()
	toTimestamp := now.UnixMilli()

	payload := homerRequestPayload{
		Timestamp: TimeRange{
			From: fromTimestamp,
			To:   toTimestamp,
		},
		Param: homerRequestParam{
			Limit:   1,
			OrLogic: false,
			Search: map[string]any{
				"1_call": map[string]any{
					"callid": []string{callID},
				},
			},
			Transaction: map[string]any{},
			Whitelist:   h.loadBalancerIPs,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "error marshalling payload for call ID %s", callID)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", homerAPIEndpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.Wrapf(err, "error creating request for call ID %s", callID)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth-Token", h.homerAuthToken)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "error sending request for call ID %s", callID)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 300 {
		return nil, errors.Errorf("Homer API request failed for call ID %s: status %s", callID, resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading response body for call ID %s", callID)
	}

	responseData := homerAPIResponse{}
	if errUnmarshal := json.Unmarshal(respBody, &responseData); errUnmarshal != nil {
		return nil, errors.Wrapf(errUnmarshal, "error unmarshalling response for call ID %s", callID)
	}

	res := []*sip.Msg{}
	for _, message := range responseData.Data.Messages {
		tmp, err := sip.ParseMsg([]byte(message.Raw))
		if err != nil {
			return nil, errors.Wrapf(err, "error parsing SIP message for call ID %s", callID)
		}

		res = append(res, tmp)
	}

	return res, nil
}
