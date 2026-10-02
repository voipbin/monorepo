package callhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jart/gosip/sip"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

	return filterSIPMessages(responseData.Data.Messages, callID), nil
}

// filterSIPMessages parses the Homer rows of the given Call-ID. Rows of another Call-ID and rows that are not
// parseable SIP (the correlation can add non-SIP rows) are skipped instead of failing the whole recovery.
func filterSIPMessages(rows []HomerSIPMessageDetail, callID string) []*sip.Msg {
	log := logrus.WithFields(logrus.Fields{
		"func":    "filterSIPMessages",
		"call_id": callID,
	})

	res := []*sip.Msg{}
	for _, row := range rows {
		if row.CallID != callID {
			log.Debugf("Skipping a row of another Call-ID. row_id: %d, row_call_id: %s", row.ID, row.CallID)
			continue
		}

		tmp, err := sip.ParseMsg(sipHeaderOnly(row.Raw))
		if err != nil {
			// correlated non-SIP rows (e.g. RTCP, logs) of the same call land here; summarized below.
			log.Debugf("Skipping a row that could not be parsed as SIP. row_id: %d, err: %v", row.ID, err)
			continue
		}

		if tmp.CallID != callID {
			log.Debugf("Skipping a message of another Call-ID. row_id: %d, message_call_id: %s", row.ID, tmp.CallID)
			continue
		}

		res = append(res, tmp)
	}

	log.Infof("Filtered the Homer rows. rows: %d, sip_messages: %d", len(rows), len(res))

	return res
}

// sipHeaderOnly returns the SIP message without its body and with Content-Length 0. Recovery needs only headers,
// and the SIP parser also parses SDP bodies strictly (e.g. a declined dynamic payload without rtpmap fails), which
// would drop a valid INVITE or 2xx. It expects the CRLF wire format Homer stores; other input is returned as is
// (and fails in the parser as before).
func sipHeaderOnly(raw string) []byte {
	idx := strings.Index(raw, "\r\n\r\n")
	if idx < 0 {
		return []byte(raw)
	}

	lines := strings.Split(raw[:idx], "\r\n")
	for i, line := range lines {
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue // folded continuation of the previous header
		}
		name, _, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if strings.EqualFold(name, "Content-Length") || strings.EqualFold(name, "l") {
			lines[i] = name + ": 0"
		}
	}

	return []byte(strings.Join(lines, "\r\n") + "\r\n\r\n")
}
