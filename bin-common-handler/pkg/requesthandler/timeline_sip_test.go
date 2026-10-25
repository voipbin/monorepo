package requesthandler

import (
	"context"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	tmsipmessage "monorepo/bin-timeline-manager/models/sipmessage"
)

func Test_TimelineV1SIPWrappers(t *testing.T) {
	callID := uuid.FromStringOrNil("c5d6e7f8-0a1b-4c2d-8e4f-e6f708192a3b")

	// the body keys are sorted by encoding/json, not in the order of the source.
	const requestBody = `{"call_id":"c5d6e7f8-0a1b-4c2d-8e4f-e6f708192a3b","from_time":"2026-03-04T05:00:00Z","sip_call_id":"sip-call-a","to_time":"2026-03-04T06:00:00Z"}`

	analysisWant := &tmsipmessage.SIPAnalysisResponse{
		SIPMessages: []*tmsipmessage.SIPMessage{
			{Timestamp: "2026-03-04T05:00:01Z", Method: "INVITE", SrcIP: "192.0.2.10", SrcPort: 5060, DstIP: "192.0.2.20", DstPort: 5061, Raw: "INVITE sip:a@example.com SIP/2.0"},
		},
		RTCPStats: &tmsipmessage.RTCPStats{MOS: 4.2, Jitter: 3, PacketLossPct: 0.5, RTT: 40},
	}
	const analysisResponse = `{"sip_messages":[{"timestamp":"2026-03-04T05:00:01Z","method":"INVITE","src_ip":"192.0.2.10","src_port":5060,"dst_ip":"192.0.2.20","dst_port":5061,"raw":"INVITE sip:a@example.com SIP/2.0"}],"rtcp_stats":{"mos":4.2,"jitter":3,"packet_loss_pct":0.5,"rtt":40}}`

	cases := []rpcCase{
		{
			name:           "analysis_get",
			queue:          timelineQueue,
			expectURI:      "/v1/sip/analysis",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     requestBody,
			responseData:   analysisResponse,
			call: func(h *requestHandler) (any, error) {
				return h.TimelineV1SIPAnalysisGet(context.Background(), callID, "sip-call-a", "2026-03-04T05:00:00Z", "2026-03-04T06:00:00Z")
			},
			expectRes: analysisWant,
		},
		{
			// the pcap payload is base64 of the four byte pcap magic number.
			name:           "pcap_get",
			queue:          timelineQueue,
			expectURI:      "/v1/sip/pcap",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     requestBody,
			responseData:   `{"data":"1MOyoQ=="}`,
			call: func(h *requestHandler) (any, error) {
				return h.TimelineV1SIPPcapGet(context.Background(), callID, "sip-call-a", "2026-03-04T05:00:00Z", "2026-03-04T06:00:00Z")
			},
			expectRes:  []byte{0xd4, 0xc3, 0xb2, 0xa1},
			statusMode: statusCustom,
		},
		{
			name:           "pcap_get_empty_data",
			queue:          timelineQueue,
			expectURI:      "/v1/sip/pcap",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     requestBody,
			responseData:   `{"data":""}`,
			call: func(h *requestHandler) (any, error) {
				return h.TimelineV1SIPPcapGet(context.Background(), callID, "sip-call-a", "2026-03-04T05:00:00Z", "2026-03-04T06:00:00Z")
			},
			expectRes:  []byte{},
			statusMode: statusCustom,
		},
	}

	runRPCCases(t, cases)
}

func Test_TimelineV1SIPPcapGet_responses(t *testing.T) {
	callID := uuid.FromStringOrNil("c5d6e7f8-0a1b-4c2d-8e4f-e6f708192a3b")

	tests := []struct {
		name string

		statusCode int
		data       string

		expectErr   string
		expectBytes []byte
	}{
		{
			name:        "status just below the error boundary is accepted",
			statusCode:  299,
			data:        `{"data":"1MOyoQ=="}`,
			expectBytes: []byte{0xd4, 0xc3, 0xb2, 0xa1},
		},
		{
			name:       "status at the error boundary is rejected",
			statusCode: 300,
			data:       `{"data":"1MOyoQ=="}`,
			expectErr:  "status 300",
		},
		{
			name:       "body that is not json",
			statusCode: 200,
			data:       `not-json`,
			expectErr:  "could not parse PCAP response",
		},
		{
			name:       "payload that is not base64",
			statusCode: 200,
			data:       `{"data":"***"}`,
			expectErr:  "could not decode PCAP data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := &requestHandler{sock: mockSock}

			mockSock.EXPECT().RequestPublish(gomock.Any(), timelineQueue, gomock.Any()).Return(&sock.Response{
				StatusCode: tt.statusCode,
				DataType:   ContentTypeJSON,
				Data:       []byte(tt.data),
			}, nil)

			res, err := h.TimelineV1SIPPcapGet(context.Background(), callID, "sip-call-a", "2026-03-04T05:00:00Z", "2026-03-04T06:00:00Z")
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if string(res) != string(tt.expectBytes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectBytes, res)
			}
		})
	}
}
