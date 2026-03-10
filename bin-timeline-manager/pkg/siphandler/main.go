package siphandler

//go:generate mockgen -package siphandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/sirupsen/logrus"

	"monorepo/bin-timeline-manager/models/sipmessage"
	"monorepo/bin-timeline-manager/pkg/homerhandler"
)

const (
	gcsRTPPrefix    = "rtp-recordings/"
	gcsTimeout      = 30 * time.Second
	maxRTPPcapFiles = 20
)

// RFC 1918 private address ranges
var privateIPBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

// isPrivateIP checks if an IP address is in RFC 1918 private ranges.
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// SIPHandler interface for SIP message and PCAP operations.
type SIPHandler interface {
	GetSIPAnalysis(ctx context.Context, sipCallID string, fromTime, toTime time.Time) (*sipmessage.SIPAnalysisResponse, error)
	GetPcap(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]byte, error)
}

type sipHandler struct {
	homerHandler homerhandler.HomerHandler
	gcsReader    GCSReader
	gcsBucket    string
}

// NewSIPHandler creates a new SIPHandler.
func NewSIPHandler(homerHandler homerhandler.HomerHandler, gcsReader GCSReader, gcsBucket string) SIPHandler {
	return &sipHandler{
		homerHandler: homerHandler,
		gcsReader:    gcsReader,
		gcsBucket:    gcsBucket,
	}
}

// GetSIPAnalysis retrieves SIP messages and RTCP stats for a given SIP call ID and time range.
func (h *sipHandler) GetSIPAnalysis(ctx context.Context, sipCallID string, fromTime, toTime time.Time) (*sipmessage.SIPAnalysisResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "GetSIPAnalysis",
		"sip_callid": sipCallID,
	})
	log.WithFields(logrus.Fields{
		"from_time": fromTime,
		"to_time":   toTime,
	}).Info("SIPHandler called - fetching SIP analysis")

	messages, err := h.homerHandler.GetSIPMessages(ctx, sipCallID, fromTime, toTime)
	if err != nil {
		log.Errorf("Could not get SIP messages from Homer. err: %v", err)
		return nil, err
	}

	// Extract RTCP stats from BYE messages before filtering (the BYE with X-RTP-Stat
	// typically comes from internal IPs and would be filtered out).
	rtcpStats := sipmessage.ExtractRTCPStatsFromMessages(messages)
	if rtcpStats != nil {
		log.WithField("rtcp_stats", rtcpStats).Debug("Extracted RTCP stats from X-RTP-Stat header.")
	} else {
		log.Debug("No RTCP stats found in SIP messages.")
	}

	// Filter out messages where both src and dst are internal IPs
	filtered := make([]*sipmessage.SIPMessage, 0, len(messages))
	for _, msg := range messages {
		if isPrivateIP(msg.SrcIP) && isPrivateIP(msg.DstIP) {
			continue // Skip internal-to-internal messages
		}
		filtered = append(filtered, msg)
	}

	log.WithFields(logrus.Fields{
		"total_count":    len(messages),
		"filtered_count": len(filtered),
	}).Debug("Filtered internal messages.")

	res := &sipmessage.SIPAnalysisResponse{
		SIPMessages: filtered,
		RTCPStats:   rtcpStats,
	}

	return res, nil
}

// fetchRTPPcaps downloads RTP pcap files from GCS for the given SIP Call-ID.
// Returns open file handles and a cleanup function. Caller must call cleanup after use.
func (h *sipHandler) fetchRTPPcaps(ctx context.Context, sipCallID string) ([]*os.File, func(), error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "fetchRTPPcaps",
		"sip_callid": sipCallID,
	})

	if h.gcsReader == nil || h.gcsBucket == "" {
		return nil, func() {}, nil
	}

	gcsCtx, cancel := context.WithTimeout(ctx, gcsTimeout)
	defer cancel()

	prefix := gcsRTPPrefix + sipCallID + "-"
	objects, err := h.gcsReader.ListObjects(gcsCtx, prefix)
	if err != nil {
		log.Warnf("Could not list RTP pcap objects from GCS, continuing without RTP: %v", err)
		return nil, func() {}, nil
	}

	if len(objects) == 0 {
		log.Debug("No RTP pcap files found in GCS.")
		return nil, func() {}, nil
	}

	if len(objects) > maxRTPPcapFiles {
		log.WithFields(logrus.Fields{
			"found": len(objects),
			"limit": maxRTPPcapFiles,
		}).Warnf("Too many RTP pcap files, truncating to %d", maxRTPPcapFiles)
		objects = objects[:maxRTPPcapFiles]
	}

	log.WithField("count", len(objects)).Debugf("Found RTP pcap files in GCS. sip_callid: %s", sipCallID)

	type downloadResult struct {
		file *os.File
		err  error
	}

	results := make([]downloadResult, len(objects))
	var wg sync.WaitGroup

	for i, objName := range objects {
		wg.Add(1)
		go func(idx int, name string) {
			defer wg.Done()

			tmpFile, err := os.CreateTemp("", "rtp-pcap-*.pcap")
			if err != nil {
				results[idx] = downloadResult{err: fmt.Errorf("could not create temp file: %w", err)}
				return
			}

			if err := h.gcsReader.DownloadObject(gcsCtx, name, tmpFile); err != nil {
				_ = tmpFile.Close()
				_ = os.Remove(tmpFile.Name())
				results[idx] = downloadResult{err: fmt.Errorf("could not download %s: %w", name, err)}
				return
			}

			if _, err := tmpFile.Seek(0, 0); err != nil {
				_ = tmpFile.Close()
				_ = os.Remove(tmpFile.Name())
				results[idx] = downloadResult{err: fmt.Errorf("could not seek temp file: %w", err)}
				return
			}

			log.WithField("object", filepath.Base(name)).Debugf("Downloaded RTP pcap file. object: %s", name)
			results[idx] = downloadResult{file: tmpFile}
		}(i, objName)
	}

	wg.Wait()

	var files []*os.File
	var cleanupPaths []string

	for i, r := range results {
		if r.err != nil {
			log.WithField("object", objects[i]).Warnf("Skipping RTP pcap download: %v", r.err)
			continue
		}
		files = append(files, r.file)
		cleanupPaths = append(cleanupPaths, r.file.Name())
	}

	cleanup := func() {
		for _, f := range files {
			_ = f.Close()
		}
		for _, p := range cleanupPaths {
			_ = os.Remove(p)
		}
	}

	return files, cleanup, nil
}

// GetPcap retrieves PCAP data for a given SIP call ID and time range.
// It fetches SIP (hepid 1) and RTCP (hepid 5) packets from Homer, filters out
// internal-to-internal packets, then merges with unfiltered RTP pcaps from GCS.
func (h *sipHandler) GetPcap(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]byte, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "GetPcap",
		"sip_callid": sipCallID,
	})
	log.WithFields(logrus.Fields{
		"from_time": fromTime,
		"to_time":   toTime,
	}).Info("SIPHandler called - fetching PCAP data")

	// Fetch SIP PCAP (hepid 1)
	sipPcapData, err := h.homerHandler.GetPcap(ctx, sipCallID, fromTime, toTime)
	if err != nil {
		log.Errorf("Could not get SIP PCAP data from Homer. err: %v", err)
		return nil, err
	}
	log.WithField("sip_pcap_size", len(sipPcapData)).Debug("Retrieved SIP PCAP data.")

	if len(sipPcapData) == 0 {
		log.Debug("No SIP PCAP data available.")
		return []byte{}, nil
	}

	// Fetch RTCP PCAP (hepid 5) - non-fatal if this fails
	rtcpPcapData, err := h.homerHandler.GetRTCPPcap(ctx, sipCallID, fromTime, toTime)
	if err != nil {
		log.Warnf("Could not get RTCP PCAP data from Homer, continuing with SIP only: %v", err)
	} else {
		log.WithField("rtcp_pcap_size", len(rtcpPcapData)).Debug("Retrieved RTCP PCAP data.")
	}

	// Merge SIP and RTCP from Homer
	var homerData []byte
	if len(rtcpPcapData) > 0 {
		homerData, err = mergePcaps(sipPcapData, rtcpPcapData)
		if err != nil {
			log.Warnf("Could not merge SIP and RTCP PCAPs, using SIP only: %v", err)
			homerData = sipPcapData
		} else {
			log.WithField("homer_merged_size", len(homerData)).Debug("Merged SIP and RTCP PCAPs.")
		}
	} else {
		homerData = sipPcapData
	}

	// Filter internal-to-internal packets from Homer data (SIP/RTCP only)
	filteredHomerData, err := filterInternalPackets(homerData)
	if err != nil {
		log.Warnf("Could not filter Homer PCAP data, using unfiltered: %v", err)
		filteredHomerData = homerData
	} else {
		log.WithFields(logrus.Fields{
			"original_size": len(homerData),
			"filtered_size": len(filteredHomerData),
		}).Debug("Filtered internal packets from Homer PCAP.")
	}

	// Fetch RTP pcap files from GCS (best-effort, NOT filtered)
	rtpFiles, cleanup, rtpErr := h.fetchRTPPcaps(ctx, sipCallID)
	if rtpErr != nil {
		log.Warnf("Could not fetch RTP pcaps from GCS: %v", rtpErr)
	}
	defer cleanup()

	// If no RTP files, return filtered Homer data
	if len(rtpFiles) == 0 {
		return filteredHomerData, nil
	}

	// Merge filtered Homer data with unfiltered RTP pcaps
	var sources []io.Reader
	sources = append(sources, bytes.NewReader(filteredHomerData))
	for _, f := range rtpFiles {
		sources = append(sources, f)
	}

	mergedData, err := mergeMultiplePcaps(sources)
	if err != nil {
		log.Warnf("Could not merge with RTP PCAPs, returning filtered Homer data: %v", err)
		return filteredHomerData, nil
	}

	log.WithFields(logrus.Fields{
		"homer_filtered_size": len(filteredHomerData),
		"merged_size":         len(mergedData),
		"rtp_file_count":      len(rtpFiles),
	}).Debug("Merged filtered Homer data with unfiltered RTP PCAPs.")

	// Rewrite IP addresses in RTP packets to match SDP endpoints so Wireshark
	// can correlate RTP streams with the SIP/SDP negotiation.
	messages, msgErr := h.homerHandler.GetSIPMessages(ctx, sipCallID, fromTime, toTime)
	if msgErr != nil {
		log.Warnf("Could not get SIP messages for SDP parsing, RTP IPs will not be rewritten: %v", msgErr)
		return mergedData, nil
	}

	endpoints := parseSDPEndpoints(messages)
	if len(endpoints) > 0 {
		rewritten, rwErr := rewriteRTPPacketIPs(mergedData, endpoints)
		if rwErr != nil {
			log.Warnf("Could not rewrite RTP packet IPs: %v", rwErr)
		} else {
			mergedData = rewritten
			log.WithField("endpoint_count", len(endpoints)).Debug("Rewrote RTP packet IPs to match SDP.")
		}
	}

	return mergedData, nil
}

// mergePcaps merges two PCAP byte slices into a single PCAP, sorted by timestamp.
func mergePcaps(pcap1, pcap2 []byte) ([]byte, error) {
	return mergeMultiplePcaps([]io.Reader{
		bytes.NewReader(pcap1),
		bytes.NewReader(pcap2),
	})
}

// readerEntry tracks a pcap reader and its currently buffered packet.
type readerEntry struct {
	reader   *pcapgo.Reader
	ci       gopacket.CaptureInfo
	data     []byte
	done     bool
	linkType layers.LinkType
	snaplen  uint32
}

// mergeMultiplePcaps merges N pcap sources into a single pcap sorted by timestamp.
func mergeMultiplePcaps(sources []io.Reader) ([]byte, error) {
	if len(sources) == 0 {
		return []byte{}, nil
	}

	log := logrus.WithField("func", "mergeMultiplePcaps")

	var entries []*readerEntry
	for i, src := range sources {
		reader, err := pcapgo.NewReader(src)
		if err != nil {
			log.WithField("source_index", i).Warnf("Could not open pcap source, skipping: %v", err)
			continue
		}

		entry := &readerEntry{
			reader:   reader,
			linkType: reader.LinkType(),
			snaplen:  reader.Snaplen(),
		}

		data, ci, err := reader.ReadPacketData()
		if err != nil {
			entry.done = true
		} else {
			entry.ci = ci
			entry.data = data
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return []byte{}, nil
	}

	primaryLinkType := entries[0].linkType

	var included []*readerEntry
	var maxSnaplen uint32
	for _, e := range entries {
		if e.linkType != primaryLinkType {
			log.WithFields(logrus.Fields{
				"expected": primaryLinkType,
				"actual":   e.linkType,
			}).Warn("Link type mismatch, excluding source from merge.")
			continue
		}
		included = append(included, e)
		if e.snaplen > maxSnaplen {
			maxSnaplen = e.snaplen
		}
	}

	if len(included) == 0 {
		return []byte{}, nil
	}

	var buf bytes.Buffer
	writer := pcapgo.NewWriter(&buf)
	if err := writer.WriteFileHeader(maxSnaplen, primaryLinkType); err != nil {
		return nil, fmt.Errorf("could not write pcap header: %w", err)
	}

	for {
		minIdx := -1
		for i, e := range included {
			if e.done {
				continue
			}
			if minIdx == -1 || e.ci.Timestamp.Before(included[minIdx].ci.Timestamp) {
				minIdx = i
			}
		}

		if minIdx == -1 {
			break
		}

		if err := writer.WritePacket(included[minIdx].ci, included[minIdx].data); err != nil {
			return nil, fmt.Errorf("could not write packet: %w", err)
		}

		data, ci, err := included[minIdx].reader.ReadPacketData()
		if err != nil {
			included[minIdx].done = true
		} else {
			included[minIdx].ci = ci
			included[minIdx].data = data
		}
	}

	return buf.Bytes(), nil
}

// filterInternalPackets removes packets where both src and dst IPs are internal.
func filterInternalPackets(pcapData []byte) ([]byte, error) {
	reader, err := pcapgo.NewReader(bytes.NewReader(pcapData))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	writer := pcapgo.NewWriter(&buf)
	if err := writer.WriteFileHeader(reader.Snaplen(), reader.LinkType()); err != nil {
		return nil, err
	}

	for {
		data, ci, err := reader.ReadPacketData()
		if err != nil {
			break // End of file or error
		}

		// Parse the packet
		packet := gopacket.NewPacket(data, reader.LinkType(), gopacket.Default)

		// Extract IP layer
		var srcIP, dstIP net.IP
		if ipv4Layer := packet.Layer(layers.LayerTypeIPv4); ipv4Layer != nil {
			ipv4, _ := ipv4Layer.(*layers.IPv4)
			srcIP = ipv4.SrcIP
			dstIP = ipv4.DstIP
		} else if ipv6Layer := packet.Layer(layers.LayerTypeIPv6); ipv6Layer != nil {
			ipv6, _ := ipv6Layer.(*layers.IPv6)
			srcIP = ipv6.SrcIP
			dstIP = ipv6.DstIP
		}

		// Skip if both IPs are internal
		if srcIP != nil && dstIP != nil {
			if isPrivateIP(srcIP.String()) && isPrivateIP(dstIP.String()) {
				continue
			}
		}

		// Write packet to output
		if err := writer.WritePacket(ci, data); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// sdpMediaEndpoint represents a media endpoint extracted from SDP negotiation.
type sdpMediaEndpoint struct {
	IP   net.IP
	Port int
}

// parseSDPEndpoints extracts media endpoints (IP + port) from SDP bodies in SIP messages.
// It parses INVITE and 200 OK messages to find the negotiated media addresses.
func parseSDPEndpoints(messages []*sipmessage.SIPMessage) []sdpMediaEndpoint {
	var endpoints []sdpMediaEndpoint
	for _, msg := range messages {
		if msg.Raw == "" {
			continue
		}

		sdpBody := extractSDPBody(msg.Raw)
		if sdpBody == "" {
			continue
		}

		var sessionIP string
		var mediaPort int
		var mediaIP string

		for _, line := range strings.Split(sdpBody, "\n") {
			line = strings.TrimRight(line, "\r")
			line = strings.TrimSpace(line)

			if strings.HasPrefix(line, "c=IN IP4 ") {
				ip := strings.TrimSpace(line[len("c=IN IP4 "):])
				if mediaPort == 0 {
					sessionIP = ip // session-level (before first m= line)
				} else {
					mediaIP = ip // media-level (after m= line, overrides session)
				}
			}

			if strings.HasPrefix(line, "m=audio ") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					port, err := strconv.Atoi(fields[1])
					if err == nil {
						mediaPort = port
						mediaIP = "" // reset media-level IP for this section
					}
				}
			}
		}

		if mediaPort > 0 {
			ip := mediaIP
			if ip == "" {
				ip = sessionIP
			}
			if ip != "" {
				parsedIP := net.ParseIP(ip)
				if parsedIP != nil {
					endpoints = append(endpoints, sdpMediaEndpoint{IP: parsedIP.To4(), Port: mediaPort})
				}
			}
		}
	}

	return endpoints
}

// extractSDPBody returns the SDP body from a raw SIP message (everything after the first blank line).
func extractSDPBody(rawSIP string) string {
	for _, sep := range []string{"\r\n\r\n", "\n\n"} {
		if idx := strings.Index(rawSIP, sep); idx >= 0 {
			return rawSIP[idx+len(sep):]
		}
	}
	return ""
}

// rewriteRTPPacketIPs rewrites IP addresses in pcap packets to match SDP endpoints.
// For each UDP packet whose src or dst port matches an SDP media port, the corresponding
// IP address is rewritten to the SDP-advertised IP. This allows Wireshark to correlate
// RTP streams with SIP/SDP negotiation when the capture was taken behind NAT.
func rewriteRTPPacketIPs(pcapData []byte, endpoints []sdpMediaEndpoint) ([]byte, error) {
	if len(endpoints) == 0 {
		return pcapData, nil
	}

	// Build port → IP mapping (including RTCP port = media port + 1)
	portToIP := make(map[int]net.IP)
	for _, ep := range endpoints {
		portToIP[ep.Port] = ep.IP
		portToIP[ep.Port+1] = ep.IP // RTCP
	}

	reader, err := pcapgo.NewReader(bytes.NewReader(pcapData))
	if err != nil {
		return nil, err
	}

	linkType := reader.LinkType()

	var buf bytes.Buffer
	writer := pcapgo.NewWriter(&buf)
	if err := writer.WriteFileHeader(reader.Snaplen(), linkType); err != nil {
		return nil, err
	}

	for {
		data, ci, err := reader.ReadPacketData()
		if err != nil {
			break
		}

		rewritten := rewritePacketIPs(data, linkType, portToIP)
		if err := writer.WritePacket(ci, rewritten); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// rewritePacketIPs rewrites IP addresses in a single packet based on port-to-IP mapping.
// It modifies raw packet bytes directly and recalculates the IPv4 header checksum.
func rewritePacketIPs(data []byte, linkType layers.LinkType, portToIP map[int]net.IP) []byte {
	packet := gopacket.NewPacket(data, linkType, gopacket.NoCopy)

	ipv4Layer := packet.Layer(layers.LayerTypeIPv4)
	if ipv4Layer == nil {
		return data
	}

	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		return data
	}

	udp, _ := udpLayer.(*layers.UDP)
	srcPort := int(udp.SrcPort)
	dstPort := int(udp.DstPort)

	newSrcIP, srcRewrite := portToIP[srcPort]
	newDstIP, dstRewrite := portToIP[dstPort]

	if !srcRewrite && !dstRewrite {
		return data
	}

	// Determine Ethernet header length
	var ipOffset int
	switch linkType {
	case layers.LinkTypeEthernet:
		ipOffset = 14
		if len(data) > 16 {
			etherType := uint16(data[12])<<8 | uint16(data[13])
			if etherType == 0x8100 { // 802.1Q VLAN tag
				ipOffset = 18
			}
		}
	case layers.LinkTypeRaw:
		ipOffset = 0
	default:
		return data
	}

	ihl := int(data[ipOffset]&0x0f) * 4
	if ihl < 20 || len(data) < ipOffset+ihl+8 {
		return data
	}

	// Copy data to avoid modifying the original
	newData := make([]byte, len(data))
	copy(newData, data)

	if srcRewrite {
		ip4 := newSrcIP.To4()
		if ip4 != nil {
			copy(newData[ipOffset+12:ipOffset+16], ip4)
		}
	}
	if dstRewrite {
		ip4 := newDstIP.To4()
		if ip4 != nil {
			copy(newData[ipOffset+16:ipOffset+20], ip4)
		}
	}

	// Recalculate IPv4 header checksum
	newData[ipOffset+10] = 0
	newData[ipOffset+11] = 0
	var sum uint32
	for i := 0; i < ihl; i += 2 {
		sum += uint32(newData[ipOffset+i])<<8 | uint32(newData[ipOffset+i+1])
	}
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	checksum := ^uint16(sum)
	newData[ipOffset+10] = byte(checksum >> 8)
	newData[ipOffset+11] = byte(checksum)

	// Zero out UDP checksum (optional for IPv4, avoids recalculating pseudo-header)
	udpOffset := ipOffset + ihl
	newData[udpOffset+6] = 0
	newData[udpOffset+7] = 0

	return newData
}
