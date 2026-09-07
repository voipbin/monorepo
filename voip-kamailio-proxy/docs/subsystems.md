# voip-kamailio-proxy — Subsystems

This service is class **A**: a standalone Go service, deployed on its own and not co-located with any SIP daemon.

## Relationship to the Kamailio SIP daemon

Despite the name, this service does not start, stop, configure, or communicate
with the Kamailio SIP daemon. It never did at runtime; it used to be deployed
onto the same hosts, and that co-location is now gone.

The SIP OPTIONS health check goes straight to the carrier hostname carried in
the RPC request, over an ordinary UDP socket. Kamailio is not in the path, so
Kamailio's own state has no bearing on whether a health check succeeds.

The Kamailio daemon, its `kamailio.cfg`, its control interfaces, the PSTN
whitelist and TLS transport are owned by a separate repository
(`monorepo-voip`, `voip-kamailio-docker` and `voip-kamailio-ansible`) and are
documented there. Sections describing them used to live in this file, which
made this service look like it managed the daemon.

## Deployment Notes

### Container model

The Dockerfile (`voip-kamailio-proxy/Dockerfile`) builds only the Go binary. The service runs as its own standalone container (a Komodo stack, 2 replicas, on the `production` Docker network); no Kamailio daemon is installed in the runtime image or co-located in the same container.

### Instance identity

On startup, the proxy reads the MAC address from `--interface_name` (default `eth0`) to derive a unique instance ID. This ID is used to name the volatile RabbitMQ queue `voip.kamailio.<mac>.request`, but the queue is declared and consumed without ever being addressed by any publisher; the queue that is actually addressed is the shared permanent queue `voip.kamailio.request`.

Unlike `voip-asterisk-proxy`, `voip-kamailio-proxy` does **not** register its address in Redis or patch Kubernetes annotations. Identity is purely through RabbitMQ queue naming.

### Network policy considerations

Outbound UDP on port 5060 must be permitted from the proxy container to the SIP provider IPs. If network policies are restrictive, health checks will always time out, and every provider will be marked unhealthy.

```yaml
# Example: allow egress to SIP providers on UDP 5060
egress:
  - ports:
      - port: 5060
        protocol: UDP
```
