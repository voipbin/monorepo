# Deploy voip-kamailio-proxy as a standalone Komodo stack — Design

Date: 2026-09-07 (v6 — design review rounds 1-5 applied, approved)
Jira: VOIP-1486
Branch: `VOIP-1486-Deploy-kamailio-proxy-komodo-stack`
Repos touched: `voipbin/monorepo` (this), `voipbin/monorepo-etc` (`infra-prometheus`)

## Problem (issue analysis)

`bin-route-manager` runs an unconditional health-check ticker every 30 s
(`cmd/route-manager/main.go:152-153`, `internal/config/config.go:47`) that RPCs
`voip.kamailio.request` for every active provider. **Nothing consumes that queue.**
`voip-kamailio-proxy` ran on the GCP VM fleet via Ansible
(`monorepo-voip/voip-kamailio-ansible/…/docker-compose.yml:84-102`, PRs #78/#79/#80) but was
never ported to Komodo when that fleet was cut over to bm-nyc-01 on 2026-08-13.

Live on bm-nyc-01 (2026-09-07): no proxy container, no `voip.kamailio.request` queue, both
route-manager replicas logging `circuit breaker … half-open -> open` and
`context deadline exceeded` every 30 s, and `route_providers.health_checked_at` frozen at
2026-08-12 for both active providers (26 days). Calls are unaffected —
`bin-route-manager/pkg/routehandler/` never reads `health_status` — but the field ships to
customers over REST (`openapi.yaml:6761`) and provider webhooks
(`models/provider/webhook.go:30-31`), so it is publishing a 26-day-old claim.

The full analysis (9 revisions, 8 review rounds, two consecutive approvals) is the source for
every constraint below.

## Decision

**A standalone Komodo stack on the `production` network, deployed from monorepo** — not a
sidecar on the Kamailio stack.

The sidecar model existed for one reason: the SIP OPTIONS probe's source IP should match the
SIP proxy's so carrier ACLs would not drop it. Measured on bm-nyc-01, that premise fails twice:

1. **Host networking would not deliver it anyway.** Kamailio's SIP public IP is
   `199.127.61.42` (the `kamailio-ext` veth); the host's default egress is `104.243.38.39`
   (`enp7s0`). `SendOptionsCheck` dials with a nil local address
   (`pkg/siphandler/health.go:40`), so the kernel picks the route's source — never Kamailio's.
2. **The carrier does not filter on it.** Probing `sip.telnyx.com` bound to `104.243.38.39`,
   `199.127.61.42`, `172.24.0.246`, and from **inside a `production`-network container**
   (`172.24.128.85`) all returned `SIP/2.0 200 Keepalive P20` in ~40 ms. Our Telnyx
   integration authenticates by FQDN connection with credentials, not source IP
   (`bin-route-manager/pkg/telnyxclient/main.go:31-46`).

Dropping host networking removes the constraints that made a sidecar expensive:

| | Sidecar (rejected) | **Standalone stack (chosen)** |
|---|---|---|
| SIP interruption | Certain — the kamailio deploy job rewrites that service's image tag every commit (`monorepo-voip/.circleci/config_work.yml:1057`), so `up -d` recreates it | **None.** The Kamailio stack is untouched |
| Image reference | Manual digest pin. `IMAGE_TAG_PLACEHOLDER` is rejected by the deploy guard (`:1059-1062`, regression-pinned at `voip-kamailio-docker/tests/komodo-config.bats:556-585`) because its sed only matches `voipbin/voip-kamailio:` | `__IMAGE_TAG__` + `render-image-tag.sh`; `$CIRCLE_SHA1` is meaningful because the image is built in this repo |
| Stale image | Last deployed tag `bac8645d` is 7 commits behind HEAD, spanning the distroless migration (`ac066fd2c`) and the toolchain upgrade (`e15975e58`) | Pipeline builds and deploys the current commit |
| Existing tests | `komodo-config.bats`'s position-dependent `sed` ranges (`:84`, `:599`, `:611`, `:618`, `:625`) would need rework | Untouched |
| RabbitMQ | Host netns has no Docker DNS → `127.0.0.1:5672` | `[[BIN_MANAGER__RABBITMQ_ADDRESS]]`, same as 34 peers |
| Metrics | Docker SD cannot compute an address for a host-network container | Standard dns_sd path |

The cost accepted: the probe's source IP is now a container address SNAT'd to
`104.243.38.39`, permanently different from Kamailio's `199.127.61.42`. With the only active
real carrier authenticating by FQDN this is free today; see "Known limitation".

## Change 1 — `monorepo/voip-kamailio-proxy` (this repo)

### `komodo/docker-compose.yml` (new)

Modelled on `bin-route-manager/komodo/docker-compose.yml`, which is the pattern 33 services
already use.

```yaml
services:
  kamailio-proxy:
    image: voipbin/voip-kamailio-proxy:__IMAGE_TAG__
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
    restart: always
    deploy:
      replicas: 2
    environment:
      - RABBITMQ_ADDRESS=[[BIN_MANAGER__RABBITMQ_ADDRESS]]
      - SIP_TIMEOUT=5s
      - PROMETHEUS_ENDPOINT=/metrics
      - PROMETHEUS_LISTEN_ADDRESS=:2112
    networks:
      default: {}
networks:
  default:
    name: production
    external: true
```

Decisions this file encodes, each load-bearing:

- **Service key `kamailio-proxy`.** It is the container's DNS name on the `production`
  network and therefore the scrape target name. Not `-manager`-suffixed, which is why the
  Prometheus choice below must be (b) — see Change 2.
- **`RABBITMQ_QUEUE_LISTEN` deliberately omitted.** The binary defaults to
  `QueueNameKamailioRequest` = `voip.kamailio.request` (`cmd/kamailio-proxy/init.go:20`),
  which is the only queue route-manager publishes to. Setting it explicitly adds a way to get
  it wrong.
- **`INTERFACE_NAME` deliberately omitted.** The default is `eth0` (`init.go:22`), and a
  `production`-network container has exactly `lo` and `eth0`, with a MAC (verified with
  `docker run --network production busybox ip -o link show`). The Ansible value `ens4` was a
  GCP artifact; `enp7s0` would have been needed only under host networking.
- **`restart: always`**, matching all 38 bin-*-manager service blocks. This matters more than
  it looks: a failed `getKamailioID` returns from `main()` with **exit code 0**
  (`main.go:34-38`), so `on-failure` would never restart it.
- **`replicas: 2`**, matching 31 of 33 peers, and run through the repo's own eligibility gate
  (`docs/workflows/manager-replica-scaling.md:14-37`) rather than assumed:
  1. *No per-pod queue* — this service **does** declare one (`main.go:41`,
     `voip.kamailio.<mac>.request`), which item 1 flags as disqualifying. **Exempt, verified**:
     the only publisher path is `bin-common-handler/pkg/requesthandler/send_request.go:343-344`
     → `QueueNameKamailioRequest`, and a repo-wide grep finds nothing publishing to a per-MAC
     kamailio queue. The volatile queue is declared and consumed but never addressed, and it is
     `autoDelete` with `x-expires: 1800000` (`rabbitmqhandler/queue.go:70-78`), so redeploys
     leave no orphans. This exemption goes in the compose comment, not just here.
  2. *No container-name-dependent env/DNS* — none; nothing resolves its own container name.
  3-6. *External naming / ordering / state / singleton work* — the permanent queue is a
     competing consumer (RPC replies go to the caller's reply-to, not a shared queue), the
     service holds no state between checks (`docs/domain.md`: "Health checks are stateless"),
     and the once-per-cycle guarantee lives in route-manager's redsync lock, not here.
- **`PROMETHEUS_LISTEN_ADDRESS=:2112`**, the binary default, stated explicitly because the
  Ansible precedent's `:9102` would never be scraped by a `port: 2112` dns_sd job.
- **The env rule, stated plainly** (it is not "always omit defaults"): the Prometheus pair is
  written out to match all 33 bin-*-manager peers; `SIP_TIMEOUT` is carried forward from the
  Ansible precedent so the probe budget is visible next to the RPC's 10 s timeout;
  `RABBITMQ_QUEUE_LISTEN` and `INTERFACE_NAME` are omitted because writing them is the only
  way to get them wrong, and both defaults are pinned by a Go test (see Tests).
- **No `container_name`** (Compose rejects it with `replicas > 1`) and **no resource limits**
  (all 38 bin-*-manager blocks omit them; `mem_limit`/`cpus` is a monorepo-voip convention).
- **No `command:`**, unlike 37 of the 38 peer service blocks (e.g.
  `bin-route-manager/komodo/docker-compose.yml:38`; the one exception is
  `bin-sentinel-manager`'s third-party `sentinel-docker-socket-proxy`): this image sets
  `ENTRYPOINT ["/app/bin/kamailio-proxy"]` (`Dockerfile:17`), so the binary already runs.
- **No `cap_add: NET_RAW`.** The Ansible definition had it, but `SendOptionsCheck` uses an
  ordinary `net.DialUDP` socket.
- **No `depends_on`.** Cross-project references are impossible, and the RabbitMQ handler
  retries connection forever at 1 s intervals (`rabbitmqhandler/main.go:244`), which absorbs
  any ordering concern.

### `.circleci/config_work.yml`

Add a deploy job mirroring `bin-route-manager-deploy` (`config_work.yml:1708-1721`, workflow entry `:610-613`; `:1038-1051` is the byte-identical `bin-campaign-manager-deploy`):

```yaml
  voip-kamailio-proxy-deploy:
    docker:
      - image: cimg/base:2024.02
    resource_class: small
    steps:
      - checkout
      - run:
          name: Render image tag into Komodo compose fragment
          command: .circleci/scripts/render-image-tag.sh voip-kamailio-proxy/komodo/docker-compose.yml "$CIRCLE_SHA1"
      - run:
          name: Deploy voip-kamailio-proxy via Komodo API
          command: |
            CC_KOMODO_POLL_ATTEMPTS=60 CC_KOMODO_POLL_INTERVAL_S=3 \
              .circleci/scripts/komodo-api-deploy.sh voip-kamailio-proxy voip-kamailio-proxy/komodo/docker-compose.yml
```

and extend the workflow (`:834-846`) so `-deploy` requires `-build`, **with
`<<: *context_production`** on the deploy entry — without it the Komodo credentials are not
injected. The path filter already exists (`.circleci/config.yml:56`), so adding files under
`voip-kamailio-proxy/` triggers the pipeline. Komodo stack name = `voip-kamailio-proxy`;
`komodo-api-deploy.sh` creates the stack if it does not exist. This is monorepo's first
`voip-*` Komodo deploy target.

### Tests — `.circleci/tests/voip-kamailio-proxy-komodo.bats` (new)

The `shell-tests` job globs `.circleci/tests/*.bats` (`config_work.yml:2183`), so a new file is
picked up automatically and no existing suite enumerates services.

**Known trigger gap**: `shell-tests` is armed by `.circleci/tests/.*` (`config.yml:59-62`), not
by `voip-kamailio-proxy/.*` (`:56`), so a later edit to the compose file alone would not re-run
this suite. This would also be the first suite in that directory to assert on files outside
`.circleci/` (the three existing ones are script-unit suites). Accepted rather than solved: the
single highest-value assertion — the `__IMAGE_TAG__` placeholder being present — is
independently enforced at deploy time by `render-image-tag.sh:64-67`, which `exit 1`s when it is
missing.

**No PyYAML.** That job installs only `bats` and `mawk` on `cimg/base:2024.02`
(`config_work.yml:2176-2180`), no existing suite under `.circleci/tests/` imports `yaml`, and
the sibling repo has already taken a CI failure from exactly this assumption
(`monorepo-etc/.circleci/config_work.yml:643-658`: "ubuntu-2204:current has no PyYAML
preinstalled, unlike every local dev machine this branch was reviewed on"). Assertions are
therefore `grep`/`awk` only. Where an assertion needs structure (the workflow entry's context),
pin the literal lines instead of parsing YAML.

Assertions:

1. the compose declares the service key `kamailio-proxy` (`grep -c '^  kamailio-proxy:' == 1`)
   and no second top-level service key (`grep -c '^  [a-z].*:$'` under `services:` == 1, via an
   `awk` range from `^services:` to `^networks:`). YAML validity is not asserted here — the
   deploy job's own `render-image-tag.sh` + Komodo push is what would reject a malformed file,
   and asserting it would need a parser this job has no dependency for.
2. image is `voipbin/voip-kamailio-proxy:__IMAGE_TAG__` (placeholder present, no literal tag).
3. `render-image-tag.sh` on a temp copy substitutes it and leaves no `__IMAGE_TAG__`.
4. `restart: always`; `deploy.replicas == 2`; no `container_name`.
5. env: `RABBITMQ_ADDRESS` is exactly `[[BIN_MANAGER__RABBITMQ_ADDRESS]]`;
   `PROMETHEUS_LISTEN_ADDRESS` is `:2112`; `SIP_TIMEOUT` present; **`RABBITMQ_QUEUE_LISTEN` and
   `INTERFACE_NAME` are absent as env entries**. The absence assertion must anchor on the env
   form — `grep -cE '^\s*-\s*RABBITMQ_QUEUE_LISTEN=' == 0` — **not** a bare name grep, because
   the compose comments deliberately mention both names to explain the omission — write those
   comments, the same way the replica exemption is required to be commented. A bare grep would fail against the file this design
   specifies.
6. no `cap_add`, no `depends_on`, no `mem_limit`/`cpus` — anchored on the key form
   (`^\s*cap_add:`, `^\s*depends_on:`, `^\s*mem_limit:`, `^\s*cpus:`), not bare names, for the
   same reason as assertion 5: the compose comments will mention these keys while explaining
   why they are absent.
7. network is external `production`.
8. `config_work.yml`: `voip-kamailio-proxy-deploy` exists as a job key, its command lines
   reference `render-image-tag.sh` and `komodo-api-deploy.sh` with the stack name
   `voip-kamailio-proxy`, and **its workflow entry carries the context**. Assert the literal the
   design actually writes — `<<: *context_production` — not `context: [production]`: every entry
   in this file uses the merge-key alias (`config_work.yml:3-5` defines the anchor, `:610-613`
   is bin-route-manager's deploy entry), and resolving the alias needs a YAML parser this job
   does not have. Pin the alias line plus the dependency by extracting the workflow entry's
   line range with `awk` and asserting **two lines**: `requires:` followed by
   `^ *- voip-kamailio-proxy-build$`. `requires:` is a **block sequence** everywhere in this
   file — `grep -c 'requires: \['` over `.circleci/` returns 0 — so a flow-sequence literal
   like `requires: [voip-kamailio-proxy-build]` would fail against a correct implementation,
   the same trap as `context: [production]` above.

**A Go test pins the other half of the omission contract.** Asserting only that the compose
file lacks `RABBITMQ_QUEUE_LISTEN`/`INTERFACE_NAME` pins the decision, not the thing it rests
on: if someone edits `init.go:20` or `:22`, the compose stays "correct" while the service
listens on the wrong queue or fails `getKamailioID` with exit 0. Add to
`cmd/kamailio-proxy/main_test.go`:

```go
// The Komodo compose deliberately omits RABBITMQ_QUEUE_LISTEN, INTERFACE_NAME
// and relies on PROMETHEUS_LISTEN_ADDRESS's default matching the dns_sd port.
// See voip-kamailio-proxy/komodo/docker-compose.yml and
// monorepo-etc infra-prometheus's kamailio-proxy job.
defaultRabbitMQQueueListen == "voip.kamailio.request"  // the literal, not the const
defaultInterfaceName        == "eth0"
defaultPrometheusListenAddress == ":2112"
```

### Docs

- `komodo/README.md` (new): what the stack is, the queue contract, why
  `RABBITMQ_QUEUE_LISTEN`/`INTERFACE_NAME` are omitted, the deploy path, the verification
  commands from §Rollout, and the known limitation.
- Correct the co-location claims this change invalidates: `CLAUDE.md:3`
  ("co-located with a Kamailio SIP proxy daemon"), `docs/architecture.md` ("same pod"),
  `docs/subsystems.md` ("Sidecar pattern … the pod's network namespace is shared"),
  `docs/dependencies.md:58` ("Co-located SIP proxy … shares the pod").
- `docs/operations.md` is the on-call doc for a service that is now a Docker/Komodo stack and
  is wrong throughout, not only at line 7: `:7` (claims the service exits immediately when
  RabbitMQ is unreachable; it retries forever), `:13` "reachable from the pod", `:24` "the
  pod's primary interface", `:39` "from within the pod", `:56`
  `kubectl logs -f <pod-name> -c kamailio-proxy` (the actual command is `docker logs`), `:86`
  `<pod-ip>`. Fix all of them in this pass. (`docs/domain.md:38`'s "exits immediately" refers
  to the interface-lookup path and is correct — leave it.)
- `CLAUDE.md`'s "Instance identity: MAC address of `--interface_name` … used to name the
  volatile queue" is now semantically empty (a random container MAC, no Kamailio to identify).
  Reword alongside line 3.
- The same vacated claim appears in three more files and must go with them:
  `docs/domain.md:20` ("routing … to a specific Kamailio pod"), `docs/architecture.md:7`
  ("same pod") and `:56` ("routes requests to one specific Kamailio pod"),
  `docs/subsystems.md:3` ("the same container or pod"), `:63-66` (the whole "Sidecar pattern"
  / shared-namespace passage), `:70` ("targeted routing … to a specific pod") and `:74-76` (the
  "Startup order" passage, which assumes Kamailio is listening on the proxy's own `:5060`). Note
  `docs/subsystems.md:80`'s egress warning stays — it is still true and the issue analysis
  used it — but "proxy pod" becomes "proxy container".
- **`docs/architecture.md:26-27`**, the ASCII diagram, terminates the proxy's arrow at
  "Kamailio SIP Proxy / (co-located daemon)". This is the most misleading surviving instance —
  it already contradicts `:30` ("not forwarded to Kamailio") — and must be redrawn so the
  probe arrow goes to the SIP provider, not to Kamailio.
- `bin-route-manager/docs/operations.md` describes the health check as route-manager probing
  carriers directly — i.e. a topology in which this outage is impossible. The wrong mechanism
  recurs past the first mention and the follow-on text gives operational instructions derived
  from it, so fix the block as a whole: `:81-82` (the mechanism), `:83-84` ("this doubles
  outbound probe traffic to carriers, so … cut over promptly"), and `:88-100` (the P23b
  paragraph's "only one replica probes carriers per tick" / "doubled probing"). The redsync
  lock and its fail-open tradeoff are still real; what changes is that the probe is issued by
  `voip-kamailio-proxy` over RPC, so the doubling is of RPC requests, not of direct SIP
  traffic from route-manager. Also add `health_check_interval` to its Configuration table.

## Change 2 — `monorepo-etc/infra-prometheus`

**A dedicated `kamailio-proxy` scrape job, not membership in `voipbin-managers`.**

Joining `voipbin-managers` is the tempting option because it inherits `ManagerServiceGone`'s
zero-target coverage. It is rejected because `infra-prometheus/tests/config.bats:399-420`
asserts the `ManagerServiceGone` literal set equals the dns_sd `names:` set, extracting with
`r'"service",\s*"([a-z0-9-]+-manager)"'` (`:415`, asserted at `:417`) — and that test runs in
CI (`monorepo-etc/.circleci/config_work.yml:663`). Adding `kamailio-proxy` puts it in
`configured` but not `alerted`, so **infra-prometheus CI goes red**. Making it fit would mean
either renaming the service to end in `-manager` (wrong for a `voip-*` service, and that name
becomes the container and DNS name) or changing a regex 32 services share. The dedicated job
avoids all five sync points at the cost of one guard alert.

```yaml
  - job_name: kamailio-proxy
    metrics_path: /metrics
    dns_sd_configs:
      - names: [kamailio-proxy]
        type: A
        port: 2112
    relabel_configs:
      - source_labels: [__meta_dns_name]
        regex: (.+)
        target_label: service
        action: replace
```

dns_sd, not `static_configs`: with `replicas: 2` a single fixed hostname resolves to whichever
replica DNS round-robin picks per scrape (`prometheus.yml:297-305` records exactly this
reasoning). The `service` relabel mirrors `:374-378` so `InstanceDown`'s
`{{ $labels.service }}` renders.

`alert-rules.yml` gains a zero-target guard shaped like `AsteriskTargetsMissing`
(`:223-224`), since `InstanceDown` cannot fire for a target discovery never produced:

```yaml
      # zero-target guard: InstanceDown cannot fire for a target discovery
      # never produced. SYNC POINTS for the literal 2 below: deploy.replicas
      # in monorepo/voip-kamailio-proxy/komodo/docker-compose.yml, the
      # ReplicaDegraded variant, alert-rules_test.yml, tests/config.bats.
      - alert: KamailioProxyTargetsMissing
        expr: (count(up{job="kamailio-proxy"}) or vector(0)) < 1
        for: 10m
        labels:
          severity: critical

      # 1-of-2 only. NO `or vector(0)` - deliberately, mirroring
      # ReplicaDegraded (alert-rules.yml:183-188). At zero targets the count
      # series disappears and this rule cannot fire; that boundary belongs to
      # the zero-target guard above (alert-rules.yml:155-161 states the same
      # three-way split for the manager fleet). Adding `or vector(0)` here would put
      # this rule and the zero-target guard on the same state, which is the
      # ownership split the fleet comment above forbids. (It would NOT make
      # the guard fire during a redeploy - that rule's `for: 10m` absorbs a
      # rollout, exactly as its AsteriskTargetsMissing model states at
      # alert-rules.yml:221-222.)
      # `for: 0s` matches the fleet's 2026-08-29 decision: a redeploy is
      # EXPECTED to produce this warning; that is how you know the replica
      # was actually recreated.
      - alert: KamailioProxyReplicaDegraded
        expr: count(up{job="kamailio-proxy"}) < 2
        for: 0s
        labels:
          severity: warning
```

**The alert that actually detects this ticket's outage is a third one, and it is nearly free.**
Neither rule above can catch the failure mode that produced this ticket: `initProm` runs in
`init()` before `main()`, so `/metrics` answers — and `up == 1` — even when `listenRun` died
silently and no consumer exists (`pkg/listenhandler/main.go:58-68`). The signal that
distinguishes "running" from "actually consuming" is already scraped: the `rabbitmq` job
(`prometheus.yml:96-98`, kbudde exporter with `RABBIT_EXPORTERS=exchange,node,queue`, no
`metric_relabel_configs`) exposes `rabbitmq_queue_consumers` — 91 series live on bm-nyc-01
today, e.g. `asterisk.call.request = 2`.

```yaml
      # THE detector for this ticket's failure mode, and it belongs in the
      # `rabbitmq` group (not instance-health) because it is sourced from
      # rabbitmq_* series, like its neighbour RabbitMQNoConsumers.
      # `< 1`, not `< 2`: one consumer is a fully working health-check
      # pipeline (the permanent queue is competing-consumer, and ConsumeRPC
      # registers exactly one AMQP consumer per replica -
      # bin-common-handler/pkg/rabbitmqhandler/consume.go:48-68), so 1-of-2
      # is redundancy loss and belongs to KamailioProxyReplicaDegraded.
      # `or vector(0)` is required here and means something different from
      # the target rules: it covers the queue not existing at all, which is
      # exactly what the 26-day outage looked like.
      # `and on() (up{job="rabbitmq"} == 1)` keeps a broker/exporter outage
      # from being reported as a proxy fault; RabbitMQDown covers that case.
      # `on()` is MANDATORY here, not stylistic: the left side is label-free
      # (max()/vector(0)) and the right side is labelled, so a bare `and`
      # (as at alert-rules.yml:254, where both sides share labels) would
      # never match and the alert would be permanently silent.
      - alert: KamailioProxyHealthCheckStalled
        expr: |
          ((max(rabbitmq_queue_consumers{queue="voip.kamailio.request"}) or vector(0)) < 1)
          and on() (up{job="rabbitmq"} == 1)
        for: 10m
        labels:
          severity: critical
```

**Overlap with the existing `RabbitMQNoConsumers`** (`alert-rules.yml:499-510`,
`rabbitmq_queue_messages_ready > 0 and rabbitmq_queue_consumers == 0`, unlabelled so it already
applies to this queue): that rule would catch a dead consumer on an **existing** queue that has
**pending messages**. It missed this outage on both counts — the queue never existed, and RPC
publishes to a nonexistent queue leave nothing ready. The new rule covers the queue-absent case
and the idle case (`messages_ready == 0` between 30 s cycles). Post-deploy, a dead proxy with
pending RPCs fires both; that is acceptable duplication for a critical path, and the new rule's
summary must be specific enough to tell them apart.

**Annotations for all three are label-free.** `count(...)` and `max(...)` strip every label, so
no `{{ $labels.* }}` renders — `alert-rules_test.yml:842` pins exactly that empty-string outcome
for a neighbouring rule. Do not copy a `{{ $labels.service }}` line from an adjacent rule — it
would render empty. `{{ $value }}` is unaffected by label stripping and **is** available: the
model rule uses it (`alert-rules.yml:229`, "only {{ $value }} of 6 targets discovered"), and it
is the only dynamic context these rules can carry (0-of-2 vs 1-of-2), so use it in `summary`.
The CI assertion bans templates in the **runbook** only (`tests/config.bats:622`,
`assert '{{' not in ann['runbook']`), not in `summary`/`impact`.

All three rules carry the three-annotation schema (`alert-rules.yml:53`) with no Go templates in
the runbook.

**`InstanceDown`'s runbook is deliberately NOT touched.** Adding a `kamailio-proxy` step there
would break monorepo-etc CI in four pinned places — `tests/config.bats:273-286` asserts
literally `'5. kamailio job:' in rb`, `'6. Logs:' in rb` and `'5. Logs:' not in rb`, and
`alert-rules_test.yml:842,864` pin the entire rendered runbook verbatim for the kamailio-job
and asterisk-job cases — which is the exact failure class used above to reject option (a). It
is also unnecessary: the runbook's existing **step 2** (`docker ps --filter name={{ $labels.job }}`)
works for this target, because the job name, the compose service key and the container-name
substring are all `kamailio-proxy`. (Step 3 does *not* cover it — it is textually scoped to the
`voipbin-managers` job and routes the reader to bin-*-manager projects. The residual gap is that
no step names this service explicitly.) The triage detail specific to this service lives
in `komodo/README.md` and in the new alert's own runbook.

Two prose registries enumerate the covered alerts and must be updated alongside the new cases,
though neither is CI-asserted: `alert-rules_test.yml:6-9` and the step comment in
`monorepo-etc/.circleci/config_work.yml:578-584`.

Tests, following the `AsteriskTargetsMissing` precedent: promtool cases in
`alert-rules_test.yml` covering (1) zero targets → `KamailioProxyTargetsMissing` fires,
(2) zero targets → `KamailioProxyReplicaDegraded` **silent** (this is what pins the missing
`or vector(0)`, and it is the case the fleet's own comment says belongs to the other rule),
(3) both up → both silent, (4) both down (`up == 0`, two series) → both silent, that is
`InstanceDown`'s job, (5) one of two → `KamailioProxyReplicaDegraded` fires and
`KamailioProxyTargetsMissing` silent, (6) `rabbitmq_queue_consumers` absent with
`up{job="rabbitmq"} == 1` → `KamailioProxyHealthCheckStalled` fires, (7) same but
`up{job="rabbitmq"} == 0` → silent (pins the exporter-outage guard), (8) consumers == 1 **with
`up{job="rabbitmq"} == 1` present in the input series** → silent. That series is what makes the
case discriminating: without it the `and on()` guard silences the rule regardless of threshold,
so a `< 2` mutation would pass.

Each case carries explicit `eval_time`s, and the two `for:` values are pinned behaviourally the
way this file already does it (`alert-rules_test.yml:739-746`: "The 9m probe … pins `for: 10m`
behaviourally for this rule too (a `for: 5m` or `for: 0m` mutation would fire here)"): for the
two `for: 10m` rules, a 9m probe asserting still-pending plus a 12m probe asserting fired; for
`KamailioProxyReplicaDegraded`'s `for: 0s`, a probe at the first evaluation asserting it has
already fired (a `for: 10m` mutation would be silent there). Cases (1) and (2) also need the
filler-series trick the zero-target precedent uses (`alert-rules_test.yml:782-787`: "The
unrelated asterisk-proxy series is present only so the test has some input"), since a case with
no input series at all evaluates nothing.

Alongside those promtool cases, `tests/config.bats` gets assertions for the job shape (dns_sd
not static_configs, port 2112, the `service` relabel) and for each new alert's three-annotation
shape.

Naming and placement: `alert-rules.yml:306` already has `KamailioTargetMissing` for the SIP
proxy's own xhttp_prom target (VOIP-1474), so the three new rules must lead their
`summary`/`impact` with "provider health-check RPC worker", never a bare "Kamailio", and the two
`up`-based rules go in the `instance-health` group (the precedent tests assert
`grp['instance-health']`) and the `rabbitmq_queue_consumers` rule goes in the **`rabbitmq`**
group (`alert-rules.yml:459`), next to `RabbitMQNoConsumers`, because the file segregates rules
by metric source.

While editing `prometheus.yml`, add the missing `dns_sd_configs` bullet to the header's target-style list
(`:5-29` enumerates the other styles but not this one; `config.bats:314-326` only pins the
docker_sd bullet, so this edit is free).

Docs: add the dashboard-free service to `docker-compose_monitoring/README.md`'s job list and
record in `docs/follow-ups.md` that this target's only signal is liveness — the service
registers no custom metrics (`promhttp.Handler()` only, `docs/operations.md:108,110`), and
the health check's real health is observable from `route_providers.health_checked_at`.

## Known limitation

The probe's source IP is now permanently different from Kamailio's SIP IP. Today every active
carrier authenticates by FQDN, so this costs nothing. If a source-IP-ACL carrier is ever
added, calls would work while its health check reported `unhealthy`. Routing does not consult
`health_status`, so this is a reporting defect, not an outage. Record it in
`komodo/README.md`; the fix would be an egress rule or a carrier-side ACL entry for
`104.243.38.39`.

## Rollout

No SIP interruption: the Kamailio stack is not touched.

1. Merge Change 1, then click `build-approval` **on the `voip-kamailio-proxy` workflow**.
   **Order matters in both directions**: merging alone does not deploy, and approving on an
   unmerged branch runs test → build → deploy straight to production with no further gate.
   **There will be more than one approval button.** Change 1 edits
   `bin-route-manager/docs/operations.md`, which matches `bin-route-manager/.*`
   (`.circleci/config.yml:41`) and arms that service's workflow — whose approval performs a
   live Komodo redeploy of both route-manager replicas. Leave it untouched. (If that risk is
   judged not worth carrying, the alternative is to drop the route-manager doc fix from this PR
   and land it separately; the design keeps it here so the on-call doc stops describing a
   topology in which this outage is impossible.)
2. Verify on bm-nyc-01 — **not by whether the container is running**. `listenhandler.Run()`
   spawns a goroutine and returns `nil` unconditionally (`pkg/listenhandler/main.go:58-68`),
   so a failed `QueueCreate`/`ConsumeRPC` leaves the container `Up` with no consumer, which
   looks exactly like today's outage. Also note the CI gate cannot prove 2-of-2:
   `check_stack_running` reads `.container.state` per compose service, not per container
   (`docs/workflows/manager-replica-scaling.md:106-123`), so a green deploy proves at most
   1-of-2. Required checks:
   - `docker ps --filter name=kamailio-proxy` → **exactly 2** running containers
   - `docker exec infra-rabbitmq rabbitmqctl list_queues name consumers | grep voip.kamailio.request`
     → **consumers == 2** (one `ConsumeRPC` registration per replica). `>= 1` would pass a
     half-failed deploy
   - route-manager logs: no `circuit breaker` or `context deadline exceeded` for that queue
   - `route_providers.health_checked_at` advancing on a ~30 s cadence
   - `sip.telnyx.com` → `healthy`, `pchero21.com` → `unhealthy`
3. Merge Change 2, approve `infra-prometheus`. Then `count(up{job="kamailio-proxy"}) == 2`,
   and all three new alerts silent.

**Rollback.** This is the stack's first deploy, so there is no previous pipeline run to
re-approve; the only rollback is stopping or deleting the stack in Komodo. Before Change 2 is
merged, that returns the system to today's state — failing health checks and a stale
`health_status`, degraded but not an outage. **After Change 2 is merged, stopping the stack leaves two alerts firing**:
`KamailioProxyTargetsMissing` and `KamailioProxyHealthCheckStalled`, both after 10 minutes. It
does **not** leave `KamailioProxyReplicaDegraded` firing — at zero targets
`count(up{job="kamailio-proxy"})` returns empty and, with no `or vector(0)`, that rule cannot
match; it fires for at most one evaluation window during teardown and then resolves on its own.
So a rollback at that point must revert Change 2 or silence those two.

## Out of scope

- Making routing consult `health_status` (a product decision).
- Cleaning up the 8 soft-deleted dummy providers.
- Removing the dead `voip-kamailio-ansible` proxy definition — worth doing to prevent drift,
  but it is a different repo and a different concern; track separately if wanted.
