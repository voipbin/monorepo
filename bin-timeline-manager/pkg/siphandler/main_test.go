package siphandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-timeline-manager/models/sipmessage"
	"monorepo/bin-timeline-manager/pkg/homerhandler"
)

func TestGetSIPAnalysis(t *testing.T) {
	fromTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		sipCallID      string
		homerMessages  []*sipmessage.SIPMessage
		homerErr       error
		wantErr        bool
		wantCount      int
		wantRTCPStats  bool
		wantMOS        float64
	}{
		{
			name:      "homer error returns error",
			sipCallID: "call-1",
			homerErr:  fmt.Errorf("homer unavailable"),
			wantErr:   true,
		},
		{
			name:      "no messages returns empty result",
			sipCallID: "call-1",
			homerMessages: []*sipmessage.SIPMessage{},
			wantCount:     0,
			wantRTCPStats: false,
		},
		{
			name:      "filters internal-to-internal messages",
			sipCallID: "call-1",
			homerMessages: []*sipmessage.SIPMessage{
				{Method: "INVITE", SrcIP: "203.0.113.1", DstIP: "10.96.4.18", Raw: "INVITE sip:test SIP/2.0\r\n"},
				{Method: "100", SrcIP: "10.96.4.18", DstIP: "10.164.0.20", Raw: "SIP/2.0 100 Trying\r\n"},
				{Method: "200", SrcIP: "10.96.4.18", DstIP: "203.0.113.1", Raw: "SIP/2.0 200 OK\r\n"},
			},
			wantCount:     2, // INVITE and 200 kept, 100 (internal-to-internal) filtered
			wantRTCPStats: false,
		},
		{
			name:      "extracts RTCP stats from internal BYE before filtering",
			sipCallID: "call-1",
			homerMessages: []*sipmessage.SIPMessage{
				{Method: "INVITE", SrcIP: "203.0.113.1", DstIP: "10.96.4.18", Raw: "INVITE sip:test SIP/2.0\r\n"},
				{Method: "200", SrcIP: "10.96.4.18", DstIP: "203.0.113.1", Raw: "SIP/2.0 200 OK\r\n"},
				{
					Method: "BYE",
					SrcIP:  "10.164.0.20",
					DstIP:  "10.96.4.18",
					Raw: "BYE sip:anonymous@10.96.4.18:5060 SIP/2.0\r\n" +
						"X-RTP-Stat: MOS=3.8;Jitter=7;PacketLossPct=0;RTT=260682;RTPStat=RTP: 258452 bytes, 1509 packets, 0 errors; RTCP:  1248 bytes, 18 packets, 12 errors\r\n" +
						"Content-Length: 0\r\n",
				},
			},
			wantCount:     2, // BYE is internal-to-internal, gets filtered
			wantRTCPStats: true,
			wantMOS:       3.8,
		},
		{
			name:      "BYE without X-RTP-Stat yields no RTCP stats",
			sipCallID: "call-1",
			homerMessages: []*sipmessage.SIPMessage{
				{Method: "INVITE", SrcIP: "203.0.113.1", DstIP: "10.96.4.18", Raw: "INVITE sip:test SIP/2.0\r\n"},
				{
					Method: "BYE",
					SrcIP:  "10.164.0.20",
					DstIP:  "10.96.4.18",
					Raw:    "BYE sip:anonymous@10.96.4.18:5060 SIP/2.0\r\nContent-Length: 0\r\n",
				},
			},
			wantCount:     1,
			wantRTCPStats: false,
		},
		{
			name:      "external BYE with X-RTP-Stat is kept in messages and stats extracted",
			sipCallID: "call-1",
			homerMessages: []*sipmessage.SIPMessage{
				{
					Method: "BYE",
					SrcIP:  "203.0.113.1",
					DstIP:  "10.96.4.18",
					Raw: "BYE sip:test@10.96.4.18 SIP/2.0\r\n" +
						"X-RTP-Stat: MOS=4.2;Jitter=3;PacketLossPct=0;RTT=50\r\n" +
						"Content-Length: 0\r\n",
				},
			},
			wantCount:     1, // external BYE is kept
			wantRTCPStats: true,
			wantMOS:       4.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockHomer := homerhandler.NewMockHomerHandler(mc)
			mockHomer.EXPECT().GetSIPMessages(gomock.Any(), tt.sipCallID, fromTime, toTime).Return(tt.homerMessages, tt.homerErr)

			h := NewSIPHandler(mockHomer)

			resp, err := h.GetSIPAnalysis(context.Background(), tt.sipCallID, fromTime, toTime)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(resp.SIPMessages) != tt.wantCount {
				t.Errorf("SIPMessages count = %d, want %d", len(resp.SIPMessages), tt.wantCount)
			}

			if tt.wantRTCPStats {
				if resp.RTCPStats == nil {
					t.Fatal("expected RTCPStats, got nil")
				}
				if resp.RTCPStats.MOS != tt.wantMOS {
					t.Errorf("MOS = %v, want %v", resp.RTCPStats.MOS, tt.wantMOS)
				}
			} else {
				if resp.RTCPStats != nil {
					t.Errorf("expected nil RTCPStats, got %+v", resp.RTCPStats)
				}
			}
		})
	}
}
