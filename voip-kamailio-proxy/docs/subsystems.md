# voip-kamailio-proxy — Subsystems

This service is registered as class **A+sub**, but the `sub` is a naming heuristic: it is a standalone Go service, deployed on its own and not co-located with any SIP daemon. See `CLAUDE.md`.

## Native Daemon Overview

There is no native daemon in this service's deployment. The section below explains
why the name suggests otherwise.

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

## Configuration

All configuration is environment variables, read in `cmd/kamailio-proxy/init.go`.
The values used in production, and which defaults are deliberately left unset,
are documented in [../komodo/README.md](../komodo/README.md); the full table with
failure modes is in [operations.md](operations.md).

## Deployment Notes

### Container model

The Dockerfile (`voip-kamailio-proxy/Dockerfile`) builds only the Go binary. The service runs as its own standalone container (a Komodo stack, 2 replicas, on the `production` Docker network); no Kamailio daemon is installed in the runtime image or co-located in the same container.

### Instance identity

On startup, the proxy reads the MAC address from `--interface_name` (default `eth0`) to derive a unique instance ID. This ID is used to name the volatile RabbitMQ queue `voip.kamailio.<mac>.request`, but the queue is declared and consumed without ever being addressed by any publisher; the queue that is actually addressed is the shared permanent queue `voip.kamailio.request`.

Unlike `voip-asterisk-proxy`, `voip-kamailio-proxy` does **not** register its address anywhere. Identity is purely through RabbitMQ queue naming.

### Egress

Outbound UDP on port 5060 must reach the SIP provider IPs. The container sits on
the `production` Docker bridge, which SNATs outbound traffic to the host's
default egress address; there is no policy object to configure. If egress is
blocked upstream, every health check times out and every provider is marked
unhealthy, regardless of whether the providers are actually reachable from
elsewhere.
