# Call SIP Timeline & PCAP API Implementation Plan

**Status:** Implemented. The code samples below reflect the original implementation. PR #411 (2026-02-08) subsequently renamed `sip-messages` to `sip-analysis`, added RTCP stats, and merged RTCP packets into PCAP downloads. See [2026-02-08-timeline-rtcp-stats-design.md](2026-02-08-timeline-rtcp-stats-design.md) for the current state.

**Goal:** Add API endpoints to retrieve SIP message timeline and PCAP download for calls via Homer integration.

**Architecture:** API-manager fetches call data, extracts channel's SIP Call-ID, then calls timeline-manager which queries Homer and returns SIP messages or PCAP files uploaded to GCS.

**Tech Stack:** Go, RabbitMQ RPC, Homer API, Google Cloud Storage, Redis caching

---

## Task 1: Add SIP Message Model to bin-timeline-manager

**Files:**
- Create: `bin-timeline-manager/models/sipmessage/sipmessage.go`

**Step 1: Create the SIP message model**

```go
package sipmessage

// SIPMessage represents a single SIP message from Homer
type SIPMessage struct {
	Timestamp string `json:"timestamp"`
	Method    string `json:"method"`
	SrcIP     string `json:"src_ip"`
	SrcPort   int    `json:"src_port"`
	DstIP     string `json:"dst_ip"`
	DstPort   int    `json:"dst_port"`
	Raw       string `json:"raw"`
}

// SIPMessagesResponse is the response for SIP messages list
type SIPMessagesResponse struct {
	CallID     string        `json:"call_id"`
	SIPCallID  string        `json:"sip_call_id"`
	Messages   []*SIPMessage `json:"messages"`
}

// PcapResponse is the response for PCAP download
type PcapResponse struct {
	CallID      string `json:"call_id"`
	DownloadURI string `json:"download_uri"`
	ExpiresAt   string `json:"expires_at"`
}
```

**Step 2: Run verification**

Run: `cd bin-timeline-manager && go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add bin-timeline-manager/models/sipmessage/sipmessage.go
git commit -m "bin-timeline-manager: Add SIP message models

- bin-timeline-manager: Add SIPMessage struct for Homer message data
- bin-timeline-manager: Add SIPMessagesResponse for API response
- bin-timeline-manager: Add PcapResponse for PCAP download URL"
```

---

## Task 2: Add Homer Handler to bin-timeline-manager

**Files:**
- Create: `bin-timeline-manager/pkg/homerhandler/main.go`

**Step 1: Create Homer handler interface and implementation**

```go
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
	maxSIPMessages      = 50
	defaultTimeout      = 30 * time.Second
	timeBufferSeconds   = 30
)

// HomerHandler interface for Homer API operations
type HomerHandler interface {
	GetSIPMessages(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]*sipmessage.SIPMessage, error)
	GetPcap(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]byte, error)
}

type homerHandler struct {
	httpClient     *http.Client
	homerAPIAddr   string
	homerAuthToken string
}

// NewHomerHandler creates a new HomerHandler
func NewHomerHandler(homerAPIAddr, homerAuthToken string) HomerHandler {
	return &homerHandler{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		homerAPIAddr:   homerAPIAddr,
		homerAuthToken: homerAuthToken,
	}
}

// Homer API request/response types
type timeRange struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

type homerRequestParam struct {
	Limit       int    `json:"limit,omitempty"`
	Search      any    `json:"search"`
	Transaction any    `json:"transaction"`
}

type homerRequestPayload struct {
	Timestamp timeRange         `json:"timestamp"`
	Param     homerRequestParam `json:"param"`
}

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

type homerResponseData struct {
	Messages []homerSIPMessageDetail `json:"messages"`
}

type homerAPIResponse struct {
	Data  homerResponseData `json:"data"`
	Total int               `json:"total,omitempty"`
}

func (h *homerHandler) GetSIPMessages(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]*sipmessage.SIPMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GetSIPMessages",
		"sip_call_id": sipCallID,
	})

	if h.homerAPIAddr == "" || h.homerAuthToken == "" {
		return nil, fmt.Errorf("homer API not configured")
	}

	// Add buffer to time range
	fromTS := fromTime.Add(-timeBufferSeconds * time.Second).UnixMilli()
	toTS := toTime.Add(timeBufferSeconds * time.Second).UnixMilli()

	payload := homerRequestPayload{
		Timestamp: timeRange{From: fromTS, To: toTS},
		Param: homerRequestParam{
			Limit: 200, // Fetch more, we'll truncate
			Search: map[string]any{
				"1_call": map[string]any{
					"callid": []string{sipCallID},
				},
			},
			Transaction: map[string]any{},
		},
	}

	endpoint := fmt.Sprintf("%s/api/v3/call/transaction", h.homerAPIAddr)
	respData, err := h.doHomerRequest(ctx, endpoint, payload)
	if err != nil {
		log.Errorf("Homer request failed: %v", err)
		return nil, err
	}

	// Sort by timestamp
	sort.Slice(respData.Data.Messages, func(i, j int) bool {
		return respData.Data.Messages[i].MicroTS < respData.Data.Messages[j].MicroTS
	})

	// Convert and truncate to max messages
	result := make([]*sipmessage.SIPMessage, 0, maxSIPMessages)
	for i, msg := range respData.Data.Messages {
		if i >= maxSIPMessages {
			break
		}
		result = append(result, &sipmessage.SIPMessage{
			Timestamp: time.UnixMicro(msg.MicroTS).Format(time.RFC3339Nano),
			Method:    msg.Method,
			SrcIP:     msg.SrcIP,
			SrcPort:   msg.SrcPort,
			DstIP:     msg.DstIP,
			DstPort:   msg.DstPort,
			Raw:       msg.Raw,
		})
	}

	return result, nil
}

func (h *homerHandler) GetPcap(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]byte, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GetPcap",
		"sip_call_id": sipCallID,
	})

	if h.homerAPIAddr == "" || h.homerAuthToken == "" {
		return nil, fmt.Errorf("homer API not configured")
	}

	// Add buffer to time range
	fromTS := fromTime.Add(-timeBufferSeconds * time.Second).UnixMilli()
	toTS := toTime.Add(timeBufferSeconds * time.Second).UnixMilli()

	payload := homerRequestPayload{
		Timestamp: timeRange{From: fromTS, To: toTS},
		Param: homerRequestParam{
			Limit: 1000,
			Search: map[string]any{
				"1_call": map[string]any{
					"callid":  []string{sipCallID},
					"type":    "string",
					"hepid":   1,
				},
			},
			Transaction: map[string]any{},
		},
	}

	endpoint := fmt.Sprintf("%s/api/v3/export/call/messages/pcap", h.homerAPIAddr)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal payload")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth-Token", h.homerAuthToken)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "homer request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Errorf("Homer PCAP request failed: %s", resp.Status)
		return nil, fmt.Errorf("homer request failed: %s", resp.Status)
	}

	pcapData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read pcap response")
	}

	return pcapData, nil
}

func (h *homerHandler) doHomerRequest(ctx context.Context, endpoint string, payload homerRequestPayload) (*homerAPIResponse, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal payload")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth-Token", h.homerAuthToken)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "homer request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("homer request failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response")
	}

	var result homerAPIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}

	return &result, nil
}
```

**Step 2: Generate mocks**

Run: `cd bin-timeline-manager && go generate ./pkg/homerhandler/...`
Expected: `mock_main.go` generated

**Step 3: Run verification**

Run: `cd bin-timeline-manager && go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add bin-timeline-manager/pkg/homerhandler/
git commit -m "bin-timeline-manager: Add Homer handler for SIP message retrieval

- bin-timeline-manager: Add HomerHandler interface with GetSIPMessages and GetPcap
- bin-timeline-manager: Implement Homer API client for /api/v3/call/transaction
- bin-timeline-manager: Implement Homer PCAP export via /api/v3/export/call/messages/pcap
- bin-timeline-manager: Add 30-second time buffer for call start/end
- bin-timeline-manager: Limit SIP messages to 50 max"
```

---

## Task 3: Add SIP Handler to bin-timeline-manager

**Files:**
- Create: `bin-timeline-manager/pkg/siphandler/main.go`

**Step 1: Create SIP handler**

```go
package siphandler

//go:generate mockgen -package siphandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-timeline-manager/models/sipmessage"
	"monorepo/bin-timeline-manager/pkg/homerhandler"
)

// SIPHandler interface for SIP timeline operations
type SIPHandler interface {
	GetSIPMessages(ctx context.Context, sipCallID string, fromTime, toTime time.Time) (*sipmessage.SIPMessagesResponse, error)
	GetPcap(ctx context.Context, callID uuid.UUID, sipCallID string, fromTime, toTime time.Time) (*sipmessage.PcapResponse, error)
}

type sipHandler struct {
	reqHandler   requesthandler.RequestHandler
	homerHandler homerhandler.HomerHandler
}

// NewSIPHandler creates a new SIPHandler
func NewSIPHandler(
	reqHandler requesthandler.RequestHandler,
	homerHandler homerhandler.HomerHandler,
) SIPHandler {
	return &sipHandler{
		reqHandler:   reqHandler,
		homerHandler: homerHandler,
	}
}

func (h *sipHandler) GetSIPMessages(ctx context.Context, sipCallID string, fromTime, toTime time.Time) (*sipmessage.SIPMessagesResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GetSIPMessages",
		"sip_call_id": sipCallID,
	})

	messages, err := h.homerHandler.GetSIPMessages(ctx, sipCallID, fromTime, toTime)
	if err != nil {
		log.Errorf("Failed to get SIP messages: %v", err)
		return nil, fmt.Errorf("failed to get SIP messages")
	}

	return &sipmessage.SIPMessagesResponse{
		SIPCallID: sipCallID,
		Messages:  messages,
	}, nil
}

func (h *sipHandler) GetPcap(ctx context.Context, callID uuid.UUID, sipCallID string, fromTime, toTime time.Time) (*sipmessage.PcapResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GetPcap",
		"call_id":     callID,
		"sip_call_id": sipCallID,
	})

	// Get PCAP data from Homer
	pcapData, err := h.homerHandler.GetPcap(ctx, sipCallID, fromTime, toTime)
	if err != nil {
		log.Errorf("Failed to get PCAP: %v", err)
		return nil, fmt.Errorf("failed to get PCAP data")
	}

	if len(pcapData) == 0 {
		return nil, fmt.Errorf("no PCAP data available")
	}

	// Upload to storage via storage-manager
	filename := fmt.Sprintf("sip-trace-%s-%d.pcap", callID.String(), time.Now().Unix())

	res, err := h.reqHandler.StorageV1FileCreate(ctx, uuid.Nil, "pcap", callID, filename, "", pcapData)
	if err != nil {
		log.Errorf("Failed to upload PCAP to storage: %v", err)
		return nil, fmt.Errorf("failed to store PCAP")
	}

	return &sipmessage.PcapResponse{
		CallID:      callID.String(),
		DownloadURI: res.URIDownload,
		ExpiresAt:   res.TMDownloadExpire,
	}, nil
}
```

**Step 2: Generate mocks**

Run: `cd bin-timeline-manager && go generate ./pkg/siphandler/...`
Expected: `mock_main.go` generated

**Step 3: Run verification**

Run: `cd bin-timeline-manager && go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add bin-timeline-manager/pkg/siphandler/
git commit -m "bin-timeline-manager: Add SIP handler for timeline operations

- bin-timeline-manager: Add SIPHandler interface
- bin-timeline-manager: Implement GetSIPMessages using HomerHandler
- bin-timeline-manager: Implement GetPcap with storage-manager upload"
```

---

## Task 4: Add Listen Handler Endpoints to bin-timeline-manager

**Files:**
- Modify: `bin-timeline-manager/pkg/listenhandler/main.go`
- Create: `bin-timeline-manager/pkg/listenhandler/v1_sip.go`
- Create: `bin-timeline-manager/pkg/listenhandler/models/request/sip.go`

**Step 1: Update listenhandler main.go to add sipHandler dependency and new routes**

Add to imports:
```go
"monorepo/bin-timeline-manager/pkg/siphandler"
```

Add regex patterns after existing ones:
```go
var (
	regV1SIPMessages = regexp.MustCompile("/v1/sip/messages$")
	regV1SIPPcap     = regexp.MustCompile("/v1/sip/pcap$")
)
```

Update listenHandler struct:
```go
type listenHandler struct {
	sockHandler  sockhandler.SockHandler
	eventHandler eventhandler.EventHandler
	sipHandler   siphandler.SIPHandler
}
```

Update NewListenHandler:
```go
func NewListenHandler(
	sockHandler sockhandler.SockHandler,
	eventHandler eventhandler.EventHandler,
	sipHandler siphandler.SIPHandler,
) ListenHandler {
	return &listenHandler{
		sockHandler:  sockHandler,
		eventHandler: eventHandler,
		sipHandler:   sipHandler,
	}
}
```

Add cases to processRequest switch:
```go
case regV1SIPMessages.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
	requestType = "/sip/messages"
	response, err = h.v1SIPMessagesPost(ctx, m)

case regV1SIPPcap.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
	requestType = "/sip/pcap"
	response, err = h.v1SIPPcapPost(ctx, m)
```

**Step 2: Create request models**

Create `bin-timeline-manager/pkg/listenhandler/models/request/sip.go`:

```go
package request

import "github.com/gofrs/uuid"

// V1SIPMessagesPost is the request for SIP messages
type V1SIPMessagesPost struct {
	CallID    uuid.UUID `json:"call_id"`
	SIPCallID string    `json:"sip_call_id"`
	FromTime  string    `json:"from_time"`
	ToTime    string    `json:"to_time"`
}

// V1SIPPcapPost is the request for PCAP download
type V1SIPPcapPost struct {
	CallID    uuid.UUID `json:"call_id"`
	SIPCallID string    `json:"sip_call_id"`
	FromTime  string    `json:"from_time"`
	ToTime    string    `json:"to_time"`
}
```

**Step 3: Create v1_sip.go handlers**

Create `bin-timeline-manager/pkg/listenhandler/v1_sip.go`:

```go
package listenhandler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/request"
)

func (h *listenHandler) v1SIPMessagesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1SIPMessagesPost",
		"request": m,
	})

	var req request.V1SIPMessagesPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal request: %v", err)
		return simpleResponse(400), nil
	}

	fromTime, err := time.Parse(time.RFC3339, req.FromTime)
	if err != nil {
		log.Errorf("Invalid from_time: %v", err)
		return simpleResponse(400), nil
	}

	toTime, err := time.Parse(time.RFC3339, req.ToTime)
	if err != nil {
		log.Errorf("Invalid to_time: %v", err)
		return simpleResponse(400), nil
	}

	res, err := h.sipHandler.GetSIPMessages(ctx, req.SIPCallID, fromTime, toTime)
	if err != nil {
		log.Errorf("Could not get SIP messages: %v", err)
		return simpleResponse(500), nil
	}

	res.CallID = req.CallID.String()

	data, err := json.Marshal(res)
	if err != nil {
		log.Errorf("Could not marshal response: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1SIPPcapPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1SIPPcapPost",
		"request": m,
	})

	var req request.V1SIPPcapPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal request: %v", err)
		return simpleResponse(400), nil
	}

	fromTime, err := time.Parse(time.RFC3339, req.FromTime)
	if err != nil {
		log.Errorf("Invalid from_time: %v", err)
		return simpleResponse(400), nil
	}

	toTime, err := time.Parse(time.RFC3339, req.ToTime)
	if err != nil {
		log.Errorf("Invalid to_time: %v", err)
		return simpleResponse(400), nil
	}

	res, err := h.sipHandler.GetPcap(ctx, req.CallID, req.SIPCallID, fromTime, toTime)
	if err != nil {
		log.Errorf("Could not get PCAP: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		log.Errorf("Could not marshal response: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
```

**Step 4: Run verification**

Run: `cd bin-timeline-manager && go generate ./... && go build ./...`
Expected: Build succeeds

**Step 5: Commit**

```bash
git add bin-timeline-manager/pkg/listenhandler/
git commit -m "bin-timeline-manager: Add SIP messages and PCAP listen handlers

- bin-timeline-manager: Add /v1/sip/messages endpoint
- bin-timeline-manager: Add /v1/sip/pcap endpoint
- bin-timeline-manager: Add request models for SIP endpoints
- bin-timeline-manager: Wire sipHandler to listenHandler"
```

---

## Task 5: Update bin-timeline-manager Configuration

**Files:**
- Modify: `bin-timeline-manager/internal/config/config.go`

**Step 1: Add Homer configuration fields**

Add to Config struct:
```go
HomerAPIAddress string
HomerAuthToken  string
```

Add to Bootstrap function flags:
```go
cmd.Flags().String("homer_api_address", "", "Homer API address")
_ = viper.BindPFlag("homer_api_address", cmd.Flags().Lookup("homer_api_address"))

cmd.Flags().String("homer_auth_token", "", "Homer auth token")
_ = viper.BindPFlag("homer_auth_token", cmd.Flags().Lookup("homer_auth_token"))
```

Add to LoadGlobalConfig:
```go
HomerAPIAddress: viper.GetString("homer_api_address"),
HomerAuthToken:  viper.GetString("homer_auth_token"),
```

**Step 2: Run verification**

Run: `cd bin-timeline-manager && go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add bin-timeline-manager/internal/config/config.go
git commit -m "bin-timeline-manager: Add Homer configuration

- bin-timeline-manager: Add HOMER_API_ADDRESS config
- bin-timeline-manager: Add HOMER_AUTH_TOKEN config"
```

---

## Task 6: Update bin-timeline-manager Main to Wire New Handlers

**Files:**
- Modify: `bin-timeline-manager/cmd/timeline-manager/main.go`

**Step 1: Import new packages**

Add imports:
```go
"monorepo/bin-timeline-manager/pkg/homerhandler"
"monorepo/bin-timeline-manager/pkg/siphandler"
```

**Step 2: Create handler instances in run function**

After existing handler creation, add:
```go
// Create Homer handler
homerH := homerhandler.NewHomerHandler(cfg.HomerAPIAddress, cfg.HomerAuthToken)

// Create SIP handler
sipH := siphandler.NewSIPHandler(reqHandler, homerH)
```

**Step 3: Update ListenHandler creation**

Update the NewListenHandler call to include sipHandler:
```go
listenH := listenhandler.NewListenHandler(sockHandler, eventH, sipH)
```

**Step 4: Run verification**

Run: `cd bin-timeline-manager && go mod tidy && go build ./...`
Expected: Build succeeds

**Step 5: Commit**

```bash
git add bin-timeline-manager/cmd/timeline-manager/main.go
git commit -m "bin-timeline-manager: Wire Homer and SIP handlers

- bin-timeline-manager: Create HomerHandler with config
- bin-timeline-manager: Create SIPHandler with dependencies
- bin-timeline-manager: Pass sipHandler to ListenHandler"
```

---

## Task 7: Add RPC Methods to bin-common-handler RequestHandler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/timeline.go`

**Step 1: Add new RPC methods**

Add methods to RequestHandler interface and implementation:

```go
// TimelineV1SIPMessagesGet gets SIP messages for a call
func (h *requestHandler) TimelineV1SIPMessagesGet(ctx context.Context, callID uuid.UUID, sipCallID string, fromTime, toTime string) (*sipmessage.SIPMessagesResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TimelineV1SIPMessagesGet",
		"call_id":     callID,
		"sip_call_id": sipCallID,
	})

	uri := "/v1/sip/messages"

	req := map[string]any{
		"call_id":     callID,
		"sip_call_id": sipCallID,
		"from_time":   fromTime,
		"to_time":     toTime,
	}

	m, err := h.sendRequestTimeline(ctx, uri, sock.RequestMethodPost, req)
	if err != nil {
		log.Errorf("Could not send request: %v", err)
		return nil, err
	}

	if m.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %d", m.StatusCode)
	}

	var res sipmessage.SIPMessagesResponse
	if err := json.Unmarshal(m.Data, &res); err != nil {
		log.Errorf("Could not unmarshal response: %v", err)
		return nil, err
	}

	return &res, nil
}

// TimelineV1SIPPcapGet gets PCAP download URL for a call
func (h *requestHandler) TimelineV1SIPPcapGet(ctx context.Context, callID uuid.UUID, sipCallID string, fromTime, toTime string) (*sipmessage.PcapResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TimelineV1SIPPcapGet",
		"call_id":     callID,
		"sip_call_id": sipCallID,
	})

	uri := "/v1/sip/pcap"

	req := map[string]any{
		"call_id":     callID,
		"sip_call_id": sipCallID,
		"from_time":   fromTime,
		"to_time":     toTime,
	}

	m, err := h.sendRequestTimeline(ctx, uri, sock.RequestMethodPost, req)
	if err != nil {
		log.Errorf("Could not send request: %v", err)
		return nil, err
	}

	if m.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %d", m.StatusCode)
	}

	var res sipmessage.PcapResponse
	if err := json.Unmarshal(m.Data, &res); err != nil {
		log.Errorf("Could not unmarshal response: %v", err)
		return nil, err
	}

	return &res, nil
}
```

**Step 2: Add to interface in main.go**

Add to RequestHandler interface:
```go
TimelineV1SIPMessagesGet(ctx context.Context, callID uuid.UUID, sipCallID string, fromTime, toTime string) (*sipmessage.SIPMessagesResponse, error)
TimelineV1SIPPcapGet(ctx context.Context, callID uuid.UUID, sipCallID string, fromTime, toTime string) (*sipmessage.PcapResponse, error)
```

**Step 3: Run verification**

Run: `cd bin-common-handler && go generate ./... && go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add bin-common-handler/pkg/requesthandler/
git commit -m "bin-common-handler: Add Timeline SIP RPC methods

- bin-common-handler: Add TimelineV1SIPMessagesGet RPC method
- bin-common-handler: Add TimelineV1SIPPcapGet RPC method"
```

---

## Task 8: Add Channel RPC Method to bin-common-handler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/call.go` (or create channel.go)

**Step 1: Add ChannelGet RPC method**

We need to get the channel to retrieve the SIP Call-ID. Add:

```go
// CallV1ChannelGet gets a channel by ID
func (h *requestHandler) CallV1ChannelGet(ctx context.Context, channelID string) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "CallV1ChannelGet",
		"channel_id": channelID,
	})

	uri := fmt.Sprintf("/v1/channels/%s", channelID)

	m, err := h.sendRequestCall(ctx, uri, sock.RequestMethodGet, nil)
	if err != nil {
		log.Errorf("Could not send request: %v", err)
		return nil, err
	}

	if m.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %d", m.StatusCode)
	}

	var res channel.Channel
	if err := json.Unmarshal(m.Data, &res); err != nil {
		log.Errorf("Could not unmarshal response: %v", err)
		return nil, err
	}

	return &res, nil
}
```

**Step 2: Add to RequestHandler interface**

```go
CallV1ChannelGet(ctx context.Context, channelID string) (*channel.Channel, error)
```

**Step 3: Add channel endpoint to bin-call-manager listenhandler**

If not exists, add `/v1/channels/{id}` endpoint to bin-call-manager.

**Step 4: Run verification**

Run: `cd bin-common-handler && go generate ./... && go build ./...`
Expected: Build succeeds

**Step 5: Commit**

```bash
git add bin-common-handler/pkg/requesthandler/
git commit -m "bin-common-handler: Add Channel RPC method

- bin-common-handler: Add CallV1ChannelGet RPC method for SIP Call-ID retrieval"
```

---

## Task 9: Add Service Handler Methods to bin-api-manager

**Files:**
- Create: `bin-api-manager/pkg/servicehandler/timeline_sip.go`

**Step 1: Create timeline SIP service handler methods**

```go
package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-timeline-manager/models/sipmessage"
)

// TimelineSIPMessagesGet gets SIP messages for a call
func (h *serviceHandler) TimelineSIPMessagesGet(
	ctx context.Context,
	a *amagent.Agent,
	callID uuid.UUID,
) (*sipmessage.SIPMessagesResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TimelineSIPMessagesGet",
		"customer_id": a.CustomerID,
		"call_id":     callID,
	})

	// Get call to verify ownership and get timing
	call, err := h.callGet(ctx, callID)
	if err != nil {
		log.Infof("Could not get call: %v", err)
		return nil, fmt.Errorf("call not found")
	}

	// Check permission
	if !h.hasPermission(ctx, a, call.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("Agent has no permission")
		return nil, fmt.Errorf("permission denied")
	}

	// Get channel to retrieve SIP Call-ID
	if call.ChannelID == "" {
		log.Info("Call has no channel ID")
		return nil, fmt.Errorf("no SIP data available for this call")
	}

	channel, err := h.reqHandler.CallV1ChannelGet(ctx, call.ChannelID)
	if err != nil {
		log.Errorf("Could not get channel: %v", err)
		return nil, fmt.Errorf("no SIP data available for this call")
	}

	if channel.SIPCallID == "" {
		log.Info("Channel has no SIP Call-ID")
		return nil, fmt.Errorf("no SIP data available for this call")
	}

	// Determine time range from call
	fromTime := call.TMCreate
	toTime := call.TMHangup
	if toTime == "" || toTime == "9999-01-01T00:00:00.000000Z" {
		toTime = call.TMUpdate // Use update time if not hung up
	}

	// Call timeline-manager
	res, err := h.reqHandler.TimelineV1SIPMessagesGet(ctx, callID, channel.SIPCallID, fromTime, toTime)
	if err != nil {
		log.Errorf("Could not get SIP messages: %v", err)
		return nil, fmt.Errorf("upstream service unavailable")
	}

	return res, nil
}

// TimelineSIPPcapGet gets PCAP download URL for a call
func (h *serviceHandler) TimelineSIPPcapGet(
	ctx context.Context,
	a *amagent.Agent,
	callID uuid.UUID,
) (*sipmessage.PcapResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TimelineSIPPcapGet",
		"customer_id": a.CustomerID,
		"call_id":     callID,
	})

	// Get call to verify ownership and get timing
	call, err := h.callGet(ctx, callID)
	if err != nil {
		log.Infof("Could not get call: %v", err)
		return nil, fmt.Errorf("call not found")
	}

	// Check permission
	if !h.hasPermission(ctx, a, call.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("Agent has no permission")
		return nil, fmt.Errorf("permission denied")
	}

	// Get channel to retrieve SIP Call-ID
	if call.ChannelID == "" {
		log.Info("Call has no channel ID")
		return nil, fmt.Errorf("no SIP data available for this call")
	}

	channel, err := h.reqHandler.CallV1ChannelGet(ctx, call.ChannelID)
	if err != nil {
		log.Errorf("Could not get channel: %v", err)
		return nil, fmt.Errorf("no SIP data available for this call")
	}

	if channel.SIPCallID == "" {
		log.Info("Channel has no SIP Call-ID")
		return nil, fmt.Errorf("no SIP data available for this call")
	}

	// Determine time range from call
	fromTime := call.TMCreate
	toTime := call.TMHangup
	if toTime == "" || toTime == "9999-01-01T00:00:00.000000Z" {
		toTime = call.TMUpdate
	}

	// Call timeline-manager
	res, err := h.reqHandler.TimelineV1SIPPcapGet(ctx, callID, channel.SIPCallID, fromTime, toTime)
	if err != nil {
		log.Errorf("Could not get PCAP: %v", err)
		return nil, fmt.Errorf("upstream service unavailable")
	}

	return res, nil
}
```

**Step 2: Run verification**

Run: `cd bin-api-manager && go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add bin-api-manager/pkg/servicehandler/timeline_sip.go
git commit -m "bin-api-manager: Add Timeline SIP service handlers

- bin-api-manager: Add TimelineSIPMessagesGet with call ownership validation
- bin-api-manager: Add TimelineSIPPcapGet with call ownership validation
- bin-api-manager: Resolve SIP Call-ID from channel data"
```

---

## Task 10: Add HTTP Handlers to bin-api-manager

**Files:**
- Modify: `bin-api-manager/pkg/listenhandler/main.go` (add routes)
- Create: `bin-api-manager/pkg/listenhandler/v1_timelines_sip.go`

**Step 1: Add regex patterns to main.go**

```go
var (
	regV1TimelinesCallSIPMessages = regexp.MustCompile("/v1/timelines/call/" + regUUID + "/sip-messages$")
	regV1TimelinesCallPcap        = regexp.MustCompile("/v1/timelines/call/" + regUUID + "/pcap$")
)
```

**Step 2: Add cases to route handler**

```go
case regV1TimelinesCallSIPMessages.MatchString(r.URL.Path) && r.Method == http.MethodGet:
	h.TimelineCallSIPMessagesGet(w, r)

case regV1TimelinesCallPcap.MatchString(r.URL.Path) && r.Method == http.MethodGet:
	h.TimelineCallPcapGet(w, r)
```

**Step 3: Create v1_timelines_sip.go**

```go
package listenhandler

import (
	"net/http"
	"regexp"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

var regTimelineCallID = regexp.MustCompile(`/v1/timelines/call/([0-9a-f-]+)/`)

func (h *listenHandler) TimelineCallSIPMessagesGet(w http.ResponseWriter, r *http.Request) {
	log := logrus.WithFields(logrus.Fields{
		"func": "TimelineCallSIPMessagesGet",
	})

	// Extract call ID from path
	matches := regTimelineCallID.FindStringSubmatch(r.URL.Path)
	if len(matches) < 2 {
		log.Info("Invalid path")
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	callID, err := uuid.FromString(matches[1])
	if err != nil {
		log.Infof("Invalid call ID: %v", err)
		h.writeError(w, http.StatusBadRequest, "invalid call ID")
		return
	}

	// Get agent from context
	a, err := h.getAgent(r)
	if err != nil {
		log.Infof("Could not get agent: %v", err)
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	res, err := h.serviceHandler.TimelineSIPMessagesGet(r.Context(), a, callID)
	if err != nil {
		log.Infof("Could not get SIP messages: %v", err)
		switch err.Error() {
		case "call not found":
			h.writeError(w, http.StatusNotFound, err.Error())
		case "permission denied":
			h.writeError(w, http.StatusForbidden, err.Error())
		case "no SIP data available for this call":
			h.writeError(w, http.StatusNotFound, err.Error())
		default:
			h.writeError(w, http.StatusBadGateway, "upstream service unavailable")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}

func (h *listenHandler) TimelineCallPcapGet(w http.ResponseWriter, r *http.Request) {
	log := logrus.WithFields(logrus.Fields{
		"func": "TimelineCallPcapGet",
	})

	// Extract call ID from path
	matches := regTimelineCallID.FindStringSubmatch(r.URL.Path)
	if len(matches) < 2 {
		log.Info("Invalid path")
		h.writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	callID, err := uuid.FromString(matches[1])
	if err != nil {
		log.Infof("Invalid call ID: %v", err)
		h.writeError(w, http.StatusBadRequest, "invalid call ID")
		return
	}

	// Get agent from context
	a, err := h.getAgent(r)
	if err != nil {
		log.Infof("Could not get agent: %v", err)
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	res, err := h.serviceHandler.TimelineSIPPcapGet(r.Context(), a, callID)
	if err != nil {
		log.Infof("Could not get PCAP: %v", err)
		switch err.Error() {
		case "call not found":
			h.writeError(w, http.StatusNotFound, err.Error())
		case "permission denied":
			h.writeError(w, http.StatusForbidden, err.Error())
		case "no SIP data available for this call":
			h.writeError(w, http.StatusNotFound, err.Error())
		default:
			h.writeError(w, http.StatusBadGateway, "upstream service unavailable")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}
```

**Step 4: Run verification**

Run: `cd bin-api-manager && go build ./...`
Expected: Build succeeds

**Step 5: Commit**

```bash
git add bin-api-manager/pkg/listenhandler/
git commit -m "bin-api-manager: Add Timeline SIP HTTP handlers

- bin-api-manager: Add GET /v1/timelines/call/{id}/sip-messages endpoint
- bin-api-manager: Add GET /v1/timelines/call/{id}/pcap endpoint
- bin-api-manager: Map errors to appropriate HTTP status codes"
```

---

## Task 11: Add Channel Endpoint to bin-call-manager

**Files:**
- Modify: `bin-call-manager/pkg/listenhandler/main.go`
- Create: `bin-call-manager/pkg/listenhandler/v1_channels.go`

**Step 1: Add regex pattern**

```go
var regV1ChannelsID = regexp.MustCompile("/v1/channels/[^/]+$")
```

**Step 2: Add case to processRequest**

```go
case regV1ChannelsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
	requestType = "/channels/{id}"
	response, err = h.v1ChannelsIDGet(ctx, m)
```

**Step 3: Create v1_channels.go**

```go
package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
)

func (h *listenHandler) v1ChannelsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1ChannelsIDGet",
		"request": m,
	})

	// Extract channel ID from URI
	parts := strings.Split(m.URI, "/")
	if len(parts) < 3 {
		log.Error("Invalid URI")
		return simpleResponse(400), nil
	}
	channelID := parts[len(parts)-1]

	res, err := h.channelHandler.Get(ctx, channelID)
	if err != nil {
		log.Errorf("Could not get channel: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		log.Errorf("Could not marshal response: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
```

**Step 4: Run verification**

Run: `cd bin-call-manager && go build ./...`
Expected: Build succeeds

**Step 5: Commit**

```bash
git add bin-call-manager/pkg/listenhandler/
git commit -m "bin-call-manager: Add Channel GET endpoint

- bin-call-manager: Add GET /v1/channels/{id} endpoint for internal RPC"
```

---

## Task 12: Add OpenAPI Schema Definitions

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`
- Create: `bin-openapi-manager/openapi/paths/timelines/call_id_sip_messages.yaml`
- Create: `bin-openapi-manager/openapi/paths/timelines/call_id_pcap.yaml`

**Step 1: Add schema definitions to openapi.yaml**

Add to components/schemas:

```yaml
TimelineManagerSIPMessage:
  type: object
  properties:
    timestamp:
      type: string
      format: date-time
    method:
      type: string
    src_ip:
      type: string
    src_port:
      type: integer
    dst_ip:
      type: string
    dst_port:
      type: integer
    raw:
      type: string

TimelineManagerSIPMessagesResponse:
  type: object
  properties:
    call_id:
      type: string
      format: uuid
    sip_call_id:
      type: string
    messages:
      type: array
      items:
        $ref: '#/components/schemas/TimelineManagerSIPMessage'

TimelineManagerPcapResponse:
  type: object
  properties:
    call_id:
      type: string
      format: uuid
    download_uri:
      type: string
      format: uri
    expires_at:
      type: string
      format: date-time
```

**Step 2: Add path definitions**

Create path files referencing the schemas.

**Step 3: Run verification**

Run: `cd bin-openapi-manager && go generate ./...`
Expected: Models regenerated

**Step 4: Commit**

```bash
git add bin-openapi-manager/
git commit -m "bin-openapi-manager: Add Timeline SIP API schemas

- bin-openapi-manager: Add TimelineManagerSIPMessage schema
- bin-openapi-manager: Add TimelineManagerSIPMessagesResponse schema
- bin-openapi-manager: Add TimelineManagerPcapResponse schema
- bin-openapi-manager: Add path definitions for new endpoints"
```

---

## Task 13: Run Full Verification

**Step 1: Run verification for all affected services**

```bash
for svc in bin-timeline-manager bin-common-handler bin-api-manager bin-call-manager bin-openapi-manager; do
  echo "=== $svc ===" && \
  (cd $svc && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m) || echo "FAILED: $svc"
done
```

**Step 2: Fix any issues found**

**Step 3: Final commit**

```bash
git add -A
git commit -m "All services: Final verification and fixes

- All affected services pass tests and linting"
```

---

## Task 14: Update Kubernetes Deployment (if needed)

**Files:**
- Modify: `bin-timeline-manager/k8s/deployment.yaml`

**Step 1: Add environment variables**

```yaml
env:
  - name: HOMER_API_ADDRESS
    valueFrom:
      secretKeyRef:
        name: timeline-manager-secrets
        key: homer-api-address
  - name: HOMER_AUTH_TOKEN
    valueFrom:
      secretKeyRef:
        name: timeline-manager-secrets
        key: homer-auth-token
```

**Step 2: Commit**

```bash
git add bin-timeline-manager/k8s/
git commit -m "bin-timeline-manager: Add Homer env vars to k8s deployment

- bin-timeline-manager: Add HOMER_API_ADDRESS from secret
- bin-timeline-manager: Add HOMER_AUTH_TOKEN from secret"
```

---

## Summary

This plan implements:
1. **bin-timeline-manager**: Homer handler, SIP handler, listen endpoints
2. **bin-common-handler**: RPC methods for timeline SIP and channel operations
3. **bin-api-manager**: Service handlers and HTTP endpoints
4. **bin-call-manager**: Channel GET endpoint
5. **bin-openapi-manager**: API schema definitions

Total estimated tasks: 14
