# Selective Codec Outbound — Kamailio Handover

**Date:** 2026-05-03  
**For:** New session implementing the Kamailio side of the selective codec feature  
**Companion PR:** [#879](https://github.com/voipbin/monorepo/pull/879) — Go side (already approved, not yet merged)

---

## What Has Already Been Done

The Go side is fully implemented and reviewed in PR #879 on the `monorepo` repo
(branch `NOJIRA-selective-codec-outbound`). No work is needed there.

### Summary of Go changes

| Service | Change |
|---|---|
| `bin-customer-manager` | `customer.Metadata.OutboundCodecs string` — admin-set via `PUT /v1/customers/{id}/metadata`, JSON key `outbound_codecs`, e.g. `"PCMU,PCMA,G729"` |
| `bin-call-manager` | New metadata key `codecs` (per-call override); at channel setup time, writes `PJSIP_HEADER(add,VBOUT-CODECS)` = codec list as an Asterisk channel variable |

### What Asterisk now sends

On every outgoing INVITE where `outbound_codecs` is configured, Asterisk adds:

```
VBOUT-CODECS: PCMU,PCMA,G729
```

The value is a comma-separated, preference-ordered list of SIP codec names matching
standard SDP `a=rtpmap` names (e.g. `PCMU`, `PCMA`, `G722`, `G729`, `OPUS`).

**If neither customer-level nor per-call codec config is set, the header is absent.**
This means: absent header = pass-through, use current default behavior.

Two-level priority (already resolved by Go before the header is set):
1. Per-call `metadata["codecs"]` overrides the customer default.
2. Customer `outbound_codecs` is used when no per-call override is present.

---

## What Kamailio Must Do

When Kamailio receives an outgoing INVITE from Asterisk (internal → external path)
with a `VBOUT-CODECS` header:

1. **Build RTPEngine codec filter directives** from the header value.
2. **Call `rtpengine_offer()`** with those directives instead of the hardcoded
   `codec-transcode=PCMU codec-transcode=PCMA` defaults.
3. **Strip the `VBOUT-CODECS` header** before forwarding the INVITE to the PSTN
   provider (it is an internal VoIPBIN header, not meant for external parties).

When the header is **absent**, keep existing behavior exactly as-is (no change).

---

## Kamailio Codebase Location

**Repository:** `~/gitvoipbin/monorepo-voip/`  
**Service:** `voip-kamailio-docker/`  
**Main config template:** `voip-kamailio-docker/templates/kamailio.cfg`  
**CLAUDE.md:** `voip-kamailio-docker/CLAUDE.md` (read before starting)

The `kamailio.cfg` file is a **template** — it uses `VARIABLE_NAME` placeholders that
`scripts/generate_defines.py` expands into `#!substdef` directives at container startup.
Do NOT use hardcoded IPs or env values; use the existing `DEFINE_NAME` constants already
in the file.

---

## Exact Code Location

The change lives entirely in `route[request_from_internal_rtpengine]` (~line 305).

The route handles all SIP requests from Asterisk. The relevant branch is the
**new outbound call** block (`!has_totag()`), roughly lines 382–426:

```
else {
    // new outbound call request from internal  ← THIS IS WHERE THE CHANGE GOES
    ...
    if ($nh(P) =~ "(i?)WSS" || $nh(P) =~ "(i?)WS") {
        rtpengine_offer( ... WS/WSS path ... );
    }
    else {
        rtpengine_offer(              ← MODIFY THIS BLOCK
            "replace-origin "
            "replace-session-connection "
            "direction=priv "
            "direction=pub "
            "codec-mask=all "
            "codec-transcode=PCMU "
            "codec-transcode=PCMA "
            "codec-transcode-telephone-event "
            "codec-except=VP8 "
            "codec-offer=telephone-event "
            "ICE=remove "
            "$hdr(VBOUT-SDP_Transport) "
            "rtcp-mux-accept "
        );
    }
```

The **WS/WSS path** (WebRTC) does not need to change — codec selection is always
forced to OPUS for WebRTC regardless of customer preference.

Only the **non-WS path** (PSTN/SIP) is affected.

---

## Implementation Design

### Converting the comma-separated value to RTPEngine directives

`VBOUT-CODECS: PCMU,PCMA,G729` must become:

```
codec-filter=PCMU codec-filter=PCMA codec-filter=G729
```

RTPEngine's `codec-filter=<name>` directive is a **whitelist**: only codecs matching
at least one `codec-filter` entry are kept in the offered SDP. Combined with
`codec-mask=all` (which instructs RTPEngine to accept any codec from Asterisk), this
ensures:
- Asterisk can offer its full codec set internally.
- Only the customer-specified codecs appear in the outgoing INVITE to the provider.
- If the provider's 200 OK contains only a subset of those codecs, RTPEngine handles
  the media bridging.

`codec-transcode-telephone-event` should always be added regardless of the customer
codec list, since telephone-event (DTMF) is needed for all PSTN calls.

### Kamailio script approach

Kamailio's `{s.replace,old,new}` transformation can convert the comma-separated list:

```
"PCMU,PCMA,G729"
  → apply {s.replace,","," codec-filter="}
  → "PCMU codec-filter=PCMA codec-filter=G729"
  → prepend "codec-filter="
  → "codec-filter=PCMU codec-filter=PCMA codec-filter=G729"
```

Suggested implementation (insert before the existing non-WS `rtpengine_offer` call):

```kamailio
if ($hdr(VBOUT-CODECS) != $null && $hdr(VBOUT-CODECS) != "") {
    // Build RTPEngine codec-filter directives from comma-separated customer list.
    // e.g. "PCMU,PCMA,G729" → "codec-filter=PCMU codec-filter=PCMA codec-filter=G729"
    $var(codecs_raw) = $hdr(VBOUT-CODECS);
    $var(codec_filter) = "codec-filter=" + $(var(codecs_raw){s.replace,","," codec-filter="});
    xlog("L_INFO", "[request_from_internal_rtpengine] RTPENGINE: Selective codec, codecs=$hdr(VBOUT-CODECS), filter=$var(codec_filter)\n");
    rtpengine_offer(
        "replace-origin "
        "replace-session-connection "
        "direction=priv "
        "direction=pub "
        "codec-mask=all "
        "$var(codec_filter) "
        "codec-transcode-telephone-event "
        "codec-except=VP8 "
        "codec-offer=telephone-event "
        "ICE=remove "
        "$hdr(VBOUT-SDP_Transport) "
        "rtcp-mux-accept "
    );
}
else {
    // No customer codec preference — use existing default transcode behavior.
    rtpengine_offer(
        "replace-origin "
        "replace-session-connection "
        "direction=priv "
        "direction=pub "
        "codec-mask=all "
        "codec-transcode=PCMU "
        "codec-transcode=PCMA "
        "codec-transcode-telephone-event "
        "codec-except=VP8 "
        "codec-offer=telephone-event "
        "ICE=remove "
        "$hdr(VBOUT-SDP_Transport) "
        "rtcp-mux-accept "
    );
}
```

### Stripping the internal header before forwarding

The `VBOUT-CODECS` header must not reach the PSTN provider. Add a `remove_hf` call
**after** the `rtpengine_offer()` block (still inside the `!has_totag()` branch) and
**before** `forward()` is called in `route[request_from_internal]`:

```kamailio
// Strip internal VoIPBIN headers before forwarding to provider
if (is_present_hf("VBOUT-CODECS")) {
    remove_hf("VBOUT-CODECS");
}
```

The best place is at the end of `route[request_from_internal_rtpengine]` (after all
codec/RTPEngine logic), unconditionally, so it fires whether the header was used or
not. This is safe because the header is absent on calls without codec config anyway.

> **Note on `VBOUT-SDP_Transport`:** The existing code does not currently strip
> `VBOUT-SDP_Transport` before forwarding. That is pre-existing behavior — do not
> change it in this session (separate concern).

---

## Existing Pattern Reference — `VBOUT-SDP_Transport`

This feature follows the exact same pattern as `VBOUT-SDP_Transport`:

| Aspect | `VBOUT-SDP_Transport` | `VBOUT-CODECS` (new) |
|---|---|---|
| Set by | Asterisk via `PJSIP_HEADER(add,...)` | Asterisk via `PJSIP_HEADER(add,...)` |
| Blocked from provider override | Yes (`reservedTechHeaderKeys`) | Yes (`reservedTechHeaderKeys`) |
| Read by Kamailio | `$hdr(VBOUT-SDP_Transport)` | `$hdr(VBOUT-CODECS)` |
| Used in `rtpengine_offer()` | Passed directly as a single directive | Transformed into `codec-filter=` directives |
| Cached in Redis | Yes (`kamailio.$ci:sdp_transport`) | No (stateless per-request) |
| Stripped before forward | Not currently stripped | Must be stripped |

---

## Config Validation

After editing `kamailio.cfg`, validate the syntax before committing:

```bash
cd ~/gitvoipbin/monorepo-voip/voip-kamailio-docker
docker-compose run --rm kamailio kamailio -c -f /config/kamailio.cfg
```

Expected output: `Listening on ...` with no `ERROR` lines.

---

## Commit and Branch

Work in a new worktree branched from `main` of the `monorepo-voip` repo:

```bash
cd ~/gitvoipbin/monorepo-voip
git worktree add .worktrees/NOJIRA-selective-codec-outbound-kamailio -b NOJIRA-selective-codec-outbound-kamailio
```

Commit message format (matches the monorepo-voip convention):

```
NOJIRA-selective-codec-outbound-kamailio

- voip-kamailio-docker: Handle VBOUT-CODECS header in request_from_internal_rtpengine
- voip-kamailio-docker: Strip VBOUT-CODECS before forwarding to provider
```

---

## Testing

### Unit-level (config syntax)
```bash
docker-compose run --rm kamailio kamailio -c -f /config/kamailio.cfg
# Expected: no ERROR output
```

### Functional test (manual)
1. Set `outbound_codecs = "PCMU"` on a test customer via the admin API.
2. Make an outbound call from that customer.
3. Capture SIP traffic (e.g. via Homer or tcpdump on the Kamailio host).
4. Verify the outgoing INVITE to the PSTN provider contains only `PCMU` in the SDP
   `m=audio` line codec list.
5. Verify `VBOUT-CODECS` header is **not** present in the forwarded INVITE.

### Regression test
6. Make an outbound call from a customer with **no** `outbound_codecs` set.
7. Verify SDP contains `PCMU` and `PCMA` (existing default behavior, unchanged).

---

## Edge Cases to Be Aware Of

| Case | Expected behavior |
|---|---|
| `VBOUT-CODECS` absent | Existing `codec-transcode=PCMU/PCMA` path, no change |
| `VBOUT-CODECS` is empty string | Same as absent — fall through to default path |
| Only one codec, e.g. `"PCMU"` | `codec-filter=PCMU` — single entry, works correctly |
| CRLF in value | Already rejected by `setChannelVariableCodecs` in Go (header not set), so cannot reach Kamailio |
| Re-INVITE (has_totag) | Handled by the `has_totag()` branch which does not read `VBOUT-CODECS` — correct, since the codec preference was already established at call setup |
| Failover channel | Same `c.Metadata` is used, so `VBOUT-CODECS` is set again on the new channel — Kamailio handles it identically |
