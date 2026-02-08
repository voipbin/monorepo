package sipmessage

import (
	"regexp"
	"strconv"
	"strings"
)

// SIPMessage represents a single SIP message from Homer.
type SIPMessage struct {
	Timestamp string `json:"timestamp"`
	Method    string `json:"method"`
	SrcIP     string `json:"src_ip"`
	SrcPort   int    `json:"src_port"`
	DstIP     string `json:"dst_ip"`
	DstPort   int    `json:"dst_port"`
	Raw       string `json:"raw"`
}

// RTCPStats holds parsed RTCP quality metrics from the X-RTP-Stat SIP header.
type RTCPStats struct {
	MOS           float64 `json:"mos"`
	Jitter        int     `json:"jitter"`
	PacketLossPct float64 `json:"packet_loss_pct"`
	RTT           int     `json:"rtt"`
	RTPBytes      int     `json:"rtp_bytes"`
	RTPPackets    int     `json:"rtp_packets"`
	RTPErrors     int     `json:"rtp_errors"`
	RTCPBytes     int     `json:"rtcp_bytes"`
	RTCPPackets   int     `json:"rtcp_packets"`
	RTCPErrors    int     `json:"rtcp_errors"`
}

// SIPAnalysisResponse is the response for the SIP analysis endpoint, containing
// both SIP messages and RTCP quality stats.
type SIPAnalysisResponse struct {
	SIPMessages []*SIPMessage `json:"sip_messages"`
	RTCPStats   *RTCPStats    `json:"rtcp_stats"`
}

// regexRTPStat matches the RTP portion of the RTPStat value.
// Example: "RTP: 258452 bytes, 1509 packets, 0 errors"
var regexRTPStat = regexp.MustCompile(`RTP:\s*(\d+)\s*bytes,\s*(\d+)\s*packets,\s*(\d+)\s*errors`)

// regexRTCPStat matches the RTCP portion of the RTPStat value.
// Example: "RTCP:  1248 bytes, 18 packets, 12 errors"
var regexRTCPStat = regexp.MustCompile(`RTCP:\s*(\d+)\s*bytes,\s*(\d+)\s*packets,\s*(\d+)\s*errors`)

// ParseXRTPStat parses the X-RTP-Stat header value into RTCPStats.
// Format: MOS=3.8;Jitter=7;PacketLossPct=0;RTT=260682;RTPStat=RTP: 258452 bytes, 1509 packets, 0 errors; RTCP:  1248 bytes, 18 packets, 12 errors
// Returns nil if the value is empty or unparseable.
func ParseXRTPStat(value string) *RTCPStats {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	stats := &RTCPStats{}
	parsed := false

	// Split into key=value portion and RTPStat portion.
	// RTPStat contains semicolons internally, so we find it first and handle separately.
	rtpStatIdx := strings.Index(value, "RTPStat=")
	var kvPart, rtpStatPart string
	if rtpStatIdx >= 0 {
		kvPart = strings.TrimRight(value[:rtpStatIdx], ";")
		rtpStatPart = value[rtpStatIdx+len("RTPStat="):]
	} else {
		kvPart = value
	}

	// Parse key=value pairs
	for _, pair := range strings.Split(kvPart, ";") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		eqIdx := strings.Index(pair, "=")
		if eqIdx < 0 {
			continue
		}

		key := strings.TrimSpace(pair[:eqIdx])
		val := strings.TrimSpace(pair[eqIdx+1:])

		switch key {
		case "MOS":
			stats.MOS, _ = strconv.ParseFloat(val, 64)
			parsed = true
		case "Jitter":
			stats.Jitter, _ = strconv.Atoi(val)
			parsed = true
		case "PacketLossPct":
			stats.PacketLossPct, _ = strconv.ParseFloat(val, 64)
			parsed = true
		case "RTT":
			stats.RTT, _ = strconv.Atoi(val)
			parsed = true
		}
	}

	// Parse RTPStat portion
	if rtpStatPart != "" {
		if m := regexRTPStat.FindStringSubmatch(rtpStatPart); len(m) == 4 {
			stats.RTPBytes, _ = strconv.Atoi(m[1])
			stats.RTPPackets, _ = strconv.Atoi(m[2])
			stats.RTPErrors, _ = strconv.Atoi(m[3])
			parsed = true
		}
		if m := regexRTCPStat.FindStringSubmatch(rtpStatPart); len(m) == 4 {
			stats.RTCPBytes, _ = strconv.Atoi(m[1])
			stats.RTCPPackets, _ = strconv.Atoi(m[2])
			stats.RTCPErrors, _ = strconv.Atoi(m[3])
			parsed = true
		}
	}

	if !parsed {
		return nil
	}

	return stats
}

// ExtractXRTPStat extracts the X-RTP-Stat header value from a raw SIP message.
// The header name comparison is case-insensitive per RFC 3261.
// Returns empty string if the header is not found.
func ExtractXRTPStat(rawSIPMessage string) string {
	const headerPrefix = "x-rtp-stat:"
	for _, line := range strings.Split(rawSIPMessage, "\n") {
		line = strings.TrimRight(line, "\r")
		if len(line) > len(headerPrefix) && strings.EqualFold(line[:len(headerPrefix)], headerPrefix) {
			return strings.TrimSpace(line[len(headerPrefix):])
		}
	}
	return ""
}

// ExtractRTCPStatsFromMessages scans SIP messages for BYE messages containing
// X-RTP-Stat headers and returns parsed RTCP stats. If multiple BYE messages
// contain X-RTP-Stat, the last one is used.
func ExtractRTCPStatsFromMessages(messages []*SIPMessage) *RTCPStats {
	var result *RTCPStats
	for _, msg := range messages {
		if !strings.EqualFold(msg.Method, "BYE") {
			continue
		}
		xrtpStat := ExtractXRTPStat(msg.Raw)
		if xrtpStat == "" {
			continue
		}
		if parsed := ParseXRTPStat(xrtpStat); parsed != nil {
			result = parsed
		}
	}
	return result
}

// PcapResponse is the response for PCAP download.
type PcapResponse struct {
	CallID      string `json:"call_id"`
	DownloadURI string `json:"download_uri"`
	ExpiresAt   string `json:"expires_at"`
}
