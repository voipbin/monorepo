# VOIP-1685 shared secretcrypto / confighandler and email_configs (PR1)

Scope of this PR: `bin-common-handler/pkg/secretcrypto`, `bin-common-handler/pkg/confighandler`,
compatibility tests, and the first adopter `bin-email-manager` (tables `email_configs` and
`email_config_histories`, a key-supplier refactor of the email handler, and `email-control config`).
The number-manager cipher replacement is a separate PR (PR2) and is not part of this one.

Admission note: the root rule asks for three consuming services before a package enters
`bin-common-handler`. After this PR and PR2 there are three planned consumers (email-manager, number-manager,
and ai-manager later), but this PR has one consumer (email-manager). The packages are placed in
`bin-common-handler` from the start for that reason; this is stated as a deviation from the admission rule,
not as a general pattern.

## 1. Ticket review (issue analysis)

Base: `origin/main` 39fd641d0 (after `git fetch`). Result: valid.

Claims checked against the code:

| Claim | Result | Evidence |
|---|---|---|
| bin-common-handler, bin-number-manager, bin-email-manager unchanged since 8e54831dc | confirmed | `git diff --stat 8e54831dc origin/main -- <three dirs>` is empty |
| Handler constructors take plain key strings | confirmed | `cmd/email-manager/main.go:155`, `cmd/email-control/main.go:70`, `pkg/emailhandler/main.go:66-90` |
| The three env reads of email-manager are `SENDGRID_API_KEY`, `MAILGUN_API_KEY`, `MAILGUN_WEBHOOK_SIGNING_KEY` | confirmed, and these are the only ones | `internal/config/config.go:65-67`; `grep Getenv` finds none outside viper |
| Mailgun domain is not a config value | it is the constant `defaultMailgunDomain`; not a registry key | `pkg/emailhandler/engine_mailgun.go:20` |
| Empty key behavior today | Mailgun and SendGrid clients are built with the empty key and the send fails, then the next provider is tried; an empty signing key rejects every Mailgun webhook and logs one warning at start | `engine_mailgun.go:34-48`, `send.go:20-47`, `hook.go:168-171`, `cmd/email-manager/main.go:151` |
| Number-manager cipher: `nonce||ct||tag`, AAD = provider name, 1024 byte limit, version in a column | confirmed | `bin-number-manager/pkg/secretcrypto/main.go:21` (`MaxPlaintextBytes`), `:98-125`, `:128-150` |
| ai-manager MCP cipher: ct and nonce separate, AAD nil, no length limit | confirmed | `bin-ai-manager/pkg/mcpserverhandler/secret.go:104-120,123-144` |
| ai-manager engine base key: `version(2B BE)||nonce(12)||ct||tag`, AAD nil | confirmed | `bin-ai-manager/pkg/aihandler/engine_base_key.go:13,27-70` |
| Error text of the ai-manager key parser quotes the key entry (`%q`) | confirmed | `secret.go:50,56` |
| Migration infrastructure fits new tables | confirmed: `migrations/` has only `000001_baseline`; `email_schema_migrations` version table; `migrationhandler.Run` runs before cache and RabbitMQ | `cmd/email-manager/main.go:121-126`, `migrations/embed.go` |
| Existing precedent for a platform-wide provider table | `number_providers`, resolution db > env, undecryptable never falls back | `bin-number-manager/CLAUDE.md` "Provider Credentials" |

Not refuted: all stated premises. Wrong claims: none.

Duplicates and prior work: `gh pr list --state all --search "configs"` and `git log -- bin-common-handler/pkg`
show no `confighandler` or shared `secretcrypto`. VOIP-1681 (Mailgun signature verification) is already in main and
is the reason `mailgunWebhookSigningKey` exists in `emailHandler`.

Impact and hidden assumptions found (each one is resolved in section 2):

1. New read path. Every provider attempt and every Mailgun webhook reads the database once. A read error is an
   error (never "no row", never an env fallback), so that provider attempt fails and the next provider is tried;
   a webhook with a read error is rejected. Latency is one primary key read; the context of the caller bounds it.
   This is the only new failure mode, and it exists only when the database is unreachable. With no row the
   observable behavior is identical (decision R1).
2. The Mailgun and SendGrid clients are built once at construction. A rotated key needs a client built at use
   time. Existing engine tests inject a client (`engine_mailgun_test.go:91`, `engine_sendgrid_test.go:70`,
   `error_paths_test.go:330-334`), so an injection point must stay (decision E2).
3. Test coupling: about 40 references in tests to `emailHandler` fields `engineMailgun`, `engineSendgrid`,
   `mailgunWebhookSigningKey` and `NewEmailHandler` (`main_test.go:40`). The refactor keeps the field names and
   changes only the key type (string to `SecretFunc`); the existing tests are edited mechanically
   (a `staticSecret` helper), not rewritten. "Same behavior with no row" is shown by the existing tests passing
   after only those mechanical edits, plus new regression tests (decision E1, test plan T4).
4. `config.Config` accepts the three values as command line flags too (viper). The env source of the resolver
   is the viper value (flag or env). Otherwise a flag-only deployment would change silently (decision E3).
5. The master key env is new (decision M1 to M3). `bin-email-manager/komodo/docker-compose.yml` cannot name a
   Komodo variable that may not exist yet. `k8s/deployment.yml` carries `SENDGRID_API_KEY` and
   `MAILGUN_API_KEY` (lines 64-73) but not the signing key, and is the legacy manifest; the master key is not
   added there.
6. `email-control` today builds Redis and RabbitMQ for every email command (`initHandler`). The new `config`
   commands need only the database and the master key and use their own initializer (decision C1).
   Secret input must not appear on the command line or in logs (decision C2).
7. Three ciphertext layouts exist. PR1 supports all three read and write paths through the shared library
   (the compatibility tests cover all three). AAD is optional, the plaintext limit is an option (default
   unlimited, number-manager passes 1024 in PR2), the 2 byte version limit (65535) applies only to the packed
   layout. The key parser error never quotes the key entry (requirement, tested).
8. Dependencies and blast radius: both packages use only the standard library plus `gofrs/uuid` and
   `go.uber.org/mock` (test), all already in `bin-common-handler/go.mod`. They import nothing from number-manager;
   sentinel errors live in the library and PR2 maps them. A new package that no service imports does not change
   any service build, but `bin-common-handler` is a `replace` target of every service, so all services are
   built once (`go build ./...`) to prove it (test plan T6).
9. Rotation: rows carry their own `key_version`, so replicas holding older and newer key lists decrypt each row
   with the version in the row as long as every replica has every version (same as today's cipher users).

10. A row exists but cannot be decrypted (the master key is empty, or the version of the row is not in the key
    list, or the ciphertext is damaged). The precedent (number-manager) never falls back to env here. The
    resolver reports source `db_undecryptable` and returns an error; the provider is unusable until the row is
    cleared or the key list is restored (decision R2). The reverse case is the premise of "no row means the
    same as today": no row and no master key must start and run exactly as now. The key parse error, an empty
    master key, and a missing table never stop the service from starting (decision M2).
11. Schema and docs obligations: two new tables (add-only, no destructive operation). They must be listed in
    `bin-dbscheme-manager/docs/schema-ownership.md`; the hardened schema verification applies (impact analysis in
    the PR body, upgrade/downgrade/upgrade on a scratch database with both MariaDB and MySQL). Service docs to
    update: `bin-email-manager/CLAUDE.md`, `docs/operations.md` (config table), `docs/domain.md`,
    `docs/dependencies.md`, and `bin-common-handler/CLAUDE.md` plus `docs/architecture.md`.

Unverified: runtime behavior of Komodo with an undefined `[[VARIABLE]]` reference, and the replica count
(not checked from the repository). Both are avoided by decision M3 (the compose file is not changed in this PR).

## 2. Design (direction and decisions)

Recommended approach: one table pair per service, a typed registry in code, a shared resolver and cipher in
`bin-common-handler`. Alternatives rejected: (a) keep per-service cipher copies (the two layouts already
diverge and the number-manager key parser had a different error policy); (b) generalize `number_providers`
(one secret per row, no history, cannot hold the two Mailgun secrets independently); (c) a central config
service (out of scope). Not built: caching, change events, `reencrypt`, customer scoped keys, a management
API, auto import of env values into the database, a generic on/off flag store.

### 2.1 Resolver behavior

- R1 (no row = today, with a healthy database). Resolution order is db, then env, then default. With no row the value is the
  viper value of the env name exactly as now; an empty value is "not configured" and is returned as an
  empty secret with source `none` and no error. Callers proceed exactly as today: the engines build a client
  with the empty key and the send fails (failover continues), the webhook is rejected by the existing
  "signing key is not configured" check. A store read error (not "not found") is returned as an error and is
  never turned into "no row" or an env value. For a send this fails that provider attempt (logged at error
  level like the existing failures, and the next provider is tried). For a webhook the existing verification
  error path is used, so the response code and log level are the same as for the existing "not configured"
  rejection. The database call uses the caller context. A missing table is the same as a read error. Rejecting the webhook
  on a read error is deliberate: an unverified request must not be accepted. Whether the provider retries the
  rejected delivery was not verified through the hook-manager path. "Identical to today" therefore holds when the database is
  reachable and has no row; an unreachable database is the one new failure mode. During such an outage both providers fail even when an env
  key exists, because the key is read from the database first (the status update after a send needs the database
  as well).
- R2 (row but undecryptable). A secret row whose key version is not in the key list, whose key list is
  empty, or whose ciphertext fails authentication yields source `db_undecryptable` and an error. There is no
  fallback to env. A row whose `value_type` does not match the registry yields `db_invalid`. Both are shown
  by `Describe`/`email-control config list`.
- R3 (atomic groups). A group marked `Atomic` has one anchor key (its single Secret key, else its single
  Select key; anything else is a registration error). If the anchor has a row, every key of the group is read
  from the database or the default only; otherwise from env or default only. The three email keys are
  deliberately independent (no group): the Mailgun API key (sending) and the webhook signing key (receiving)
  have different purposes and are rotated separately, unlike the credential set of one provider account in
  `number_providers`. Atomic groups are exercised by library tests only.
- R4 (registry guards, `NewRegistry` returns an error). Duplicate or malformed name (`^[a-z0-9][a-z0-9._]{0,127}$`);
  an `Env` that is a reserved bootstrap name (a list in `confighandler`: `DATABASE_DSN`, `DATABASE_DSN_ASTERISK`,
  `DATABASE_DSN_BIN`, `POSTGRESQL_DSN`, `CLICKHOUSE_ADDRESS`, `CLICKHOUSE_DATABASE`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`,
  `REDIS_DATABASE`, `REDIS_DB`, `REDIS_PASSWORD`, `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `LOG_LEVEL`) or ends
  with `_ENCRYPTION_KEYS`; a default of the wrong type; `Min>Max`; a `bool` type unless the key is `KindSelect` and a member of an `Atomic` group (a provider selection such as
  "enabled" next to its credential). A standalone on/off key, `KindParam` + bool, `KindSecret` + bool and
  `KindSelect` + bool outside a group are all rejected, each pinned by a test; a Secret with a default or a non-string type; an Enum without values;
  an invalid Atomic group.
- R5 (secrets). `confighandler.Secret` prints as `[REDACTED]` for every fmt verb, JSON and text marshal;
  `Reveal()` is the only accessor. History rows of secret or `Sensitive` keys never carry value columns
  (no code path writes them); only action, actor, versions and time are recorded. The cipher binds a
  ciphertext to its row with the AAD `email_configs 0x00 <config_key>`.
- R5b (definitions). `env_shadowed` is true when the source is `db` (or `db_undecryptable`) and the env value is
  non-empty. `Describe` lists every registry key, with or without a row, never a value. `Clear` deletes the row,
  so the key resolves from env or default again; when the env value was already removed it becomes `none`
  (stated in `docs/operations.md` and in the `clear` help text).
- R6 (writes). `Set`/`SetSecret`/`Clear` validate against the registry, use the optimistic lock (`revision`)
  and write the row and the history in the same transaction (store contract). The lock keeps the row, its revision and its history consistent (no torn insert, no lost history row); it does
  not preserve what a person read earlier, because `Set`, `SetSecret` and `Clear` are overwrites. A retry means:
  read the row again, take its current revision (or insert if it is still missing), and write again, at most
  three attempts, then `ErrConflict`. Concurrent writers are therefore last-writer-wins and each write has its own
  history row. A duplicate primary key on insert (MySQL error 1062; the table has no other unique key) is reported by the
  store as `ErrConflict` and is retried the same way.
  `Clear` on a missing row is a no-op without history. `SetSecret` fails with `ErrKeyNotConfigured` when
  there is no master key, and `ErrTooLong` over the key's `MaxLen`.

The bool rule is narrower than the earlier notes (a bool is a provider selection inside an atomic group only).
The timestamp range check of the webhook stays after the HMAC (only a genuine request learns the reason), so a
well formed unsigned request costs one key read; this is accepted.

### 2.2 secretcrypto

Same format as today: AES-256-GCM, 12 byte nonce, key list `<version>:<base64 32B>[,...]`, highest version
for writes, row version for reads. Differences from the two copies: AAD is a `[]byte` argument (nil means the
legacy no-AAD mode, which is byte-identical to `gcm.Seal(nil, n, pt, nil)`); `WithMaxPlaintext`; sentinel
errors without service types; the parser error never contains key text, version text, or base64 error text.
Adapters (no data migration): `JoinSplit`/`SplitBlob` (ai MCP columns), `PackVersioned`/`UnpackVersioned`
(engine base key, version 1..65535). Error text of `Decrypt` is constant (`ErrDecrypt`), never the plaintext.

### 2.3 email-manager

- E1 (supplier functions). `emailhandler.SecretFunc func(ctx context.Context) (string, error)`.
  `NewEmailHandler(db, reqHandler, notifyHandler, sendgridKey, mailgunKey, mailgunSigningKey SecretFunc)`.
  The struct fields keep their names (`engineMailgun`, `engineSendgrid`, `mailgunWebhookSigningKey`); only the
  type of `mailgunWebhookSigningKey` changes. `verifyMailgunSignature` gets a `ctx` parameter (its only caller
  is `hookMailgun`, which has one) and checks the structure of the signature first (nil, missing fields, timestamp,
  hex); the key is resolved immediately before the HMAC, so malformed requests cause no database read. The
  empty-key rejection stays before the HMAC. Only the error text of requests that are rejected either way can
  change (the existing tests assert only that an error is returned). Existing tests change through a `staticSecret(s)` helper for the key
  fields and the extra argument; the behavior assertions are not edited.
- E2 (clients at use time). `NewEngineMailgun(reqHandler, key SecretFunc)` and `NewEngineSendgrid(...)` keep
  the existing `client` field as an optional override: when it is non-nil `Send` uses it (this is what the
  existing engine tests inject, with their test base URL and transport, so those literals compile unchanged);
  when it is nil `Send` resolves the key and builds the provider client for that call. A new test proves the
  resolved key reaches the client constructor and that a rotated key is used by the next call. A supplier
  error is returned from `Send` as an error (the failover continues).
- E3 (env source). `internal/config` gets `EnvValue(name)` over the viper-loaded `Config` for the three names
  and the new master key field `ConfigEncryptionKeys` (viper key `email_config_encryption_keys`, env
  `EMAIL_CONFIG_ENCRYPTION_KEYS`; the two names match because `AutomaticEnv` looks up the upper cased viper key first,
  so an unprefixed `CONFIG_ENCRYPTION_KEYS` must never be read; a test sets it and expects it to be ignored). Unknown names return "not set". The master key is bound to the environment
  only (`BindEnv`, no command line flag), so it never appears in a process list; the existing API key flags are
  not changed.
- E4 (registry). `internal/configkeys`: table names, key names, `NewRegistry()`:
  `sendgrid.api_key`, `mailgun.api_key`, `mailgun.webhook_signing_key`, all `KindSecret`, `MaxLen` 512,
  env names unchanged, no defaults. A golden test pins the name to kind mapping.
- E5 (store). `dbhandler` implements `confighandler.Store` on the same `*sql.DB` (squirrel, transaction for
  row plus history, no cache, optimistic update `... WHERE config_key=? AND revision=?`, 0 rows is `ErrConflict`).
  `DBHandler` gets one method `ConfigStore()`.
- E6 (startup). After the migrations, the daemon builds the cipher from `EMAIL_CONFIG_ENCRYPTION_KEYS`; a
  parse error is logged (without key text) and the daemon continues with no key (M2). It logs, once, the
  source and `env_shadowed` of the three keys (`Describe`); a store error there is a warning. The old
  "signing key not configured" warning becomes: warn when the resolved source is `none`.

### 2.4 Master key placement

- M1: one new env `EMAIL_CONFIG_ENCRYPTION_KEYS` for email-manager and email-control (same list, so the
  CLI can write what the daemon reads). Existing key lists are not reused (rotation scope stays separate).
- M2: no key, an unparsable key, or a missing table never stops the daemon. Only secret writes are refused,
  and a secret row cannot be read.
- M3: `komodo/docker-compose.yml` is not changed in this PR, because naming a Komodo variable that does not
  exist yet may break the stack deploy, and that behavior was not checked. Before the first DB write, the operator
  creates the variable and the compose line is added in a follow-up change. This is called out in the PR description.

### 2.5 email-control

- C1: `email-control config list|set|clear` uses its own initializer: database plus cipher only (no Redis,
  no RabbitMQ). Flags are read with `cmd.Flags().GetString`, not through the global viper (whose `AutomaticEnv`
  could fill `key` or `value` from the environment). The existing `get`, `list` and `delete` commands build the
  email handler with the same resolver-backed suppliers. The help text says that the daemon applies the migration
  first (or `migrate up` on a local database). `set` and `clear` require `--actor-id` (valid, non-nil UUID) and `--key`.
- C2: the value of a secret key is read from standard input only (at most `MaxLen+3` bytes are read; one trailing `\n` or `\r\n` is removed; then the length in bytes must be
  1..`MaxLen`, else `ErrTooLong` or an empty error; a value that contains a NUL, CR or LF byte is rejected). There is no value flag for secret keys; non-secret keys take `--value`. Nothing
  secret is logged or printed. `list` prints key, kind, source, `env_shadowed`, `has_secret`, key version,
  revision, updated_by and update time, never a value. History source is `cli`.

### 2.6 Compatibility and rollout

The two tables are added to both tables of `bin-dbscheme-manager/docs/schema-ownership.md` (the per service table and
the per table list) in this PR.

Add-only tables (`000002_email_configs_create_table_all`, `000003_email_config_histories_create_table_all`,
one DDL statement per file), no destructive operation. Old code ignores the tables; new code reads them only after the
migration ran. Rolling back the code leaves empty tables. After a DB value replaces an env value, the env value
should stay valid until the rollback window closes (operator procedure, documented in `docs/operations.md`).

## 3. Design (interfaces, implementation plan, verification)

### 3.1 `bin-common-handler/pkg/secretcrypto`

```go
var (
    ErrKeyNotConfigured = errors.New("secret encryption key is not configured")
    ErrTooLong          = errors.New("secret is longer than the limit")
    ErrDecrypt          = errors.New("could not decrypt the secret")
    ErrInvalidKeySet    = errors.New("invalid encryption key list")
    ErrMalformed        = errors.New("malformed ciphertext layout")
)
type Option func(*Crypto)
func WithMaxPlaintext(n int) Option // n <= 0: unlimited
func New(raw string, opts ...Option) (*Crypto, error) // "" is valid (no key); errors wrap ErrInvalidKeySet with the entry INDEX only
func (c *Crypto) HasKey() bool            // nil safe
func (c *Crypto) CurrentVersion() int
func (c *Crypto) Versions() []int         // ascending
func (c *Crypto) Encrypt(plaintext string, aad []byte) (blob []byte, version int, err error) // nonce||ct||tag; no key: ErrKeyNotConfigured; over the limit: ErrTooLong; the empty plaintext is allowed
func (c *Crypto) Decrypt(blob []byte, version int, aad []byte) (string, error) // nil receiver or no key at all: ErrKeyNotConfigured; unknown version, short blob, authentication failure: ErrDecrypt
func AAD(table, key string) []byte        // table 0x00 key
func AADLiteral(s string) []byte
func JoinSplit(ct, nonce []byte) ([]byte, error)  // both empty: (nil, nil); only one empty, nonce != 12 bytes, ct < 16 bytes: ErrMalformed
func SplitBlob(blob []byte) (ct, nonce []byte, err error) // blob < 28 bytes: ErrMalformed
func PackVersioned(version int, blob []byte) ([]byte, error) // version 1..65535, blob >= 28 bytes
func UnpackVersioned(packed []byte) (int, []byte, error)     // version 0 or short input: ErrMalformed
```

Internals: one `newGCM(key)`; nonce from `crypto/rand`; `Seal(nonce, nonce, pt, aad)`; key slices are copied at parse;
no function formats the key, the entry, the plaintext or the blob into an error. Error values are wrapped with `%w` and a constant text.

### 3.2 `bin-common-handler/pkg/confighandler`

Files: `main.go` (doc, types), `registry.go`, `secret.go`, `store.go`, `resolve.go`, `handler.go`. Signatures:

```go
type Kind string   // KindParam, KindSecret, KindSelect
type Type string   // TypeString, TypeInt, TypeBool, TypeDuration, TypeEnum
type Source string // SourceDB "db", SourceEnv "env", SourceDefault "default", SourceNone "none", SourceDBUndecryptable "db_undecryptable", SourceDBInvalid "db_invalid"

type Key struct {
    Name string; Kind Kind; Type Type; Env string; Default any   // Default: string, int64, bool, time.Duration by Type
    Min, Max *int64        // int: value; duration: nanoseconds
    MaxLen int             // string and secret: bytes
    Enum []string; Group string; Atomic bool; Sensitive bool
}
func NewRegistry(keys []Key) (*Registry, error)
func (r *Registry) Keys() []Key             // declaration order
func (r *Registry) Lookup(name string) (Key, bool)

// Registration rules (each returns an error naming the key and the rule, never a value):
// name format and uniqueness; Env not reserved and not *_ENCRYPTION_KEYS; Default type matches Type;
// Min<=Max; Enum non-empty for TypeEnum and Default in Enum; KindSecret: TypeString, no Default,
// 1 <= MaxLen <= 2020 (the column holds 2048 bytes, minus nonce and tag); KindParam: TypeString needs MaxLen;
// Sensitive on a KindSecret key is redundant and allowed (a secret is always sensitive); an empty Env skips the env step;
// TypeBool only for KindSelect inside an Atomic group; Atomic requires Group != "" and every key of a group
// has the same Atomic value; an Atomic group has exactly one anchor (its single Secret key, else its single Select key).

type Secret struct{ /* unexported */ }
func NewSecret(v string) Secret
func (s Secret) Reveal() string
// String, GoString, Format (all verbs), MarshalJSON, MarshalText render "[REDACTED]"; the zero Secret renders the same.

// each is created with errors.New in the implementation
var (ErrNotFound, ErrConflict, ErrUnknownKey, ErrWrongKind, ErrUndecryptable, ErrInvalid error)
var ErrKeyNotConfigured = secretcrypto.ErrKeyNotConfigured // re-exported, same value
var ErrTooLong = secretcrypto.ErrTooLong

type Row struct {
    Key, ValueType string; ValueJSON []byte; SecretCiphertext []byte; KeyVersion int
    Revision uint32; UpdatedBy *uuid.UUID; TMCreate, TMUpdate time.Time
}
type History struct {
    ID uuid.UUID; Key, Action string; IsSecret bool; OldValueJSON, NewValueJSON []byte
    OldKeyVersion, NewKeyVersion int; ActorID uuid.UUID; Source string; TMCreate time.Time
}
type Store interface {
    Get(ctx context.Context, key string) (*Row, error)                                     // ErrNotFound when absent
    List(ctx context.Context) ([]*Row, error)
    Insert(ctx context.Context, row *Row, hist *History) error                              // duplicate key: ErrConflict
    Update(ctx context.Context, row *Row, expectRevision uint32, hist *History) error       // sets revision = expectRevision+1; 0 rows: ErrConflict
    Delete(ctx context.Context, key string, expectRevision uint32, hist *History) error     // 0 rows: ErrConflict
}
// Row has no db tags: the store converts explicitly. Every store method that writes does row and history in one transaction.

type EnvLookup func(name string) (string, bool) // a missing or empty value means "not set"

type Resolved struct {
    Key string; Kind Kind; Type Type; Source Source; HasSecret, EnvShadowed bool
    KeyVersion int; Revision uint32; UpdatedBy *uuid.UUID; TMUpdate *time.Time
}
type Handler interface {
    String(ctx context.Context, key string) (string, Resolved, error)
    Int(ctx context.Context, key string) (int64, Resolved, error)
    Bool(ctx context.Context, key string) (bool, Resolved, error)
    Duration(ctx context.Context, key string) (time.Duration, Resolved, error)
    Secret(ctx context.Context, key string) (Secret, Resolved, error)
    Set(ctx context.Context, actor uuid.UUID, key, raw string) (*Resolved, error)
    SetSecret(ctx context.Context, actor uuid.UUID, key, secret string) (*Resolved, error)
    Clear(ctx context.Context, actor uuid.UUID, key string) (*Resolved, error)
    Describe(ctx context.Context) ([]*Resolved, error)   // registry order, rows of unregistered keys are ignored
    Registry() *Registry
}
func NewHandler(reg *Registry, store Store, c *secretcrypto.Crypto, env EnvLookup, table string, opts ...Option) Handler
func WithSource(s string) Option // history source, default "api"
```

`ErrUndecryptable` wraps the underlying `secretcrypto` error (`errors.Is` works for both `ErrKeyNotConfigured` and `ErrDecrypt`), so a row that cannot be decrypted
because there is no key at all and one whose version is missing both satisfy `errors.Is(err, ErrUndecryptable)`.

Type to row mapping: the stored `value_type` is `secret` for `KindSecret` and the registry `Type` string otherwise (`string`, `int`, `bool`,
`duration`, `enum`); any other value in a row is `db_invalid`.

Write misuse: `Set` on a `KindSecret` key and `SetSecret` on any other kind return `ErrWrongKind`; `SetSecret` with an empty value
returns `ErrInvalid`; `actor == uuid.Nil` returns `ErrInvalid` for all three writers (the history column is NOT NULL and the nil
actor is reserved). The daemon's and the CLI's cipher are both built (in `emailconfig.New`) with `WithMaxPlaintext(2020)` as a second guard besides the registry `MaxLen`.

Mocks: `Store` and `Handler` get `//go:generate mockgen` files in the package (repository convention); in-package tests also use a
small in-memory fake store whose contract (conflict on stale revision, not found, duplicate) is checked by one shared test function that
runs against the fake and, in the email module, against the sqlite store, so both implement the same contract.

Behavior details: `Resolved` is filled even when the read returns an error (so callers and `Describe` can show `db_undecryptable`
and `db_invalid`). A row of the wrong `value_type`, an unparsable or out of range `value_json`: `SourceDBInvalid`, `ErrInvalid`. An
env value that does not parse or violates its range: `ErrInvalid` with `SourceEnv`. A typed read of a `KindSecret` key and `Secret()`
of any other kind: `ErrWrongKind`. `Set` stores the JSON of the typed value (duration in its string form). Unexported fields
`now func() time.Time` and `newID func() uuid.UUID` are set by in-package tests (deterministic history). A retry builds a new history
row from the freshly read row (old key version included). `Describe` uses `Store.List` once; typed reads use `Store.Get` (and one
more `Get` for the anchor of an atomic group). History for `Set`: `IsSecret` is `Kind==Secret || Sensitive`; then `OldValueJSON` and
`NewValueJSON` stay nil by construction (they are assigned only inside the non-secret branch).

### 3.3 `bin-email-manager`

- Migrations: `000002_email_configs_create_table_all.{up,down}.sql` and `000003_email_config_histories_create_table_all.{up,down}.sql`
  (DDL in 3.6, `CREATE TABLE IF NOT EXISTS`). The first line of the `000002` down file is exactly `-- DOWN-LOSSY: dropping the table loses every stored setting row, including encrypted secrets.` and of the `000003` down file exactly `-- DOWN-LOSSY: dropping the table loses every setting change history row.`, followed by the `DROP TABLE IF EXISTS` statement. `migrations_test.go` gets a test that the two new
  tables exist in the embedded files, the files pass `migrationcheck`, and the up files hold one statement each.
  Sqlite test schema: `scripts/database_scripts/table_email_configs.sql` and `table_email_config_histories.sql` (no CHECK, no JSON functions).
- `internal/configkeys`: `Table = "email_configs"`, `HistoryTable`, key name constants, `NewRegistry()`, golden test.
- `internal/config`: field `ConfigEncryptionKeys`, viper key `email_config_encryption_keys` bound with `BindEnv` only (no flag),
  `EnvValue(name)`.
- `internal/emailconfig`: `New(cfg *config.Config, store confighandler.Store, opts ...confighandler.Option) (confighandler.Handler, error)`; `initConfigHandler` passes `confighandler.WithSource("cli")`, the daemon uses the default (it never writes) (parse policy M2: a parse
  error is logged without key text and the handler is built with no key; the registry is the only error source) and
  `Supplier(h confighandler.Handler, key string, opts ...SupplierOption) emailhandler.SecretFunc` (option `WithTimeout(d)` for tests), which applies a 5 second default timeout to the caller context
  (`Send` is started with `context.Background()` in `email.go:65`, so a stalled database must not block a send forever) and converts
  `Secret` to string. `Describe`-based startup log lives here too (`LogSources`): `none` warns, `db_undecryptable` and `db_invalid` log errors.
- `pkg/dbhandler/config_store.go`: `NewConfigStore(db *sql.DB) confighandler.Store` (squirrel, explicit column lists and scans,
  `binary(16)` through `uuid.UUID`, `datetime(6)` through a small time scanner that accepts `time.Time` and `[]byte`/string,
  error classifier `isDuplicateKey(err)` for MySQL error 1062 and the sqlite constraint error). `handler.ConfigStore()` returns
  `NewConfigStore(h.db)`; the `DBHandler` interface gains `ConfigStore() confighandler.Store`; `go generate` rebuilds `mock_main.go`.
- Wiring: `cmd/email-manager`: `createDBHandler` is unchanged in signature; `run(dbHandler, cfg)` builds the config handler from
  `dbHandler.ConfigStore()`, then the three suppliers, then `NewEmailHandler`, then logs the sources with a 5 second context.
  `cmd/email-control`: `initEmailHandler(sqlDB, cache)` builds the same suppliers from `dbhandler.NewConfigStore(sqlDB)`; the new
  `config` command group (`config.go`, child of the root) uses `initConfigHandler()` = database + cipher only, with an injectable
  reader (`cmd.InOrStdin()`). Existing tests that need edits: `cmd/email-manager/main_test.go` and `run_test.go`,
  `cmd/email-control/main_test.go` and `handler_test.go`, `pkg/emailhandler/main_test.go:40`, and the `emailHandler` literals of the
  emailhandler tests that set `mailgunWebhookSigningKey` (mechanical `staticSecret`).
- `pkg/emailhandler`: `type SecretFunc func(ctx context.Context) (string, error)`; engines with an optional `client` override and the constructor seams `newClient func(apiKey string) *mailgun.MailgunImpl` (mailgun) and `newClient func(apiKey string) *sendgrid.Client` (sendgrid), defaulting to the real constructors;
  `verifyMailgunSignature(ctx, sig)` resolves the key right before the HMAC.
- CLI output of `config list`: key, kind, source, env_shadowed, has_secret, key_version, revision, updated_by, tm_update (JSON); the kind
  comes from `Handler.Registry()`.

### 3.4 Order of work

1. secretcrypto + tests (including frozen vectors) 2. confighandler + tests 3. email migrations + store + registry + config
4. emailhandler refactor + tests 5. daemon and CLI wiring + tests 6. docs and schema-ownership 7. verification in each module,
build of every service, scratch upgrade/downgrade/upgrade and the store/CHECK run on MariaDB 12.3 and MySQL 8.0 containers.

### 3.5 Verification (no test code is written in the design stage)

- T1 compatibility vectors. A throwaway program in scratch imports the current `bin-number-manager/pkg/secretcrypto` and holds
  verbatim copies of the ai-manager `SecretCrypto` (`mcpserverhandler/secret.go`) and of the packing code of `sealEngineBaseKey` /
  its unpacking counterpart (`aihandler/engine_base_key.go`), builds a fake key in code, encrypts one fake plaintext in each of the
  three layouts, prints hex. The vectors are frozen in `secretcrypto/compat_test.go` (the key is regenerated in the test from a byte
  sequence; no key literal). Expect: the new library decrypts all three; wrong AAD, wrong version, truncated blob and a flipped bit
  fail. The reverse direction (new write, old decrypt, all three layouts) runs once in the same program and is saved as evidence.
- T2 parser and primitives. Entries with bad base64, bad length, bad version, duplicates: the error contains none of a distinctive
  marker placed in the input; empty is valid; version order; nil safe methods; `WithMaxPlaintext`; adapters' edge cases
  (JoinSplit both empty / one empty / wrong nonce length / short ct, Pack version 0 / 65536, Unpack short); empty plaintext round trip.
- T3 confighandler matrix. Resolution: db > env > default, none, `db_undecryptable` (no key, unknown version, flipped bit; never env),
  `db_invalid` (wrong type, bad JSON, out of range), store read error vs not found, env invalid value, `env_shadowed`, atomic group
  (anchor row present: members read only from db/default; absent: env/default only). Registry guards: duplicate, format, reserved
  env (each listed name and the suffix), default type, Min>Max, enum, secret rules incl. MaxLen bounds, the forbidden bool
  placements (standalone, `KindParam`, `KindSecret`, `KindSelect` outside a group, `KindSelect` in a group with `Atomic=false`) and the one allowed, Atomic without group, mixed Atomic, anchor count. Writes: `Set` types, ranges, `ErrWrongKind`,
  `SetSecret` (`ErrKeyNotConfigured`, `ErrTooLong`, AAD binding: the stored blob does not decrypt under another key name or table),
  `Clear` (existing and no-op), conflict retry (success on 2nd attempt; `ErrConflict` after 3; duplicate insert race), history
  contents for every action, `Describe` (all keys, no values, unregistered rows ignored). `Secret`: String, `%v`, `%+v`, `%#v`,
  `%s`, `%q`, `%x`, JSON, Text, zero value, inside a struct and a map.
- T4 mutation checks in a scratch copy, each must turn at least one test red: AAD ignored on encrypt; fixed nonce; version ignored;
  parser error includes the entry; `NewValueJSON` filled for a secret; `OldValueJSON` filled for a secret; env fallback on
  undecryptable; store read error treated as no row; revision condition dropped in the fake store contract test; retry limit exceeded;
  reserved-name check removed; bool placement check removed; `Secret.String` reveals; `env_shadowed` computation removed; `Clear` writes history
  for a missing row; `IsSecret` false for secrets; AAD without the table part; `MaxLen` check removed; atomic anchor ignored; email side: webhook
  key read before the structure checks; supplier error swallowed as an empty key; `config list` printing a value. CLI and guard mutants: NUL/CR/LF check removed; read limit removed; nil `--actor-id` check removed; `ErrWrongKind` check removed; `actor==Nil` check
  removed; supplier timeout removed; `WithSource("cli")` not passed. Each of the email side and CLI mutants must also turn a test red. Commands and results saved as evidence.
- T5 email "no row = today" regression. With a fake store without rows and the three values in env: the engines receive the same
  key strings; an empty key still builds a client with the empty key, the send fails and the failover continues; an empty signing key
  rejects every webhook; the existing emailhandler tests pass after only the mechanical edits.
- T6 email-specific: row wins over env; undecryptable row is rejected without env fallback; store error propagates; a rotated key is
  used by the next call (client constructor sees it); webhook structure failure causes zero store reads; supplier timeout; `email-control
  config` set/clear/list with a fake store (stdin secret, `\r\n`, boundary lengths `MaxLen`/`MaxLen+1`, NUL/CR/LF inside, `--actor-id`
  missing or nil UUID, list output contains no value, no-key set fails; after set and clear the history source in the fake store is `cli`); unprefixed `CONFIG_ENCRYPTION_KEYS` ignored; daemon helper builds
  without a key; startup log lines contain no value.
- T7 store on real engines. In scratch MariaDB 12.3 and MySQL 8.0 containers: migrations up, down, up; the real `configStore`
  (build-tagged integration test or a one-off program kept in scratch evidence) runs insert, get, update with the right and a stale
  revision, delete, list, duplicate insert (1062 mapping), the transaction rollback when the history insert fails, plus one valid and
  several invalid rows for the CHECK constraints. Sqlite unit tests cover the same paths with the sqlite error mapping; the classifier
  is unit tested for both drivers' error values.
- T8 builds and gates: in `bin-common-handler` and `bin-email-manager` run the full sequence (`go mod tidy`, `go mod vendor`,
  `go generate ./...`, `go test ./...`, `golangci-lint run -v --timeout 5m`); `go build ./...` in every `bin-*` module that replaces
  `bin-common-handler` after refreshing its vendor directory (or with `-mod=mod`), to prove the library change does not break them;
  `scripts/check-*.sh`; `scripts/check-service-migrations.sh bin-email-manager`; `scripts/verify-service-migrations.sh
  bin-email-manager` if its tools exist locally.
- Coverage target: at least 90 percent statements for the two new packages and the new email files.
- Public text rule: the PR body, `bin-common-handler/CLAUDE.md` and the docs state the admission exception and its reason only, not who
  approved it.

### 3.6 DDL

```sql
CREATE TABLE IF NOT EXISTS email_configs (
  config_key         varchar(128)    NOT NULL,
  value_type         varchar(16)     NOT NULL,
  value_json         json            DEFAULT NULL,
  secret_ciphertext  varbinary(2048) DEFAULT NULL,
  key_version        int             NOT NULL DEFAULT 0,
  revision           int unsigned    NOT NULL DEFAULT 1,
  updated_by         binary(16)      DEFAULT NULL,
  tm_create          datetime(6)     NOT NULL,
  tm_update          datetime(6)     NOT NULL,
  PRIMARY KEY (config_key),
  CONSTRAINT chk_email_configs_one_of CHECK ((value_json IS NULL) <> (secret_ciphertext IS NULL)),
  CONSTRAINT chk_email_configs_type CHECK (value_type IN ('string','int','bool','duration','enum','secret')),
  CONSTRAINT chk_email_configs_secret_type CHECK ((value_type = 'secret') = (secret_ciphertext IS NOT NULL)),
  CONSTRAINT chk_email_configs_json CHECK (value_json IS NULL OR JSON_TYPE(value_json) <> 'NULL'),
  CONSTRAINT chk_email_configs_version CHECK (
    (secret_ciphertext IS NULL AND key_version = 0) OR (secret_ciphertext IS NOT NULL AND key_version > 0))
);

CREATE TABLE IF NOT EXISTS email_config_histories (
  id               binary(16)   NOT NULL,
  config_key       varchar(128) NOT NULL,
  action           varchar(16)  NOT NULL,
  is_secret        tinyint(1)   NOT NULL,
  old_value_json   json         DEFAULT NULL,
  new_value_json   json         DEFAULT NULL,
  old_key_version  int          NOT NULL DEFAULT 0,
  new_key_version  int          NOT NULL DEFAULT 0,
  actor_id         binary(16)   NOT NULL,
  source           varchar(16)  NOT NULL,
  tm_create        datetime(6)  NOT NULL,
  PRIMARY KEY (id),
  KEY idx_email_config_histories_key (config_key, tm_create),
  KEY idx_email_config_histories_create (tm_create),
  CONSTRAINT chk_email_config_histories_secret CHECK (is_secret = 0 OR (old_value_json IS NULL AND new_value_json IS NULL)),
  CONSTRAINT chk_email_config_histories_action CHECK (action IN ('set','clear')),
  CONSTRAINT chk_email_config_histories_source CHECK (source IN ('cli','api'))
);
```

Narrowed from the general idea on purpose: no `float` type, and only the actions and sources that exist in this PR
(`reencrypt` and `system` are added by the migration that introduces them). Impact analysis for the PR body: both
statements create new tables (no destructive operation); old code never touches them; new code reads them only after
the startup migration; rolling the code back leaves two empty tables; no data loss possible; lock time is nil (new
tables).
