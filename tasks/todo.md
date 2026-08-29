# VOIP-1407: Cutover -- remove fanout dual publish and per-service event exchanges

Status: ISSUE ANALYSIS APPROVED (stage 1 of 4 complete -- R10 Approve, R11 Approve,
2 consecutive). Proceed to design stage. rev.10 after R1 Request Changes (1 CRITICAL:
missing asterisk-proxy publisher), R2 Request Changes (1 CRITICAL: missing
consumer-side fanout residue that VOIP-1406 deliberately left as rollback surface; 2
HIGH/MEDIUM count and cross-reference errors; 2 MEDIUM scope-closure contradictions; 2
LOW; 1 process finding), R3 Request Changes (1 HIGH: the 28-exchange deletion set was a
placeholder, never enumerated, and its correct exclusion of
`bin-manager.webhook-manager.event` was undocumented; 4 MEDIUM: stale-binding
self-healing loss, a wording contradiction, dangling cross-references from
renumbering, and failure semantics presented as both settled and open; 3 LOW; 1 process
finding, closed with a fresh full re-verification), and R4 Request Changes (1 HIGH: the
rev.4 "QueueUnbind against a missing exchange is safe" proof relied on pod liveness
despite the call's own error being log-only and non-fatal to `Run()`, replaced with a
same-channel-ordering argument already latent in §1's own data; 6 LOW: mis-numbered
finding attribution, an over-broad stale-binding-policy citation, an unqualified "zero
references" claim, an unexplained 36->33 constant-count derivation, two unflagged
`subscribeTargets` structural exceptions, and one hardcoded-literal exchange-name call
site the methodology's own grep pattern cannot see), and R5 Request Changes (1 HIGH:
the rev.5 same-channel-ordering argument was itself invalid on two counts --
`TopicCreateWithKind` opens a separate channel, and a binding snapshot cannot date
which boot created a durable binding -- replaced with direct current-boot evidence, a
live `consumers` count on the two affected queues; 1 MEDIUM: an unfounded causal claim
about why `bin-manager.webhook-manager.event` no longer exists, corrected to an
honestly-uncertain explanation; 6 LOW: gomock-expectation undercount [7->8], a wrong
line citation, a doc-mention undercount, a stale round-count in the recommendation, a
mislocated code comment, and an ambiguous metrics-impact denominator), and R6 Request
Changes (2 HIGH: the rev.6 live-`consumers`-count proof was confounded by RabbitMQ's
automatic reconnect path, which can re-register consumers on a fresh channel without
ever re-running the boot-time `QueueUnbind` -- making a live count consistent with
either a clean boot or a silently-failed one later healed by reconnect; and the
exchange's absence was only established in the present tense, not at the time the
pods in question actually booted -- both replaced with direct container-log evidence
(absence of the call's own `CRITICAL` error string, and absence of any
reconnect/redeclare log line, across each container's full lifetime from its
`StartedAt` timestamp) plus a `docker ps` replica-count citation; 2 MEDIUM: an
unsupported "matches replica count" claim, and an uncited bind-vs-unbind AMQP
condition conflation; 4 LOW: a dangling cross-reference, an unqualified migration
hypothesis, an overgeneralized restart claim, and un-elided `'...'` command output),
and R7 Request Changes (2 CRITICAL: the rev.7 container-log-absence proof assumed
unbounded log retention that these services' own committed Docker log-rotation config
(`max-size: "10m"`, `max-file: "3"`) does not provide, and separately the webhook
`QueueUnbind` call sits behind an unexamined `else if` guard on a prior `QueueBind`,
so log-absence could not distinguish "unbind succeeded" from "unbind never executed
because the bind failed first" -- both replaced with a direct controlled experiment: an
isolated scratch queue on the real production broker (RabbitMQ 3.13.7), `queue.unbind`
against a guaranteed-nonexistent exchange, observed to raise no exception with the
channel remaining usable afterward; 1 HIGH recommending exactly this experiment as the
fix; 2 LOW: a stale §7 recommendation and an overgeneralized restart-confirmation
claim), and R8 Request Changes (the reviewer explicitly confirmed the rev.8 unbind
experiment itself was sound and a genuinely different, non-inferential proof strategy
from all four predecessors -- no further attempt needed on that specific result; 1
HIGH: the claim is two-sided [`QueueBind` fails, `QueueUnbind` doesn't] but only the
unbind arm was tested, with the bind arm resting on an uncorroborated source-comment
citation, fixed by adding a second experimental arm -- a `queue.bind` against a
missing exchange -- as a positive control in the same harness, which reproduced the
exact 404 and channel death; 2 MEDIUM: two unstated bridging premises [Python/Go AMQP
client equivalence; scratch-queue-property immateriality] now stated explicitly; 5
LOW: an arithmetic slip ["five" vs "four" inference attempts], a version-string typo,
an imprecise "no state modified" claim, reproducibility gaps in the shown commands,
and a run-on sentence burying its own conclusion), and R9 Request Changes (the
reviewer confirmed the two-arm experiment itself genuinely closes both halves of §1's
claim and is substantively airtight; 1 MEDIUM: the client-equivalence bridging premise
asserted "neither client exposes a nowait form," false for amqp091-go's `QueueBind`
[which does take a `noWait` parameter] -- fixed by citing the true, already-verified
justification instead [every production call site passes `noWait=false`]; 2 LOW: an
imprecise amqp091-go line citation, and a stale "Seven review rounds" in §7 that had
not been updated to include R8).

## Issue Analysis (2026-08-29, rev.10)

### 1. Issue validity: VALID. Precondition evidence, and what it does and does NOT prove

Live broker inspection on bm-nyc-01 (104.243.38.39), via SSH + RabbitMQ management API
(port **80**), re-run fresh for this revision (2026-08-29, R3 finding 9 -- prior
rounds' reviewers lacked shell access to independently reproduce this):

```
$ curl -s -u voipbin:$PASS http://$IP:80/api/exchanges/%2f/bin-manager.event/bindings/source
total bindings: 66
  agent-manager.subscribe: 4     number-manager.subscribe: 2
  ai-manager.subscribe: 10       queue-manager.subscribe: 3
  billing-manager.subscribe: 14  registrar-manager.subscribe: 2
  call-manager.subscribe: 4      schedule-manager.subscribe: 1
  campaign-manager.subscribe: 2  storage-manager.subscribe: 2
  conference-manager.subscribe: 2  tag-manager.subscribe: 1
  contact-manager.subscribe: 1   timeline-manager.subscribe: 1
  conversation-manager.subscribe: 4  transcribe-manager.subscribe: 3
  direct-manager.subscribe: 1    transfer-manager.subscribe: 3
  flow-manager.subscribe: 1      webhook-manager.subscribe: 5
# = 65 PatternForEventType + 1 timeline "#" -- matches VOIP-1406 design §5 exactly.
```

**The 28 fanout exchanges this ticket's §4.4 runbook deletes, enumerated explicitly
(R3 finding 1 -- rev.3 used an unresolved placeholder here), each individually queried
for existence + binding count:**

```
$ for ex in ai-manager agent-manager billing-manager call-manager campaign-manager \
    conference-manager contact-manager conversation-manager customer-manager \
    direct-manager email-manager flow-manager message-manager number-manager \
    outdial-manager pipecat-manager queue-manager registrar-manager route-manager \
    schedule-manager sentinel-manager storage-manager tag-manager talk-manager \
    transcribe-manager transfer-manager tts-manager webchat-manager; do
    code=$(curl -s -o /dev/null -w '%{http_code}' -u voipbin:$PASS \
      "http://$IP:80/api/exchanges/%2f/bin-manager.$ex.event")
    [ "$code" = 200 ] && curl -s -u voipbin:$PASS \
      "http://$IP:80/api/exchanges/%2f/bin-manager.$ex.event/bindings/source" \
      | python3 -c 'import json,sys; d=json.load(sys.stdin); print(len([b for b in d if b["destination_type"]=="queue"]))'
  done
# -> all 28 EXIST, all report 0 queue bindings.
```

**Also queried, to settle exactly which `QueueName*Event`-family exchanges are NOT in
the 28-exchange deletion set, and why (R3 finding 1's second half):**

```
bin-manager.api-manager.event:      ABSENT (404) -- api-manager never published events; dead constant, never had an exchange.
bin-manager.rag-manager.event:      ABSENT (404) -- same, rag-manager.
bin-manager.timeline-manager.event: ABSENT (404) -- same, timeline-manager is consumer-only, never published.
bin-manager.user-manager.event:     ABSENT (404) -- no user-manager service exists; dead constant.
bin-manager.webhook-manager.event:  ABSENT (404) -- cause unestablished (see next paragraph);
                                     VOIP-1296 removed the code that used to declare it, and
                                     nothing has re-declared it since, but that alone does not
                                     explain a durable exchange's disappearance.
bin-manager.webhook-manager.event.topic: EXISTS -- a DIFFERENT exchange (topic-kind, VOIP-1258's mechanism,
                                     permanently out of scope, not part of the fanout family at all).
```

`bin-manager.webhook-manager.event` is correctly excluded from the 28-exchange deletion
set not because deleting it would be risky, but because **it does not exist to
delete**. Its exact CAUSE of absence is not established and this document does not
claim one (R5 finding 2 -- an earlier draft asserted "VOIP-1296's cutover deleted it,"
which is mechanically implausible: ceasing to DECLARE a durable exchange does not
DELETE it -- an explicit `exchange.delete` or a broker rebuild would be required, and
neither is evidenced). What IS verified: VOIP-1296 removed webhook-manager's own
publish path to this exchange (`bin-webhook-manager/pkg/webhookhandler/
routingkey.go:102`, `webhook_test.go:416-417`), and zero code anywhere in
`bin-webhook-manager` constructs a plain `NewNotifyHandler` that would re-declare it
(§3). Its exact cause is left unestablished deliberately -- one plausible mechanism
(unverified, offered only as a hypothesis, not relied on by any proof below): this
session's memory of the bare-metal migration to bm-nyc-01 (VOIP-1325) suggests a
single fresh broker where only actively-declared exchanges exist post-migration, which
would also explain the 4 dead constants' exchanges being absent the same way. This
matters for the empirical proof directly below, not for §2b.

```
$ curl -s -u voipbin:$PASS http://$IP:80/api/exchanges/%2f/asterisk.all.event/bindings/source \
    | python3 -c 'import json,sys; d=json.load(sys.stdin); print([b["destination"] for b in d if b["destination_type"]=="queue"])'
['bin-manager.call-manager.subscribe', 'bin-manager.timeline-manager.subscribe']
```

Sampled 5 subscribe queues' FULL binding lists individually (billing/conversation/
transfer/storage/tag): each queue's only event source is `bin-manager.event` (plus the
harmless default-exchange self-bind and the unrelated `bin-manager.delay` retry
exchange). Separately, THIS session's own VOIP-1406 deploy roll-out included a real
restart of billing-manager-1 specifically (observed "Up 8 seconds" shortly after that
deploy, with the same clean binding state) -- scoped to that one service, not
generalized to the other four sampled queues, which were not individually confirmed
post-restart.

**§1's central empirical claim: `QueueBind`/`QueueSubscribe` 404 (and kill the channel)
when the target exchange is missing, but `QueueUnbind` does NOT. Settled this revision
with a two-arm DIRECT CONTROLLED EXPERIMENT against the real production broker,
including a positive control, rather than any further inference from production STATE.**
(R8 finding 1 -- four prior attempts across rev.4/5/6/7 were each inference-based and
each rested on a premise that turned out false, unverifiable, or unproven: process
liveness doesn't distinguish a swallowed error from success; a same-channel "ordering"
argument used `TopicCreateWithKind`, which actually opens a SEPARATE channel; a live
consumer-count argument is confounded by RabbitMQ's automatic reconnect path; and a
container-log-absence argument assumed unbounded log retention that these services' own
committed Docker log-rotation config (`max-size: "10m"`, `max-file: "3"`,
`bin-agent-manager/komodo/docker-compose.yml` and `bin-timeline-manager/komodo/
docker-compose.yml`) does not actually provide, while the call itself sits behind a
conditional `else if` that a log-absence argument cannot distinguish from "never
executed." This revision instead tests the underlying AMQP/broker-semantics question
directly, for BOTH operations, so the null result on unbind is interpretable against a
proven-working test harness rather than merely asserted.)

```
$ docker exec infra-rabbitmq rabbitmqctl version
3.13.7
$ cat > /tmp/unbind_test.py << 'PYEOF'
import pika
# Client equivalence: pika's queue_unbind/queue_bind and amqp091-go's Channel.QueueUnbind
# (bin-common-handler/pkg/rabbitmqhandler/queue.go:195-218, calling the driver at :201)/
# QueueBind (queue.go:166-191, calling the driver at :172) emit the identical AMQP 0-9-1
# queue.unbind/queue.bind methods. pika's BlockingChannel exposes no nowait form for
# either; amqp091-go's Channel.QueueBind does take a noWait parameter, but every
# production call site passes false (queue.go:159's QueueSubscribe wrapper -- the legacy
# fanout path this experiment targets -- and each service's topicPatterns binds), and
# Channel.QueueUnbind has no noWait parameter at all. So both calls, on both clients,
# synchronously await the broker's -ok and surface a broker rejection as a returned
# error -- the broker-side resolution path and the observable outcome are the same.
creds = pika.PlainCredentials("voipbin", "<redacted, read via `docker exec infra-rabbitmq printenv RABBITMQ_DEFAULT_PASS`>")
params = pika.ConnectionParameters(host="rabbitmq", credentials=creds)  # "rabbitmq" is
# infra-rabbitmq's Docker network alias on the `production` network (docker inspect
# infra-rabbitmq --format '{{json .NetworkSettings.Networks}}' confirms Aliases: [rabbitmq]).

# ARM 1: unbind against a missing exchange -- the claim under test.
conn1 = pika.BlockingConnection(params)
ch1 = conn1.channel()
ch1.queue_declare(queue="voip1407-scratch-test-2", durable=False, auto_delete=True, exclusive=True)
try:
    ch1.queue_unbind(queue="voip1407-scratch-test-2",
                      exchange="voip1407-does-not-exist-exchange-2", routing_key="")
    print("ARM1 UNBIND-MISSING-EXCHANGE: no exception")
except Exception as e:
    print("ARM1 UNBIND-MISSING-EXCHANGE: EXCEPTION:", type(e).__name__, str(e))
try:
    ch1.queue_declare(queue="voip1407-scratch-test-2", durable=False, auto_delete=True,
                       exclusive=True, passive=True)
    print("ARM1 CHANNEL-STILL-ALIVE:", "yes")
except Exception as e:
    print("ARM1 CHANNEL-STILL-ALIVE: NO:", type(e).__name__, str(e))
conn1.close()

# ARM 2 (positive control): bind against a missing exchange -- expected to fail and kill
# the channel, per bin-timeline-manager/pkg/subscribehandler/main.go:147-152's own comment
# and the 2026-07-14 VOIP-1258 production incident. Proves the harness CAN detect a broker
# error, so ARM 1's null result is a finding, not a blind spot.
conn2 = pika.BlockingConnection(params)
ch2 = conn2.channel()
ch2.queue_declare(queue="voip1407-scratch-test-3", durable=False, auto_delete=True, exclusive=True)
try:
    ch2.queue_bind(queue="voip1407-scratch-test-3",
                    exchange="voip1407-does-not-exist-exchange-3", routing_key="")
    print("ARM2 BIND-MISSING-EXCHANGE: no exception (unexpected)")
except Exception as e:
    print("ARM2 BIND-MISSING-EXCHANGE: EXCEPTION:", type(e).__name__, str(e))
try:
    ch2.queue_declare(queue="voip1407-scratch-test-3", durable=False, auto_delete=True,
                       exclusive=True, passive=True)
    print("ARM2 CHANNEL-STILL-ALIVE-AFTER-BIND-FAIL: yes (unexpected)")
except Exception as e:
    print("ARM2 CHANNEL-STILL-ALIVE-AFTER-BIND-FAIL: NO (expected):", type(e).__name__, str(e))
conn2.close()
PYEOF
$ docker run --rm --network production -v /tmp/unbind_test.py:/test.py python:3-slim \
    bash -c "pip install pika -q && python3 /test.py"
```
```
ARM1 UNBIND-MISSING-EXCHANGE: no exception
ARM1 CHANNEL-STILL-ALIVE: yes
ARM2 BIND-MISSING-EXCHANGE: EXCEPTION: ChannelClosedByBroker (404, "NOT_FOUND - no exchange 'voip1407-does-not-exist-exchange-3' in vhost '/'")
ARM2 CHANNEL-STILL-ALIVE-AFTER-BIND-FAIL: NO (expected): ChannelWrongStateError Channel is closed.
```

**This is a direct observation with a working positive control, not an inference
chain and not a one-sided assertion**: run against the real production broker
(RabbitMQ 3.13.7 -- version-sensitivity matters for AMQP error-handling semantics, so
the exact production version was used, not assumed), using `pika` in a throwaway
container attached to the `production` Docker network, against scratch queues
(non-durable/exclusive/auto-delete -- self-deleting on connection close, isolated from
and never touching any real production queue, exchange, or binding; only transient
scratch state was created, and it was gone by the time each connection closed). The
durability/exclusivity difference between the scratch queues and the real subscribe
queues is immaterial here: RabbitMQ resolves the SOURCE exchange named in `queue.bind`/
`queue.unbind` before reaching any durability-dependent binding-table logic, so a
missing exchange short-circuits identically regardless of the destination queue's
properties -- ARM 2's exact-match 404 (`"no exchange '...' in vhost '/'"`, an
exchange-resolution error, not a queue or binding error) confirms this directly. ARM 1:
`queue.unbind` against a guaranteed-nonexistent exchange raised no exception, and a
subsequent operation on the SAME channel (a passive queue-declare) succeeded, proving
the channel was not closed. ARM 2 (the positive control): `queue.bind` against a
guaranteed-nonexistent exchange raised `ChannelClosedByBroker (404, NOT_FOUND)`
immediately, and the same subsequent same-channel operation then failed with
`ChannelWrongStateError` ("Channel is closed") -- exactly the failure mode
`bin-timeline-manager/pkg/subscribehandler/main.go:147-152`'s own comment describes
("makes QueueSubscribe fail with an AMQP 404, which closes the shared channel and
takes this service down at boot"), reproduced directly rather than merely cited.

Together the two arms answer the question every prior inference attempt was trying to
indirectly establish, for BOTH halves of §1's claim, and do not depend on log
retention, reconnect history, container uptime, or the specific conditional structure
of any one service's `Run()`. `bin-agent-manager/pkg/subscribehandler/main.go:118-120`
and `bin-timeline-manager/pkg/subscribehandler/main.go:177-179`'s `QueueUnbind(queue,
"", "bin-manager.webhook-manager.event", nil)` calls -- and any other leftover
`QueueUnbind` call anywhere in this migration referencing an absent exchange -- are
therefore confirmed safe no-ops at the broker-protocol level; conversely, §2b's
"deleting the exchange kills `Run()` on the next fanout `QueueSubscribe`" claim and
§4 item 4's runbook precondition are now backed by a reproduction of the exact failure,
not just a source-comment citation.

(For completeness: the webhook `QueueUnbind` call in both services actually sits behind
an `else if` guard on a prior `QueueBind` to the (existing) VOIP-1258 topic exchange, so
it only executes when that bind succeeds -- irrelevant to the experiment above, which
tests both operations in isolation, but worth noting for anyone reading the surrounding
source. Also: nothing in either service's `subscribeTargets`/`fanoutUnbindTargets` lists
re-binds `QueueNameWebhookEvent`, so the unbind call was already redundant before this
ticket, on top of being safe.)

This distinction is the reason §2b's danger is specifically about the fanout
`QueueSubscribe` BIND loop, not about any leftover `QueueUnbind` calls that might
reference an already- or soon-to-be-deleted exchange.

**What this precondition proves, precisely:** every CONSUMER has migrated off the 28
fanout exchanges onto `bin-manager.event`, for exactly the 28 exchanges enumerated
above, with a real-restart confirmation for one sampled service (billing-manager -- R7
finding 5, correcting an earlier overgeneralization to "every consumer"; the other
sampled queues are point-in-time snapshots, not individually restart-confirmed).
**What it does NOT prove, and rev.2 wrongly implied
it did**: it says nothing about whether the CODE that produced this steady state still
contains a live fanout `QueueSubscribe` (bind) path that would try to rebind to those
exchanges on a FUTURE restart. It does (§2b) -- this is R2's CRITICAL finding, addressed
below.

**Rollback / point-of-no-return, corrected (R2 finding 4 -- was self-contradictory and
mis-cited §4.2 instead of §4.3 twice):**

| Action | Reversible? | How |
|---|---|---|
| Revert the notifyhandler/publisher code (§4 item 1) | Yes | Dual publish resumes cleanly; `TopicCreate(queueEvent)` re-declares an exchange that was never deleted. |
| Revert the consumer-side code (§2b/§4 item 2, the fanout-`QueueSubscribe`-removal half) | Yes, UNTIL the exchanges are deleted | Re-adding the fanout subscribe loop just re-binds to exchanges that still exist. |
| Delete the 28 exchanges (§4 item 4's runbook) | **NO, for any service whose consumer code has not yet been redeployed without the fanout QueueSubscribe loop** | A `QueueSubscribe`/`QueueBind` call to a deleted exchange 404s the channel; per the CURRENT code (`bin-tag-manager/pkg/subscribehandler/main.go:110-113` and identically in the other 19 services), that failure `return`s from `Run()` -- the subscribe handler never starts, and there is no fallback because the fanout exchange the fallback depends on is gone. **This -- not the publisher-side change -- is the actual point of no return**, and it is gated on BOTH publisher and consumer code being fully redeployed first (§4 item 4). By contrast, `QueueUnbind` calls left referencing a deleted exchange (e.g. the legacy VOIP-1258 lines in agent/timeline-manager, §1) are NOT part of this risk -- `QueueUnbind` against a missing exchange does not 404 (empirically proven, §1). |

### 2. Two independent code changes required, not one -- corrects rev.1/rev.2's publish-only framing

**(a) Publish side** (bin-common-handler/pkg/notifyhandler + ~62 call sites) -- unchanged
from rev.2, detailed in §3. Its "degrade, don't abort" design intent (the reason
`topicDisabled` exists today) is documented in the VOIP-1406/1404 design docs' own §2,
not in this document -- see §3's failure-semantics paragraph for what changes here.

**(b) Consumer side (R2 CRITICAL finding, verified against source this revision) --
entirely missing from rev.1/rev.2, and larger in service-count than (a).** The VOIP-1406
design doc explicitly assigns this ticket the removal of the fanout rollback surface it
deliberately built and left in place:
- `docs/plans/2026-08-29-voip-1406-consumer-topic-migration-design.md:90-91`: *"Fanout
  `QueueSubscribe` calls in step 1 stay in the code during 1406 (they are the rollback
  surface and the degrade path). **VOIP-1407 deletes them.**"*
- Same doc, :129-130 (sentinel defensive declare, call-manager and timeline-manager
  only): *"The declare is deleted in **VOIP-1407** together with the fanout
  QueueSubscribe lines."*
- Same doc, :10: *"Blocks: VOIP-1407 (Follow-up C: remove fanout publish **+
  per-service fanout exchanges**)"*.

Verified directly against `bin-tag-manager/pkg/subscribehandler/main.go` (representative
-- identical shape in the other 19 VOIP-1406 services): `Run()` does
`QueueCreate` -> loop `QueueSubscribe(subscribeQueue, target)` over `h.subscribeTargets`
(**a bind with no declare** -- `sockhandler.QueueSubscribe` is
`QueueBind(name, "", exchange, false, nil)`, `bin-common-handler/pkg/rabbitmqhandler/
queue.go:158-159`) -> `TopicCreateWithKind(bin-manager.event)` -> `QueueBind` each
`topicPatterns` entry -> on full success, `QueueUnbind` each `fanoutUnbindTargets`
entry -> `go ConsumeMessage(...)`. **The fanout `QueueSubscribe` loop's error path
`return`s immediately** (`main.go:110-113`: `if errSubscribe := ...; errSubscribe != nil
{ ... return errSubscribe }`) -- a bind to a deleted exchange kills `Run()` outright,
with NO fallback, because the fanout leg the "stay fully on fanout" degrade path (the
VOIP-1406 design's own intent, referenced above in (a)) depends on no longer exists.
Verified this same shape (fanout-subscribe-loop-error-returns-immediately) in 6 services
beyond tag-manager, spread across the list rather than assuming uniformity: agent, call,
billing, timeline, transfer, webhook -- all six `return errSubscribe` (or the equivalent
`return nil, errSubscribe` in a constructor-shaped `Run`) with no fallback.

**Concrete consequence if §4.4's exchange deletion runs while this code is still
deployed anywhere:** the next restart of ANY of the 20 VOIP-1406 services (pod restart,
rolling deploy, crash recovery -- not a hypothetical, a routine operational event) fails
to start its subscribe handler. This is a full, silent, service-wide event-intake outage
for that service, discovered only at the next restart, potentially long after the
exchanges were deleted.

**Scope this adds** (mirrors VOIP-1406's own service list, 20 services: agent, ai,
billing, call, campaign, conference, contact, conversation, direct, flow, number, queue,
registrar, schedule, storage, tag, timeline, transcribe, transfer, webhook):
- Delete the fanout `QueueSubscribe` loop and the `subscribeTargets`
  package-level/wiring var (fed from each `cmd/*-manager/main.go`) in each service's
  `pkg/subscribehandler/main.go`. **Two structural shape exceptions (R4 finding 6),
  already called out by the VOIP-1406 design doc (:100-107) and worth restating here so
  the design stage doesn't size this as 20 identical edits**: `bin-webhook-manager/pkg/
  subscribehandler/main.go:97,105` receives `subscribeTargets` as a comma-joined
  `string` split inside `Run()`, not a slice like the other 19; `bin-timeline-manager/
  pkg/subscribehandler/main.go:27-54` declares it as a package-level var (`:30` is one
  entry inside it, `QueueNameAsteriskEventAll` -- the one entry that must NOT be
  deleted, per the bullet below) with no `cmd/` wiring at all (nothing to touch there
  beyond the var itself).
- Delete `fanoutUnbindTargets` (no longer meaningful once there is nothing to unbind
  from) and the `QueueUnbind` step -- `topicPatterns`/`QueueBind` becomes the sole
  intake mechanism. Whether a topic-declare/bind failure at startup should now be FATAL
  (there is no fanout left to degrade to) is an explicit open ruling, tracked once in
  §6 (deferred item 6) -- not asserted as decided here.
- call-manager and timeline-manager: delete the sentinel defensive `TopicCreate`
  declare (VOIP-1406 design doc :124-130 explicitly assigns this to VOIP-1407) --
  **but do NOT touch the `asterisk.all.event` `QueueSubscribe`**, which is permanent
  (the asterisk-proxy publish leg §3 item 1 keeps alive has no topic counterpart, so
  its two consumers keep subscribing to it forever; this is the one fanout
  `QueueSubscribe` line across all 20 services that must NOT be deleted).
- Update every service's `binding_golden_test.go` (the fanout-target pins become
  unnecessary) and `docs/architecture.md` events section.
- **Precondition on the consumer-side change itself, not just on §4 item 4's exchange
  deletion (R3 finding 2, citations corrected R4 finding 2)**: VOIP-1406's design doc
  (:225-232) established a stale-binding policy that TOLERATES resurrected fanout
  bindings from two live triggers -- an image rollback, or a 2-replica rolling-deploy
  window where an old-image pod's `redeclareAll()` replays its tracked fanout bind
  after the new pod already unbound it -- remediated via manual `rabbitmqadmin` unbind
  or roll-forward ("No automated cleanup in 1406" -- the doc is explicit that this is
  NOT automatic). The actual automatic self-heal this ticket removes lives in §2's own
  per-boot template (:66-74) and reconnect note (:92-94): every service's `Run()`
  already re-runs the bind-then-unbind sequence on every boot, so a stray fanout
  binding left by either trigger above gets swept the next time that service restarts
  for ANY reason, even without a human running the manual remediation. Removing the
  `QueueUnbind` loop removes THIS automatic path entirely: any fanout binding stray at
  the moment this consumer-side change deploys becomes permanent double delivery,
  unremediable by any subsequent boot (manual remediation per :225-232 still works, it
  is simply no longer optional), until the exchange itself is deleted (§4 item 4).
  **This consumer-side change
  should therefore only roll out to a service once a stale-binding sweep (the same
  broker-binding inspection §1 already demonstrates) confirms zero stray fanout
  bindings for that service**, not on a fixed schedule independent of that check.
  **A pre-rollout sweep alone is insufficient (R5 finding 9)**: it cannot catch a
  stray created DURING the rollout window itself by trigger (a) above (an old-image
  pod's `Run()` re-subscribing to fanout during a rolling deploy) -- since the new
  image no longer sweeps, such a stray would otherwise persist undetected until §4
  item 4's exchange deletion. Add a POST-rollout per-service re-sweep to this
  precondition as well, mirroring §4 item 4's own post-deletion re-check.
- **Sequencing, corrected (R3 finding 2 -- rev.3's "can deploy independently" was too
  strong)**: the publish-side change (§3) and this consumer-side change do not depend
  on EACH OTHER's code being live first, and can ship in separate PRs/commits. But
  each one has its OWN precondition before it is safe to roll out (publish-side: none
  beyond normal deploy hygiene; consumer-side: the stale-binding sweep above), and BOTH
  must be fully deployed to EVERY service, with a confirmed restart-survival check (not
  just a point-in-time binding snapshot -- §1), before §4 item 4's exchange deletion
  runs.

### 3. Publish side: mechanism and call-site inventory (unchanged substance from rev.2, count corrected)

`publishEvent()` ALWAYS calls `publishDirectEvent` (fanout, via `h.queueNotify`) first;
only on success, and only if `h.topicEnabled` (opt-in via `WithGlobalTopicPublish()`,
default off), does it call `publishTopicEvent` (global topic). Topic-publish failure is
swallowed. `topicDisabled`: a construction-time topic-declare failure silently suppresses
topic publish for that instance's lifetime, fanout keeps working -- this "degrade, don't
abort" behavior must invert once fanout is gone for the paths that drop it.

Delayed-publish (`delay > 0`) never reaches `publishTopicEvent`; confirmed dead code --
`PublishEventRaw`/`PublishEvent` hardcode `delay=0`, no interface method exposes delay,
`DelaySecond`/`DelayMinute`/`DelayHour` (notifyhandler package) have zero production
callers. Do not confuse with `requesthandler.DelayNow`/`DelaySecond`/`DelayMinute`/
`DelayHour` (`bin-common-handler/pkg/requesthandler/main.go:156-159`) -- different
package; of THOSE, only `DelayNow` has any caller at all, and only in
`requesthandler`'s own tests -- "live" was too strong a word for any of them (R2 finding
7); none are touched by this ticket regardless.

**Metrics (R2 finding 3, corrected -- rev.2's "no action needed" ruling was wrong
scope):** `promNotifyTotal` (fires in `publishEvent()` itself, survives removing
`publishDirectEvent`) is fine. `promNotifyProcessTime` is observed at exactly THREE call
sites: `publishDirectEvent` (removed by this ticket for every instance except
asterisk-proxy's), `publishDirectEventWithKey` (webhook-manager's VOIP-1258 path only),
`publishDelayedEvent` (dead code, previous paragraph). **Removing `publishDirectEvent`
silences `<namespace>_notify_process_time` for the 27 daemons that dual-publish via
`publishEvent()` (§3's 27 dual-publish daemons -- the ones becoming topic-only per §4
item 1) -- their publish-latency signal disappears entirely** (R5 finding 8,
denominator corrected: NOT "27 of the 28 real publishers" -- of the 29 processes that
actually publish something today (27 dual-publish daemons + asterisk-proxy +
webhook-manager), asterisk-proxy KEEPS the metric because its fanout leg and
`publishDirectEvent` call are deliberately preserved (§4 item 1), and webhook-manager
was never affected in the first place, since it observes `promNotifyProcessTime` via
the unrelated `publishDirectEventWithKey` call site) -- `publishTopicEvent` deliberately
does not observe it today. External-consumer audit (`~/gitvoipbin/
monorepo-etc`, all Grafana dashboards + alert rules, all four metric names): zero
matches, so nothing breaks visibly, but the signal itself disappears platform-wide.
**Design ruling required, not a note**: either make `publishTopicEvent` observe
`notify_process_time` (folding topic latency into the existing histogram under its
existing name), or explicitly accept the loss of publish-latency observability. Do not
carry this into implementation undecided.

**Call-site inventory, recount verified this revision (R2 finding 2 -- rev.2's 55 was
undercounted by 2):**

- **55 dual-publish call sites** (`WithGlobalTopicPublish()` present) across 49 files,
verified this revision via `/usr/bin/grep -c "WithGlobalTopicPublish()"` per file (not
reconstructed by hand -- rev.2's attempt to do so miscounted; raw counts below are the
source of truth):
  - **5 multi-call files**: `bin-registrar-manager/cmd/registrar-control/main.go` (3,
    lines 109/148/176); `bin-storage-manager/cmd/storage-control/main.go` (2, lines
    72/93); `bin-queue-manager/cmd/queue-control/main.go` (2, lines 68/513);
    `bin-customer-manager/cmd/customer-control/main.go` (2, lines 452/511);
    `bin-ai-manager/cmd/ai-control/main.go` (2, lines 425/473).
  - **44 single-call files**, one each: every remaining daemon (agent/ai/billing/call/
    campaign/conference/contact/conversation/customer/direct/email/flow/message/number/
    outdial/pipecat/queue/registrar/route/schedule/sentinel/storage/tag/talk/transcribe/
    tts/webchat -- 27 daemons) and every remaining `-control` binary that has one
    (agent/billing/call/campaign/conference/contact/conversation/direct/email/flow/
    message/number/outdial/pipecat/route/tag/transcribe -- 17 controls; schedule/
    sentinel/talk/tts/webchat have no `-control` DUAL-PUBLISH construction -- talk-control
    and tts-control DO construct a fanout-only `NotifyHandler`, listed separately below
    as items 4/5 of the 5 fanout-only sites, not double-counted here).
  - Arithmetic: (3×1) + (2×4) + (44×1) = 3 + 8 + 44 = **55**.
- **5 fanout-ONLY call sites** (no `WithGlobalTopicPublish()`), each individually
  dispositioned:
  1. `voip-asterisk-proxy/cmd/asterisk-proxy/main.go:107` (`QueueNameAsteriskEventAll`)
     -- **the one real fanout-only publisher that must keep working**, feeding
     `asterisk.all.event` (permanently retained, §1/§5). Sole caller of
     `PublishEventRaw` is `voip-asterisk-proxy/pkg/eventhandler/ari_handler.go:76`.
  2. `bin-transfer-manager/cmd/transfer-manager/main.go:137` -- dead wiring, zero
     publish-method calls anywhere in `pkg/transferhandler/*.go`.
  3. `bin-transfer-manager/cmd/transfer-control/main.go:66` -- same, CLI side.
  4. `bin-talk-manager/cmd/talk-control/main.go:57` -- pre-existing broken wiring
     (empty exchange name), independent of this ticket.
  5. `bin-tts-manager/cmd/tts-control/main.go:38` -- dead wiring, zero publish-method
     calls anywhere in `pkg/ttshandler/*.go`.
- **2 `NewNotifyHandlerForExistingExchange` call sites** (webhook-manager,
  webhook-control; VOIP-1258's scope-based topic mechanism). Both constructed WITHOUT
  `WithGlobalTopicPublish()`. Grepped the ENTIRE `bin-webhook-manager/pkg/webhookhandler`
  package (not just one file): the only method ever called on this instance anywhere is
  `PublishEventWithRoutingKey` (`routingkey.go:182`) -- confirmed genuinely orthogonal
  to `publishEvent()`'s fanout/topic switch at the runtime level. The narrower risk is
  at the constructor/option-surface level only: if §4.1 changes `initGlobalTopicExchange`
  to run unconditionally (dropping the `topicEnabled` gate) instead of preserving an
  explicit opt-out, this instance would harmlessly redeclare `bin-manager.event` a
  second time (idempotent). **Ruling needed in design (unchanged from rev.2, not yet
  resolved -- see §6)**: keep or strengthen the existing in-code warning at
  `notifyhandler/main.go:226-228` against ever adding `WithGlobalTopicPublish()` to this
  specific construction path, since a future maintainer doing so WOULD triple-publish.
  Note also (R2 finding 7): the comment at `notifyhandler/main.go:70-76` claims
  webhook-manager/webhook-control make "a fanout-bound `NewNotifyHandler` call in the
  same process" -- they do not (verified: only `NewNotifyHandlerForExistingExchange`
  anywhere in their `cmd/` trees); this stale comment should be corrected as part of
  §4.1's edit, not left as a landmine for the next reader.
- Total: 55 + 5 + 2 = **62**.

**Search methodology, corrected (R2 finding 6):** re-swept repo-wide with `/usr/bin/grep`
(NOT the IDE-integrated Grep tool, which R1 and R2's reviewers both independently found
silently drops at least one match --
`bin-pipecat-manager/cmd/pipecat-manager/main.go:125` -- from both repo-wide and
directory-scoped searches; **any future enumeration in this ticket's implementation
phase must use `/usr/bin/grep`, not the IDE tool, and should re-verify against this
document's counts rather than trusting either blindly**; also note (R4 finding 7) that
this whole 62-site count was built by grepping for `WithGlobalTopicPublish()`/
`NewNotifyHandler`/`NewNotifyHandlerForExistingExchange` call syntax, which cannot see
a call that hardcodes its exchange-name argument as a string literal instead of using
the `QueueName*Event` constant --
`bin-conversation-manager/cmd/conversation-control/main.go:58` does exactly this
(`"bin-manager.conversation-manager.event"` inline); it IS counted here because it also
happens to call `WithGlobalTopicPublish()`, but any FUTURE sweep keyed off the
constants alone (e.g. for the exchange-deletion runbook's precondition check) would
silently miss it -- worth a dedicated literal-string grep at implementation time) across
`**/cmd/**/*.go` and
`**/pkg/**/*.go` (not just `bin-*/cmd/*/main.go` -- the narrower glob is what produced
R1's CRITICAL miss). Confirmed additionally clean (construct no `NotifyHandler`
anywhere): `voip-kamailio-proxy`, `voip-rtpengine-proxy`, `bin-contact-manager/cmd/
case-control`, plus rev.2's original five (`bin-timeline-manager`, `bin-hook-manager`,
`bin-api-manager`, `bin-rag-manager`, `bin-trigger-sender`).

### 4. What "remove fanout dual publish" concretely requires (scope for design stage)

1. **Publish side** -- `bin-common-handler/pkg/notifyhandler`: preserve exactly one
   fanout-publishing construction path for asterisk-proxy (§3 item 1's leg); make every
   other instance topic-only. Design must choose the mechanism (invert `WithGlobalTopicPublish`
   into an opt-OUT used only by asterisk-proxy; a separate constructor reserved for it;
   or another split) -- **not yet decided, carried to design (§6)**. Failure semantics
   for the now-fanout-less instances (fatal vs. degrade on topic-declare failure) --
   **not yet decided, carried to design (§6)**. `queueEvent` parameter/`h.queueNotify`
   field's fate (drop from the topic-only constructor signature vs. keep unused for
   compatibility) -- **not yet decided, carried to design (§6)**. Also carried: 36
   `bin-common-handler/models/outline/queuename.go` constants match `QueueName*Event*`;
   excluding `QueueNameEvent` (the global topic target), `QueueNameAsteriskEventAll`
   (permanently retained, §5), and `QueueNameWebhookEventTopic` (VOIP-1258, §1) leaves
   the **33 per-service fanout constants**, which break down as the 28 real, deletable
   exchanges (§1) + 4 fully dead in Go source (`QueueNameAPIEvent`/`RagEvent`/
   `TimelineEvent`/`UserEvent` -- never had an exchange, zero non-definition references
   in any `.go` file, though `APIEvent`/`UserEvent` still appear in
   `docs/reference/rabbitmq-queues-reference.md:45,220` and `TimelineEvent` in the
   2026-03-15 centralize-clickhouse-writes plan AND design docs (two, not one) -- doc
   mentions only, no code) + 1 legacy (`QueueNameWebhookEvent` -- exchange already gone,
   constant still referenced by the two `QueueUnbind` sites in §1 plus **8** gomock
   expectations (`bin-agent-manager/pkg/subscribehandler/main_test.go:76,174` [2];
   `bin-timeline-manager/pkg/subscribehandler/run_topic_migration_test.go:49,109,148,191`
   [4]; `run_sentinel_test.go:49` [1]; `run_ordering_test.go:48` [1]), so deleting it
   touches those tests too); design should decide whether to delete the 4-dead and 1-legacy
   constants (and update the doc mentions and mock expectations) in this ticket or
   leave them (non-blocking either way, §6 item 8).
   `notify_process_time` observability (§3) -- **not yet
   decided, carried to design (§6)**. Dead delayed-publish cleanup -- **optional, not
   blocking, carried to design (§6)**.

2. **Consumer side** -- 20 services' `pkg/subscribehandler/main.go` + `cmd/*/main.go`
   wiring + `binding_golden_test.go` + `docs/architecture.md` (§2b, in full above).
   call-manager/timeline-manager additionally lose the sentinel defensive declare, but
   NOT the `asterisk.all.event` subscribe (permanent).

3. **`WithGlobalTopicPublish` / per-call-site disposition**: 62 total sites (§3) each
   resolved individually -- 55 lose the option (become the new topic-only default,
   whatever shape that takes), 1 (asterisk-proxy) keeps its fanout leg via whatever
   mechanism §4.1 lands on, 3 (transfer×2, tts-control) are dead wiring -- fix or leave,
   design's call, 1 (talk-control) is an independent pre-existing bug, 2
   (webhook-manager/-control) need the option-surface safeguard reviewed (§3).

4. **Per-service fanout exchange deletion** (ticket step 3, R1 finding 6 + R2 finding 1
   both incorporated): a ONE-TIME documented broker-admin cleanup (`rabbitmqadmin`/
   management API), matching VOIP-1406's stale-binding-runbook precedent.
   **Precondition, corrected and widened per R2**: EVERY publisher daemon+control binary
   (§3) AND every one of the 20 consumer services (§2b) that declares, binds, or
   subscribes to one of the 28 exchanges must be rebuilt/redeployed with that code
   removed, with a CONFIRMED restart-survival check per service (not a single
   point-in-time snapshot -- §1), before deletion runs. Add a post-deletion re-check
   (re-list exchanges/bindings after a defined soak window) to catch resurrection from
   an un-redeployed `-control` CLI invocation (these run ad hoc, not as long-running
   pods, so a stale binary run any time after deploy but before deletion is a live
   resurrection risk independent of the pod-restart risk).

5. **Docs**: `docs/reference/rabbitmq-queues-reference.md`'s entire dual-publish framing
   rewritten for a topic-only world (except the permanently-retained asterisk leg,
   documented as such, not as migration debris) + every touched service's
   `docs/architecture.md`/`docs/dependencies.md` (publish-side prose for the ~27
   publisher services AND events-section prose for the 20 consumer services -- union,
   not a simple sum, since most services are both) + `bin-common-handler/docs/
   architecture.md`'s `notifyhandler` section.

### 5. Scope boundary (explicit)

- OUT: `asterisk.all.event` exchange itself and its two consumer bindings (permanent).
- IN (was OUT in rev.1, corrected R1): asterisk-proxy's PUBLISH side -- the one fanout
  leg that must be deliberately preserved.
- OUT: VOIP-1258's `NewNotifyHandlerForExistingExchange` RUNTIME path (confirmed
  orthogonal) -- but its constructor/option-surface interaction stays IN SCOPE for
  review (§3).
- OUT: talk-control's pre-existing broken wiring (independent follow-up).
- OUT (pending design ruling, low priority): fixing transfer-manager's/transfer-control's
  and tts-control's dead `notifyHandler` construction -- cosmetic, all three confirmed
  zero live publish calls.
- IN: bin-common-handler/pkg/notifyhandler mechanism redesign (§4.1).
- IN (was ENTIRELY MISSING in rev.1/rev.2, R2 CRITICAL): all 20 consumer services'
  fanout-`QueueSubscribe`-removal (§2b/§4.2), sequenced BEFORE §4.4's exchange deletion,
  independent of but no less mandatory than the publish-side change.
- IN: per-call-site disposition for all 62 publish-side sites (§4.3).
- IN: broker-side one-time exchange-deletion runbook with a WIDENED precondition
  (publish-side AND consumer-side, all with restart-survival confirmation) and
  post-deletion resurrection check (§4.4).
- IN: the rabbitmq-queues-reference.md rewrite + per-service docs sync, both sides.

### 6. Open items -- design-stage rulings (NOT resolved by issue analysis; consolidated
handoff list, replacing rev.2's contradictory "all resolved" framing -- R2 finding 5)

Genuinely resolved by this analysis (facts established, no design ruling needed):
1. transfer-manager, transfer-control, and tts-control all construct a fanout-only
   `NotifyHandler` but make zero live publish-method calls anywhere in their respective
   handler packages -- confirmed dead wiring for all three (whether to actually DELETE
   the dead construction, as opposed to merely knowing it's dead, is deferred item 7
   below -- a fact being established here does not resolve what to do about it).
2. Third-party `notifyhandler` consumer check: searched every directory reachable from
   this session's shell access -- the full `monorepo` tree (all `**/cmd/**` and
   `**/pkg/**`, §3's methodology paragraph) and `monorepo-etc` (infra/ops only, no Go
   code). No third consumer found in either. This does not rule out a consumer in a
   repository this session cannot see (e.g. `voipbin-go`, `python-sdk`, `install/`,
   `sandbox` were not independently confirmed absent-of-reference, only not found within
   reach) -- stated as the search's actual scope, not as an absolute guarantee.

Deferred to the design stage, each requiring an explicit ruling before implementation:
1. **§4 item 1 mechanism**: how exactly to preserve asterisk-proxy's fanout leg while
   making every other publish-side instance topic-only (opt-out option / separate
   constructor / other).
2. **§4 item 1 failure semantics (publish side)**: fatal vs. degrade on topic-declare
   failure for the now-fanout-less publish instances.
3. **§4 item 1 constructor signature**: keep or drop `queueEvent`/`h.queueNotify` for
   the topic-only path.
4. **§3 `notify_process_time` observability**: fold into `publishTopicEvent` or
   knowingly drop the signal.
5. **§3 webhook-manager option-surface safeguard**: strengthen/keep the in-code warning
   against ever opting `NewNotifyHandlerForExistingExchange` into the (redesigned)
   topic-publish option, to prevent a future triple-publish.
6. **§2b consumer-side failure semantics**: once the fanout `QueueSubscribe` fallback is
   removed, should a topic-declare/bind failure at consumer startup become fatal
   (matching item 2's publish-side ruling, for consistency) rather than today's "stay
   on fanout" degrade -- there is no fanout left to degrade to, so SOME explicit
   decision replaces today's silent-degrade branch.
7. **Dead-wiring cleanup** (transfer-manager/-control, tts-control -- fact established
   in "resolved" item 1 above): fix now or leave -- non-blocking either way.
8. **Delayed-publish dead code cleanup** (§3) and **the 5 vestigial `QueueName*Event`
   constants' cleanup** (§4 item 1: 4 fully-dead + 1 legacy, per §1): clean up now or
   leave -- non-blocking either way, can be decided together since both are dead-code
   removal calls of the same weight.

### 7. Recommendation

Ticket is valid; the zero-fanout-binding precondition is met, evidenced with fresh raw
broker output covering exactly the 28 deleted exchanges plus the 5 excluded ones (with
reasons), and now explicitly scoped to what it does and does not prove (§1). Nine
review rounds each surfaced real gaps: R1 caught a missing real publisher
(asterisk-proxy) whose omission would have caused a full call-processing outage; R2
caught an entirely missing half of the ticket's scope (consumer-side fanout residue
that VOIP-1406's own design doc explicitly assigned here) whose omission would have
caused a platform-wide subscribe-handler boot failure on the next restart after
exchange deletion; R3 caught an unenumerated deletion set, a stale-binding self-healing
gap, and several internal-consistency defects from the rev.2->rev.3 rewrite; R4, R5, R6,
and R7 each caught a successive flaw in four DIFFERENT inference-based proofs of one
specific empirical claim (that `QueueUnbind` against a missing exchange is safe, and
`QueueBind`/`QueueSubscribe` against one is not) -- process liveness, a same-channel
argument that used a call which actually opens a separate channel, a consumer-count
argument confounded by RabbitMQ's automatic reconnect path, and a container-log-absence
argument undone by these services' own committed log-rotation config and an unhandled
conditional guard around the call being tested. Four inference attempts across four
rounds is what it took to learn the right lesson: **§1's claim is now settled with a
two-arm direct controlled experiment against the real production broker, including a
positive control** (isolated scratch queues, RabbitMQ 3.13.7; `queue.unbind` against a
guaranteed-nonexistent exchange raised no exception and left the channel alive;
`queue.bind` against a guaranteed-nonexistent exchange, run as a control in the same
harness, raised the exact 404 and killed the channel exactly as the codebase's own
incident comment describes) rather than a further inference from broker or process
state -- the standard this ticket should have held §1 to from the first round. R8
found the experiment itself sound but the write-up one-sided (only the unbind arm was
tested despite the claim covering both operations) plus two unstated bridging premises
(Python/Go client equivalence, scratch-queue-property immateriality); all three are
now addressed with the second experimental arm and explicit premise statements above.
R9 confirmed the two-arm experiment genuinely closes both halves of the claim and is
substantively airtight, and found only one false clause inside an otherwise-correct
bridging premise (amqp091-go's `QueueBind` does take a `noWait` parameter, contrary to
a prior "neither client exposes a nowait form" claim -- corrected here to the true and
sufficient justification: every production call site passes `noWait=false`) plus two
trivial citation/wording fixes, all applied above. All findings across all nine rounds
are now incorporated. Eight design-stage rulings are handed off explicitly in §6, none
of which block starting the design stage itself -- they ARE the design stage's job.
Proceed to design.
