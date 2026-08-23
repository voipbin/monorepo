# VOIP-1385: Short customer SIP domain — implementation plan

Date: 2026-08-23
Design: docs/plans/2026-08-23-voip-1385-short-sip-domain-design.md (rev 2,
approved). This plan implements it. Deploy/runbook steps (DNS, cert, env
flip, batch execution) are NOT code and stay in the design's section 8.

## PR structure (one PR per repository — structural necessity)

| Repo | Branch | Content |
|---|---|---|
| monorepo | VOIP-1385-short-sip-domain | design doc + all Go/schema/docs work (tasks 1-6) |
| monorepo-javascript | VOIP-1385-short-sip-domain-frontend | task 7 (square-talk, square-admin) |
| monorepo-voip | VOIP-1385-short-sip-domain-kamailio | task 8 (kamailio dual-suffix + tls profile) |

Merging follows the design's deploy order; PR bodies state their gate.

## Task 1 — bin-dbscheme-manager: schema migration

- `alembic revision -m "add table registrar_customer_domains"` (generated id,
  never hand-picked), upgrade/downgrade per the design's DDL:
  customer_id BINARY(16) PK, domain_label VARCHAR(64) UNIQUE,
  realm VARCHAR(255) UNIQUE, tm_create/tm_update DATETIME(6). No tm_delete
  (deliberate, documented deviation).
- Do NOT run alembic upgrade (policy; CEO/CTO executes).

## Task 2 — bin-registrar-manager: domain model + handler + RPC

New package layout (follows existing per-entity conventions):
- `models/customerdomain/customerdomain.go`: CustomerDomain{CustomerID,
  DomainLabel, Realm, TMCreate, TMUpdate} with db/json tags. field.go IS
  created (Field constants for the fields-map Update convention);
  WebhookMessage is omitted (internal-only entity, no webhook).
- `pkg/dbhandler`: CustomerDomainCreate/Get/GetByRealm/Update/Delete +
  Redis cache (realm -> row) with invalidation on Update/Delete. Table const
  `registrar_customer_domains`.
- `pkg/customerdomainhandler`:
  - `EnsureByCustomerID(ctx, customerID)` — idempotent get-or-create.
    Label shape is gated by registrar-manager env
    `DOMAIN_SHORT_LABEL_ENABLED` (default false, added in code review
    round 3 to close the 1b->step-2 deploy window): flag false -> legacy
    label = customer uuid string (same shape as backfill); flag true ->
    4-char base36 lowercase via crypto/rand, retry on duplicate-key,
    reserved list {pstn, sip, echo, reg, www, api}. realm =
    label + "." + baseDomainExtension. The migration batch generates short
    labels regardless of the flag. Flag flips at deploy step 4.
  - `GetByCustomerID(ctx, customerID)` — lookup-only (no side effects).
  - `GetByRealm(ctx, realm)` — cache-backed, lookup-only.
  - `DeleteByCustomerID(ctx, customerID)`.
  - Lazy-create semantics (design §4): the EXTENSION-CREATE path
    (extensionhandler) calls `EnsureByCustomerID` (lazy-create fallback if
    the customer_created event was missed); READ paths (contacthandler
    endpoint reconstruction) call `GetByCustomerID` only — a contact lookup
    must never create a domain row as a side effect.
- Generator move (models/common must stay DB-free): `GenerateRealmExtension`
  and `GenerateEndpointExtension` are REPLACED by handler methods
  `customerdomainhandler.RealmGet(ctx, customerID)` and
  `EndpointGet(ctx, customerID, extension)` (ctx+error signatures).
  models/common keeps only `SetBaseDomainNames`/validation +
  `GenerateRealmTrunkDomain` (trunk unchanged). Call sites updated:
  extensionhandler/extension.go:55,58; contacthandler/contact.go:18,36.
- Event lifecycle: subscribehandler gains `customer_created` ->
  EnsureByCustomerID; `customer_deleted` cascade additionally calls
  DeleteByCustomerID (after existing trunk/extension cascade).
- Constructor wiring: NewSubscribeHandler, NewContactHandler, and
  NewExtensionHandler gain the customerdomainhandler dependency;
  cmd/registrar-manager/main.go (and registrar-control main) wire it.
- Listenhandler RPC (mirror the trunk domain_name route):
  `regV1CustomerDomainsRealm = /v1/customer_domains/realm/<url-escaped realm>$`
  (GET) -> customerdomainhandler.GetByRealm -> marshal domain model directly
  (style A per layering rules).
- Extension create path: `DomainName` field now stores the FULL realm
  (was bare uuid). SERVING RULE (read/webhook value): serve `Realm` when
  set; when Realm is NULL (legacy pre-Feb-2024 rows) serve computed
  `<customer_id>.registrar.<base>`. NEVER serve the raw stored DomainName
  column for legacy rows — existing rows hold the bare uuid there until the
  batch rewrites it, and the frontends consume this value verbatim from
  deploy step 2 onward (a bare uuid would break webphones). The stored
  column becomes authoritative only for rows written post-deploy/post-batch,
  and by then it equals Realm anyway.
- Stale comment cleanup: models/extension/webhook.go:19 + extension.go
  domain_name comments — drop "used by the kamailio's INVITE validation"
  from DomainName (Realm/Username/Password comments stay).

## Task 3 — bin-registrar-manager: migration batch (registrar-control)

- New registrar-control subcommands:
  - `domain-migrate --dry-run|--execute [--customer-id <uuid>] --log <path>`
  - `domain-migrate-rollback --log <path> --execute`
- Per customer (identity-preserving re-provision, NO extension
  Delete/Create handlers):
  1. EnsureByCustomerID with a NEW short label (or explicit update when a
     backfilled uuid-realm row exists: update label+realm).
  2. Per extension: build new realm/endpoint strings; dbhandler-level
     AstEndpointDelete/AstAORDelete/AstAuthDelete (old ids) +
     AstAORCreate/AstAuthCreate/AstEndpointCreate (new ids, same
     username/password); SIPAuthUpdate (realm); ExtensionUpdate in place
     (realm, domain_name, endpoint_id, aor_id, auth_id) — id, extension,
     username, password, direct_id, direct_hash UNTOUCHED; publish ONE
     extension_updated event; NO billing RPC.
  3. NEW dbhandler method AstContactDeleteByEndpoint(old endpoint) — row
     delete + contact-cache invalidation.
  4. Invalidate realm->customer cache entries (old + new).
  5. Append JSON line {customer_id, old_realm, new_realm, old_label,
     new_label, extensions: [...]} to the log (rollback source).
- Rollback command replays the log in reverse direction with the same
  identity-preserving mechanics.
- Backfill mode: `domain-backfill --execute` inserts rows for all existing
  customers with their CURRENT uuid realm (label = customer uuid string;
  varchar(64) fits 36 chars) — pure additive, run before the call-manager
  cutover deploy (design gate 1c).
- Interruption safety: per-extension operations idempotent (re-run
  converges); batch resumes by skipping customers whose row already carries
  a short realm.

## Task 4 — bin-common-handler: RPC client

- requesthandler: `RegistrarV1CustomerDomainGetByRealm(ctx, realm)
  (*rmcustomerdomain.CustomerDomain, error)` in a new
  registrar_customerdomain.go, mirroring registrar_trunks.go GetByDomainName
  (domain models preferred over DTOs per layering exception note). Mock
  regeneration via go generate.

## Task 5 — bin-call-manager: reverse-parse removal + .reg.-only classification

- projectconfig: `DomainRegistrarSuffix` becomes `".reg." + base` ONLY —
  no legacy constant (CEO/CTO decision: window downtime accepted;
  `.registrar.` domains classify as none/rejected). Kamailio keeps its
  `.registrar.` auth-gate regex (security, removed in cleanup step 6).
- start_incoming_domain_type_registrar.go: replace TrimSuffix+FromStringOrNil
  with `RegistrarV1CustomerDomainGetByRealm(ctx, domain)`; on error/miss ->
  log + reject the call (fail closed; mirror trunk file's error handling).
- ParseSIPURI (models/common/domain.go): reduce to pure textual split
  returning (extension, domain string, error) — NO uuid parsing;
  ari_contact.go caller then resolves customer via the same RPC and builds
  the filters; unknown realm -> log warn, skip refresh (explicit, not
  silent).
- Remove now-dead uuid-parsing imports/consts; keep suffix classification.

## Task 5b — service docs sync (root CLAUDE.md CRITICAL rule)

- bin-registrar-manager docs updated in the same commit: architecture.md
  (new listenhandler route + customer_created event), domain.md
  (CustomerDomain entity), operations.md if config-affecting.
- bin-call-manager docs: architecture.md/domain.md for the lookup change
  and the `.reg.`-only classification (legacy rejection pinned by test).
- Run `bash docs/reference/extractor.sh <service>` where applicable.

## Task 6 — docs/openapi (monorepo)

- bin-openapi-manager openapi.yaml:6515 domain_name description -> "The full
  SIP domain (realm) of the extension, e.g. ab12.reg.voipbin.net" (drop the
  stale kamailio clause).
- Gens regeneration ORDER (per bin-openapi-manager/CLAUDE.md): go generate
  in bin-openapi-manager FIRST, then in bin-api-manager (regenerates the
  openapi_redoc/api.html embed), then full verification workflow in
  bin-api-manager.
- RST updates + clean sphinx rebuild (`rm -rf build` first) +
  `git add -f bin-api-manager/docsdev/build/` (root .gitignore excludes
  build/)
  (extension_overview/tutorial/struct, quickstart_extension,
  trunk_overview_domain_name, direct_hash_overview, self_hosting_envvars):
  new domain format, note that TCP/TLS remains the fallback for oversized
  messages.

## Task 7 — monorepo-javascript (separate PR)

- square-talk: config/hostname.js registrar constructors deleted;
  config.js getSipRegistrarUrl/getSipUri take the full domain (from the
  extension API's domain_name) instead of customerId; webphone.js plumbs
  the extension's domain_name into both (WSS URL = wss://<domain_name>).
- square-admin: hostname.js same removal; extensions_detail.js Registration
  Address = `${extension}@${domain_name from API}`.
- Follow that repo's lint/build conventions; per-repo verification (build).

## Task 8 — monorepo-voip (separate PR)

- CERT gating choice: CERT_REG_FULLCHAIN_B64/CERT_REG_PRIVKEY_B64 form
  their OWN all-or-none pair gate (NOT merged into the existing 4-var
  registrar/sip gate), so image deploys do not fail before the new Komodo
  variables are registered; the trailing `unset CERT_*` line gains the new
  vars. Design deploy step 3 ships cert + config together either way.
- kamailio.cfg template: validate_authentication gate regex accepts both
  `.reg.BASE_DOMAIN` and `.registrar.BASE_DOMAIN`.
- tls.cfg template: add `[server:...]` SNI profile for `*.reg.BASE_DOMAIN`
  with cert path `/certs/reg.BASE_DOMAIN/...`; entrypoint.sh + komodo
  compose gain CERT_REG_FULLCHAIN_B64/CERT_REG_PRIVKEY_B64 handling
  (mirror the existing CERT_REGISTRAR_* block, all-or-none gating included);
  docker-compose local bind-mount path unchanged.
- kamailio -c validation with rendered config (same harness as the pike PR).

## Verification (per monorepo policy, per touched service)

- Go services: go mod tidy && go mod vendor && go generate ./... &&
  go test ./... && golangci-lint run -v --timeout 5m — for
  bin-registrar-manager, bin-call-manager, bin-common-handler,
  bin-openapi-manager, bin-api-manager (openapi-manager first, then
  api-manager — gens ordering), bin-dbscheme-manager (as applicable).
- Unit tests per the design's section 10 item 1 (aggressive coverage),
  including the re-provision identity assertions and idempotent re-run.
- RST: clean rebuild, build output committed.
- Kamailio: kamailio -c on 6.0.1 image with rendered config.
- Frontends: repo build + lint.
- Code review loop: minimum 3 rounds, 2 consecutive approvals, across all
  three repos' diffs.
- Sandbox E2E, byte capture, and TLS/WSS cert validation (design section
  10 items 2-5) are POST-merge/deploy verification per the design's deploy
  order — not gating the PRs, stated in PR bodies.

## Explicitly out of scope (this PR set)

- DNS record creation, cert issuance, Komodo variable registration, env
  flip, production batch execution (runbook steps, CEO/CTO-driven).
- Legacy suffix cleanup (design step 6).
- Vanity labels.
- SDK regeneration (voipbin-go, python-sdk, vn CLI, MCP) + release notes:
  separate repos, description-only change (plain string type, non-breaking);
  executed post-merge alongside the deploy, tracked on VOIP-1385.
- monorepo-monitoring config updates (api-validator/sip-validator uuid
  defaults) — runbook-adjacent, updated alongside deploy step 5 per design
  section 9; tracked on VOIP-1385.
