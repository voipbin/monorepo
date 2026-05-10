# voip-kamailio-proxy — Subsystems

This service is class **A+sub**: the Go proxy runs co-located with a Kamailio SIP proxy daemon in the same container or pod.

## Native Daemon Overview

### Kamailio SIP Proxy

[Kamailio](https://www.kamailio.org/) is an open-source SIP server that handles SIP routing, registration, authentication, and load balancing. In VoIPbin, each Kamailio instance acts as the SIP signaling front-door for inbound and outbound calls.

The Go proxy does **not** start, stop, or communicate directly with Kamailio at runtime. The two processes are co-deployed and serve complementary roles:

| Component | Responsibility |
|-----------|---------------|
| Kamailio daemon | SIP signaling (REGISTER, INVITE, BYE, ACK), routing, authentication |
| Go proxy | RabbitMQ bridge, SIP OPTIONS health checks on behalf of platform services |

The Go proxy performs SIP OPTIONS health checks itself (raw UDP to port 5060) without routing through Kamailio. Kamailio is not involved in health check processing.

### Kamailio control interfaces

Kamailio exposes management interfaces that may be used by operators, though the Go proxy does not use them at runtime:

| Interface | Protocol | Default | Purpose |
|-----------|----------|---------|---------|
| XMLRPC | HTTP | `localhost:5060` (or `/RPC`) | Remote procedure calls (reload config, list dialogs, etc.) |
| Binrpc (`kamctl`) | TCP | `localhost:9998` | Native binary RPC used by `kamctl` CLI |
| SIP port | UDP/TCP | `0.0.0.0:5060` | SIP signaling |

## Configuration

### Kamailio main configuration (`kamailio.cfg`)

Kamailio is configured via its own `kamailio.cfg` file. The Go proxy does not modify or read this file. Key parameters relevant to the combined deployment:

```
# Listen on all interfaces for SIP
listen=udp:0.0.0.0:5060

# Enable XMLRPC for management (optional)
loadmodule "xmlrpc.so"

# Route to RTPEngine for media proxying (if integrated)
loadmodule "rtpengine.so"
```

The Ansible role `voip-kamailio-ansible` manages Kamailio configuration in production.

### PSTN whitelist

Kamailio supports a PSTN IP whitelist via the `PSTN_WHITELIST_IPS` environment variable (injected by the Ansible deployment). Whitelisted IPs bypass SIP authentication for inbound PSTN calls.

### TLS / SRTP

If TLS SIP transport is required, configure `tls.cfg` and load `tls.so` in `kamailio.cfg`. The Go proxy has no TLS configuration — it uses plain UDP for OPTIONS health checks.

## Deployment Notes

### Container model

The Dockerfile (`voip-kamailio-proxy/Dockerfile`) builds only the Go binary. Kamailio is expected to be installed in the runtime environment separately:

1. **Sidecar pattern**: Kamailio runs as a separate container in the same Kubernetes pod, sharing the network namespace. The Go proxy and Kamailio both see `localhost:5060`.
2. **Single container**: Kamailio and the proxy binary are installed in the same image.

In either case, the pod's network namespace is shared so that SIP OPTIONS probes sent by the Go proxy reach Kamailio on `localhost:5060`.

### Instance identity

On startup, the proxy reads the MAC address from `--interface_name` (default `eth0`) to derive a unique instance ID. This ID is used to name the volatile RabbitMQ queue `voip.kamailio.<mac>.request`, enabling targeted routing from upstream services to a specific pod.

Unlike `voip-asterisk-proxy`, `voip-kamailio-proxy` does **not** register its address in Redis or patch Kubernetes annotations. Identity is purely through RabbitMQ queue naming.

### Startup order

Kamailio and the Go proxy start independently. If Kamailio is not ready when a health check arrives, the OPTIONS packet will be answered (or not) by whatever is listening on port 5060. The Go proxy handles this gracefully — any response is healthy, no response is unhealthy.

### Network policy considerations

Outbound UDP on port 5060 must be permitted from the proxy pod to the SIP provider IPs. If network policies are restrictive, health checks will always time out even if Kamailio is running correctly.

```yaml
# Example: allow egress to SIP providers on UDP 5060
egress:
  - ports:
      - port: 5060
        protocol: UDP
```
