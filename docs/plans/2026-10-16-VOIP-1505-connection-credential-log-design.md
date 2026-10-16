# VOIP-1505 connection credential log design

## Goal
`bin-common-handler/pkg/rabbitmqhandler` must not write the broker connection URI (which carries the userinfo) or the userinfo echoed back in a dial error into any log field or message.

## Current behavior
- `Close()` logs `r.uri` as the `url` field (Info).
- `connect()` builds a logger with `url: r.uri`, so every line of the connect loop (Debug, and the retry Errorf) carries it.
- `err` of `amqp.Dial` is logged with `%v`. `url.Parse` masks the password only for some error kinds (bad port, bad escape) and leaves the user name; a userinfo it rejects (`net/url: invalid userinfo`, e.g. a space) is echoed unmasked. amqp091 rejects whitespace before parsing, other dial errors contain host and port only.
- bin-rag-manager `cmd/rag-manager` passes the PostgreSQL DSN to `sql.Open`/`Ping` and `migrate.New`; the same parse error reaches the startup error log and cobra's output with the password in clear text.

## Change
1. Remove the `url` field from `Close()` and `connect()`. `connect()` uses `WithField("func", "connect")`. No new address field is added.
2. Add `redactUserinfo(err)`: replaces `scheme://userinfo@` (up to the last `@` before the next `"`) with `scheme://***@` in the error text (regexp), used for the retry error line. The rest of the dial error text is unchanged.
3. Add the package variable `amqpDial = amqp.Dial` as a test seam, since the connect loop cannot be run without a broker.

No signature, queue, or metric change. The success path logs no host, as before.

4. bin-rag-manager: local `redactDSNError` (same regexp; the admission rule forbids exporting it from common-handler) applied to the `migrate.New` error, and `openPostgres(dsn)` (Open + Ping, errors redacted) replaces the inline code in `runService`.

## Tests (RED on the old code, GREEN after)
- `Test_Close_doesNotLogCredentials`: Close() output has neither the user, the password, nor the host.
- `Test_connect_doesNotLogCredentials`: stubbed dial error echoing the URI, no `url` field and no user or password in the output.
- `Test_redactUserinfo`: table test, including a userinfo with a space.
- rag-manager: `Test_redactDSNError`, `Test_openPostgres_doesNotLeakPassword`, `Test_runMigrations_doesNotLeakPassword` with a DSN that net/url rejects.

## Impact on consumers
Consumers reach this package through `sockhandler` and a local `replace` of `monorepo/bin-common-handler`; `vendor/` is not committed and the Dockerfiles regenerate it, so no consumer file changes. All services using `sockhandler` pick it up on their next build.

## Scope
This change is limited to the log and error paths of rabbitmqhandler and the bin-rag-manager startup connection and migration.
