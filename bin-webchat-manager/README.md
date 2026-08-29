# webchat-manager

## Events Published

Exchange: the global topic exchange `bin-manager.event` (see "Global topic exchange" below — as of VOIP-1407 the previous fanout exchange `bin-manager.webchat-manager.event` is no longer published to).

| Event type | Trigger | Topic routing key |
|-----------|---------|-------------------|
| `message.EventTypeMessageCreated` (`webchat_message_created`) | A webchat message was created, inbound or outbound (`pkg/messagehandler/create.go`) | `webchat-manager.webchat.<session-id>.message_created` |
| `session.EventTypeSessionEnded` (`webchat_session_ended`) | A session ended, so the visitor-side WS client can close cooperatively (`pkg/sessionhandler/db.go`) | `webchat-manager.webchat.<session-id>.session_ended` |

Both publish sites are guarded by `if h.notifyHandler != nil` — a handler constructed without a NotifyHandler publishes nothing at all.

### Global topic exchange (VOIP-1405 / VOIP-1407)

`cmd/webchat-manager` (the service's only binary — there is no `webchat-control`) constructs its NotifyHandler with `notifyhandler.WithGlobalTopicPublish()`, so every event is published to the global topic exchange `bin-manager.event` with the routing key `webchat-manager.<resource>.<subscription-id>.<action>`. **As of VOIP-1407, this is the sole publish path** — the previous per-service fanout exchange `bin-manager.webchat-manager.event` is no longer published to, and (per the operational runbook in `docs/reference/rabbitmq-queues-reference.md`) will eventually be deleted from the broker. No code in this service changed for VOIP-1407; the behavior change (dual publish → topic-only) comes entirely from `bin-common-handler/pkg/notifyhandler`'s shared library update. A topic publish failure now propagates to the caller as an error (previously it was swallowed silently).

Both event types collapse onto the **same** resource segment `webchat` (the key schema splits the event type on its first `_`), and both resolve to the **same** subscription address: `Session` by its own id, and `*message.Message` through its `eventtopic.SubscriptionIdentifier` override returning the parent `SessionID` rather than the message's own id (VOIP-1405 Category B — a message id first appears in the event that announces it, so it is not an address anyone can bind to in advance). The consequence is that a single binding pattern follows an entire visitor conversation:

```
webchat-manager.webchat.<session-id>.#
```

`models/session/routingkey_golden_test.go` pins both halves of that property. See the monorepo design docs `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md` and `docs/plans/2026-08-27-voip-1405-topic-publisher-rollout-design.md`.

This section lives in the README because this service has no `docs/architecture.md`/`docs/domain.md` yet — see the note under `## Deployment`. When [VOIP-1352](https://voipbin.atlassian.net/browse/VOIP-1352) generates the doc suite, this section moves to `docs/architecture.md` (events + cmd wiring) and `docs/domain.md` (the override rationale).

## Deployment

bin-webchat-manager deploys via Komodo (VOIP-1347 Tier 1 rollout, following the
VOIP-1342/bin-call-manager pilot pattern) instead of the older SSH +
`versions.lock` (`ssh-deploy.sh`) path.

- **Stack definition:** `bin-webchat-manager/komodo/docker-compose.yml` (git is
  the source of truth for structure; Komodo only executes it on request).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes the built
  image tag, then `.circleci/scripts/komodo-api-deploy.sh` pushes the file's
  content to Komodo and triggers a deploy, gated by the
  `bin-webchat-manager-deploy` job's poll/running checks.
- **Full design and cutover procedure:**
  [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md](../docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md)
  (in the monorepo root, not this service's own `docs/`).

Note: bin-webchat-manager does not have a `docs/operations.md` file (only
`docs/plans/`) — see [VOIP-1352](https://voipbin.atlassian.net/browse/VOIP-1352)
for generating the full architecture/operations/domain/dependencies doc
suite that `docs/reference/extractor.sh` produces for other services. Once
that ticket lands, this `## Deployment` section should move to the new
`docs/operations.md` and be removed from here.
