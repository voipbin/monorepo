# Komodo stack: voip-kamailio-proxy

Deployment definition for the provider health-check RPC worker. VOIP-1486.

## What this stack runs

`voip-kamailio-proxy` answers exactly one RabbitMQ RPC, `POST /v1/providers/health`,
by sending a raw UDP SIP OPTIONS packet to the carrier named in the request and
reporting whether anything answered. Any SIP response code counts as healthy; a
timeout or network error counts as unhealthy.

`bin-route-manager` calls it once every 30 seconds from a background ticker
(`bin-route-manager/pkg/healthcheckhandler`) and writes the verdict to each
provider's `health_status`. Without this stack running, that ticker's RPCs go
unanswered and provider health stops being updated.

## Why it is a standalone stack, not a sidecar

The service was previously deployed by `voip-kamailio-ansible` onto the GCP
Kamailio VMs, alongside the Kamailio SIP daemon, on the premise that the probe
had to originate from Kamailio's own SIP source address. That premise was
measured and does not hold: Kamailio's SIP address and the host's default egress
address already differ, and the carrier answers OPTIONS from every source
address tested. The probe needs egress to the carrier and nothing else, so the
service runs on its own on the `production` bridge network.

One consequence is worth stating plainly: the probe leaves from the container's
egress address, not from the SIP proxy's public address. A carrier that
whitelists by source IP could answer the probe while rejecting real SIP traffic,
or the reverse.

## Queue contract

| Queue | Role |
|---|---|
| `voip.kamailio.request` | The shared RPC queue. Every replica is a competing consumer, so exactly one replica handles each request. This is the only queue anything publishes to. |
| `voip.kamailio.<mac>.request` | Declared and consumed per replica, derived from the MAC of `eth0`. Nothing in the monorepo publishes to it. It is `autoDelete` with `x-expires` of 30 minutes, so redeploys leave no orphans. |

Because the per-instance queue is never addressed, the usual bar on running
multiple replicas of a service that declares one does not apply here. See
`docs/workflows/manager-replica-scaling.md`.

## Environment variables

Set explicitly:

| Variable | Value | Why |
|---|---|---|
| `RABBITMQ_ADDRESS` | Komodo variable | Broker address. |
| `SIP_TIMEOUT` | `5s` | The per-probe budget. Visible next to route-manager's 10 second RPC timeout, so the relationship between the two is legible. |
| `PROMETHEUS_ENDPOINT` | `/metrics` | Matches the bin-*-manager peers. |
| `PROMETHEUS_LISTEN_ADDRESS` | `:2112` | The default, written out on purpose: the Ansible precedent used `:9102`, which the planned scrape job will not reach. See Monitoring below. |

Deliberately omitted, because their defaults are already the values needed and
writing them out is the only way to get them wrong:

| Variable | Default | Source |
|---|---|---|
| `RABBITMQ_QUEUE_LISTEN` | `voip.kamailio.request` | `cmd/kamailio-proxy/init.go` |
| `INTERFACE_NAME` | `eth0` | `cmd/kamailio-proxy/init.go` |

All three defaults are pinned by `Test_defaults` in
`cmd/kamailio-proxy/main_test.go`.

## Deployment

CircleCI workflow `voip-kamailio-proxy`: test, then build (behind the
`build-approval` gate), then `voip-kamailio-proxy-deploy`, which renders the
image tag into this compose file and pushes it to Komodo as the stack
`voip-kamailio-proxy`. Approving `build-approval` deploys to production.

## Monitoring

The stack exposes metrics on `:2112`, but **nothing scrapes it yet.** Prometheus
discovers the `bin-*-manager` fleet from a static list of service names, and
`kamailio-proxy` is not on it. Until a scrape job is added in `monorepo-etc`
(`infra-prometheus`, the second half of VOIP-1486):

- there is no alert if this stack disappears or drops to one replica, and
- the post-deploy check in `docs/workflows/manager-replica-scaling.md`,
  `count(up{job="voipbin-managers", service="kamailio-proxy"}) == 2`, returns
  empty rather than 2. That is the expected result right now, not a failed
  deploy.

Until then, verify by hand as below.

## Verification

```bash
bats .circleci/tests/voip-kamailio-proxy-komodo.bats
python3 -c "import yaml; yaml.safe_load(open('voip-kamailio-proxy/komodo/docker-compose.yml'))"
cd voip-kamailio-proxy && go test ./...
```

After deploying, on the host:

```bash
docker ps --filter name=kamailio-proxy          # expect 2 containers
docker exec infra-rabbitmq rabbitmqctl list_queues name consumers \
  | grep voip.kamailio.request                  # expect 2
```

`rabbitmqctl` is not installed on the host; the broker runs in the
`infra-rabbitmq` container.

Provider health updating is the real signal: `health_checked_at` should advance
every 30 seconds, and route-manager's log should stop reporting health-check
failures.

## Known limits

- `restart: always` is required, not stylistic. A failed `getKamailioID` returns
  from `main()` with exit code 0, so `on-failure` would never restart it.
- A failure to declare or consume the queue leaves the container `Up` with no
  consumer registered. Container liveness alone does not prove the service is
  working; the consumer count and `health_checked_at` do.
