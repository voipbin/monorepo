package homerhandler

//go:generate mockgen -package homerhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-timeline-manager/models/sipmessage"
)

const (
	// defaultHTTPTimeout is the default timeout for HTTP requests to Homer API.
	defaultHTTPTimeout = 30 * time.Second

	// defaultTimeBuffer is the buffer added/subtracted to the time range for Homer searches.
	defaultTimeBuffer = 10 * time.Minute

	// defaultSIPMessageLimit is the maximum number of SIP messages returned.
	defaultSIPMessageLimit = 50

	// homerSearchLimit is the limit parameter sent in Homer API requests.
	homerSearchLimit = 200

	// homerPcapSearchLimit is the limit parameter sent in Homer API PCAP requests.
	homerPcapSearchLimit = 1000
)

// HomerHandler interface for Homer API operations.
type HomerHandler interface {
	GetSIPMessages(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]*sipmessage.SIPMessage, error)
	GetPcap(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]byte, error)
	GetRTCPPcap(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]byte, error)
}

type homerHandler struct {
	homerAPIAddr   string
	homerAuthToken string
	httpClient     *http.Client
}

// NewHomerHandler creates a new HomerHandler.
func NewHomerHandler(homerAPIAddr, homerAuthToken string) HomerHandler {
	return &homerHandler{
		homerAPIAddr:   homerAPIAddr,
		homerAuthToken: homerAuthToken,
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
	}
}

// timeRange represents the timestamp range for Homer API requests.
type timeRange struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

// homerRequestParam represents the param field in Homer API requests.
type homerRequestParam struct {
	Limit       int `json:"limit"`
	Search      any `json:"search"`
	Transaction any `json:"transaction"`
}

// homerRequestPayload represents the full Homer API request payload.
type homerRequestPayload struct {
	Timestamp timeRange         `json:"timestamp"`
	Param     homerRequestParam `json:"param"`
}

// homerSIPMessageDetail represents a single SIP message from the Homer API response.
type homerSIPMessageDetail struct {
	CallID  string `json:"callid"`
	Raw     string `json:"raw"`
	MicroTS int64  `json:"micro_ts"`
	Method  string `json:"method"`
	SrcIP   string `json:"srcIp"`
	DstIP   string `json:"dstIp"`
	SrcPort int    `json:"srcPort"`
	DstPort int    `json:"dstPort"`
}

// homerResponseData represents the data field in the Homer API response.
type homerResponseData struct {
	Messages []homerSIPMessageDetail `json:"messages"`
}

// homerAPIResponse represents the full Homer API response.
type homerAPIResponse struct {
	Data  homerResponseData `json:"data"`
	Total int               `json:"total,omitempty"`
}

// GetSIPMessages retrieves SIP messages from Homer for a given SIP call ID and time range.
func (h *homerHandler) GetSIPMessages(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]*sipmessage.SIPMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "GetSIPMessages",
		"sip_callid": sipCallID,
	})
	log.WithFields(logrus.Fields{
		"homer_addr": h.homerAPIAddr,
		"from_time":  fromTime,
		"to_time":    toTime,
	}).Info("HomerHandler called - querying Homer API for SIP messages")

	if h.homerAPIAddr == "" || h.homerAuthToken == "" {
		return nil, fmt.Errorf("missing Homer API address or auth token")
	}

	if sipCallID == "" {
		return nil, fmt.Errorf("sip call ID cannot be empty")
	}

	homerAPIEndpoint := fmt.Sprintf("%s/api/v3/call/transaction", h.homerAPIAddr)

	// Add buffer to time range
	fromTimestamp := fromTime.Add(-defaultTimeBuffer).UnixMilli()
	toTimestamp := toTime.Add(defaultTimeBuffer).UnixMilli()

	payload := homerRequestPayload{
		Timestamp: timeRange{
			From: fromTimestamp,
			To:   toTimestamp,
		},
		Param: homerRequestParam{
			Limit: homerSearchLimit,
			Search: map[string]any{
				"1_call": map[string]any{
					"callid": []string{sipCallID},
				},
			},
			Transaction: map[string]any{},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal Homer request payload for SIP call ID %s", sipCallID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, homerAPIEndpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.Wrapf(err, "could not create HTTP request for SIP call ID %s", sipCallID)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth-Token", h.homerAuthToken)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send HTTP request for SIP call ID %s", sipCallID)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("homer API returned non-success status %s for SIP call ID %s", resp.Status, sipCallID)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read response body for SIP call ID %s", sipCallID)
	}

	var apiResponse homerAPIResponse
	if err := json.Unmarshal(respBody, &apiResponse); err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal Homer API response for SIP call ID %s", sipCallID)
	}

	log.WithField("message_count", len(apiResponse.Data.Messages)).Debug("Received SIP messages from Homer.")

	// Convert Homer messages to SIPMessage model
	messages := make([]*sipmessage.SIPMessage, 0, len(apiResponse.Data.Messages))
	for _, msg := range apiResponse.Data.Messages {
		// Convert micro_ts (milliseconds) to a timestamp string
		// Note: Despite the field name, Homer returns milliseconds, not microseconds
		ts := time.UnixMilli(msg.MicroTS).UTC().Format(time.RFC3339Nano)

		messages = append(messages, &sipmessage.SIPMessage{
			Timestamp: ts,
			Method:    msg.Method,
			SrcIP:     msg.SrcIP,
			SrcPort:   msg.SrcPort,
			DstIP:     msg.DstIP,
			DstPort:   msg.DstPort,
			Raw:       msg.Raw,
		})
	}

	// Sort by timestamp ascending
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp < messages[j].Timestamp
	})

	// Limit to max messages
	if len(messages) > defaultSIPMessageLimit {
		messages = messages[:defaultSIPMessageLimit]
	}

	return messages, nil
}

// GetPcap retrieves a PCAP file from Homer for a given SIP call ID and time range.
func (h *homerHandler) GetPcap(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]byte, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "GetPcap",
		"sip_callid": sipCallID,
	})
	log.WithFields(logrus.Fields{
		"homer_addr": h.homerAPIAddr,
		"from_time":  fromTime,
		"to_time":    toTime,
	}).Info("HomerHandler called - querying Homer API for PCAP data")

	if h.homerAPIAddr == "" || h.homerAuthToken == "" {
		return nil, fmt.Errorf("missing Homer API address or auth token")
	}

	if sipCallID == "" {
		return nil, fmt.Errorf("sip call ID cannot be empty")
	}

	homerAPIEndpoint := fmt.Sprintf("%s/api/v3/export/call/messages/pcap", h.homerAPIAddr)

	// Add buffer to time range
	fromTimestamp := fromTime.Add(-defaultTimeBuffer).UnixMilli()
	toTimestamp := toTime.Add(defaultTimeBuffer).UnixMilli()

	payload := homerRequestPayload{
		Timestamp: timeRange{
			From: fromTimestamp,
			To:   toTimestamp,
		},
		Param: homerRequestParam{
			Limit: homerPcapSearchLimit,
			Search: map[string]any{
				"1_call": map[string]any{
					"callid": []string{sipCallID},
					"type":   "string",
					"hepid":  1,
				},
			},
			Transaction: map[string]any{},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal Homer PCAP request payload for SIP call ID %s", sipCallID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, homerAPIEndpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.Wrapf(err, "could not create HTTP request for PCAP export for SIP call ID %s", sipCallID)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth-Token", h.homerAuthToken)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send HTTP request for PCAP export for SIP call ID %s", sipCallID)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("homer API returned non-success status %s for PCAP export for SIP call ID %s", resp.Status, sipCallID)
	}

	pcapData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read PCAP response body for SIP call ID %s", sipCallID)
	}

	log.WithField("pcap_size", len(pcapData)).Debug("Received PCAP data from Homer.")

	return pcapData, nil
}

// GetRTCPPcap retrieves RTCP PCAP data (hepid 5) from Homer for a given SIP call ID and time range.
func (h *homerHandler) GetRTCPPcap(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]byte, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "GetRTCPPcap",
		"sip_callid": sipCallID,
	})
	log.WithFields(logrus.Fields{
		"homer_addr": h.homerAPIAddr,
		"from_time":  fromTime,
		"to_time":    toTime,
	}).Info("HomerHandler called - querying Homer API for RTCP PCAP data")

	if h.homerAPIAddr == "" || h.homerAuthToken == "" {
		return nil, fmt.Errorf("missing Homer API address or auth token")
	}

	if sipCallID == "" {
		return nil, fmt.Errorf("sip call ID cannot be empty")
	}

	homerAPIEndpoint := fmt.Sprintf("%s/api/v3/export/call/messages/pcap", h.homerAPIAddr)

	// Add buffer to time range
	fromTimestamp := fromTime.Add(-defaultTimeBuffer).UnixMilli()
	toTimestamp := toTime.Add(defaultTimeBuffer).UnixMilli()

	payload := homerRequestPayload{
		Timestamp: timeRange{
			From: fromTimestamp,
			To:   toTimestamp,
		},
		Param: homerRequestParam{
			Limit: homerPcapSearchLimit,
			Search: map[string]any{
				"5_default": map[string]any{
					"callid": []string{sipCallID},
					"type":   "string",
					"hepid":  5,
				},
			},
			Transaction: map[string]any{},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal Homer RTCP PCAP request payload for SIP call ID %s", sipCallID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, homerAPIEndpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.Wrapf(err, "could not create HTTP request for RTCP PCAP export for SIP call ID %s", sipCallID)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth-Token", h.homerAuthToken)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send HTTP request for RTCP PCAP export for SIP call ID %s", sipCallID)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("homer API returned non-success status %s for RTCP PCAP export for SIP call ID %s", resp.Status, sipCallID)
	}

	pcapData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read RTCP PCAP response body for SIP call ID %s", sipCallID)
	}

	log.WithField("pcap_size", len(pcapData)).Debug("Received RTCP PCAP data from Homer.")

	return pcapData, nil
}
