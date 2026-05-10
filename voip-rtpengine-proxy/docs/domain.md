# voip-rtpengine-proxy — Domain

## Domain Entities

### Command

An incoming RabbitMQ RPC request routed to the proxy. The `type` field determines dispatch.

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Command type: `"ng"`, `"exec"`, or `"kill"` |
| `id` | string (UUID) | Process identifier for `exec`/`kill` commands |
| `command` | string | Executable name (`exec`) or RTPEngine NG command name (`ng`) |
| `parameters` | []string | Additional CLI arguments for `exec` commands |
| `data` | map | Full raw payload forwarded to RTPEngine for `ng` commands |

### NG Protocol Command

An RTPEngine [NG protocol](https://github.com/sipwise/rtpengine#ng-control-protocol) message sent over UDP/bencode. The proxy generates a random cookie, sends `<cookie> <bencode>`, and waits for a matching `<cookie> <bencode>` reply.

| Concept | Detail |
|---------|--------|
| Transport | UDP |
| Encoding | bencode |
| Wire format | `<16-hex-char-cookie> <bencode-payload>` |
| Timeout | Configurable via `RTPENGINE_NG_TIMEOUT` (default `5s`) |

### Tracked Process

A tcpdump process managed by the process manager, identified by a UUID.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Caller-assigned unique identifier |
| `pcap_path` | string | `/tmp/<id>.pcap` — local capture file |
| `safety_timeout` | duration | Maximum capture duration (default 20 min); auto-kills on expiry |

### Proxy Identity

Each proxy instance registers itself in Redis using its IPv4 address:

| Redis key | Value | TTL |
|-----------|-------|-----|
| `rtpengine.<ip>.address-internal` | IPv4 address | 24 hours, refreshed every 5 min |

Upstream services use this key to construct volatile queue names for targeted routing.

## Key Business Rules

1. **Command type is mandatory.** A missing or empty `type` field returns HTTP 400 immediately without touching RTPEngine or the process manager.

2. **`exec` IDs must be UUIDs.** The process manager validates the `id` field against a UUID regex before starting any process. Non-UUID IDs are rejected with 400.

3. **Only whitelisted commands can be executed.** The process manager validates the `command` field against an allowlist before calling `exec.Command`. Commands not on the list are rejected.

4. **Parameters are sanitized.** The `-w` flag is explicitly blocked in incoming parameters — the proxy always constructs the pcap write path (`/tmp/<id>.pcap`) itself to prevent path traversal.

5. **Maximum 20 concurrent captures.** The process manager enforces a cap of 20 concurrent tcpdump processes. New `exec` requests beyond this limit return an error.

6. **Safety timeout auto-kills captures.** Each `exec` starts a 20-minute timer. If `kill` is not called before the timer fires, the process is automatically killed and the pcap uploaded.

7. **GCS upload is best-effort with one retry.** On kill, the proxy uploads the pcap file to GCS. If upload fails, it retries once. If both fail, the local file is kept for manual recovery. If no GCS bucket is configured, the file is deleted after the process exits.

8. **Pcap watcher handles RTPEngine's own recordings.** If `RTPENGINE_RECORDING_DIR` is set, the watcher monitors that directory and uploads completed pcap files to GCS automatically (without explicit `kill` commands).

9. **Orphan pcap files are cleaned on startup.** Any `/tmp/<uuid>.pcap` files left from a previous run are removed during `CleanOrphans()` at startup.
