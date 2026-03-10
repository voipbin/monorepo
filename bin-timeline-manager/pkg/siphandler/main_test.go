package siphandler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"go.uber.org/mock/gomock"

	"monorepo/bin-timeline-manager/models/sipmessage"
	"monorepo/bin-timeline-manager/pkg/homerhandler"
)

func TestNewSIPHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHomer := homerhandler.NewMockHomerHandler(ctrl)
	handler := NewSIPHandler(mockHomer, nil, "")

	if handler == nil {
		t.Error("NewSIPHandler() returned nil")
	}
}

func TestSIPHandler_Interface(t *testing.T) {
	// Ensure sipHandler implements SIPHandler interface
	var _ SIPHandler = (*sipHandler)(nil)
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantPri bool
	}{
		{name: "10.x.x.x private", ip: "10.0.0.1", wantPri: true},
		{name: "10.x.x.x boundary", ip: "10.255.255.255", wantPri: true},
		{name: "172.16-31.x.x private", ip: "172.16.0.1", wantPri: true},
		{name: "172.31.x.x private", ip: "172.31.255.255", wantPri: true},
		{name: "192.168.x.x private", ip: "192.168.1.1", wantPri: true},
		{name: "192.168.x.x boundary", ip: "192.168.255.255", wantPri: true},
		{name: "public IP", ip: "8.8.8.8", wantPri: false},
		{name: "public IP 2", ip: "203.0.113.1", wantPri: false},
		{name: "172.15.x.x not private", ip: "172.15.0.1", wantPri: false},
		{name: "172.32.x.x not private", ip: "172.32.0.1", wantPri: false},
		{name: "11.x.x.x not private", ip: "11.0.0.1", wantPri: false},
		{name: "invalid IP", ip: "invalid", wantPri: false},
		{name: "empty string", ip: "", wantPri: false},
		{name: "localhost", ip: "127.0.0.1", wantPri: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPrivateIP(tt.ip)
			if result != tt.wantPri {
				t.Errorf("isPrivateIP(%q) = %v, want %v", tt.ip, result, tt.wantPri)
			}
		})
	}
}

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

			h := NewSIPHandler(mockHomer, nil, "")

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

func TestGetPcap(t *testing.T) {
	fromTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	// Helper to create a minimal valid PCAP
	createPcap := func(srcIP, dstIP string) []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(65536, layers.LinkTypeEthernet)

		// Create a simple UDP/SIP packet
		eth := &layers.Ethernet{
			SrcMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			DstMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			EthernetType: layers.EthernetTypeIPv4,
		}
		ip := &layers.IPv4{
			Version:  4,
			SrcIP:    []byte(srcIP),
			DstIP:    []byte(dstIP),
			Protocol: layers.IPProtocolUDP,
		}
		udp := &layers.UDP{
			SrcPort: 5060,
			DstPort: 5060,
		}
		_ = udp.SetNetworkLayerForChecksum(ip)

		payload := []byte("INVITE sip:test SIP/2.0")
		packetBuf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{ComputeChecksums: true}
		_ = gopacket.SerializeLayers(packetBuf, opts, eth, ip, udp, gopacket.Payload(payload))
		packetData := packetBuf.Bytes()

		ci := gopacket.CaptureInfo{
			Timestamp:     time.Now(),
			CaptureLength: len(packetData),
			Length:        len(packetData),
		}
		_ = writer.WritePacket(ci, packetData)

		return buf.Bytes()
	}

	tests := []struct {
		name          string
		sipCallID     string
		sipPcapData   []byte
		sipPcapErr    error
		rtcpPcapData  []byte
		rtcpPcapErr   error
		wantErr       bool
		wantEmpty     bool
	}{
		{
			name:        "SIP PCAP error returns error",
			sipCallID:   "call-1",
			sipPcapErr:  fmt.Errorf("pcap fetch failed"),
			wantErr:     true,
		},
		{
			name:        "empty SIP PCAP returns empty",
			sipCallID:   "call-1",
			sipPcapData: []byte{},
			wantEmpty:   true,
		},
		{
			name:         "SIP PCAP only success",
			sipCallID:    "call-1",
			sipPcapData:  createPcap("\x0a\x00\x00\x01", "\xcb\x00\x71\x01"), // 10.0.0.1 -> 203.0.113.1
			rtcpPcapErr:  fmt.Errorf("rtcp unavailable"),
			wantErr:      false,
		},
		{
			name:         "SIP and RTCP PCAP merge success",
			sipCallID:    "call-1",
			sipPcapData:  createPcap("\x0a\x00\x00\x01", "\xcb\x00\x71\x01"), // 10.0.0.1 -> 203.0.113.1
			rtcpPcapData: createPcap("\xcb\x00\x71\x01", "\x0a\x00\x00\x01"), // 203.0.113.1 -> 10.0.0.1
			wantErr:      false,
		},
		{
			name:         "RTCP PCAP empty falls back to SIP only",
			sipCallID:    "call-1",
			sipPcapData:  createPcap("\x0a\x00\x00\x01", "\xcb\x00\x71\x01"),
			rtcpPcapData: []byte{},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockHomer := homerhandler.NewMockHomerHandler(mc)
			mockHomer.EXPECT().GetPcap(gomock.Any(), tt.sipCallID, fromTime, toTime).Return(tt.sipPcapData, tt.sipPcapErr)

			if tt.sipPcapErr == nil && len(tt.sipPcapData) > 0 {
				mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), tt.sipCallID, fromTime, toTime).Return(tt.rtcpPcapData, tt.rtcpPcapErr)
			}

			h := NewSIPHandler(mockHomer, nil, "")

			result, err := h.GetPcap(context.Background(), tt.sipCallID, fromTime, toTime)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantEmpty {
				if len(result) != 0 {
					t.Errorf("expected empty result, got %d bytes", len(result))
				}
			} else {
				if len(result) == 0 {
					t.Error("expected non-empty result, got empty")
				}
			}
		})
	}
}

func TestMergePcaps(t *testing.T) {
	// Helper to create a PCAP with a single packet at a specific timestamp
	createTimestampedPcap := func(ts time.Time) []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(65536, layers.LinkTypeEthernet)

		eth := &layers.Ethernet{
			SrcMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			DstMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			EthernetType: layers.EthernetTypeIPv4,
		}
		ip := &layers.IPv4{
			Version:  4,
			SrcIP:    []byte{10, 0, 0, 1},
			DstIP:    []byte{10, 0, 0, 2},
			Protocol: layers.IPProtocolUDP,
		}
		udp := &layers.UDP{SrcPort: 5060, DstPort: 5060}
		_ = udp.SetNetworkLayerForChecksum(ip)

		packetBuf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{ComputeChecksums: true}
		_ = gopacket.SerializeLayers(packetBuf, opts, eth, ip, udp, gopacket.Payload([]byte("test")))
		packetData := packetBuf.Bytes()

		ci := gopacket.CaptureInfo{
			Timestamp:     ts,
			CaptureLength: len(packetData),
			Length:        len(packetData),
		}
		_ = writer.WritePacket(ci, packetData)

		return buf.Bytes()
	}

	t.Run("merge sorts by timestamp", func(t *testing.T) {
		ts1 := time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC)
		ts2 := time.Date(2026, 1, 1, 0, 0, 2, 0, time.UTC)

		pcap1 := createTimestampedPcap(ts1)
		pcap2 := createTimestampedPcap(ts2)

		merged, err := mergePcaps(pcap1, pcap2)
		if err != nil {
			t.Fatalf("mergePcaps() error = %v", err)
		}

		// Verify merged PCAP has 2 packets
		reader, err := pcapgo.NewReader(bytes.NewReader(merged))
		if err != nil {
			t.Fatalf("failed to read merged PCAP: %v", err)
		}

		count := 0
		for {
			_, _, err := reader.ReadPacketData()
			if err != nil {
				break
			}
			count++
		}

		if count != 2 {
			t.Errorf("merged PCAP has %d packets, want 2", count)
		}
	})

	t.Run("invalid pcap1 skips source", func(t *testing.T) {
		ts := time.Now()
		result, err := mergePcaps([]byte("invalid"), createTimestampedPcap(ts))
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		// Invalid first source is skipped, second source passes through.
		reader, rErr := pcapgo.NewReader(bytes.NewReader(result))
		if rErr != nil {
			t.Fatalf("failed to read result: %v", rErr)
		}
		count := 0
		for {
			_, _, readErr := reader.ReadPacketData()
			if readErr != nil {
				break
			}
			count++
		}
		if count != 1 {
			t.Errorf("expected 1 packet from valid source, got %d", count)
		}
	})

	t.Run("invalid pcap2 skips source", func(t *testing.T) {
		ts := time.Now()
		result, err := mergePcaps(createTimestampedPcap(ts), []byte("invalid"))
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		// Invalid second source is skipped, first source passes through.
		reader, rErr := pcapgo.NewReader(bytes.NewReader(result))
		if rErr != nil {
			t.Fatalf("failed to read result: %v", rErr)
		}
		count := 0
		for {
			_, _, readErr := reader.ReadPacketData()
			if readErr != nil {
				break
			}
			count++
		}
		if count != 1 {
			t.Errorf("expected 1 packet from valid source, got %d", count)
		}
	})
}

func TestFilterInternalPackets(t *testing.T) {
	// Helper to create a PCAP with specific src/dst IPs
	createPcapWithIPs := func(srcIP, dstIP string) []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(65536, layers.LinkTypeEthernet)

		eth := &layers.Ethernet{
			SrcMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			DstMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			EthernetType: layers.EthernetTypeIPv4,
		}
		ip := &layers.IPv4{
			Version:  4,
			SrcIP:    []byte(srcIP),
			DstIP:    []byte(dstIP),
			Protocol: layers.IPProtocolUDP,
		}
		udp := &layers.UDP{SrcPort: 5060, DstPort: 5060}
		_ = udp.SetNetworkLayerForChecksum(ip)

		packetBuf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{ComputeChecksums: true}
		_ = gopacket.SerializeLayers(packetBuf, opts, eth, ip, udp, gopacket.Payload([]byte("test")))
		packetData := packetBuf.Bytes()

		ci := gopacket.CaptureInfo{
			Timestamp:     time.Now(),
			CaptureLength: len(packetData),
			Length:        len(packetData),
		}
		_ = writer.WritePacket(ci, packetData)

		return buf.Bytes()
	}

	tests := []struct {
		name      string
		srcIP     string
		dstIP     string
		wantKept  bool
	}{
		{name: "internal to internal filtered", srcIP: "\x0a\x00\x00\x01", dstIP: "\x0a\x00\x00\x02", wantKept: false},
		{name: "internal to external kept", srcIP: "\x0a\x00\x00\x01", dstIP: "\xcb\x00\x71\x01", wantKept: true},
		{name: "external to internal kept", srcIP: "\xcb\x00\x71\x01", dstIP: "\x0a\x00\x00\x01", wantKept: true},
		{name: "external to external kept", srcIP: "\xcb\x00\x71\x01", dstIP: "\xcb\x00\x71\x02", wantKept: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pcap := createPcapWithIPs(tt.srcIP, tt.dstIP)
			filtered, err := filterInternalPackets(pcap)
			if err != nil {
				t.Fatalf("filterInternalPackets() error = %v", err)
			}

			reader, err := pcapgo.NewReader(bytes.NewReader(filtered))
			if err != nil {
				t.Fatalf("failed to read filtered PCAP: %v", err)
			}

			count := 0
			for {
				_, _, err := reader.ReadPacketData()
				if err != nil {
					break
				}
				count++
			}

			if tt.wantKept && count != 1 {
				t.Errorf("expected packet to be kept, but got %d packets", count)
			}
			if !tt.wantKept && count != 0 {
				t.Errorf("expected packet to be filtered, but got %d packets", count)
			}
		})
	}

	t.Run("invalid PCAP returns error", func(t *testing.T) {
		_, err := filterInternalPackets([]byte("invalid pcap"))
		if err == nil {
			t.Error("expected error for invalid PCAP, got nil")
		}
	})
}

func TestGetPcap_EmptyRTCP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fromTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	// Helper to create a minimal valid PCAP
	createPcap := func() []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(65536, layers.LinkTypeEthernet)

		eth := &layers.Ethernet{
			SrcMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			DstMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			EthernetType: layers.EthernetTypeIPv4,
		}
		ip := &layers.IPv4{
			Version:  4,
			SrcIP:    []byte{10, 0, 0, 1},
			DstIP:    []byte{203, 0, 113, 1},
			Protocol: layers.IPProtocolUDP,
		}
		udp := &layers.UDP{SrcPort: 5060, DstPort: 5060}
		_ = udp.SetNetworkLayerForChecksum(ip)

		packetBuf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{ComputeChecksums: true}
		_ = gopacket.SerializeLayers(packetBuf, opts, eth, ip, udp, gopacket.Payload([]byte("test")))
		packetData := packetBuf.Bytes()

		ci := gopacket.CaptureInfo{
			Timestamp:     time.Now(),
			CaptureLength: len(packetData),
			Length:        len(packetData),
		}
		_ = writer.WritePacket(ci, packetData)

		return buf.Bytes()
	}

	mockHomer := homerhandler.NewMockHomerHandler(ctrl)
	sipPcap := createPcap()
	mockHomer.EXPECT().GetPcap(gomock.Any(), "call-1", fromTime, toTime).Return(sipPcap, nil)
	mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), "call-1", fromTime, toTime).Return([]byte{}, nil)

	h := NewSIPHandler(mockHomer, nil, "")

	result, err := h.GetPcap(context.Background(), "call-1", fromTime, toTime)
	if err != nil {
		t.Fatalf("GetPcap() error = %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestGetPcap_MergeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fromTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	// Create a valid SIP PCAP
	createPcap := func() []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(65536, layers.LinkTypeEthernet)
		return buf.Bytes()
	}

	mockHomer := homerhandler.NewMockHomerHandler(ctrl)
	sipPcap := createPcap()
	invalidRTCP := []byte("invalid pcap")
	mockHomer.EXPECT().GetPcap(gomock.Any(), "call-1", fromTime, toTime).Return(sipPcap, nil)
	mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), "call-1", fromTime, toTime).Return(invalidRTCP, nil)

	h := NewSIPHandler(mockHomer, nil, "")

	result, err := h.GetPcap(context.Background(), "call-1", fromTime, toTime)
	if err != nil {
		t.Fatalf("GetPcap() error = %v", err)
	}
	// Should fall back to SIP only when merge fails
	if len(result) == 0 {
		t.Error("expected fallback to SIP pcap")
	}
}

func TestFilterInternalPackets_IPv6(t *testing.T) {
	// Create a PCAP with IPv6 packets
	var buf bytes.Buffer
	writer := pcapgo.NewWriter(&buf)
	_ = writer.WriteFileHeader(65536, layers.LinkTypeEthernet)

	eth := &layers.Ethernet{
		SrcMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
		DstMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		EthernetType: layers.EthernetTypeIPv6,
	}
	ipv6 := &layers.IPv6{
		Version:    6,
		SrcIP:      []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		DstIP:      []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		NextHeader: layers.IPProtocolUDP,
	}
	udp := &layers.UDP{SrcPort: 5060, DstPort: 5060}
	_ = udp.SetNetworkLayerForChecksum(ipv6)

	packetBuf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{ComputeChecksums: true}
	_ = gopacket.SerializeLayers(packetBuf, opts, eth, ipv6, udp, gopacket.Payload([]byte("test")))
	packetData := packetBuf.Bytes()

	ci := gopacket.CaptureInfo{
		Timestamp:     time.Now(),
		CaptureLength: len(packetData),
		Length:        len(packetData),
	}
	_ = writer.WritePacket(ci, packetData)

	pcap := buf.Bytes()

	filtered, err := filterInternalPackets(pcap)
	if err != nil {
		t.Fatalf("filterInternalPackets() error = %v", err)
	}

	// IPv6 packets should be kept (not in private IPv4 ranges)
	reader, err := pcapgo.NewReader(bytes.NewReader(filtered))
	if err != nil {
		t.Fatalf("failed to read filtered PCAP: %v", err)
	}

	count := 0
	for {
		_, _, err := reader.ReadPacketData()
		if err != nil {
			break
		}
		count++
	}

	if count != 1 {
		t.Errorf("expected 1 IPv6 packet to be kept, got %d", count)
	}
}

func TestFilterInternalPackets_NoIPLayer(t *testing.T) {
	// Create a PCAP with a packet that has no IP layer
	var buf bytes.Buffer
	writer := pcapgo.NewWriter(&buf)
	_ = writer.WriteFileHeader(65536, layers.LinkTypeEthernet)

	eth := &layers.Ethernet{
		SrcMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
		DstMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		EthernetType: layers.EthernetTypeARP,
	}

	packetBuf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	_ = gopacket.SerializeLayers(packetBuf, opts, eth, gopacket.Payload([]byte("arp payload")))
	packetData := packetBuf.Bytes()

	ci := gopacket.CaptureInfo{
		Timestamp:     time.Now(),
		CaptureLength: len(packetData),
		Length:        len(packetData),
	}
	_ = writer.WritePacket(ci, packetData)

	pcap := buf.Bytes()

	filtered, err := filterInternalPackets(pcap)
	if err != nil {
		t.Fatalf("filterInternalPackets() error = %v", err)
	}

	// Packets without IP layer should be kept
	reader, err := pcapgo.NewReader(bytes.NewReader(filtered))
	if err != nil {
		t.Fatalf("failed to read filtered PCAP: %v", err)
	}

	count := 0
	for {
		_, _, err := reader.ReadPacketData()
		if err != nil {
			break
		}
		count++
	}

	if count != 1 {
		t.Errorf("expected 1 non-IP packet to be kept, got %d", count)
	}
}

func TestMergePcaps_LinkTypeMismatch(t *testing.T) {
	createPcapWithLinkType := func(linkType layers.LinkType) []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(65536, linkType)
		return buf.Bytes()
	}

	pcap1 := createPcapWithLinkType(layers.LinkTypeEthernet)
	pcap2 := createPcapWithLinkType(layers.LinkTypeRaw)

	// With mergeMultiplePcaps, mismatched sources are excluded (not errored).
	result, err := mergePcaps(pcap1, pcap2)
	if err != nil {
		t.Fatalf("expected no error for link type mismatch (excluded), got: %v", err)
	}

	reader, err := pcapgo.NewReader(bytes.NewReader(result))
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}

	count := 0
	for {
		_, _, err := reader.ReadPacketData()
		if err != nil {
			break
		}
		count++
	}
	if count != 0 {
		t.Errorf("expected 0 packets, got %d", count)
	}
}

func TestGetPcap_FilterError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fromTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	mockHomer := homerhandler.NewMockHomerHandler(ctrl)
	// Return invalid PCAP data that will fail filtering
	invalidPcap := []byte("invalid pcap data")
	mockHomer.EXPECT().GetPcap(gomock.Any(), "call-1", fromTime, toTime).Return(invalidPcap, nil)
	mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), "call-1", fromTime, toTime).Return([]byte{}, nil)

	h := NewSIPHandler(mockHomer, nil, "")

	result, err := h.GetPcap(context.Background(), "call-1", fromTime, toTime)
	if err != nil {
		t.Fatalf("GetPcap() error = %v", err)
	}
	// Should return unfiltered data when filtering fails
	if !bytes.Equal(result, invalidPcap) {
		t.Error("expected unfiltered data when filter fails")
	}
}

func TestGetSIPAnalysis_NilMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fromTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	mockHomer := homerhandler.NewMockHomerHandler(ctrl)
	mockHomer.EXPECT().GetSIPMessages(gomock.Any(), "call-1", fromTime, toTime).Return(nil, nil)

	h := NewSIPHandler(mockHomer, nil, "")

	resp, err := h.GetSIPAnalysis(context.Background(), "call-1", fromTime, toTime)
	if err != nil {
		t.Fatalf("GetSIPAnalysis() error = %v", err)
	}

	if resp.SIPMessages == nil {
		t.Error("expected non-nil SIPMessages slice")
	}
	if len(resp.SIPMessages) != 0 {
		t.Errorf("expected empty SIPMessages, got %d", len(resp.SIPMessages))
	}
}

func TestGetPcap_MergeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fromTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	// Create valid PCAPs
	createPcap := func(srcIP, dstIP string) []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(65536, layers.LinkTypeEthernet)

		eth := &layers.Ethernet{
			SrcMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			DstMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			EthernetType: layers.EthernetTypeIPv4,
		}
		ip := &layers.IPv4{
			Version:  4,
			SrcIP:    []byte(srcIP),
			DstIP:    []byte(dstIP),
			Protocol: layers.IPProtocolUDP,
		}
		udp := &layers.UDP{SrcPort: 5060, DstPort: 5060}
		_ = udp.SetNetworkLayerForChecksum(ip)

		packetBuf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{ComputeChecksums: true}
		_ = gopacket.SerializeLayers(packetBuf, opts, eth, ip, udp, gopacket.Payload([]byte("test")))
		packetData := packetBuf.Bytes()

		ci := gopacket.CaptureInfo{
			Timestamp:     time.Now(),
			CaptureLength: len(packetData),
			Length:        len(packetData),
		}
		_ = writer.WritePacket(ci, packetData)

		return buf.Bytes()
	}

	mockHomer := homerhandler.NewMockHomerHandler(ctrl)
	sipPcap := createPcap("\x0a\x00\x00\x01", "\xcb\x00\x71\x01")
	rtcpPcap := createPcap("\xcb\x00\x71\x01", "\x0a\x00\x00\x01")
	mockHomer.EXPECT().GetPcap(gomock.Any(), "call-1", fromTime, toTime).Return(sipPcap, nil)
	mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), "call-1", fromTime, toTime).Return(rtcpPcap, nil)

	h := NewSIPHandler(mockHomer, nil, "")

	result, err := h.GetPcap(context.Background(), "call-1", fromTime, toTime)
	if err != nil {
		t.Fatalf("GetPcap() error = %v", err)
	}
	if len(result) == 0 {
		t.Error("expected merged result")
	}

	// Verify merged PCAP has packets
	reader, err := pcapgo.NewReader(bytes.NewReader(result))
	if err != nil {
		t.Fatalf("failed to read merged PCAP: %v", err)
	}

	count := 0
	for {
		_, _, err := reader.ReadPacketData()
		if err != nil {
			break
		}
		count++
	}

	if count == 0 {
		t.Error("expected at least one packet in merged PCAP")
	}
}

func TestIsPrivateIP_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantPri bool
	}{
		{name: "10.0.0.0 start of range", ip: "10.0.0.0", wantPri: true},
		{name: "172.16.0.0 start of range", ip: "172.16.0.0", wantPri: true},
		{name: "192.168.0.0 start of range", ip: "192.168.0.0", wantPri: true},
		{name: "9.255.255.255 not private", ip: "9.255.255.255", wantPri: false},
		{name: "11.0.0.0 not private", ip: "11.0.0.0", wantPri: false},
		{name: "172.15.255.255 not private", ip: "172.15.255.255", wantPri: false},
		{name: "172.32.0.0 not private", ip: "172.32.0.0", wantPri: false},
		{name: "192.167.255.255 not private", ip: "192.167.255.255", wantPri: false},
		{name: "192.169.0.0 not private", ip: "192.169.0.0", wantPri: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPrivateIP(tt.ip)
			if result != tt.wantPri {
				t.Errorf("isPrivateIP(%q) = %v, want %v", tt.ip, result, tt.wantPri)
			}
		})
	}
}

func TestMergePcaps_EmptyPcaps(t *testing.T) {
	// Test merging two empty PCAPs (only headers)
	createEmptyPcap := func() []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(65536, layers.LinkTypeEthernet)
		return buf.Bytes()
	}

	pcap1 := createEmptyPcap()
	pcap2 := createEmptyPcap()

	merged, err := mergePcaps(pcap1, pcap2)
	if err != nil {
		t.Fatalf("mergePcaps() error = %v", err)
	}

	// Should have valid PCAP header but no packets
	reader, err := pcapgo.NewReader(bytes.NewReader(merged))
	if err != nil {
		t.Fatalf("failed to read merged PCAP: %v", err)
	}

	count := 0
	for {
		_, _, err := reader.ReadPacketData()
		if err != nil {
			break
		}
		count++
	}

	if count != 0 {
		t.Errorf("expected 0 packets in merged empty PCAPs, got %d", count)
	}
}

func TestFilterInternalPackets_Mixed(t *testing.T) {
	// Test PCAP with mix of internal-internal, internal-external, and external-external
	var buf bytes.Buffer
	writer := pcapgo.NewWriter(&buf)
	_ = writer.WriteFileHeader(65536, layers.LinkTypeEthernet)

	addPacket := func(srcIP, dstIP string) {
		eth := &layers.Ethernet{
			SrcMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			DstMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			EthernetType: layers.EthernetTypeIPv4,
		}
		ip := &layers.IPv4{
			Version:  4,
			SrcIP:    []byte(srcIP),
			DstIP:    []byte(dstIP),
			Protocol: layers.IPProtocolUDP,
		}
		udp := &layers.UDP{SrcPort: 5060, DstPort: 5060}
		_ = udp.SetNetworkLayerForChecksum(ip)

		packetBuf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{ComputeChecksums: true}
		_ = gopacket.SerializeLayers(packetBuf, opts, eth, ip, udp, gopacket.Payload([]byte("test")))
		packetData := packetBuf.Bytes()

		ci := gopacket.CaptureInfo{
			Timestamp:     time.Now(),
			CaptureLength: len(packetData),
			Length:        len(packetData),
		}
		_ = writer.WritePacket(ci, packetData)
	}

	// Add 3 packets: internal-internal, internal-external, external-external
	addPacket("\x0a\x00\x00\x01", "\x0a\x00\x00\x02")    // internal-internal (filtered)
	addPacket("\x0a\x00\x00\x01", "\xcb\x00\x71\x01")    // internal-external (kept)
	addPacket("\xcb\x00\x71\x01", "\xcb\x00\x71\x02")    // external-external (kept)

	pcap := buf.Bytes()

	filtered, err := filterInternalPackets(pcap)
	if err != nil {
		t.Fatalf("filterInternalPackets() error = %v", err)
	}

	reader, err := pcapgo.NewReader(bytes.NewReader(filtered))
	if err != nil {
		t.Fatalf("failed to read filtered PCAP: %v", err)
	}

	count := 0
	for {
		_, _, err := reader.ReadPacketData()
		if err != nil {
			break
		}
		count++
	}

	if count != 2 {
		t.Errorf("expected 2 packets after filtering, got %d", count)
	}
}

func TestMergeMultiplePcaps(t *testing.T) {
	// Helper to create a PCAP with a single packet at a specific timestamp and link type/snaplen.
	createTimestampedPcapExt := func(ts time.Time, linkType layers.LinkType, snaplen uint32) []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(snaplen, linkType)

		eth := &layers.Ethernet{
			SrcMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			DstMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			EthernetType: layers.EthernetTypeIPv4,
		}
		ip := &layers.IPv4{
			Version:  4,
			SrcIP:    []byte{10, 0, 0, 1},
			DstIP:    []byte{10, 0, 0, 2},
			Protocol: layers.IPProtocolUDP,
		}
		udp := &layers.UDP{SrcPort: 5060, DstPort: 5060}
		_ = udp.SetNetworkLayerForChecksum(ip)

		packetBuf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{ComputeChecksums: true}
		_ = gopacket.SerializeLayers(packetBuf, opts, eth, ip, udp, gopacket.Payload([]byte("test")))
		packetData := packetBuf.Bytes()

		ci := gopacket.CaptureInfo{
			Timestamp:     ts,
			CaptureLength: len(packetData),
			Length:        len(packetData),
		}
		_ = writer.WritePacket(ci, packetData)

		return buf.Bytes()
	}

	// Convenience wrapper using Ethernet and default snaplen.
	createTimestampedPcap := func(ts time.Time) []byte {
		return createTimestampedPcapExt(ts, layers.LinkTypeEthernet, 65536)
	}

	// Helper to create a Raw link-type PCAP with a single packet at a specific timestamp.
	createRawPcap := func(ts time.Time) []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(65536, layers.LinkTypeRaw)

		// For LinkTypeRaw, write a minimal IPv4 packet directly (no Ethernet layer).
		ip := &layers.IPv4{
			Version:  4,
			SrcIP:    []byte{10, 0, 0, 1},
			DstIP:    []byte{10, 0, 0, 2},
			Protocol: layers.IPProtocolUDP,
		}
		udp := &layers.UDP{SrcPort: 5060, DstPort: 5060}
		_ = udp.SetNetworkLayerForChecksum(ip)

		packetBuf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{ComputeChecksums: true}
		_ = gopacket.SerializeLayers(packetBuf, opts, ip, udp, gopacket.Payload([]byte("raw")))
		packetData := packetBuf.Bytes()

		ci := gopacket.CaptureInfo{
			Timestamp:     ts,
			CaptureLength: len(packetData),
			Length:        len(packetData),
		}
		_ = writer.WritePacket(ci, packetData)

		return buf.Bytes()
	}

	t.Run("zero sources returns empty pcap", func(t *testing.T) {
		result, err := mergeMultiplePcaps(nil)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty result, got %d bytes", len(result))
		}
	})

	t.Run("single source passthrough", func(t *testing.T) {
		ts := time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC)
		pcapData := createTimestampedPcap(ts)

		result, err := mergeMultiplePcaps([]io.Reader{bytes.NewReader(pcapData)})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		reader, err := pcapgo.NewReader(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}

		count := 0
		for {
			_, _, err := reader.ReadPacketData()
			if err != nil {
				break
			}
			count++
		}
		if count != 1 {
			t.Errorf("expected 1 packet, got %d", count)
		}
	})

	t.Run("three sources sorted by timestamp", func(t *testing.T) {
		ts1 := time.Date(2026, 1, 1, 0, 0, 2, 0, time.UTC) // middle
		ts2 := time.Date(2026, 1, 1, 0, 0, 3, 0, time.UTC) // latest
		ts3 := time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC) // earliest

		pcap1 := createTimestampedPcap(ts1)
		pcap2 := createTimestampedPcap(ts2)
		pcap3 := createTimestampedPcap(ts3)

		result, err := mergeMultiplePcaps([]io.Reader{
			bytes.NewReader(pcap1),
			bytes.NewReader(pcap2),
			bytes.NewReader(pcap3),
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		reader, err := pcapgo.NewReader(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}

		var timestamps []time.Time
		for {
			_, ci, err := reader.ReadPacketData()
			if err != nil {
				break
			}
			timestamps = append(timestamps, ci.Timestamp)
		}

		if len(timestamps) != 3 {
			t.Fatalf("expected 3 packets, got %d", len(timestamps))
		}

		// Verify timestamps are in ascending order.
		for i := 1; i < len(timestamps); i++ {
			if timestamps[i].Before(timestamps[i-1]) {
				t.Errorf("packets not sorted: ts[%d]=%v is before ts[%d]=%v", i, timestamps[i], i-1, timestamps[i-1])
			}
		}
	})

	t.Run("uses max snaplen across sources", func(t *testing.T) {
		ts := time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC)
		pcapSmall := createTimestampedPcapExt(ts, layers.LinkTypeEthernet, 1500)
		pcapLarge := createTimestampedPcapExt(ts.Add(time.Second), layers.LinkTypeEthernet, 65536)

		result, err := mergeMultiplePcaps([]io.Reader{
			bytes.NewReader(pcapSmall),
			bytes.NewReader(pcapLarge),
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		reader, err := pcapgo.NewReader(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}

		if reader.Snaplen() != 65536 {
			t.Errorf("expected snaplen 65536, got %d", reader.Snaplen())
		}
	})

	t.Run("link type mismatch excludes mismatched source", func(t *testing.T) {
		ts1 := time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC)
		ts2 := time.Date(2026, 1, 1, 0, 0, 2, 0, time.UTC)
		ts3 := time.Date(2026, 1, 1, 0, 0, 3, 0, time.UTC)

		ethPcap1 := createTimestampedPcap(ts1)       // Ethernet
		rawPcap := createRawPcap(ts2)                 // Raw (mismatched)
		ethPcap2 := createTimestampedPcap(ts3)        // Ethernet

		result, err := mergeMultiplePcaps([]io.Reader{
			bytes.NewReader(ethPcap1),
			bytes.NewReader(rawPcap),
			bytes.NewReader(ethPcap2),
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		reader, err := pcapgo.NewReader(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}

		count := 0
		for {
			_, _, err := reader.ReadPacketData()
			if err != nil {
				break
			}
			count++
		}

		// Only the 2 Ethernet packets should be included; Raw excluded.
		if count != 2 {
			t.Errorf("expected 2 packets (Raw excluded), got %d", count)
		}
	})
}

func TestGetPcap_WithGCSRTPPcaps(t *testing.T) {
	fromTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	createPcapWithTimestamp := func(srcIP, dstIP string, ts time.Time) []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(65536, layers.LinkTypeEthernet)

		eth := &layers.Ethernet{
			SrcMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			DstMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			EthernetType: layers.EthernetTypeIPv4,
		}
		ip := &layers.IPv4{
			Version:  4,
			SrcIP:    []byte(srcIP),
			DstIP:    []byte(dstIP),
			Protocol: layers.IPProtocolUDP,
		}
		udp := &layers.UDP{SrcPort: 5060, DstPort: 5060}
		_ = udp.SetNetworkLayerForChecksum(ip)

		packetBuf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{ComputeChecksums: true}
		_ = gopacket.SerializeLayers(packetBuf, opts, eth, ip, udp, gopacket.Payload([]byte("test")))
		packetData := packetBuf.Bytes()

		ci := gopacket.CaptureInfo{Timestamp: ts, CaptureLength: len(packetData), Length: len(packetData)}
		_ = writer.WritePacket(ci, packetData)

		return buf.Bytes()
	}

	t.Run("GCS returns RTP pcaps that get merged", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockHomer := homerhandler.NewMockHomerHandler(mc)
		mockGCS := NewMockGCSReader(mc)

		sipPcap := createPcapWithTimestamp("\x0a\x00\x00\x01", "\xcb\x00\x71\x01", fromTime.Add(time.Second))
		rtpPcap := createPcapWithTimestamp("\xcb\x00\x71\x01", "\x0a\x00\x00\x01", fromTime.Add(2*time.Second))

		mockHomer.EXPECT().GetPcap(gomock.Any(), "call-1", fromTime, toTime).Return(sipPcap, nil)
		mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), "call-1", fromTime, toTime).Return([]byte{}, nil)
		mockHomer.EXPECT().GetSIPMessages(gomock.Any(), "call-1", fromTime, toTime).Return(nil, nil)

		mockGCS.EXPECT().ListObjects(gomock.Any(), "rtp-recordings/call-1-").Return(
			[]string{"rtp-recordings/call-1-ssrc1.pcap"}, nil,
		)
		mockGCS.EXPECT().DownloadObject(gomock.Any(), "rtp-recordings/call-1-ssrc1.pcap", gomock.Any()).DoAndReturn(
			func(_ context.Context, _ string, dest io.Writer) error {
				_, err := dest.Write(rtpPcap)
				return err
			},
		)

		h := NewSIPHandler(mockHomer, mockGCS, "test-bucket")

		result, err := h.GetPcap(context.Background(), "call-1", fromTime, toTime)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reader, err := pcapgo.NewReader(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}

		count := 0
		for {
			_, _, err := reader.ReadPacketData()
			if err != nil {
				break
			}
			count++
		}
		if count != 2 {
			t.Errorf("expected 2 packets (SIP + RTP), got %d", count)
		}
	})

	t.Run("GCS list error degrades gracefully", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockHomer := homerhandler.NewMockHomerHandler(mc)
		mockGCS := NewMockGCSReader(mc)

		sipPcap := createPcapWithTimestamp("\x0a\x00\x00\x01", "\xcb\x00\x71\x01", fromTime.Add(time.Second))

		mockHomer.EXPECT().GetPcap(gomock.Any(), "call-1", fromTime, toTime).Return(sipPcap, nil)
		mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), "call-1", fromTime, toTime).Return([]byte{}, nil)
		mockGCS.EXPECT().ListObjects(gomock.Any(), "rtp-recordings/call-1-").Return(nil, fmt.Errorf("GCS unavailable"))

		h := NewSIPHandler(mockHomer, mockGCS, "test-bucket")

		result, err := h.GetPcap(context.Background(), "call-1", fromTime, toTime)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected SIP-only result, got empty")
		}
	})

	t.Run("GCS empty list returns SIP only", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockHomer := homerhandler.NewMockHomerHandler(mc)
		mockGCS := NewMockGCSReader(mc)

		sipPcap := createPcapWithTimestamp("\x0a\x00\x00\x01", "\xcb\x00\x71\x01", fromTime.Add(time.Second))

		mockHomer.EXPECT().GetPcap(gomock.Any(), "call-1", fromTime, toTime).Return(sipPcap, nil)
		mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), "call-1", fromTime, toTime).Return([]byte{}, nil)
		mockGCS.EXPECT().ListObjects(gomock.Any(), "rtp-recordings/call-1-").Return([]string{}, nil)

		h := NewSIPHandler(mockHomer, mockGCS, "test-bucket")

		result, err := h.GetPcap(context.Background(), "call-1", fromTime, toTime)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected SIP-only result, got empty")
		}
	})

	t.Run("GCS returns multiple RTP pcaps that get merged", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockHomer := homerhandler.NewMockHomerHandler(mc)
		mockGCS := NewMockGCSReader(mc)

		sipPcap := createPcapWithTimestamp("\x0a\x00\x00\x01", "\xcb\x00\x71\x01", fromTime.Add(time.Second))
		rtpPcap1 := createPcapWithTimestamp("\xcb\x00\x71\x01", "\x0a\x00\x00\x01", fromTime.Add(2*time.Second))
		rtpPcap2 := createPcapWithTimestamp("\xcb\x00\x71\x01", "\x0a\x00\x00\x01", fromTime.Add(3*time.Second))

		mockHomer.EXPECT().GetPcap(gomock.Any(), "call-1", fromTime, toTime).Return(sipPcap, nil)
		mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), "call-1", fromTime, toTime).Return([]byte{}, nil)
		mockHomer.EXPECT().GetSIPMessages(gomock.Any(), "call-1", fromTime, toTime).Return(nil, nil)

		mockGCS.EXPECT().ListObjects(gomock.Any(), "rtp-recordings/call-1-").Return(
			[]string{"rtp-recordings/call-1-ssrc1.pcap", "rtp-recordings/call-1-ssrc2.pcap"}, nil,
		)
		mockGCS.EXPECT().DownloadObject(gomock.Any(), "rtp-recordings/call-1-ssrc1.pcap", gomock.Any()).DoAndReturn(
			func(_ context.Context, _ string, dest io.Writer) error {
				_, err := dest.Write(rtpPcap1)
				return err
			},
		)
		mockGCS.EXPECT().DownloadObject(gomock.Any(), "rtp-recordings/call-1-ssrc2.pcap", gomock.Any()).DoAndReturn(
			func(_ context.Context, _ string, dest io.Writer) error {
				_, err := dest.Write(rtpPcap2)
				return err
			},
		)

		h := NewSIPHandler(mockHomer, mockGCS, "test-bucket")

		result, err := h.GetPcap(context.Background(), "call-1", fromTime, toTime)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reader, err := pcapgo.NewReader(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}

		count := 0
		for {
			_, _, err := reader.ReadPacketData()
			if err != nil {
				break
			}
			count++
		}
		if count != 3 {
			t.Errorf("expected 3 packets (SIP + 2 RTP), got %d", count)
		}
	})

	t.Run("GCS partial download failure merges remaining", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockHomer := homerhandler.NewMockHomerHandler(mc)
		mockGCS := NewMockGCSReader(mc)

		sipPcap := createPcapWithTimestamp("\x0a\x00\x00\x01", "\xcb\x00\x71\x01", fromTime.Add(time.Second))
		rtpPcap2 := createPcapWithTimestamp("\xcb\x00\x71\x01", "\x0a\x00\x00\x01", fromTime.Add(3*time.Second))

		mockHomer.EXPECT().GetPcap(gomock.Any(), "call-1", fromTime, toTime).Return(sipPcap, nil)
		mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), "call-1", fromTime, toTime).Return([]byte{}, nil)
		mockHomer.EXPECT().GetSIPMessages(gomock.Any(), "call-1", fromTime, toTime).Return(nil, nil)

		mockGCS.EXPECT().ListObjects(gomock.Any(), "rtp-recordings/call-1-").Return(
			[]string{"rtp-recordings/call-1-ssrc1.pcap", "rtp-recordings/call-1-ssrc2.pcap"}, nil,
		)
		// First download fails
		mockGCS.EXPECT().DownloadObject(gomock.Any(), "rtp-recordings/call-1-ssrc1.pcap", gomock.Any()).Return(fmt.Errorf("download failed"))
		// Second download succeeds
		mockGCS.EXPECT().DownloadObject(gomock.Any(), "rtp-recordings/call-1-ssrc2.pcap", gomock.Any()).DoAndReturn(
			func(_ context.Context, _ string, dest io.Writer) error {
				_, err := dest.Write(rtpPcap2)
				return err
			},
		)

		h := NewSIPHandler(mockHomer, mockGCS, "test-bucket")

		result, err := h.GetPcap(context.Background(), "call-1", fromTime, toTime)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reader, err := pcapgo.NewReader(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}

		count := 0
		for {
			_, _, err := reader.ReadPacketData()
			if err != nil {
				break
			}
			count++
		}
		// SIP + 1 successful RTP (failed one skipped)
		if count != 2 {
			t.Errorf("expected 2 packets (SIP + 1 RTP), got %d", count)
		}
	})

	t.Run("GCS disabled when bucket empty", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockHomer := homerhandler.NewMockHomerHandler(mc)

		sipPcap := createPcapWithTimestamp("\x0a\x00\x00\x01", "\xcb\x00\x71\x01", fromTime.Add(time.Second))

		mockHomer.EXPECT().GetPcap(gomock.Any(), "call-1", fromTime, toTime).Return(sipPcap, nil)
		mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), "call-1", fromTime, toTime).Return([]byte{}, nil)

		h := NewSIPHandler(mockHomer, nil, "")

		result, err := h.GetPcap(context.Background(), "call-1", fromTime, toTime)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected SIP-only result, got empty")
		}
	})
}

func TestParseSDPEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		messages []*sipmessage.SIPMessage
		wantLen  int
		wantIP   string
		wantPort int
	}{
		{
			name: "INVITE with SDP",
			messages: []*sipmessage.SIPMessage{
				{
					Method: "INVITE",
					Raw: "INVITE sip:user@example.com SIP/2.0\r\n" +
						"Content-Type: application/sdp\r\n\r\n" +
						"v=0\r\n" +
						"o=- 123 456 IN IP4 34.91.40.0\r\n" +
						"s=call\r\n" +
						"c=IN IP4 34.91.40.0\r\n" +
						"t=0 0\r\n" +
						"m=audio 36542 RTP/AVP 0 8\r\n" +
						"a=rtpmap:0 PCMU/8000\r\n",
				},
			},
			wantLen:  1,
			wantIP:   "34.91.40.0",
			wantPort: 36542,
		},
		{
			name: "200 OK with SDP",
			messages: []*sipmessage.SIPMessage{
				{
					Method: "200",
					Raw: "SIP/2.0 200 Ok\r\n" +
						"Content-Type: application/sdp\r\n\r\n" +
						"v=0\r\n" +
						"c=IN IP4 192.168.45.152\r\n" +
						"t=0 0\r\n" +
						"m=audio 7078 RTP/AVP 0 8\r\n",
				},
			},
			wantLen:  1,
			wantIP:   "192.168.45.152",
			wantPort: 7078,
		},
		{
			name: "INVITE and 200 OK produce two endpoints",
			messages: []*sipmessage.SIPMessage{
				{
					Method: "INVITE",
					Raw:    "INVITE sip:u@ex SIP/2.0\r\n\r\nv=0\r\nc=IN IP4 34.91.40.0\r\nt=0 0\r\nm=audio 36542 RTP/AVP 0\r\n",
				},
				{
					Method: "200",
					Raw:    "SIP/2.0 200 Ok\r\n\r\nv=0\r\nc=IN IP4 192.168.45.152\r\nt=0 0\r\nm=audio 7078 RTP/AVP 0\r\n",
				},
			},
			wantLen: 2,
		},
		{
			name: "media-level c= overrides session-level",
			messages: []*sipmessage.SIPMessage{
				{
					Method: "INVITE",
					Raw:    "INVITE sip:u@ex SIP/2.0\r\n\r\nv=0\r\nc=IN IP4 1.1.1.1\r\nt=0 0\r\nm=audio 5000 RTP/AVP 0\r\nc=IN IP4 2.2.2.2\r\n",
				},
			},
			wantLen:  1,
			wantIP:   "2.2.2.2",
			wantPort: 5000,
		},
		{
			name:     "no SDP in message",
			messages: []*sipmessage.SIPMessage{{Method: "BYE", Raw: "BYE sip:u@ex SIP/2.0\r\n\r\n"}},
			wantLen:  0,
		},
		{
			name:     "nil messages",
			messages: nil,
			wantLen:  0,
		},
		{
			name:     "empty raw",
			messages: []*sipmessage.SIPMessage{{Method: "INVITE", Raw: ""}},
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoints := parseSDPEndpoints(tt.messages)
			if len(endpoints) != tt.wantLen {
				t.Fatalf("expected %d endpoints, got %d", tt.wantLen, len(endpoints))
			}
			if tt.wantLen > 0 && tt.wantIP != "" {
				if endpoints[0].IP.String() != tt.wantIP {
					t.Errorf("expected IP %s, got %s", tt.wantIP, endpoints[0].IP.String())
				}
				if endpoints[0].Port != tt.wantPort {
					t.Errorf("expected port %d, got %d", tt.wantPort, endpoints[0].Port)
				}
			}
		})
	}
}

func TestRewriteRTPPacketIPs(t *testing.T) {
	// Create a UDP packet: srcIP:srcPort → dstIP:dstPort
	createUDPPcap := func(srcIP, dstIP net.IP, srcPort, dstPort uint16, ts time.Time) []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(65536, layers.LinkTypeEthernet)

		eth := &layers.Ethernet{
			SrcMAC:       net.HardwareAddr{0, 0, 0, 0, 0, 1},
			DstMAC:       net.HardwareAddr{0, 0, 0, 0, 0, 2},
			EthernetType: layers.EthernetTypeIPv4,
		}
		ip := &layers.IPv4{
			Version:  4,
			IHL:      5,
			TTL:      64,
			Protocol: layers.IPProtocolUDP,
			SrcIP:    srcIP,
			DstIP:    dstIP,
		}
		udp := &layers.UDP{
			SrcPort: layers.UDPPort(srcPort),
			DstPort: layers.UDPPort(dstPort),
		}
		_ = udp.SetNetworkLayerForChecksum(ip)

		payload := make([]byte, 160) // RTP-like payload

		serBuf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
		_ = gopacket.SerializeLayers(serBuf, opts, eth, ip, udp, gopacket.Payload(payload))

		ci := gopacket.CaptureInfo{
			Timestamp:     ts,
			CaptureLength: len(serBuf.Bytes()),
			Length:        len(serBuf.Bytes()),
		}
		_ = writer.WritePacket(ci, serBuf.Bytes())
		return buf.Bytes()
	}

	t.Run("rewrites src IP matching SDP port", func(t *testing.T) {
		// Packet: 211.187.233.49:7078 → 10.164.0.13:36542
		// SDP says port 7078 belongs to 192.168.45.152
		pcapData := createUDPPcap(
			net.ParseIP("211.187.233.49"), net.ParseIP("10.164.0.13"),
			7078, 36542, time.Now(),
		)

		endpoints := []sdpMediaEndpoint{
			{IP: net.ParseIP("192.168.45.152").To4(), Port: 7078},
			{IP: net.ParseIP("34.91.40.0").To4(), Port: 36542},
		}

		result, err := rewriteRTPPacketIPs(pcapData, endpoints)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reader, _ := pcapgo.NewReader(bytes.NewReader(result))
		data, _, _ := reader.ReadPacketData()
		packet := gopacket.NewPacket(data, layers.LinkTypeEthernet, gopacket.Default)
		ipv4 := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)

		if ipv4.SrcIP.String() != "192.168.45.152" {
			t.Errorf("expected src IP 192.168.45.152, got %s", ipv4.SrcIP)
		}
		if ipv4.DstIP.String() != "34.91.40.0" {
			t.Errorf("expected dst IP 34.91.40.0, got %s", ipv4.DstIP)
		}
	})

	t.Run("no rewrite for non-matching ports", func(t *testing.T) {
		pcapData := createUDPPcap(
			net.ParseIP("1.2.3.4"), net.ParseIP("5.6.7.8"),
			5060, 5070, time.Now(),
		)

		endpoints := []sdpMediaEndpoint{
			{IP: net.ParseIP("10.0.0.1").To4(), Port: 7078},
		}

		result, err := rewriteRTPPacketIPs(pcapData, endpoints)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reader, _ := pcapgo.NewReader(bytes.NewReader(result))
		data, _, _ := reader.ReadPacketData()
		packet := gopacket.NewPacket(data, layers.LinkTypeEthernet, gopacket.Default)
		ipv4 := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)

		if ipv4.SrcIP.String() != "1.2.3.4" {
			t.Errorf("expected unchanged src IP 1.2.3.4, got %s", ipv4.SrcIP)
		}
		if ipv4.DstIP.String() != "5.6.7.8" {
			t.Errorf("expected unchanged dst IP 5.6.7.8, got %s", ipv4.DstIP)
		}
	})

	t.Run("RTCP port rewritten too", func(t *testing.T) {
		// RTCP is media port + 1
		pcapData := createUDPPcap(
			net.ParseIP("211.187.233.49"), net.ParseIP("10.164.0.13"),
			7079, 36543, time.Now(),
		)

		endpoints := []sdpMediaEndpoint{
			{IP: net.ParseIP("192.168.45.152").To4(), Port: 7078},
			{IP: net.ParseIP("34.91.40.0").To4(), Port: 36542},
		}

		result, err := rewriteRTPPacketIPs(pcapData, endpoints)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reader, _ := pcapgo.NewReader(bytes.NewReader(result))
		data, _, _ := reader.ReadPacketData()
		packet := gopacket.NewPacket(data, layers.LinkTypeEthernet, gopacket.Default)
		ipv4 := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)

		if ipv4.SrcIP.String() != "192.168.45.152" {
			t.Errorf("expected RTCP src IP 192.168.45.152, got %s", ipv4.SrcIP)
		}
		if ipv4.DstIP.String() != "34.91.40.0" {
			t.Errorf("expected RTCP dst IP 34.91.40.0, got %s", ipv4.DstIP)
		}
	})

	t.Run("empty endpoints returns data unchanged", func(t *testing.T) {
		pcapData := createUDPPcap(
			net.ParseIP("1.2.3.4"), net.ParseIP("5.6.7.8"),
			1000, 2000, time.Now(),
		)

		result, err := rewriteRTPPacketIPs(pcapData, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(result, pcapData) {
			t.Error("expected unchanged pcap data with empty endpoints")
		}
	})
}
