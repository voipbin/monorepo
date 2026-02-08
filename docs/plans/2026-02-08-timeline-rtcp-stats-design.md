# Timeline Manager RTCP Stats Enrichment

**Status:** Implemented (PR #411, merged 2026-02-08)

## Problem

RTPEngine sends RTCP summaries to Homer via the `X-RTP-Stat` SIP header in BYE messages after calls end. However, the timeline-manager previously filtered out these BYE messages because they travel between internal IPs (e.g., `10.164.0.20` -> `10.96.4.18`). The RTCP quality data (MOS, jitter, packet loss, RTT) was lost before reaching the API consumer.

Additionally, PCAP downloads only included SIP packets (hepid 1) but not RTCP packets (hepid 5).

## Solution

1. Parse the `X-RTP-Stat` header from BYE messages **before** the internal-IP filter runs
2. Replace the `/v1/sip/messages` endpoint with `/v1/sip/analysis` that returns both SIP messages and parsed RTCP stats
3. Fetch RTCP PCAP data (hepid 5) from Homer and merge with SIP PCAP (hepid 1) in downloads

## External API

### Endpoint Change (Breaking)

- **Old:** `GET /timelines/calls/{call_id}/sip-messages`
- **New:** `GET /timelines/calls/{call_id}/sip-analysis`

### Response Format

```json
{
  "sip_messages": [
    {
      "timestamp": "2026-02-05T10:30:00.123456Z",
      "method": "INVITE",
      "src_ip": "203.0.113.1",
      "src_port": 5060,
      "dst_ip": "10.96.4.18",
      "dst_port": 5060,
      "raw": "INVITE sip:user@host SIP/2.0\r\n..."
    }
  ],
  "rtcp_stats": {
    "mos": 3.8,
    "jitter": 7,
    "packet_loss_pct": 0,
    "rtt": 260682,
    "rtp_bytes": 258452,
    "rtp_packets": 1509,
    "rtp_errors": 0,
    "rtcp_bytes": 1248,
    "rtcp_packets": 18,
    "rtcp_errors": 12
  }
}
```

- `sip_messages` is always present (required), may be empty array
- `rtcp_stats` is `null` when no X-RTP-Stat header was found (not omitted — serialized as explicit `null`)

### Field Units

- `rtt` — microseconds (divide by 1000 for milliseconds). Value of 260682 = ~261ms.
- `jitter` — milliseconds
- `mos` — Mean Opinion Score (1.0-5.0)
- `packet_loss_pct` — percentage (0-100)

## X-RTP-Stat Format

```
X-RTP-Stat: MOS=3.8;Jitter=7;PacketLossPct=0;RTT=260682;RTPStat=RTP: 258452 bytes, 1509 packets, 0 errors; RTCP:  1248 bytes, 18 packets, 12 errors
```

Fields:
- `MOS` — Mean Opinion Score (float, 1.0-5.0)
- `Jitter` — Jitter in ms (int)
- `PacketLossPct` — Packet loss percentage (float)
- `RTT` — Round-trip time in microseconds (int)
- `RTPStat` — Aggregated RTP/RTCP byte, packet, and error counts (contains internal semicolons)

## Data Model

`RTCPStats` struct in `models/sipmessage/sipmessage.go`:

```go
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

type SIPAnalysisResponse struct {
    SIPMessages []*SIPMessage `json:"sip_messages"`
    RTCPStats   *RTCPStats    `json:"rtcp_stats"`
}
```

`RTCPStats` is a pointer so it serializes as `null` when no X-RTP-Stat header was found. No `omitempty` tag — the field is always present in JSON.

## Parser

`ParseXRTPStat(value string) *RTCPStats`:

1. Split on `;` but handle `RTPStat=` specially (it contains `;` internally)
2. Parse key-value pairs for `MOS`, `Jitter`, `PacketLossPct`, `RTT`
3. Parse the `RTPStat=` substring with regex to extract RTP/RTCP bytes, packets, errors
4. Track whether any recognized field was parsed — returns `nil` for completely unrecognized input (not a zero-value struct)

`ExtractXRTPStat(rawSIPMessage string) string`:
- Case-insensitive header name matching per RFC 3261
- Returns empty string if header not found

`ExtractRTCPStatsFromMessages(messages []*SIPMessage) *RTCPStats`:
- Scans for BYE messages with X-RTP-Stat headers
- If multiple BYE messages have X-RTP-Stat, the last one wins

## Flow — SIP Analysis

In `siphandler.GetSIPAnalysis()`:

1. Fetch messages from Homer (`GetSIPMessages`, hepid 1)
2. **Extract RTCP stats** — scan all messages for BYE with X-RTP-Stat header, parse stats (before filtering)
3. Filter internal-to-internal messages (both src and dst are RFC 1918 private IPs)
4. Return `SIPAnalysisResponse{SIPMessages: filtered, RTCPStats: stats}`

Key design decision: RTCP stats are extracted **before** internal-IP filtering because the BYE message from RTPEngine travels between internal IPs and would be filtered out.

## Flow — PCAP Download

In `siphandler.GetPcap()`:

1. Fetch SIP PCAP from Homer (hepid 1)
2. If SIP PCAP is empty, return empty bytes immediately
3. Fetch RTCP PCAP from Homer (hepid 5) — **non-fatal** if this fails
4. Merge SIP and RTCP PCAPs sorted by timestamp (with link type compatibility check)
5. Filter internal-to-internal packets from merged PCAP
6. Return filtered PCAP bytes

### Homer RTCP Query

Uses `GetRTCPPcap` with `"5_default"` search key and `"hepid": 5`:

```json
{
  "param": {
    "search": {
      "5_default": {
        "callid": ["<sip-call-id>"],
        "type": "string",
        "hepid": 5
      }
    }
  }
}
```

### PCAP Merge

`mergePcaps(pcap1, pcap2 []byte) ([]byte, error)`:
- Reads packets from both PCAPs
- Validates link types match (returns error if mismatch, caller falls back to SIP-only)
- Sorts all packets by timestamp
- Writes merged PCAP using first PCAP's snaplen and link type

## Nil Pointer Guards

`bin-api-manager/pkg/servicehandler/timeline_sip.go` — both `TimelineSIPAnalysisGet` and `TimelineSIPPcapGet`:
- `fromTime` (`call.TMCreate`): if nil, return error "no SIP data available"
- `toTime` (`call.TMHangup` -> `call.TMUpdate` -> `time.Now()`): falls back to current time if both are nil

## Files Changed

| # | File | Change |
|---|------|--------|
| 1 | `bin-timeline-manager/models/sipmessage/sipmessage.go` | Add `RTCPStats`, `SIPAnalysisResponse`, `ParseXRTPStat()`, `ExtractXRTPStat()`, `ExtractRTCPStatsFromMessages()` |
| 2 | `bin-timeline-manager/models/sipmessage/sipmessage_test.go` | 18 test cases for parser, extractor, and message scanner |
| 3 | `bin-timeline-manager/pkg/siphandler/main.go` | `GetSIPAnalysis` (extract-before-filter), `GetPcap` (merge SIP+RTCP), `mergePcaps()` |
| 4 | `bin-timeline-manager/pkg/siphandler/main_test.go` | 6 test cases for GetSIPAnalysis |
| 5 | `bin-timeline-manager/pkg/homerhandler/main.go` | Add `GetRTCPPcap()` (hepid 5) |
| 6 | `bin-timeline-manager/pkg/listenhandler/main.go` | Route `/v1/sip/messages` -> `/v1/sip/analysis` |
| 7 | `bin-timeline-manager/pkg/listenhandler/v1_sip.go` | Rename handler, fix Content-Type |
| 8 | `bin-timeline-manager/pkg/listenhandler/models/request/sip.go` | Rename request struct |
| 9 | `bin-common-handler/pkg/requesthandler/timeline_sip.go` | `TimelineV1SIPMessagesGet` -> `TimelineV1SIPAnalysisGet` |
| 10 | `bin-common-handler/pkg/requesthandler/main.go` | Update interface |
| 11 | `bin-api-manager/pkg/servicehandler/timeline_sip.go` | Rename handler, add nil pointer guards |
| 12 | `bin-api-manager/pkg/servicehandler/main.go` | Update interface |
| 13 | `bin-api-manager/server/timelines_sip.go` | Rename HTTP handler |
| 14 | `bin-openapi-manager/openapi/paths/timelines/call_sip_analysis.yaml` | New (replaces `call_sip_messages.yaml`) |
| 15 | `bin-openapi-manager/openapi/openapi.yaml` | Path reference update |
