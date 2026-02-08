package sipmessage

import (
	"testing"
)

func TestParseXRTPStat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *RTCPStats
	}{
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "full X-RTP-Stat value",
			input: "MOS=3.8;Jitter=7;PacketLossPct=0;RTT=260682;RTPStat=RTP: 258452 bytes, 1509 packets, 0 errors; RTCP:  1248 bytes, 18 packets, 12 errors",
			want: &RTCPStats{
				MOS:           3.8,
				Jitter:        7,
				PacketLossPct: 0,
				RTT:           260682,
				RTPBytes:      258452,
				RTPPackets:    1509,
				RTPErrors:     0,
				RTCPBytes:     1248,
				RTCPPackets:   18,
				RTCPErrors:    12,
			},
		},
		{
			name:  "with packet loss",
			input: "MOS=2.1;Jitter=45;PacketLossPct=5.3;RTT=350;RTPStat=RTP: 100000 bytes, 800 packets, 40 errors; RTCP: 500 bytes, 10 packets, 2 errors",
			want: &RTCPStats{
				MOS:           2.1,
				Jitter:        45,
				PacketLossPct: 5.3,
				RTT:           350,
				RTPBytes:      100000,
				RTPPackets:    800,
				RTPErrors:     40,
				RTCPBytes:     500,
				RTCPPackets:   10,
				RTCPErrors:    2,
			},
		},
		{
			name:  "without RTPStat portion",
			input: "MOS=4.2;Jitter=3;PacketLossPct=0;RTT=50",
			want: &RTCPStats{
				MOS:           4.2,
				Jitter:        3,
				PacketLossPct: 0,
				RTT:           50,
			},
		},
		{
			name:  "only RTPStat portion",
			input: "RTPStat=RTP: 50000 bytes, 300 packets, 1 errors; RTCP: 600 bytes, 8 packets, 0 errors",
			want: &RTCPStats{
				RTPBytes:    50000,
				RTPPackets:  300,
				RTPErrors:   1,
				RTCPBytes:   600,
				RTCPPackets: 8,
				RTCPErrors:  0,
			},
		},
		{
			name:  "malformed numeric values default to zero",
			input: "MOS=invalid;Jitter=abc;PacketLossPct=;RTT=999",
			want: &RTCPStats{
				MOS:           0,
				Jitter:        0,
				PacketLossPct: 0,
				RTT:           999,
			},
		},
		{
			name:  "trailing semicolon",
			input: "MOS=4.0;Jitter=5;",
			want: &RTCPStats{
				MOS:    4.0,
				Jitter: 5,
			},
		},
		{
			name:  "single field only",
			input: "MOS=3.5",
			want: &RTCPStats{
				MOS: 3.5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseXRTPStat(tt.input)

			if tt.want == nil {
				if got != nil {
					t.Errorf("ParseXRTPStat() = %+v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("ParseXRTPStat() = nil, want %+v", tt.want)
			}

			if got.MOS != tt.want.MOS {
				t.Errorf("MOS = %v, want %v", got.MOS, tt.want.MOS)
			}
			if got.Jitter != tt.want.Jitter {
				t.Errorf("Jitter = %v, want %v", got.Jitter, tt.want.Jitter)
			}
			if got.PacketLossPct != tt.want.PacketLossPct {
				t.Errorf("PacketLossPct = %v, want %v", got.PacketLossPct, tt.want.PacketLossPct)
			}
			if got.RTT != tt.want.RTT {
				t.Errorf("RTT = %v, want %v", got.RTT, tt.want.RTT)
			}
			if got.RTPBytes != tt.want.RTPBytes {
				t.Errorf("RTPBytes = %v, want %v", got.RTPBytes, tt.want.RTPBytes)
			}
			if got.RTPPackets != tt.want.RTPPackets {
				t.Errorf("RTPPackets = %v, want %v", got.RTPPackets, tt.want.RTPPackets)
			}
			if got.RTPErrors != tt.want.RTPErrors {
				t.Errorf("RTPErrors = %v, want %v", got.RTPErrors, tt.want.RTPErrors)
			}
			if got.RTCPBytes != tt.want.RTCPBytes {
				t.Errorf("RTCPBytes = %v, want %v", got.RTCPBytes, tt.want.RTCPBytes)
			}
			if got.RTCPPackets != tt.want.RTCPPackets {
				t.Errorf("RTCPPackets = %v, want %v", got.RTCPPackets, tt.want.RTCPPackets)
			}
			if got.RTCPErrors != tt.want.RTCPErrors {
				t.Errorf("RTCPErrors = %v, want %v", got.RTCPErrors, tt.want.RTCPErrors)
			}
		})
	}
}

func TestExtractXRTPStat(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "BYE with X-RTP-Stat",
			raw: "BYE sip:anonymous@10.96.4.18:5060;transport=TCP SIP/2.0\r\n" +
				"Via: SIP/2.0/TCP 10.164.0.20;branch=z9hG4bK9e22\r\n" +
				"From: <sip:2002@35.204.215.63>;tag=-ESX0NH\r\n" +
				"Call-ID: dadbb998-5209-48d0-9020-3da0ced4bd8d\r\n" +
				"X-RTP-Stat: MOS=3.8;Jitter=7;PacketLossPct=0;RTT=260682;RTPStat=RTP: 258452 bytes, 1509 packets, 0 errors; RTCP:  1248 bytes, 18 packets, 12 errors\r\n" +
				"Content-Length: 0\r\n",
			want: "MOS=3.8;Jitter=7;PacketLossPct=0;RTT=260682;RTPStat=RTP: 258452 bytes, 1509 packets, 0 errors; RTCP:  1248 bytes, 18 packets, 12 errors",
		},
		{
			name: "message without X-RTP-Stat",
			raw: "INVITE sip:+821028286521@10.96.4.18 SIP/2.0\r\n" +
				"Via: SIP/2.0/UDP 192.168.42.10:5070\r\n" +
				"Content-Length: 0\r\n",
			want: "",
		},
		{
			name: "empty raw message",
			raw:  "",
			want: "",
		},
		{
			name: "lowercase header name",
			raw: "BYE sip:test@example.com SIP/2.0\r\n" +
				"x-rtp-stat: MOS=4.0;Jitter=3\r\n" +
				"Content-Length: 0\r\n",
			want: "MOS=4.0;Jitter=3",
		},
		{
			name: "mixed case header name",
			raw: "BYE sip:test@example.com SIP/2.0\r\n" +
				"X-Rtp-Stat: MOS=3.5;Jitter=10\r\n" +
				"Content-Length: 0\r\n",
			want: "MOS=3.5;Jitter=10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractXRTPStat(tt.raw)
			if got != tt.want {
				t.Errorf("ExtractXRTPStat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractRTCPStatsFromMessages(t *testing.T) {
	tests := []struct {
		name     string
		messages []*SIPMessage
		wantNil  bool
		wantMOS  float64
	}{
		{
			name:     "empty messages",
			messages: []*SIPMessage{},
			wantNil:  true,
		},
		{
			name: "no BYE message",
			messages: []*SIPMessage{
				{Method: "INVITE", Raw: "INVITE sip:test@example.com SIP/2.0\r\n"},
				{Method: "200", Raw: "SIP/2.0 200 OK\r\n"},
			},
			wantNil: true,
		},
		{
			name: "BYE without X-RTP-Stat",
			messages: []*SIPMessage{
				{Method: "BYE", Raw: "BYE sip:test@example.com SIP/2.0\r\nContent-Length: 0\r\n"},
			},
			wantNil: true,
		},
		{
			name: "BYE with X-RTP-Stat",
			messages: []*SIPMessage{
				{Method: "INVITE", Raw: "INVITE sip:test@example.com SIP/2.0\r\n"},
				{
					Method: "BYE",
					Raw: "BYE sip:test@example.com SIP/2.0\r\n" +
						"X-RTP-Stat: MOS=4.1;Jitter=5;PacketLossPct=0;RTT=100\r\n" +
						"Content-Length: 0\r\n",
				},
			},
			wantNil: false,
			wantMOS: 4.1,
		},
		{
			name: "multiple BYE with X-RTP-Stat takes last",
			messages: []*SIPMessage{
				{
					Method: "BYE",
					Raw: "BYE sip:test@example.com SIP/2.0\r\n" +
						"X-RTP-Stat: MOS=3.0;Jitter=10;PacketLossPct=1;RTT=200\r\n" +
						"Content-Length: 0\r\n",
				},
				{
					Method: "BYE",
					Raw: "BYE sip:test@example.com SIP/2.0\r\n" +
						"X-RTP-Stat: MOS=4.5;Jitter=2;PacketLossPct=0;RTT=50\r\n" +
						"Content-Length: 0\r\n",
				},
			},
			wantNil: false,
			wantMOS: 4.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractRTCPStatsFromMessages(tt.messages)
			if tt.wantNil {
				if got != nil {
					t.Errorf("ExtractRTCPStatsFromMessages() = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("ExtractRTCPStatsFromMessages() = nil, want non-nil")
			}
			if got.MOS != tt.wantMOS {
				t.Errorf("MOS = %v, want %v", got.MOS, tt.wantMOS)
			}
		})
	}
}
