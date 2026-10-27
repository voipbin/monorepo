package siphandler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"go.uber.org/mock/gomock"

	"monorepo/bin-timeline-manager/models/sipmessage"
	"monorepo/bin-timeline-manager/pkg/homerhandler"
)

// rtpxFrameSpec describes one synthetic UDP packet. All addresses are from the documentation ranges.
type rtpxFrameSpec struct {
	linkType layers.LinkType
	vlan     bool
	tcp      bool
	ipv6     bool

	srcIP, dstIP     string
	srcPort, dstPort uint16
	payload          string
}

// rtpxFrame serializes the packet of the spec with computed lengths and checksums.
func rtpxFrame(t *testing.T, spec rtpxFrameSpec) []byte {
	t.Helper()

	var stack []gopacket.SerializableLayer
	var network gopacket.NetworkLayer

	switch spec.linkType {
	case layers.LinkTypeEthernet:
		eth := &layers.Ethernet{
			SrcMAC:       net.HardwareAddr{0, 0, 0, 0, 0, 1},
			DstMAC:       net.HardwareAddr{0, 0, 0, 0, 0, 2},
			EthernetType: layers.EthernetTypeIPv4,
		}
		if spec.ipv6 {
			eth.EthernetType = layers.EthernetTypeIPv6
		}
		if spec.vlan {
			eth.EthernetType = layers.EthernetTypeDot1Q
			stack = append(stack, eth, &layers.Dot1Q{VLANIdentifier: 7, Type: layers.EthernetTypeIPv4})
		} else {
			stack = append(stack, eth)
		}
	case layers.LinkTypeNull:
		stack = append(stack, &layers.Loopback{Family: layers.ProtocolFamilyIPv4})
	case layers.LinkTypeRaw:
		// no link layer
	default:
		t.Fatalf("Wrong match. unsupported link type in the test helper: %v", spec.linkType)
	}

	proto := layers.IPProtocolUDP
	if spec.tcp {
		proto = layers.IPProtocolTCP
	}
	if spec.ipv6 {
		ip6 := &layers.IPv6{Version: 6, HopLimit: 64, NextHeader: proto, SrcIP: net.ParseIP(spec.srcIP), DstIP: net.ParseIP(spec.dstIP)}
		stack = append(stack, ip6)
		network = ip6
	} else {
		ip4 := &layers.IPv4{Version: 4, IHL: 5, TTL: 64, Protocol: proto, SrcIP: net.ParseIP(spec.srcIP).To4(), DstIP: net.ParseIP(spec.dstIP).To4()}
		stack = append(stack, ip4)
		network = ip4
	}

	if spec.tcp {
		tcp := &layers.TCP{SrcPort: layers.TCPPort(spec.srcPort), DstPort: layers.TCPPort(spec.dstPort), SYN: true, Window: 1024}
		_ = tcp.SetNetworkLayerForChecksum(network)
		stack = append(stack, tcp)
	} else {
		udp := &layers.UDP{SrcPort: layers.UDPPort(spec.srcPort), DstPort: layers.UDPPort(spec.dstPort)}
		_ = udp.SetNetworkLayerForChecksum(network)
		stack = append(stack, udp)
	}
	stack = append(stack, gopacket.Payload([]byte(spec.payload)))

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
	if err := gopacket.SerializeLayers(buf, opts, stack...); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	return buf.Bytes()
}

// rtpxZeroUDPChecksum clears the UDP checksum of an Ethernet or raw IPv4 frame without options.
func rtpxZeroUDPChecksum(frame []byte, ipOffset int) []byte {
	out := append([]byte(nil), frame...)
	out[ipOffset+20+6] = 0
	out[ipOffset+20+7] = 0

	return out
}

type rtpxPacket struct {
	ts   time.Time
	data []byte
}

// rtpxPcap builds a pcap file with the given packets.
func rtpxPcap(t *testing.T, linkType layers.LinkType, packets ...rtpxPacket) []byte {
	t.Helper()

	var buf bytes.Buffer
	w := pcapgo.NewWriter(&buf)
	if err := w.WriteFileHeader(65536, linkType); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	for _, p := range packets {
		ci := gopacket.CaptureInfo{Timestamp: p.ts, CaptureLength: len(p.data), Length: len(p.data)}
		if err := w.WritePacket(ci, p.data); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	return buf.Bytes()
}

// rtpxReadPcap returns the packet bytes of a pcap file in file order.
func rtpxReadPcap(t *testing.T, data []byte) [][]byte {
	t.Helper()

	r, err := pcapgo.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res := [][]byte{}
	for {
		d, _, errRead := r.ReadPacketData()
		if errRead != nil {
			break
		}
		res = append(res, d)
	}

	return res
}

func Test_rewritePacketIPs(t *testing.T) {
	portToIP := map[int]net.IP{
		40000: net.ParseIP("192.0.2.50").To4(),
		40001: net.ParseIP("192.0.2.50").To4(),
	}

	rtp := rtpxFrameSpec{srcIP: "198.51.100.10", dstIP: "203.0.113.9", srcPort: 40000, dstPort: 41000, payload: "rtp-data"}
	rewritten := rtp
	rewritten.srcIP = "192.0.2.50"

	ethRTP := rtp
	ethRTP.linkType = layers.LinkTypeEthernet
	ethRewritten := rewritten
	ethRewritten.linkType = layers.LinkTypeEthernet

	vlanRTP := ethRTP
	vlanRTP.vlan = true
	vlanRewritten := ethRewritten
	vlanRewritten.vlan = true

	rawRTP := rtp
	rawRTP.linkType = layers.LinkTypeRaw
	rawRewritten := rewritten
	rawRewritten.linkType = layers.LinkTypeRaw

	nullRTP := rtp
	nullRTP.linkType = layers.LinkTypeNull

	v6RTP := rtpxFrameSpec{linkType: layers.LinkTypeEthernet, ipv6: true, srcIP: "2001:db8::1", dstIP: "2001:db8::2", srcPort: 40000, dstPort: 41000, payload: "rtp-data"}

	tcpRTP := ethRTP
	tcpRTP.tcp = true

	dstOnly := ethRTP
	dstOnly.srcPort = 41000
	dstOnly.dstPort = 40000
	dstOnlyRewritten := dstOnly
	dstOnlyRewritten.dstIP = "192.0.2.50"

	noMatch := ethRTP
	noMatch.srcPort = 5060
	noMatch.dstPort = 5070

	tests := []struct {
		name string

		linkType layers.LinkType
		input    []byte
		expect   []byte
	}{
		{
			name:     "ethernet source address is rewritten and the UDP checksum is cleared",
			linkType: layers.LinkTypeEthernet,
			input:    rtpxFrame(t, ethRTP),
			expect:   rtpxZeroUDPChecksum(rtpxFrame(t, ethRewritten), 14),
		},
		{
			name:     "ethernet destination address is rewritten when only the destination port matches",
			linkType: layers.LinkTypeEthernet,
			input:    rtpxFrame(t, dstOnly),
			expect:   rtpxZeroUDPChecksum(rtpxFrame(t, dstOnlyRewritten), 14),
		},
		{
			name:     "802.1Q tagged frame is rewritten at the shifted offset",
			linkType: layers.LinkTypeEthernet,
			input:    rtpxFrame(t, vlanRTP),
			expect:   rtpxZeroUDPChecksum(rtpxFrame(t, vlanRewritten), 18),
		},
		{
			name:     "raw IP frame is rewritten without a link header",
			linkType: layers.LinkTypeRaw,
			input:    rtpxFrame(t, rawRTP),
			expect:   rtpxZeroUDPChecksum(rtpxFrame(t, rawRewritten), 0),
		},
		{
			name:     "link type without a known header length is left unchanged",
			linkType: layers.LinkTypeNull,
			input:    rtpxFrame(t, nullRTP),
			expect:   rtpxFrame(t, nullRTP),
		},
		{
			name:     "IPv6 packet is left unchanged",
			linkType: layers.LinkTypeEthernet,
			input:    rtpxFrame(t, v6RTP),
			expect:   rtpxFrame(t, v6RTP),
		},
		{
			name:     "TCP packet is left unchanged",
			linkType: layers.LinkTypeEthernet,
			input:    rtpxFrame(t, tcpRTP),
			expect:   rtpxFrame(t, tcpRTP),
		},
		{
			name:     "ports without an endpoint are left unchanged",
			linkType: layers.LinkTypeEthernet,
			input:    rtpxFrame(t, noMatch),
			expect:   rtpxFrame(t, noMatch),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := append([]byte(nil), tt.input...)

			res := rewritePacketIPs(tt.input, tt.linkType, portToIP)
			if !bytes.Equal(res, tt.expect) {
				t.Errorf("Wrong match.\nexpect: %x\ngot:    %x", tt.expect, res)
			}
			if !bytes.Equal(tt.input, before) {
				t.Errorf("Wrong match. the input packet must not be modified")
			}
		})
	}
}

func Test_rewriteRTPPacketIPs_invalid_pcap(t *testing.T) {
	endpoints := []sdpMediaEndpoint{{IP: net.ParseIP("192.0.2.50").To4(), Port: 40000}}

	res, err := rewriteRTPPacketIPs([]byte("not a pcap file at all"), endpoints)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %x", res)
	}
}

func Test_extractSDPBody(t *testing.T) {
	tests := []struct {
		name string

		raw string

		expect string
	}{
		{name: "CRLF separator", raw: "INVITE sip:a SIP/2.0\r\nVia: x\r\n\r\nv=0\r\n", expect: "v=0\r\n"},
		{name: "LF separator", raw: "INVITE sip:a SIP/2.0\nVia: x\n\nv=0\n", expect: "v=0\n"},
		{name: "no separator", raw: "INVITE sip:a SIP/2.0\r\nVia: x\r\n", expect: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := extractSDPBody(tt.raw); res != tt.expect {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expect, res)
			}
		})
	}
}

func Test_mergeMultiplePcaps_sources(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	frame := func(payload string) []byte {
		return rtpxFrame(t, rtpxFrameSpec{linkType: layers.LinkTypeEthernet, srcIP: "198.51.100.10", dstIP: "203.0.113.9", srcPort: 40000, dstPort: 41000, payload: payload})
	}

	first := rtpxPcap(t, layers.LinkTypeEthernet,
		rtpxPacket{ts: base.Add(1 * time.Second), data: frame("p1")},
		rtpxPacket{ts: base.Add(4 * time.Second), data: frame("p4")},
	)
	second := rtpxPcap(t, layers.LinkTypeEthernet,
		rtpxPacket{ts: base.Add(2 * time.Second), data: frame("p2")},
		rtpxPacket{ts: base.Add(3 * time.Second), data: frame("p3")},
		rtpxPacket{ts: base.Add(5 * time.Second), data: frame("p5")},
	)

	tests := []struct {
		name string

		sources []io.Reader

		expectPayloads []string
		expectEmpty    bool
	}{
		{
			name:           "packets of several sources are interleaved by timestamp",
			sources:        []io.Reader{bytes.NewReader(first), bytes.NewReader(second)},
			expectPayloads: []string{"p1", "p2", "p3", "p4", "p5"},
		},
		{
			name:           "the order of the sources does not decide the order of the packets",
			sources:        []io.Reader{bytes.NewReader(second), bytes.NewReader(first)},
			expectPayloads: []string{"p1", "p2", "p3", "p4", "p5"},
		},
		{
			name:           "an unreadable source is skipped",
			sources:        []io.Reader{bytes.NewReader([]byte("garbage")), bytes.NewReader(first)},
			expectPayloads: []string{"p1", "p4"},
		},
		{
			name:        "no source gives an empty result",
			sources:     nil,
			expectEmpty: true,
		},
		{
			name:        "only unreadable sources give an empty result",
			sources:     []io.Reader{bytes.NewReader([]byte("garbage")), bytes.NewReader(nil)},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := mergeMultiplePcaps(tt.sources)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil {
				t.Fatalf("Wrong match. expect: not nil, got: nil")
			}
			if tt.expectEmpty {
				if len(res) != 0 {
					t.Errorf("Wrong match. expect: empty, got: %d bytes", len(res))
				}
				return
			}

			got := []string{}
			for _, d := range rtpxReadPcap(t, res) {
				for _, marker := range []string{"p1", "p2", "p3", "p4", "p5"} {
					if bytes.Contains(d, []byte(marker)) {
						got = append(got, marker)
					}
				}
			}
			if !reflect.DeepEqual(got, tt.expectPayloads) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectPayloads, got)
			}
		})
	}
}

func Test_filterInternalPackets_invalid_pcap(t *testing.T) {
	res, err := filterInternalPackets([]byte("not a pcap file at all"))
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %x", res)
	}
}

func Test_fetchRTPPcaps_cap(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockGCS := NewMockGCSReader(mc)

	objects := []string{}
	for i := 0; i < 21; i++ {
		objects = append(objects, fmt.Sprintf("rtp-recordings/call-1-%02d.pcap", i))
	}
	mockGCS.EXPECT().ListObjects(gomock.Any(), "rtp-recordings/call-1-").Return(objects, nil)
	for i, name := range objects[:20] {
		content := fmt.Sprintf("content-%02d", i)
		mockGCS.EXPECT().DownloadObject(gomock.Any(), name, gomock.Any()).DoAndReturn(
			func(_ context.Context, _ string, dest io.Writer) error {
				_, err := dest.Write([]byte(content))
				return err
			},
		).Times(1)
	}

	h := &sipHandler{gcsReader: mockGCS, gcsBucket: "test-bucket"}

	files, cleanup, err := h.fetchRTPPcaps(context.Background(), "call-1")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if len(files) != 20 {
		t.Fatalf("Wrong match. expect: 20 files, got: %d", len(files))
	}
	names := []string{}
	for i, f := range files {
		body, errRead := io.ReadAll(f)
		if errRead != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", errRead)
		}
		if expect := fmt.Sprintf("content-%02d", i); string(body) != expect {
			t.Errorf("Wrong match. file %d expect: %s, got: %s", i, expect, body)
		}
		names = append(names, f.Name())
	}

	cleanup()
	for _, n := range names {
		if _, errStat := os.Stat(n); !os.IsNotExist(errStat) {
			t.Errorf("Wrong match. expect the temp file %s to be removed, got: %v", n, errStat)
		}
	}
}

func Test_fetchRTPPcaps_temp_files(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	closedDir := t.TempDir()

	tests := []struct {
		name string

		tmpDir   string
		download func(_ context.Context, _ string, dest io.Writer) error

		expectLeft int
	}{
		{
			name:   "temp file creation failure skips every object",
			tmpDir: missing,
			download: func(_ context.Context, _ string, _ io.Writer) error {
				return nil
			},
		},
		{
			name:   "seek failure skips the object and removes its temp file",
			tmpDir: closedDir,
			download: func(_ context.Context, _ string, dest io.Writer) error {
				f, ok := dest.(*os.File)
				if !ok {
					return fmt.Errorf("unexpected destination type %T", dest)
				}
				return f.Close()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TMPDIR", tt.tmpDir)

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockGCS := NewMockGCSReader(mc)
			mockGCS.EXPECT().ListObjects(gomock.Any(), "rtp-recordings/call-2-").Return(
				[]string{"rtp-recordings/call-2-a.pcap", "rtp-recordings/call-2-b.pcap"}, nil,
			)
			if tt.tmpDir != missing {
				mockGCS.EXPECT().DownloadObject(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.download).Times(2)
			}

			h := &sipHandler{gcsReader: mockGCS, gcsBucket: "test-bucket"}

			files, cleanup, err := h.fetchRTPPcaps(context.Background(), "call-2")
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if len(files) != 0 {
				t.Errorf("Wrong match. expect: no file, got: %d", len(files))
			}
			if cleanup == nil {
				t.Fatalf("Wrong match. expect a cleanup function, got: nil")
			}
			cleanup()

			if tt.tmpDir != missing {
				entries, errDir := os.ReadDir(tt.tmpDir)
				if errDir != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", errDir)
				}
				if len(entries) != 0 {
					t.Errorf("Wrong match. expect: no temp file left, got: %d", len(entries))
				}
			}
		})
	}
}

func Test_GetPcap_rtp_with_sdp(t *testing.T) {
	fromTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	sipFrame := rtpxFrame(t, rtpxFrameSpec{linkType: layers.LinkTypeEthernet, srcIP: "198.51.100.20", dstIP: "203.0.113.20", srcPort: 5060, dstPort: 5060, payload: "sip-message"})
	rtpSpec := rtpxFrameSpec{linkType: layers.LinkTypeEthernet, srcIP: "198.51.100.10", dstIP: "203.0.113.9", srcPort: 40000, dstPort: 41000, payload: "rtp-data"}
	rtpFrame := rtpxFrame(t, rtpSpec)

	rewrittenSpec := rtpSpec
	rewrittenSpec.srcIP = "192.0.2.50"
	rewrittenRTP := rtpxZeroUDPChecksum(rtpxFrame(t, rewrittenSpec), 14)

	sipPcap := rtpxPcap(t, layers.LinkTypeEthernet, rtpxPacket{ts: fromTime.Add(1 * time.Second), data: sipFrame})
	rtpPcap := rtpxPcap(t, layers.LinkTypeEthernet, rtpxPacket{ts: fromTime.Add(2 * time.Second), data: rtpFrame})

	sdp := &sipmessage.SIPMessage{Raw: "INVITE sip:a@example.com SIP/2.0\r\nVia: x\r\n\r\nv=0\r\nc=IN IP4 192.0.2.50\r\nm=audio 40000 RTP/AVP 0\r\n"}

	tests := []struct {
		name string

		messages   []*sipmessage.SIPMessage
		messageErr error

		expectRTP []byte
	}{
		{
			name:      "SDP endpoint rewrites the address of the RTP packet",
			messages:  []*sipmessage.SIPMessage{sdp},
			expectRTP: rewrittenRTP,
		},
		{
			name:       "SIP message lookup failure returns the merged data untouched",
			messageErr: fmt.Errorf("simulated lookup failure"),
			expectRTP:  rtpFrame,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockHomer := homerhandler.NewMockHomerHandler(mc)
			mockGCS := NewMockGCSReader(mc)

			mockHomer.EXPECT().GetPcap(gomock.Any(), "call-3", fromTime, toTime).Return(sipPcap, nil)
			mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), "call-3", fromTime, toTime).Return([]byte("not a pcap"), nil)
			mockHomer.EXPECT().GetSIPMessages(gomock.Any(), "call-3", fromTime, toTime).Return(tt.messages, tt.messageErr)
			mockGCS.EXPECT().ListObjects(gomock.Any(), "rtp-recordings/call-3-").Return([]string{"rtp-recordings/call-3-a.pcap"}, nil)
			mockGCS.EXPECT().DownloadObject(gomock.Any(), "rtp-recordings/call-3-a.pcap", gomock.Any()).DoAndReturn(
				func(_ context.Context, _ string, dest io.Writer) error {
					_, err := dest.Write(rtpPcap)
					return err
				},
			)

			h := NewSIPHandler(mockHomer, mockGCS, "test-bucket")

			res, err := h.GetPcap(context.Background(), "call-3", fromTime, toTime)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			packets := rtpxReadPcap(t, res)
			if len(packets) != 2 {
				t.Fatalf("Wrong match. expect: 2 packets, got: %d", len(packets))
			}
			if !bytes.Equal(packets[0], sipFrame) {
				t.Errorf("Wrong match. the SIP packet must be unchanged.\nexpect: %x\ngot:    %x", sipFrame, packets[0])
			}
			if !bytes.Equal(packets[1], tt.expectRTP) {
				t.Errorf("Wrong match.\nexpect: %x\ngot:    %x", tt.expectRTP, packets[1])
			}
		})
	}
}
