# voip-rtpengine-proxy — Subsystems

This service is class **A+sub**: the Go proxy runs co-located with an RTPEngine media proxy daemon in the same container or pod.

## Native Daemon Overview

### RTPEngine

[RTPEngine](https://github.com/sipwise/rtpengine) is an open-source RTP media proxy and transcoder. In VoIPbin, each RTPEngine instance handles the RTP/SRTP media streams for calls routed through the co-located Kamailio SIP proxy.

The Go proxy communicates with RTPEngine exclusively via the **NG control protocol** — a UDP/bencode-based RPC interface. The proxy does not start or stop RTPEngine; both processes are co-deployed and expected to run concurrently.

| Responsibility | Component |
|---------------|-----------|
| SIP signaling | Kamailio (separate pod/service) |
| RTP media proxying, SRTP, codec transcoding | RTPEngine daemon |
| NG protocol bridge to RabbitMQ | Go proxy (this service) |
| tcpdump capture management | Go proxy process manager |
| pcap upload to GCS | Go proxy GCS uploader |

### RTPEngine NG Control Protocol

The NG protocol is RTPEngine's primary control interface:

| Property | Value |
|----------|-------|
| Transport | UDP |
| Encoding | bencode (BitTorrent encoding) |
| Default port | `22222` |
| Wire format | `<16-hex-cookie> <bencode-dictionary>` |
| Supported commands | `offer`, `answer`, `delete`, `query`, `ping`, `start recording`, `stop recording`, etc. |

The Go proxy's `ngclient` package wraps the UDP socket with a cookie-based request/response correlation map, allowing concurrent in-flight NG commands.

### RTPEngine recording directory

When `RTPENGINE_RECORDING_DIR` is set, RTPEngine writes pcap files to that directory. The Go proxy's `pcapwatcher` package monitors this directory using file system polling and uploads closed pcap files to GCS automatically.

## Configuration

### RTPEngine daemon (`rtpengine.conf`)

RTPEngine configuration is managed outside this Go service. Key parameters:

```ini
# /etc/rtpengine/rtpengine.conf
[rtpengine]
interface = eth0
listen-ng = 127.0.0.1:22222
recording-dir = /var/lib/rtpengine-recordings
recording-method = pcap
```

The `listen-ng` value must match `RTPENGINE_NG_ADDRESS` configured in the Go proxy.

### Enabling call recording

To enable pcap-based recording via the pcap watcher:

1. Set `recording-dir` in `rtpengine.conf` to a shared directory.
2. Set `RTPENGINE_RECORDING_DIR` on the Go proxy to the same path.
3. Set `GCP_BUCKET_NAME_MEDIA` so completed pcaps are uploaded to GCS.

If `GCP_BUCKET_NAME_MEDIA` is not set, the pcap watcher is disabled at startup even if `RTPENGINE_RECORDING_DIR` is set.

### SRTP / DTLS

SRTP and DTLS-SRTP are configured on the RTPEngine daemon directly. The Go proxy passes NG commands through without inspecting or modifying SRTP parameters. Codec negotiation and transcoding are also handled entirely by RTPEngine.

## Deployment Notes

### Container model

The Dockerfile (`voip-rtpengine-proxy/Dockerfile`) builds only the Go binary. RTPEngine is expected to be present in the runtime environment:

1. **Sidecar pattern**: RTPEngine runs as a separate container in the same Kubernetes pod, sharing the network namespace. Both containers see `127.0.0.1:22222` for the NG port.
2. **Single container**: RTPEngine and the proxy binary are installed together in one image.

In either case, the pod must share a network namespace so that `127.0.0.1:22222` routes to RTPEngine.

### Instance identity and Redis registration

On startup, the proxy:

1. Reads the IPv4 address from `--interface_name` (default `eth0`).
2. Writes `rtpengine.<ip>.address-internal = <ip>` to Redis (db 1) with a 24-hour TTL.
3. Refreshes this key every 5 minutes in a background goroutine.

Upstream services use this Redis key to construct the volatile queue name `rtpengine.<ip>.request` for targeted routing to a specific pod.

### Startup order

RTPEngine must be running before the proxy begins accepting NG commands. If RTPEngine is not up when the first NG command arrives, the proxy will return a 500 timeout error. Unlike the Asterisk proxy, there is no auto-reconnect for the NG UDP client — each `Send` call dials fresh.

The proxy itself does not depend on Kamailio; NG commands can arrive from any upstream service.

### Pcap file handling

- Active captures write to `/tmp/<uuid>.pcap` on the pod's local filesystem.
- On `kill`, the file is uploaded to `gs://<GCP_BUCKET_NAME_MEDIA>/rtp-recordings/<uuid>-<unix-ts>.pcap`.
- If upload fails after one retry, the file remains at `/tmp/<uuid>.pcap` for manual recovery.
- On startup, `CleanOrphans()` removes any leftover `/tmp/<uuid>.pcap` files from a previous run.
- A safety timeout of 20 minutes auto-kills captures that were never explicitly killed.

### Volume mounts for shared recording directory

If using the pcap watcher with RTPEngine's recording directory, the directory must be shared between containers:

```yaml
volumes:
  - name: rtpengine-recordings
    emptyDir: {}
containers:
  - name: rtpengine
    volumeMounts:
      - name: rtpengine-recordings
        mountPath: /var/lib/rtpengine-recordings
  - name: rtpengine-proxy
    volumeMounts:
      - name: rtpengine-recordings
        mountPath: /var/lib/rtpengine-recordings
```
