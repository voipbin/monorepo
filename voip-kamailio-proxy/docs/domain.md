# voip-kamailio-proxy — Domain

## Domain Entities

### SIP Provider Health Check

A health check verifies that a named SIP provider is reachable and responding to SIP signaling.

| Field | Type | Description |
|-------|------|-------------|
| `hostname` | string | DNS name or IP address of the SIP provider to check |
| `status` | string | `"healthy"` or `"unhealthy"` |
| `result_code` | string | SIP response code (e.g. `"200"`, `"404"`) or `"timeout"` if no reply |

### Kamailio Instance Identity

Each proxy instance is uniquely identified by the MAC address of its network interface (`--interface_name`, default `eth0`). This identity drives:

- The volatile RabbitMQ queue name: `voip.kamailio.<mac-address>.request`
- Targeted routing from upstream services to a specific Kamailio pod

### SIP OPTIONS Message

A minimal SIP OPTIONS request used as a liveness probe:

- Sent over raw UDP to `hostname:5060`
- Contains randomly generated `Call-ID`, `branch`, and `tag` values
- Response code from the first `SIP/2.0 <code>` status line is returned to the caller

## Key Business Rules

1. **Any SIP response means healthy.** As long as the remote SIP server replies (even with 403, 404, 500, etc.), the provider is considered reachable. Only a timeout or network error marks the provider as `unhealthy`.

2. **Errors are absorbed, never propagated.** The `SIPChecker` function always returns a result struct — it never returns a Go error to the caller. Network failures translate to `unhealthy` with `result_code: "timeout"`.

3. **Health checks are stateless.** Each health check is a fresh UDP connection and SIP dialog. No session state is maintained between checks.

4. **Instance identity is derived at startup.** The MAC address is read once on startup from the named network interface. If the interface does not exist or has no MAC, the service exits immediately.

5. **Permanent queue is shared; volatile queue is instance-scoped.** Requests on the permanent queue can be handled by any running kamailio-proxy instance. Requests on the volatile queue are delivered only to the instance whose MAC matches the queue name.

6. **SIP timeout is configurable.** The `--sip_timeout` flag (default `5s`) controls the UDP read deadline for each OPTIONS check. Values are parsed as Go duration strings (e.g. `3s`, `1500ms`).
