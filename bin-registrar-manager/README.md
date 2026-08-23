# registrar-manager

Registrat manager for the voipbin project

# Usage
```
$ ./registrar-manager -h
Usage of ./registrar-manager:
  -dbDSNAst string
        database dsn for asterisk. (default "testid:testpassword@tcp(127.0.0.1:3306)/test")
  -dbDSNBin string
        database dsn for bin-manager. (default "testid:testpassword@tcp(127.0.0.1:3306)/test")
  -prom_endpoint string
        endpoint for prometheus metric collecting. (default "/metrics")
  -prom_listen_addr string
        endpoint for prometheus metric collecting. (default ":2112")
  -rabbit_addr string
        rabbitmq service address. (default "amqp://guest:guest@localhost:5672")
  -rabbit_exchange_delay string
        rabbitmq exchange name for delayed messaging. (default "bin-manager.delay")
  -rabbit_queue_listen string
        rabbitmq queue name for request listen (default "bin-manager.registrar-manager.request")
  -rabbit_queue_notify string
        rabbitmq queue name for event notify (default "bin-manager.registrar-manager.event")
  -redis_addr string
        redis address. (default "127.0.0.1:6379")
  -redis_db int
        redis database. (default 1)
  -redis_password string
        redis password
```

## Example
```
$ ./registrar-manager \
    -prom_endpoint /metrics \
    -prom_listen_addr :2112 \
    -dbDSNAstasterisk:b62160b0-ea4a-11ea-9d60-8b6c204cab46@tcp(10.126.80.5:3306)/asterisk \
    -dbDSNBin bin-manager:398e02d8-8aaa-11ea-b1f6-9b65a2a4f3a3@tcp(10.126.80.5:3306)/bin_manager \
    -rabbit_addr amqp://guest:guest@rabbitmq.voipbin.net:5672 \
    -rabbit_queue_listen bin-manager.registrar-manager.request \
    -rabbit_queue_notify bin-manager.registrar-manager.event \
    -rabbit_exchange_delay bin-manager.delay \
    -redis_addr 10.164.15.220:6379 \
    -redis_db 1
```

# Customer SIP domains (VOIP-1385)

Each customer gets one SIP domain, stored in `registrar_customer_domains`
(customer_id, domain_label, realm). Label generation is gated by
`DOMAIN_SHORT_LABEL_ENABLED` (env, default `false`):

- `false` (pre-cutover default): new customers get the legacy label, the
  customer uuid, e.g. `<uuid>.registrar.voipbin.net`
- `true` (flipped at cutover deploy): new customers get a random 4-char
  base36 label, e.g. `ab12.reg.voipbin.net`

The base suffix comes from `DOMAIN_NAME_EXTENSION`. The extension API's
`domain_name` always serves the full realm. Realm-to-customer lookup:
`GET /v1/customer_domains/realm/{realm}` (used by bin-call-manager).

## registrar-control domain migration commands

```
registrar-control domain-backfill [--execute]
registrar-control domain-migrate [--dry-run|--execute] [--customer-id <uuid>] --log <path>
registrar-control domain-migrate-rollback --log <path> [--execute]
```

`domain-backfill` seeds one row per existing customer with its current uuid
realm. `domain-migrate` re-provisions every extension to a short label under
the configured base, preserving extension id/password/direct hash, and writes
a JSON-lines rollback journal BEFORE mutating (log-before-execute).
`domain-migrate-rollback` replays the journal in reverse. Production runs are
executed by the operator, not CI.

# RabbitMQ queues
## Request Listen Queue
bin-manager.registrar-manager.request

## Event Notify Queue
bin-manager.registrar-manager.event


<!-- Updated dependencies: 2026-02-20 -->

# Deploy

`bin-registrar-manager-build` pushes the image, and a single `build-approval` gate covers
the test -> build pipeline. The CircleCI `bin-registrar-manager-deploy` job (direct SSH
deploy to bm-nyc-01) has been removed. Deploys to bm-nyc-01 are manual until
this service migrates to the Komodo-managed deploy path (see bin-call-manager
for the pattern).
