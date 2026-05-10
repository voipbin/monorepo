# voip-asterisk-proxy — Subsystems

This service is class **A+sub**: the Go proxy manages and communicates with an embedded Asterisk PBX daemon that runs in the same container or pod.

## Native Daemon Overview

### Asterisk PBX

[Asterisk](https://www.asterisk.org/) is an open-source PBX that handles SIP signaling, RTP media, and call control. In VoIPbin, each Asterisk instance is dedicated to a subset of calls (e.g., all calls routed through one Kubernetes pod).

The Go proxy does not start or stop Asterisk — they are co-deployed and expected to run concurrently. The proxy connects to Asterisk through two interfaces:

| Interface | Protocol | Default address | Purpose |
|-----------|----------|----------------|---------|
| ARI (Asterisk REST Interface) | WebSocket (upgrade from HTTP) | `localhost:8088` | Receive real-time call events; send call control commands (originate, answer, hangup, bridge, playback, etc.) |
| AMI (Asterisk Manager Interface) | TCP plain text | `127.0.0.1:5038` | Receive manager events; send manager actions (e.g., `Reload`, `Status`, custom actions) |

The proxy maintains persistent connections to both interfaces and auto-reconnects with a 1-second retry delay if either connection drops.

### ARI Stasis application

Asterisk routes calls through a Stasis application named `voipbin` (configurable via `--ari_application`). Calls entering Stasis are paused until the Go proxy (acting on instructions from upstream managers) issues ARI commands to control them. If the proxy is disconnected, calls in Stasis will be stuck until reconnection.

## Configuration

### Asterisk ARI (`ari.conf` / `http.conf`)

ARI requires Asterisk's built-in HTTP server to be enabled:

```ini
; /etc/asterisk/http.conf
[general]
enabled=yes
bindaddr=0.0.0.0
bindport=8088
```

```ini
; /etc/asterisk/ari.conf
[general]
enabled=yes
pretty=yes

[asterisk]
type=user
read_only=no
password=asterisk
```

The proxy connects as user `asterisk` (configurable via `--ari_account asterisk:asterisk`).

### Asterisk AMI (`manager.conf`)

```ini
; /etc/asterisk/manager.conf
[general]
enabled=yes
port=5038
bindaddr=127.0.0.1

[asterisk]
secret=asterisk
deny=0.0.0.0/0.0.0.0
permit=127.0.0.1/255.255.255.255
read=all
write=all
```

The proxy connects from localhost, so `bindaddr=127.0.0.1` is safe and limits exposure.

### chan_pjsip (`pjsip.conf`)

SIP endpoints and trunks are configured via `chan_pjsip`. The proxy does not manage SIP config directly — SIP registrations and trunk configuration are handled by `bin-registrar-manager` via AMI commands forwarded through the proxy.

## Deployment Notes

### Container model

The Dockerfile (`voip-asterisk-proxy/Dockerfile`) builds only the Go binary. Asterisk is expected to be present in the runtime container separately — either:

1. **Sidecar pattern**: Asterisk runs as a separate container in the same Kubernetes pod, sharing the network namespace. Both containers see `localhost` for ARI/AMI.
2. **Single container**: Asterisk and the proxy binary are installed together in one image. The proxy binary is placed under `/app/bin/` by the Dockerfile.

In either case, the Go proxy and Asterisk must share the same network namespace so that `localhost:8088` and `127.0.0.1:5038` resolve to Asterisk.

### Startup order

Asterisk must be fully started and ARI/AMI must be accepting connections before the proxy's event loops can attach. The proxy handles this via auto-reconnect: it will retry every 1 second until Asterisk becomes available. No explicit init ordering (e.g., `initContainers`) is required, but Asterisk startup time should be accounted for in readiness probes.

### Kubernetes pod identity

On startup, the proxy reads the MAC address from `--interface_name` (default `eth0`) and:

1. Patches the Kubernetes pod annotation `asterisk-id` with the MAC address (requires `patch` verb on pods).
2. Writes the pod's internal IPv4 address to Redis key `asterisk.<mac>.address-internal` with a 24-hour TTL, refreshed every 5 minutes.

Upstream services use this Redis key to construct the volatile RabbitMQ queue name for targeted RPC routing.

To disable Kubernetes annotation patching (non-k8s environments):

```bash
--kubernetes_disabled=true
# or
KUBERNETES_DISABLED=true
```

### Recording file sharing

When recordings are uploaded via `/proxy/recording_file_move`, the proxy reads from `--recording_asterisk_directory` (default `/var/spool/asterisk/recording`). In a sidecar deployment, this directory must be shared between the Asterisk container and the proxy container using a Kubernetes `emptyDir` or `hostPath` volume mount.

```yaml
volumes:
  - name: asterisk-recordings
    emptyDir: {}
containers:
  - name: asterisk
    volumeMounts:
      - name: asterisk-recordings
        mountPath: /var/spool/asterisk/recording
  - name: asterisk-proxy
    volumeMounts:
      - name: asterisk-recordings
        mountPath: /var/spool/asterisk/recording
```
