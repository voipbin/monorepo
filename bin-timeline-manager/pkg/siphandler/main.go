package siphandler

//go:generate mockgen -package siphandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"bytes"
	"context"
	"net"
	"sort"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/sirupsen/logrus"

	"monorepo/bin-timeline-manager/models/sipmessage"
	"monorepo/bin-timeline-manager/pkg/homerhandler"
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
	GetSIPInfo(ctx context.Context, sipCallID string, fromTime, toTime time.Time) (*sipmessage.SIPInfoResponse, error)
	GetPcap(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]byte, error)
}

type sipHandler struct {
	homerHandler homerhandler.HomerHandler
}

// NewSIPHandler creates a new SIPHandler.
func NewSIPHandler(homerHandler homerhandler.HomerHandler) SIPHandler {
	return &sipHandler{
		homerHandler: homerHandler,
	}
}

// GetSIPInfo retrieves SIP messages and RTCP stats for a given SIP call ID and time range.
func (h *sipHandler) GetSIPInfo(ctx context.Context, sipCallID string, fromTime, toTime time.Time) (*sipmessage.SIPInfoResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "GetSIPInfo",
		"sip_callid": sipCallID,
	})
	log.WithFields(logrus.Fields{
		"from_time": fromTime,
		"to_time":   toTime,
	}).Info("SIPHandler called - fetching SIP info")

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

	res := &sipmessage.SIPInfoResponse{
		SIPMessages: filtered,
		RTCPStats:   rtcpStats,
	}

	return res, nil
}

// GetPcap retrieves PCAP data for a given SIP call ID and time range.
// It fetches both SIP (hepid 1) and RTCP (hepid 5) packets from Homer,
// merges them, and filters out internal-to-internal packets.
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

	// Fetch RTCP PCAP (hepid 5) - non-fatal if this fails
	rtcpPcapData, err := h.homerHandler.GetRTCPPcap(ctx, sipCallID, fromTime, toTime)
	if err != nil {
		log.Warnf("Could not get RTCP PCAP data from Homer, continuing with SIP only: %v", err)
	} else {
		log.WithField("rtcp_pcap_size", len(rtcpPcapData)).Debug("Retrieved RTCP PCAP data.")
	}

	// Merge SIP and RTCP PCAPs if both are available
	var mergedData []byte
	if len(rtcpPcapData) > 0 {
		mergedData, err = mergePcaps(sipPcapData, rtcpPcapData)
		if err != nil {
			log.Warnf("Could not merge SIP and RTCP PCAPs, using SIP only: %v", err)
			mergedData = sipPcapData
		} else {
			log.WithField("merged_pcap_size", len(mergedData)).Debug("Merged SIP and RTCP PCAPs.")
		}
	} else {
		mergedData = sipPcapData
	}

	// Filter internal packets from PCAP
	filteredData, err := filterInternalPackets(mergedData)
	if err != nil {
		log.Warnf("Could not filter PCAP data, returning unfiltered: %v", err)
		return mergedData, nil
	}

	log.WithFields(logrus.Fields{
		"original_size": len(mergedData),
		"filtered_size": len(filteredData),
	}).Debug("Filtered internal packets from PCAP.")

	return filteredData, nil
}

// packetEntry holds raw packet data and its capture info for sorting during merge.
type packetEntry struct {
	ci   gopacket.CaptureInfo
	data []byte
}

// mergePcaps merges two PCAP byte slices into a single PCAP, sorted by timestamp.
func mergePcaps(pcap1, pcap2 []byte) ([]byte, error) {
	var packets []packetEntry

	// Read packets from first PCAP
	reader1, err := pcapgo.NewReader(bytes.NewReader(pcap1))
	if err != nil {
		return nil, err
	}
	snaplen := reader1.Snaplen()
	linkType := reader1.LinkType()

	for {
		data, ci, readErr := reader1.ReadPacketData()
		if readErr != nil {
			break
		}
		packets = append(packets, packetEntry{ci: ci, data: data})
	}

	// Read packets from second PCAP
	reader2, err := pcapgo.NewReader(bytes.NewReader(pcap2))
	if err != nil {
		return nil, err
	}
	for {
		data, ci, readErr := reader2.ReadPacketData()
		if readErr != nil {
			break
		}
		packets = append(packets, packetEntry{ci: ci, data: data})
	}

	// Sort by timestamp
	sort.Slice(packets, func(i, j int) bool {
		return packets[i].ci.Timestamp.Before(packets[j].ci.Timestamp)
	})

	// Write merged PCAP
	var buf bytes.Buffer
	writer := pcapgo.NewWriter(&buf)
	if err := writer.WriteFileHeader(snaplen, linkType); err != nil {
		return nil, err
	}
	for _, p := range packets {
		if err := writer.WritePacket(p.ci, p.data); err != nil {
			return nil, err
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
