package siphandler

import (
	"bytes"
	"context"
	"fmt"
	"strings"
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
	handler := NewSIPHandler(mockHomer)

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

			h := NewSIPHandler(mockHomer)

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

	t.Run("invalid pcap1 returns error", func(t *testing.T) {
		_, err := mergePcaps([]byte("invalid"), createTimestampedPcap(time.Now()))
		if err == nil {
			t.Error("expected error for invalid pcap1, got nil")
		}
	})

	t.Run("invalid pcap2 returns error", func(t *testing.T) {
		_, err := mergePcaps(createTimestampedPcap(time.Now()), []byte("invalid"))
		if err == nil {
			t.Error("expected error for invalid pcap2, got nil")
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

	h := NewSIPHandler(mockHomer)

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

	h := NewSIPHandler(mockHomer)

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
	// Create two PCAPs with different link types
	createPcapWithLinkType := func(linkType layers.LinkType) []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(65536, linkType)
		return buf.Bytes()
	}

	pcap1 := createPcapWithLinkType(layers.LinkTypeEthernet)
	pcap2 := createPcapWithLinkType(layers.LinkTypeRaw)

	_, err := mergePcaps(pcap1, pcap2)
	if err == nil {
		t.Error("expected error for link type mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "link type mismatch") {
		t.Errorf("expected link type mismatch error, got: %v", err)
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

	h := NewSIPHandler(mockHomer)

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

	h := NewSIPHandler(mockHomer)

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

	h := NewSIPHandler(mockHomer)

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
