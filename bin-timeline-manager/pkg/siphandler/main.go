package siphandler

//go:generate mockgen -package siphandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"bytes"
	"context"
	"net"
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
	GetSIPMessages(ctx context.Context, sipCallID string, fromTime, toTime time.Time) (*sipmessage.SIPMessagesResponse, error)
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

// GetSIPMessages retrieves SIP messages for a given SIP call ID and time range,
// and builds a SIPMessagesResponse.
func (h *sipHandler) GetSIPMessages(ctx context.Context, sipCallID string, fromTime, toTime time.Time) (*sipmessage.SIPMessagesResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "GetSIPMessages",
		"sip_callid": sipCallID,
	})
	log.WithFields(logrus.Fields{
		"from_time": fromTime,
		"to_time":   toTime,
	}).Info("SIPHandler called - fetching SIP messages")

	messages, err := h.homerHandler.GetSIPMessages(ctx, sipCallID, fromTime, toTime)
	if err != nil {
		log.Errorf("Could not get SIP messages from Homer. err: %v", err)
		return nil, err
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

	res := &sipmessage.SIPMessagesResponse{
		NextPageToken: "",
		Result:        filtered,
	}

	return res, nil
}

// GetPcap retrieves PCAP data for a given SIP call ID and time range.
func (h *sipHandler) GetPcap(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]byte, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "GetPcap",
		"sip_callid": sipCallID,
	})
	log.WithFields(logrus.Fields{
		"from_time": fromTime,
		"to_time":   toTime,
	}).Info("SIPHandler called - fetching PCAP data")

	pcapData, err := h.homerHandler.GetPcap(ctx, sipCallID, fromTime, toTime)
	if err != nil {
		log.Errorf("Could not get PCAP data from Homer. err: %v", err)
		return nil, err
	}

	log.WithField("pcap_size", len(pcapData)).Debug("Successfully retrieved PCAP data.")

	// Filter internal packets from PCAP
	filteredData, err := filterInternalPackets(pcapData)
	if err != nil {
		log.Warnf("Could not filter PCAP data, returning unfiltered: %v", err)
		return pcapData, nil
	}

	log.WithFields(logrus.Fields{
		"original_size": len(pcapData),
		"filtered_size": len(filteredData),
	}).Debug("Filtered internal packets from PCAP.")

	return filteredData, nil
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
