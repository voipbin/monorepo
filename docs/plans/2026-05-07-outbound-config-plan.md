# OutboundConfig Implementation Plan

> **Note:** This document was written before the table naming convention (§7.0 in [docs/conventions/database.md](../conventions/database.md)) was enforced. All references to `outbound_configs` in this document should be read as `call_outbound_configs` — that is the actual table name in production.

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a per-customer `OutboundConfig` resource (destination country whitelist + codec preference) that gates every outbound PSTN call in `bin-call-manager`, and removes `Customer.Metadata.OutboundCodecs`.

**Architecture:** The `outbound_configs` table lives in MySQL. `bin-call-manager` owns the domain (model, handler, cache, DB, listenhandler routes). `CreateCallOutgoing` fetches the config once and uses it for both codec embedding and whitelist enforcement. `bin-api-manager` proxies the CRUD surface. `bin-customer-manager` has `OutboundCodecs` removed from its `Metadata` struct.

**Tech Stack:** Go 1.21, MySQL (Squirrel query builder is NOT used here — direct SQL like the rest of call-manager's dbhandler), Redis (`go-redis/v8`), RabbitMQ RPC, `go.uber.org/mock` for mocks, `github.com/dongri/phonenumber` for ISO lookups, `golangci-lint`.

**Design doc:** `docs/plans/2026-05-07-outbound-config-design.md`

---

## Task 1: Alembic migration — create `outbound_configs` table

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<hash>_outbound_configs_create_table.py`

**Step 1: Generate migration file**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-outbound-config/bin-dbscheme-manager/bin-manager
# Ensure alembic.ini is configured (copy from .sample if needed)
alembic -c alembic.ini revision -m "outbound_configs_create_table"
```

**Step 2: Edit the generated file — add SQL to `upgrade()` and `downgrade()`**

```python
def upgrade():
    op.execute("""
        CREATE TABLE outbound_configs (
            id                    VARCHAR(36)  NOT NULL,
            customer_id           VARCHAR(36)  NOT NULL,
            name                  VARCHAR(255) NOT NULL DEFAULT '',
            detail                TEXT         NOT NULL DEFAULT '',
            destination_whitelist JSON         NOT NULL DEFAULT '[]',
            codecs                VARCHAR(255) NOT NULL DEFAULT '',
            tm_create             DATETIME(6)  DEFAULT NULL,
            tm_update             DATETIME(6)  DEFAULT NULL,
            tm_delete             DATETIME(6)  DEFAULT NULL,
            PRIMARY KEY (id),
            UNIQUE KEY uq_customer_id (customer_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)

def downgrade():
    op.execute("DROP TABLE IF EXISTS outbound_configs")
```

**Step 3: Verify migration chain**

```bash
alembic -c alembic.ini heads
# Expected: exactly one head
```

**Step 4: Commit**

```bash
git add bin-dbscheme-manager/
git commit -m "NOJIRA-outbound-config

- bin-dbscheme-manager: Add outbound_configs_create_table migration"
```

---

## Task 2: `bin-customer-manager` — remove `OutboundCodecs`, add `IsInternalSystemID`

**Files:**
- Modify: `bin-customer-manager/models/customer/metadata.go`
- Modify: `bin-customer-manager/models/customer/metadata_test.go`
- Create: `bin-customer-manager/models/customer/ids.go`
- Modify: `bin-customer-manager/pkg/listenhandler/v1_customers_test.go` (fixture cleanup)
- Modify: `bin-customer-manager/pkg/listenhandler/v1_customers_signup_test.go` (fixture cleanup)

**Step 1: Write the `IsInternalSystemID` test first**

Add to `bin-customer-manager/models/customer/metadata_test.go` (or a new `ids_test.go`):

```go
func TestIsInternalSystemID(t *testing.T) {
    tests := []struct {
        name string
        id   uuid.UUID
        want bool
    }{
        {"call-manager", IDCallManager, true},
        {"ai-manager", IDAIManager, true},
        {"system", IDSystem, true},
        {"basic-route", IDBasicRoute, true},
        {"random customer", uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"), false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := IsInternalSystemID(tt.id); got != tt.want {
                t.Errorf("IsInternalSystemID() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

Run: `cd bin-customer-manager && go test ./models/customer/...`
Expected: FAIL (function doesn't exist yet)

**Step 2: Create `bin-customer-manager/models/customer/ids.go`**

```go
package customer

import "github.com/gofrs/uuid"

// IsInternalSystemID returns true for VoIPbin internal system customer IDs
// that bypass outbound permission and whitelist checks.
func IsInternalSystemID(id uuid.UUID) bool {
    return id == IDCallManager || id == IDAIManager || id == IDSystem || id == IDBasicRoute
}
```

**Step 3: Run the test**

```bash
cd bin-customer-manager && go test ./models/customer/...
```
Expected: PASS

**Step 4: Remove `OutboundCodecs` from `models/customer/metadata.go`**

Before:
```go
type Metadata struct {
    RTPDebug       bool   `json:"rtp_debug"`
    OutboundCodecs string `json:"outbound_codecs"`
}
```

After:
```go
type Metadata struct {
    RTPDebug bool `json:"rtp_debug"`
}
```

Also remove the `MetadataKeyRTPDebug` comment referencing codecs if present. Remove the `outbound_codecs` constant if it exists.

**Step 5: Fix `metadata_test.go`**

Remove any test case that references `OutboundCodecs` or `"outbound_codecs"`.

**Step 6: Fix listenhandler fixtures**

In `pkg/listenhandler/v1_customers_test.go` and `v1_customers_signup_test.go`, remove `"outbound_codecs":""` from all expected JSON strings. Use your editor's find-replace: `,"outbound_codecs":""` → `` (empty).

**Step 7: Verify**

```bash
cd bin-customer-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all pass, lint clean.

**Step 8: Commit**

```bash
git add bin-customer-manager/
git commit -m "NOJIRA-outbound-config

- bin-customer-manager: Remove OutboundCodecs from Metadata struct
- bin-customer-manager: Add IsInternalSystemID helper to models/customer"
```

---

## Task 3: `bin-call-manager` — OutboundConfig model

**Files:**
- Create: `bin-call-manager/models/outboundconfig/outboundconfig.go`
- Create: `bin-call-manager/models/outboundconfig/webhook.go`
- Create: `bin-call-manager/models/outboundconfig/errors.go`
- Create: `bin-call-manager/models/outboundconfig/iso.go`
- Create: `bin-call-manager/models/outboundconfig/iso_test.go`

**Step 1: Create `outboundconfig.go`**

```go
package outboundconfig

import (
    "time"

    "github.com/gofrs/uuid"
)

// OutboundConfig holds per-customer outbound call configuration.
type OutboundConfig struct {
    ID                   uuid.UUID  `json:"id"`
    CustomerID           uuid.UUID  `json:"customer_id"`
    Name                 string     `json:"name"`
    Detail               string     `json:"detail"`
    DestinationWhitelist []string   `json:"destination_whitelist"` // ISO 3166 alpha-2 lowercase
    Codecs               string     `json:"codecs"`                // comma-separated; empty = server default
    TMCreate             *time.Time `json:"tm_create"`
    TMUpdate             *time.Time `json:"tm_update"`
    TMDelete             *time.Time `json:"tm_delete"`
}

// UpdateRequest uses pointer fields so callers can distinguish "absent" (nil = no change)
// from "explicit empty" (pointer to zero value = clear the field).
type UpdateRequest struct {
    Name                 *string   `json:"name,omitempty"`
    Detail               *string   `json:"detail,omitempty"`
    DestinationWhitelist *[]string `json:"destination_whitelist,omitempty"`
    Codecs               *string   `json:"codecs,omitempty"`
}
```

**Step 2: Create `webhook.go`**

```go
package outboundconfig

import "time"

import "github.com/gofrs/uuid"

// WebhookMessage is the external-facing representation of OutboundConfig.
// Always return this type through the public API — never the internal struct.
// ConvertWebhookMessage exists to future-proof against internal-only fields.
type WebhookMessage struct {
    ID                   uuid.UUID  `json:"id"`
    CustomerID           uuid.UUID  `json:"customer_id"`
    Name                 string     `json:"name"`
    Detail               string     `json:"detail"`
    DestinationWhitelist []string   `json:"destination_whitelist"`
    Codecs               string     `json:"codecs"`
    TMCreate             *time.Time `json:"tm_create"`
    TMUpdate             *time.Time `json:"tm_update"`
    TMDelete             *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts an OutboundConfig to its external form.
func ConvertWebhookMessage(c *OutboundConfig) *WebhookMessage {
    if c == nil {
        return nil
    }
    return &WebhookMessage{
        ID:                   c.ID,
        CustomerID:           c.CustomerID,
        Name:                 c.Name,
        Detail:               c.Detail,
        DestinationWhitelist: c.DestinationWhitelist,
        Codecs:               c.Codecs,
        TMCreate:             c.TMCreate,
        TMUpdate:             c.TMUpdate,
        TMDelete:             c.TMDelete,
    }
}
```

**Step 3: Create `errors.go`**

```go
package outboundconfig

import "errors"

// ErrDestinationNotWhitelisted is returned when a PSTN destination's country
// is not in the customer's OutboundConfig.DestinationWhitelist.
// bin-api-manager maps this sentinel to 400 Bad Request.
var ErrDestinationNotWhitelisted = errors.New("outbound destination country not whitelisted")
```

**Step 4: Write the ISO drift test first (`iso_test.go`)**

```go
package outboundconfig_test

import (
    "testing"

    "github.com/dongri/phonenumber"

    outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
)

// TestISOMapDrift asserts that countries returned by the phonenumber library
// for a representative E.164 set are present in the local ISOCountryCodes map.
// If this test fails after a phonenumber library update, add the new codes to iso.go.
func TestISOMapDrift(t *testing.T) {
    samples := []struct {
        number      string
        wantCountry string
    }{
        {"+12025550100", "us"},    // NANP - US
        {"+14165550100", "ca"},    // NANP - Canada
        {"+442071234567", "gb"},   // UK
        {"+33123456789", "fr"},    // France
        {"+4930123456", "de"},     // Germany
        {"+81312345678", "jp"},    // Japan
        {"+8210123456789", "kr"},  // South Korea
        {"+861012345678", "cn"},   // China
        {"+911234567890", "in"},   // India
        {"+5511987654321", "br"},  // Brazil
        {"+61212345678", "au"},    // Australia
        {"+27123456789", "za"},    // South Africa
        {"+971501234567", "ae"},   // UAE
        {"+7495123456", "ru"},     // Russia
    }

    for _, s := range samples {
        n := phonenumber.Parse(s.number, "")
        iso := phonenumber.GetISO3166ByNumber(n, false)
        got := strings.ToLower(iso.Alpha2)
        if got != s.wantCountry {
            t.Errorf("phonenumber returned %q for %s, expected %q — update test fixtures", got, s.number, s.wantCountry)
            continue
        }
        if _, ok := outboundconfig.ISOCountryCodes[got]; !ok {
            t.Errorf("ISOCountryCodes is missing %q (returned for %s) — add it to iso.go", got, s.number)
        }
    }
}
```

Add `"strings"` import.

Run: `cd bin-call-manager && go test ./models/outboundconfig/...`
Expected: FAIL (ISOCountryCodes not defined yet)

**Step 5: Create `iso.go` with the country code set**

```go
package outboundconfig

import "regexp"

// ISOCountryCodes is the set of valid ISO 3166 alpha-2 codes accepted in DestinationWhitelist.
// Entries must be lowercase. Add new codes here if the phonenumber library upgrade returns
// codes not yet in this map (you will be alerted by TestISOMapDrift).
var ISOCountryCodes = map[string]struct{}{
    "ac": {}, "ad": {}, "ae": {}, "af": {}, "ag": {}, "ai": {}, "al": {}, "am": {},
    "ao": {}, "aq": {}, "ar": {}, "as": {}, "at": {}, "au": {}, "aw": {}, "ax": {},
    "az": {}, "ba": {}, "bb": {}, "bd": {}, "be": {}, "bf": {}, "bg": {}, "bh": {},
    "bi": {}, "bj": {}, "bl": {}, "bm": {}, "bn": {}, "bo": {}, "bq": {}, "br": {},
    "bs": {}, "bt": {}, "bv": {}, "bw": {}, "by": {}, "bz": {}, "ca": {}, "cc": {},
    "cd": {}, "cf": {}, "cg": {}, "ch": {}, "ci": {}, "ck": {}, "cl": {}, "cm": {},
    "cn": {}, "co": {}, "cr": {}, "cu": {}, "cv": {}, "cw": {}, "cx": {}, "cy": {},
    "cz": {}, "de": {}, "dj": {}, "dk": {}, "dm": {}, "do": {}, "dz": {}, "ec": {},
    "ee": {}, "eg": {}, "eh": {}, "er": {}, "es": {}, "et": {}, "fi": {}, "fj": {},
    "fk": {}, "fm": {}, "fo": {}, "fr": {}, "ga": {}, "gb": {}, "gd": {}, "ge": {},
    "gf": {}, "gg": {}, "gh": {}, "gi": {}, "gl": {}, "gm": {}, "gn": {}, "gp": {},
    "gq": {}, "gr": {}, "gs": {}, "gt": {}, "gu": {}, "gw": {}, "gy": {}, "hk": {},
    "hm": {}, "hn": {}, "hr": {}, "ht": {}, "hu": {}, "id": {}, "ie": {}, "il": {},
    "im": {}, "in": {}, "io": {}, "iq": {}, "ir": {}, "is": {}, "it": {}, "je": {},
    "jm": {}, "jo": {}, "jp": {}, "ke": {}, "kg": {}, "kh": {}, "ki": {}, "km": {},
    "kn": {}, "kp": {}, "kr": {}, "kw": {}, "ky": {}, "kz": {}, "la": {}, "lb": {},
    "lc": {}, "li": {}, "lk": {}, "lr": {}, "ls": {}, "lt": {}, "lu": {}, "lv": {},
    "ly": {}, "ma": {}, "mc": {}, "md": {}, "me": {}, "mf": {}, "mg": {}, "mh": {},
    "mk": {}, "ml": {}, "mm": {}, "mn": {}, "mo": {}, "mp": {}, "mq": {}, "mr": {},
    "ms": {}, "mt": {}, "mu": {}, "mv": {}, "mw": {}, "mx": {}, "my": {}, "mz": {},
    "na": {}, "nc": {}, "ne": {}, "nf": {}, "ng": {}, "ni": {}, "nl": {}, "no": {},
    "np": {}, "nr": {}, "nu": {}, "nz": {}, "om": {}, "pa": {}, "pe": {}, "pf": {},
    "pg": {}, "ph": {}, "pk": {}, "pl": {}, "pm": {}, "pn": {}, "pr": {}, "ps": {},
    "pt": {}, "pw": {}, "py": {}, "qa": {}, "re": {}, "ro": {}, "rs": {}, "ru": {},
    "rw": {}, "sa": {}, "sb": {}, "sc": {}, "sd": {}, "se": {}, "sg": {}, "sh": {},
    "si": {}, "sj": {}, "sk": {}, "sl": {}, "sm": {}, "sn": {}, "so": {}, "sr": {},
    "ss": {}, "st": {}, "sv": {}, "sx": {}, "sy": {}, "sz": {}, "tc": {}, "td": {},
    "tf": {}, "tg": {}, "th": {}, "tj": {}, "tk": {}, "tl": {}, "tm": {}, "tn": {},
    "to": {}, "tr": {}, "tt": {}, "tv": {}, "tw": {}, "tz": {}, "ua": {}, "ug": {},
    "um": {}, "us": {}, "uy": {}, "uz": {}, "va": {}, "vc": {}, "ve": {}, "vg": {},
    "vi": {}, "vn": {}, "vu": {}, "wf": {}, "ws": {}, "xk": {}, "ye": {}, "yt": {},
    "za": {}, "zm": {}, "zw": {},
}

// codecsRegexp validates the codecs field format: alphanumeric tokens separated by commas.
var codecsRegexp = regexp.MustCompile(`^[A-Za-z0-9]+(,[A-Za-z0-9]+)*$`)

// ValidateCodecs returns true if the codecs string is empty (server default) or
// matches the expected comma-separated alphanumeric token format.
func ValidateCodecs(codecs string) bool {
    if codecs == "" {
        return true
    }
    if len(codecs) > 255 {
        return false
    }
    if strings.ContainsAny(codecs, "\r\n") {
        return false
    }
    return codecsRegexp.MatchString(codecs)
}
```

Add `"strings"` import.

**Step 6: Run tests**

```bash
cd bin-call-manager && go test ./models/outboundconfig/...
```
Expected: PASS

**Step 7: Verify build**

```bash
cd bin-call-manager && go mod tidy && go mod vendor && go build ./...
```

**Step 8: Commit**

```bash
git add bin-call-manager/models/outboundconfig/
git commit -m "NOJIRA-outbound-config

- bin-call-manager: Add OutboundConfig model, WebhookMessage, errors, ISO codes"
```

---

## Task 4: `bin-call-manager` — cachehandler methods for OutboundConfig

**Files:**
- Modify: `bin-call-manager/pkg/cachehandler/main.go` (add interface methods)
- Modify: `bin-call-manager/pkg/cachehandler/handler.go` (add implementation)
- Regenerate: `bin-call-manager/pkg/cachehandler/mock_main.go`

**Step 1: Add interface methods to `cachehandler/main.go`**

Add to the `CacheHandler` interface:

```go
// OutboundConfigGet returns a cached OutboundConfig for the given customerID.
// Returns (nil, nil) if the key exists but represents a negative-cache sentinel (no DB row).
// Returns (nil, redis.Nil error) if the key is absent entirely (cache miss).
OutboundConfigGet(ctx context.Context, customerID uuid.UUID) (*outboundconfig.OutboundConfig, error)

// OutboundConfigSet caches the config for the given customerID with a 30-minute TTL.
OutboundConfigSet(ctx context.Context, customerID uuid.UUID, c *outboundconfig.OutboundConfig) error

// OutboundConfigSetNotFound caches a "not found" sentinel for the given customerID (1-minute TTL).
// Prevents DB hammering for customers with no config row.
OutboundConfigSetNotFound(ctx context.Context, customerID uuid.UUID) error

// OutboundConfigDelete removes the cached config for the given customerID.
OutboundConfigDelete(ctx context.Context, customerID uuid.UUID) error
```

Add import: `outboundconfig "monorepo/bin-call-manager/models/outboundconfig"`

**Step 2: Implement in `handler.go`**

```go
import (
    "github.com/go-redis/redis/v8"
    outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
)

const outboundConfigNotFoundSentinel = `{"_not_found":true}`

func outboundConfigKey(customerID uuid.UUID) string {
    return fmt.Sprintf("outbound_config:%s", customerID)
}

func (h *handler) OutboundConfigGet(ctx context.Context, customerID uuid.UUID) (*outboundconfig.OutboundConfig, error) {
    key := outboundConfigKey(customerID)
    tmp, err := h.Cache.Get(ctx, key).Result()
    if err != nil {
        return nil, err // redis.Nil means cache miss
    }
    if tmp == outboundConfigNotFoundSentinel {
        return nil, nil // negative cache hit: no row in DB
    }
    var res outboundconfig.OutboundConfig
    if err := json.Unmarshal([]byte(tmp), &res); err != nil {
        return nil, err
    }
    return &res, nil
}

func (h *handler) OutboundConfigSet(ctx context.Context, customerID uuid.UUID, c *outboundconfig.OutboundConfig) error {
    key := outboundConfigKey(customerID)
    tmp, err := json.Marshal(c)
    if err != nil {
        return err
    }
    return h.Cache.Set(ctx, key, tmp, time.Minute*30).Err()
}

func (h *handler) OutboundConfigSetNotFound(ctx context.Context, customerID uuid.UUID) error {
    key := outboundConfigKey(customerID)
    return h.Cache.Set(ctx, key, outboundConfigNotFoundSentinel, time.Minute).Err()
}

func (h *handler) OutboundConfigDelete(ctx context.Context, customerID uuid.UUID) error {
    key := outboundConfigKey(customerID)
    return h.Cache.Del(ctx, key).Err()
}
```

**Step 3: Regenerate mock**

```bash
cd bin-call-manager && go generate ./pkg/cachehandler/...
```

**Step 4: Verify**

```bash
cd bin-call-manager && go mod tidy && go mod vendor && go test ./pkg/cachehandler/...
```

**Step 5: Commit**

```bash
git add bin-call-manager/pkg/cachehandler/
git commit -m "NOJIRA-outbound-config

- bin-call-manager: Add OutboundConfig cache methods to cachehandler"
```

---

## Task 5: `bin-call-manager` — dbhandler methods for OutboundConfig

**Files:**
- Create: `bin-call-manager/pkg/dbhandler/outbound_config.go`
- Create: `bin-call-manager/pkg/dbhandler/outbound_config_test.go`
- Modify: `bin-call-manager/pkg/dbhandler/main.go` (add interface methods)

**Step 1: Add interface methods to `dbhandler/main.go`**

```go
OutboundConfigCreate(ctx context.Context, c *outboundconfig.OutboundConfig) error
OutboundConfigGetByID(ctx context.Context, id uuid.UUID) (*outboundconfig.OutboundConfig, error)
OutboundConfigGetByCustomerID(ctx context.Context, customerID uuid.UUID) (*outboundconfig.OutboundConfig, error)
OutboundConfigUpdate(ctx context.Context, id uuid.UUID, req *outboundconfig.UpdateRequest) (*outboundconfig.OutboundConfig, error)
OutboundConfigList(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*outboundconfig.OutboundConfig, error)
```

Add import: `outboundconfig "monorepo/bin-call-manager/models/outboundconfig"`

**Step 2: Write tests first in `outbound_config_test.go`**

```go
package dbhandler

import (
    "context"
    "encoding/json"
    "testing"
    "time"

    "github.com/gofrs/uuid"
    "github.com/stretchr/testify/require"

    outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
)

func TestOutboundConfigCreate(t *testing.T) {
    tests := []struct {
        name    string
        config  *outboundconfig.OutboundConfig
        wantErr bool
    }{
        {
            name: "creates successfully",
            config: &outboundconfig.OutboundConfig{
                ID:                   uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001"),
                CustomerID:           uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001"),
                Name:                 "test config",
                DestinationWhitelist: []string{"us", "gb"},
                Codecs:               "PCMU,PCMA",
            },
            wantErr: false,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()
            h := &handler{/* inject test DB */}
            err := h.OutboundConfigCreate(context.Background(), tt.config)
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

Note: call-manager dbhandler tests use actual test DB injection (see existing patterns in `call_test.go`). Follow the same test setup pattern already present in the package.

Run: `cd bin-call-manager && go test ./pkg/dbhandler/...`
Expected: FAIL (method not implemented)

**Step 3: Implement `outbound_config.go`**

```go
package dbhandler

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "time"

    "github.com/gofrs/uuid"
    "github.com/sirupsen/logrus"

    outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
)

const outboundConfigTable = "outbound_configs"

func (h *handler) outboundConfigGetFromRow(row *sql.Rows) (*outboundconfig.OutboundConfig, error) {
    res := &outboundconfig.OutboundConfig{}
    var whitelistJSON []byte
    var tmCreate, tmUpdate, tmDelete sql.NullTime

    if err := row.Scan(
        &res.ID,
        &res.CustomerID,
        &res.Name,
        &res.Detail,
        &whitelistJSON,
        &res.Codecs,
        &tmCreate,
        &tmUpdate,
        &tmDelete,
    ); err != nil {
        return nil, fmt.Errorf("could not scan outbound_config row: %w", err)
    }
    if err := json.Unmarshal(whitelistJSON, &res.DestinationWhitelist); err != nil {
        return nil, fmt.Errorf("could not unmarshal destination_whitelist: %w", err)
    }
    if tmCreate.Valid { t := tmCreate.Time; res.TMCreate = &t }
    if tmUpdate.Valid { t := tmUpdate.Time; res.TMUpdate = &t }
    if tmDelete.Valid { t := tmDelete.Time; res.TMDelete = &t }
    return res, nil
}

func (h *handler) OutboundConfigCreate(ctx context.Context, c *outboundconfig.OutboundConfig) error {
    whitelistJSON, err := json.Marshal(c.DestinationWhitelist)
    if err != nil {
        return err
    }
    now := time.Now()
    _, err = h.db.ExecContext(ctx,
        `INSERT INTO outbound_configs (id, customer_id, name, detail, destination_whitelist, codecs, tm_create, tm_update)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
        c.ID, c.CustomerID, c.Name, c.Detail, whitelistJSON, c.Codecs, now, now,
    )
    return err
}

func (h *handler) OutboundConfigGetByID(ctx context.Context, id uuid.UUID) (*outboundconfig.OutboundConfig, error) {
    rows, err := h.db.QueryContext(ctx,
        `SELECT id, customer_id, name, detail, destination_whitelist, codecs, tm_create, tm_update, tm_delete
         FROM outbound_configs WHERE id = ? AND tm_delete IS NULL LIMIT 1`, id)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    if !rows.Next() {
        return nil, nil
    }
    return h.outboundConfigGetFromRow(rows)
}

func (h *handler) OutboundConfigGetByCustomerID(ctx context.Context, customerID uuid.UUID) (*outboundconfig.OutboundConfig, error) {
    rows, err := h.db.QueryContext(ctx,
        `SELECT id, customer_id, name, detail, destination_whitelist, codecs, tm_create, tm_update, tm_delete
         FROM outbound_configs WHERE customer_id = ? AND tm_delete IS NULL LIMIT 1`, customerID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    if !rows.Next() {
        return nil, nil
    }
    return h.outboundConfigGetFromRow(rows)
}

func (h *handler) OutboundConfigUpdate(ctx context.Context, id uuid.UUID, req *outboundconfig.UpdateRequest) (*outboundconfig.OutboundConfig, error) {
    // Build SET clause dynamically based on non-nil fields
    query := "UPDATE outbound_configs SET tm_update = ?"
    args := []interface{}{time.Now()}

    if req.Name != nil {
        query += ", name = ?"
        args = append(args, *req.Name)
    }
    if req.Detail != nil {
        query += ", detail = ?"
        args = append(args, *req.Detail)
    }
    if req.DestinationWhitelist != nil {
        b, err := json.Marshal(*req.DestinationWhitelist)
        if err != nil {
            return nil, err
        }
        query += ", destination_whitelist = ?"
        args = append(args, b)
    }
    if req.Codecs != nil {
        query += ", codecs = ?"
        args = append(args, *req.Codecs)
    }
    query += " WHERE id = ? AND tm_delete IS NULL"
    args = append(args, id)

    if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
        return nil, err
    }
    return h.OutboundConfigGetByID(ctx, id)
}

func (h *handler) OutboundConfigList(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*outboundconfig.OutboundConfig, error) {
    q := `SELECT id, customer_id, name, detail, destination_whitelist, codecs, tm_create, tm_update, tm_delete
          FROM outbound_configs WHERE customer_id = ? AND tm_delete IS NULL`
    args := []interface{}{customerID}
    if pageToken != "" {
        q += " AND tm_create < ?"
        args = append(args, pageToken)
    }
    q += " ORDER BY tm_create DESC LIMIT ?"
    args = append(args, pageSize)

    rows, err := h.db.QueryContext(ctx, q, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var res []*outboundconfig.OutboundConfig
    for rows.Next() {
        c, err := h.outboundConfigGetFromRow(rows)
        if err != nil {
            return nil, err
        }
        res = append(res, c)
    }
    return res, nil
}
```

**Step 4: Regenerate dbhandler mock and verify**

```bash
cd bin-call-manager && go generate ./pkg/dbhandler/... && go mod tidy && go mod vendor && go test ./pkg/dbhandler/... && golangci-lint run ./pkg/dbhandler/...
```

**Step 5: Commit**

```bash
git add bin-call-manager/pkg/dbhandler/
git commit -m "NOJIRA-outbound-config

- bin-call-manager: Add OutboundConfig CRUD methods to dbhandler"
```

---

## Task 6: `bin-call-manager` — outboundconfighandler

**Files:**
- Create: `bin-call-manager/pkg/outboundconfighandler/main.go`
- Create: `bin-call-manager/pkg/outboundconfighandler/main_test.go`
- Create: `bin-call-manager/pkg/outboundconfighandler/outbound_config.go`
- Create: `bin-call-manager/pkg/outboundconfighandler/outbound_config_test.go`
- Create: `bin-call-manager/pkg/outboundconfighandler/validate.go`
- Create: `bin-call-manager/pkg/outboundconfighandler/validate_test.go`

**Step 1: Create `main.go` — interface and constructor**

```go
package outboundconfighandler

//go:generate mockgen -package outboundconfighandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
    "context"

    "github.com/gofrs/uuid"

    outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
    "monorepo/bin-call-manager/pkg/cachehandler"
    "monorepo/bin-call-manager/pkg/dbhandler"
    "monorepo/bin-common-handler/pkg/utilhandler"
)

// OutboundConfigHandler manages OutboundConfig resources.
type OutboundConfigHandler interface {
    GetByCustomerID(ctx context.Context, customerID uuid.UUID) (*outboundconfig.OutboundConfig, error)
    Create(ctx context.Context, customerID uuid.UUID, req *outboundconfig.UpdateRequest) (*outboundconfig.OutboundConfig, error)
    Update(ctx context.Context, id uuid.UUID, req *outboundconfig.UpdateRequest) (*outboundconfig.OutboundConfig, error)
    GetByID(ctx context.Context, id uuid.UUID) (*outboundconfig.OutboundConfig, error)
    List(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*outboundconfig.OutboundConfig, error)
}

type outboundConfigHandler struct {
    utilHandler  utilhandler.UtilHandler
    db           dbhandler.DBHandler
    cacheHandler cachehandler.CacheHandler
}

// NewOutboundConfigHandler creates an OutboundConfigHandler.
func NewOutboundConfigHandler(
    utilHandler utilhandler.UtilHandler,
    db dbhandler.DBHandler,
    cacheHandler cachehandler.CacheHandler,
) OutboundConfigHandler {
    return &outboundConfigHandler{
        utilHandler:  utilHandler,
        db:           db,
        cacheHandler: cacheHandler,
    }
}
```

**Step 2: Write tests for `GetByCustomerID` first**

In `outbound_config_test.go`:

```go
package outboundconfighandler

import (
    "context"
    "errors"
    "testing"

    "github.com/go-redis/redis/v8"
    "github.com/gofrs/uuid"
    "go.uber.org/mock/gomock"

    outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
    "monorepo/bin-call-manager/pkg/cachehandler"
    "monorepo/bin-call-manager/pkg/dbhandler"
    "monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_GetByCustomerID(t *testing.T) {
    customerID := uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001")
    existingConfig := &outboundconfig.OutboundConfig{
        ID:                   uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
        CustomerID:           customerID,
        DestinationWhitelist: []string{"us"},
        Codecs:               "PCMU",
    }

    tests := []struct {
        name             string
        customerID       uuid.UUID
        cacheGetReturn   *outboundconfig.OutboundConfig
        cacheGetErr      error
        dbGetReturn      *outboundconfig.OutboundConfig
        expectCacheSet   bool
        expectCacheNotFound bool
        wantConfig       *outboundconfig.OutboundConfig
        wantErr          bool
    }{
        {
            name:           "cache hit",
            customerID:     customerID,
            cacheGetReturn: existingConfig,
            cacheGetErr:    nil,
            wantConfig:     existingConfig,
        },
        {
            name:           "cache miss, db found",
            customerID:     customerID,
            cacheGetErr:    redis.Nil,
            dbGetReturn:    existingConfig,
            expectCacheSet: true,
            wantConfig:     existingConfig,
        },
        {
            name:                "cache miss, db not found",
            customerID:          customerID,
            cacheGetErr:         redis.Nil,
            dbGetReturn:         nil,
            expectCacheNotFound: true,
            wantConfig:          nil,
        },
        {
            name:           "negative cache hit",
            customerID:     customerID,
            cacheGetReturn: nil,  // sentinel: nil config, nil err
            cacheGetErr:    nil,
            wantConfig:     nil,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()

            mockCache := cachehandler.NewMockCacheHandler(mc)
            mockDB := dbhandler.NewMockDBHandler(mc)
            mockUtil := utilhandler.NewMockUtilHandler(mc)

            // Setup cache mock
            if tt.cacheGetErr == redis.Nil {
                // cache miss — expect DB call
                mockCache.EXPECT().OutboundConfigGet(gomock.Any(), tt.customerID).Return(nil, redis.Nil)
                if tt.dbGetReturn != nil {
                    mockDB.EXPECT().OutboundConfigGetByCustomerID(gomock.Any(), tt.customerID).Return(tt.dbGetReturn, nil)
                    mockCache.EXPECT().OutboundConfigSet(gomock.Any(), tt.customerID, tt.dbGetReturn).Return(nil)
                } else {
                    mockDB.EXPECT().OutboundConfigGetByCustomerID(gomock.Any(), tt.customerID).Return(nil, nil)
                    mockCache.EXPECT().OutboundConfigSetNotFound(gomock.Any(), tt.customerID).Return(nil)
                }
            } else {
                // cache returns value (including nil for sentinel)
                mockCache.EXPECT().OutboundConfigGet(gomock.Any(), tt.customerID).Return(tt.cacheGetReturn, tt.cacheGetErr)
            }

            h := &outboundConfigHandler{
                utilHandler:  mockUtil,
                db:           mockDB,
                cacheHandler: mockCache,
            }

            got, err := h.GetByCustomerID(context.Background(), tt.customerID)
            if tt.wantErr && err == nil {
                t.Error("expected error, got nil")
            }
            if !tt.wantErr && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
            if got != tt.wantConfig {
                t.Errorf("GetByCustomerID() = %v, want %v", got, tt.wantConfig)
            }
        })
    }
}
```

Run: `cd bin-call-manager && go test ./pkg/outboundconfighandler/...`
Expected: FAIL

**Step 3: Implement `outbound_config.go`**

```go
package outboundconfighandler

import (
    "context"
    "fmt"
    "time"

    "github.com/go-redis/redis/v8"
    "github.com/gofrs/uuid"
    "github.com/sirupsen/logrus"

    outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
)

func (h *outboundConfigHandler) GetByCustomerID(ctx context.Context, customerID uuid.UUID) (*outboundconfig.OutboundConfig, error) {
    log := logrus.WithField("customer_id", customerID)

    // Cache lookup
    cached, err := h.cacheHandler.OutboundConfigGet(ctx, customerID)
    if err == nil {
        // cache hit (real row or negative sentinel)
        return cached, nil
    }
    if err != redis.Nil {
        log.Warnf("cache error getting outbound_config, falling through to DB: %v", err)
    }

    // DB lookup
    c, err := h.db.OutboundConfigGetByCustomerID(ctx, customerID)
    if err != nil {
        return nil, fmt.Errorf("db error getting outbound_config: %w", err)
    }
    if c == nil {
        _ = h.cacheHandler.OutboundConfigSetNotFound(ctx, customerID) // best-effort
        return nil, nil
    }
    _ = h.cacheHandler.OutboundConfigSet(ctx, customerID, c) // best-effort
    return c, nil
}

func (h *outboundConfigHandler) GetByID(ctx context.Context, id uuid.UUID) (*outboundconfig.OutboundConfig, error) {
    return h.db.OutboundConfigGetByID(ctx, id)
}

func (h *outboundConfigHandler) List(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*outboundconfig.OutboundConfig, error) {
    return h.db.OutboundConfigList(ctx, customerID, pageSize, pageToken)
}

func (h *outboundConfigHandler) Create(ctx context.Context, customerID uuid.UUID, req *outboundconfig.UpdateRequest) (*outboundconfig.OutboundConfig, error) {
    if err := h.validateUpdateRequest(req); err != nil {
        return nil, err
    }

    c := &outboundconfig.OutboundConfig{
        ID:         h.utilHandler.UUIDCreate(),
        CustomerID: customerID,
    }
    applyUpdateRequest(c, req)

    if err := h.db.OutboundConfigCreate(ctx, c); err != nil {
        return nil, fmt.Errorf("could not create outbound_config: %w", err)
    }
    _ = h.cacheHandler.OutboundConfigSet(ctx, customerID, c)
    return c, nil
}

func (h *outboundConfigHandler) Update(ctx context.Context, id uuid.UUID, req *outboundconfig.UpdateRequest) (*outboundconfig.OutboundConfig, error) {
    if err := h.validateUpdateRequest(req); err != nil {
        return nil, err
    }

    c, err := h.db.OutboundConfigUpdate(ctx, id, req)
    if err != nil {
        return nil, fmt.Errorf("could not update outbound_config: %w", err)
    }
    if c != nil {
        _ = h.cacheHandler.OutboundConfigDelete(ctx, c.CustomerID)
    }
    return c, nil
}

// applyUpdateRequest applies non-nil fields from req onto c.
func applyUpdateRequest(c *outboundconfig.OutboundConfig, req *outboundconfig.UpdateRequest) {
    if req == nil {
        return
    }
    if req.Name != nil { c.Name = *req.Name }
    if req.Detail != nil { c.Detail = *req.Detail }
    if req.DestinationWhitelist != nil { c.DestinationWhitelist = *req.DestinationWhitelist }
    if req.Codecs != nil { c.Codecs = *req.Codecs }
}
```

**Step 4: Create `validate.go`**

```go
package outboundconfighandler

import (
    "fmt"
    "strings"

    outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
)

func (h *outboundConfigHandler) validateUpdateRequest(req *outboundconfig.UpdateRequest) error {
    if req == nil {
        return nil
    }
    if req.DestinationWhitelist != nil {
        if err := validateWhitelist(*req.DestinationWhitelist); err != nil {
            return err
        }
    }
    if req.Codecs != nil {
        if !outboundconfig.ValidateCodecs(*req.Codecs) {
            return fmt.Errorf("codecs must be empty or a comma-separated list of alphanumeric codec names (max 255 chars, no special chars)")
        }
    }
    return nil
}

func validateWhitelist(entries []string) error {
    seen := make(map[string]struct{})
    for _, e := range entries {
        normalized := strings.ToLower(strings.TrimSpace(e))
        if _, ok := outboundconfig.ISOCountryCodes[normalized]; !ok {
            return fmt.Errorf("invalid ISO 3166 alpha-2 country code: %q", e)
        }
        if _, dup := seen[normalized]; dup {
            return fmt.Errorf("duplicate country code after normalization: %q", normalized)
        }
        seen[normalized] = struct{}{}
    }
    return nil
}
```

**Step 5: Write validate tests in `validate_test.go`**

```go
func Test_validateWhitelist(t *testing.T) {
    tests := []struct {
        name    string
        entries []string
        wantErr bool
    }{
        {"empty list", []string{}, false},
        {"valid codes", []string{"us", "gb", "kr"}, false},
        {"uppercase normalised", []string{"US", "GB"}, false},
        {"duplicate after normalize", []string{"us", "US"}, true},
        {"invalid code", []string{"xx"}, true},
        {"empty string entry", []string{""}, true},
    }
    ...
}
```

**Step 6: Regenerate mock and verify**

```bash
cd bin-call-manager && go generate ./pkg/outboundconfighandler/... && go test ./pkg/outboundconfighandler/... && golangci-lint run ./pkg/outboundconfighandler/...
```

**Step 7: Commit**

```bash
git add bin-call-manager/pkg/outboundconfighandler/
git commit -m "NOJIRA-outbound-config

- bin-call-manager: Add outboundconfighandler (CRUD, cache, validation)"
```

---

## Task 7: `bin-call-manager` — inject handler + fill `ValidateDestination` + update `CreateCallOutgoing`

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/main.go`
- Modify: `bin-call-manager/pkg/callhandler/validate.go`
- Modify: `bin-call-manager/pkg/callhandler/validate_test.go`
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call.go`
- Modify: `bin-call-manager/cmd/call-manager/main.go`

**Step 1: Write failing validate tests first**

In `validate_test.go`, add `Test_ValidateDestination` table:

```go
func Test_ValidateDestination(t *testing.T) {
    customerID := uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001")
    internalID := cucustomer.IDCallManager

    configWithUS := &outboundconfig.OutboundConfig{
        DestinationWhitelist: []string{"us"},
    }
    emptyConfig := &outboundconfig.OutboundConfig{
        DestinationWhitelist: []string{},
    }

    tests := []struct {
        name        string
        customerID  uuid.UUID
        config      *outboundconfig.OutboundConfig
        destination commonaddress.Address
        want        bool
    }{
        {
            name:        "non-tel bypasses",
            customerID:  customerID,
            config:      emptyConfig,
            destination: commonaddress.Address{Type: commonaddress.TypeSIP, Target: "sip:foo@bar.com"},
            want:        true,
        },
        {
            name:        "internal customer bypasses",
            customerID:  internalID,
            config:      emptyConfig,
            destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+12025550100"},
            want:        true,
        },
        {
            name:        "nil config denies",
            customerID:  customerID,
            config:      nil,
            destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+12025550100"},
            want:        false,
        },
        {
            name:        "empty whitelist denies",
            customerID:  customerID,
            config:      emptyConfig,
            destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+12025550100"},
            want:        false,
        },
        {
            name:        "allowed country passes",
            customerID:  customerID,
            config:      configWithUS,
            destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+12025550100"},
            want:        true,
        },
        {
            name:        "blocked country denied",
            customerID:  customerID,
            config:      configWithUS,
            destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+442071234567"},
            want:        false,
        },
        {
            name:        "unparseable number denied (fail-closed)",
            customerID:  customerID,
            config:      configWithUS,
            destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "notanumber"},
            want:        false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()
            h := &callHandler{ /* mock deps */ }
            got := h.ValidateDestination(context.Background(), tt.customerID, tt.config, tt.destination)
            if got != tt.want {
                t.Errorf("ValidateDestination() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

Run: `cd bin-call-manager && go test ./pkg/callhandler/...`
Expected: FAIL (signature mismatch)

**Step 2: Add `outboundconfighandler` dependency to `callHandler` struct and constructor in `main.go`**

In the `callHandler` struct:
```go
outboundConfigHandler outboundconfighandler.OutboundConfigHandler
```

In `NewCallHandler(...)` signature — add:
```go
outboundConfigHandler outboundconfighandler.OutboundConfigHandler,
```

In the constructor body:
```go
outboundConfigHandler: outboundConfigHandler,
```

Add import: `"monorepo/bin-call-manager/pkg/outboundconfighandler"`

**Step 3: Update `ValidateDestination` signature and implementation in `validate.go`**

Change:
```go
func (h *callHandler) ValidateDestination(ctx context.Context, customerID uuid.UUID, destination commonaddress.Address) bool
```
To:
```go
func (h *callHandler) ValidateDestination(ctx context.Context, customerID uuid.UUID, config *outboundconfig.OutboundConfig, destination commonaddress.Address) bool {
    log := logrus.WithFields(logrus.Fields{
        "func":        "ValidateDestination",
        "customer_id": customerID,
    })

    // Only enforce on PSTN destinations
    if destination.Type != commonaddress.TypeTel {
        return true
    }
    // Internal system IDs bypass the whitelist
    if cucustomer.IsInternalSystemID(customerID) {
        return true
    }
    // Parse the destination country
    country := h.getCountry(ctx, destination.Target)
    if country == "" {
        log.Infof("Could not determine country for destination; denying (fail-closed). destination: %s", destination.Target)
        return false
    }
    // nil config or empty whitelist = deny all
    if config == nil || len(config.DestinationWhitelist) == 0 {
        log.Infof("No outbound config or empty whitelist; denying. customer_id: %s", customerID)
        return false
    }
    // Membership check
    for _, allowed := range config.DestinationWhitelist {
        if allowed == country {
            return true
        }
    }
    log.Infof("Destination country %q not in whitelist. customer_id: %s", country, customerID)
    return false
}
```

Also update the `CallHandler` interface in `main.go` to match the new signature.

Add imports: `outboundconfig "monorepo/bin-call-manager/models/outboundconfig"`, `cucustomer "monorepo/bin-customer-manager/models/customer"`

**Step 4: Update `CreateCallOutgoing` in `outgoing_call.go`**

Find the existing calls to `embedCustomerCodecs` and `ValidateDestination` (around lines 167-183) and replace:

```go
// [BEFORE - remove these lines]
metadata = embedCustomerCodecs(metadata, cu.Metadata.OutboundCodecs)
// ...
if validDestination := h.ValidateDestination(ctx, customerID, destination); !validDestination {
    return nil, fmt.Errorf("could not pass the destination validation")
}
```

With:
```go
// [NEW - fetch config once for both codec embed and whitelist enforcement]
// Skip for non-tel destinations or internal system IDs (short-circuit)
if destination.Type == commonaddress.TypeTel && !cucustomer.IsInternalSystemID(customerID) {
    outboundCfg, err := h.outboundConfigHandler.GetByCustomerID(ctx, customerID)
    if err != nil {
        log.Warnf("Could not get outbound config, defaulting to deny. err: %v", err)
        outboundCfg = nil
    }
    metadata = embedCodecs(metadata, outboundCfg)
    if !h.ValidateDestination(ctx, customerID, outboundCfg, destination) {
        log.Infof("Outbound destination not in whitelist. customer_id: %s", customerID)
        return nil, outboundconfig.ErrDestinationNotWhitelisted
    }
}
```

Also update the call to `embedCustomerCodecs` in the non-tel path if any. Check for any remaining calls to the old `embedCustomerCodecs(metadata, cu.Metadata.OutboundCodecs)` and remove them.

**Step 5: Update `codec.go` — rename and adapt `embedCustomerCodecs`**

Change `embedCustomerCodecs` to `embedCodecs`:

```go
func embedCodecs(metadata map[string]any, config *outboundconfig.OutboundConfig) map[string]any {
    if config == nil || config.Codecs == "" {
        return metadata
    }
    if metadata == nil {
        metadata = map[string]any{}
    }
    if _, alreadySet := metadata[call.MetadataKeyCodecs]; alreadySet {
        return metadata // per-call override wins
    }
    metadata[call.MetadataKeyCodecs] = config.Codecs
    return metadata
}
```

**Step 6: Update `codec_test.go`**

Replace tests that used `embedCustomerCodecs` with the new `embedCodecs` signature. Add the precedence test:

```go
{
    name:     "per-call codecs already set, not overwritten",
    metadata: map[string]any{call.MetadataKeyCodecs: "G729"},
    config:   &outboundconfig.OutboundConfig{Codecs: "PCMU"},
    want:     map[string]any{call.MetadataKeyCodecs: "G729"}, // unchanged
},
```

**Step 7: Update `cmd/call-manager/main.go` to pass `outboundconfighandler`**

Find the `callhandler.NewCallHandler(...)` call and add the new handler:

```go
outboundConfigHandler := outboundconfighandler.NewOutboundConfigHandler(utilHandler, db, cache)
// ...
callHandler := callhandler.NewCallHandler(
    reqHandler, notifyHandler, db, confbridgeHandler, channelHandler, bridgeHandler,
    recordingHandler, externalMediaHandler, groupcallHandler, recoveryHandler,
    outboundConfigHandler, // NEW
)
```

Add import: `"monorepo/bin-call-manager/pkg/outboundconfighandler"`

**Step 8: Verify the full callhandler test suite**

```bash
cd bin-call-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: all pass.

**Step 9: Commit**

```bash
git add bin-call-manager/
git commit -m "NOJIRA-outbound-config

- bin-call-manager: Add outboundconfighandler to callHandler constructor
- bin-call-manager: Fill ValidateDestination with whitelist enforcement
- bin-call-manager: Update CreateCallOutgoing to fetch OutboundConfig once
- bin-call-manager: Replace embedCustomerCodecs with embedCodecs using OutboundConfig"
```

---

## Task 8: `bin-call-manager` — listenhandler routes for OutboundConfig

**Files:**
- Create: `bin-call-manager/pkg/listenhandler/v1_outbound_configs.go`
- Create: `bin-call-manager/pkg/listenhandler/v1_outbound_configs_test.go`
- Modify: `bin-call-manager/pkg/listenhandler/main.go` (register routes)

**Step 1: Write tests first**

Follow the exact table-driven pattern from `v1_calls_test.go`. Add test cases for:
- `POST /v1/outbound_configs` → 200 (create)
- `POST /v1/outbound_configs` → 409 (conflict)
- `GET /v1/outbound_configs?customer_id=<uuid>` → 200 (list)
- `GET /v1/outbound_configs/<uuid>` → 200 (get)
- `PUT /v1/outbound_configs/<uuid>` → 200 (update)

**Step 2: Implement `v1_outbound_configs.go`**

```go
package listenhandler

// processV1OutboundConfigsPost handles POST /v1/outbound_configs
func (h *listenHandler) processV1OutboundConfigsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    var req outboundconfig.UpdateRequest
    if err := json.Unmarshal(m.Data, &req); err != nil {
        return simpleResponse(400), nil
    }

    u, _ := url.Parse(m.URI)
    customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))

    c, err := h.outboundConfigHandler.Create(ctx, customerID, &req)
    if err != nil {
        if strings.Contains(err.Error(), "Duplicate entry") {
            return simpleResponse(409), nil
        }
        return simpleResponse(500), err
    }

    data, _ := json.Marshal(outboundconfig.ConvertWebhookMessage(c))
    return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1OutboundConfigsGet handles GET /v1/outbound_configs (list by customer_id)
func (h *listenHandler) processV1OutboundConfigsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    // ... parse customer_id, pageSize, pageToken
    // ... call h.outboundConfigHandler.List(...)
    // ... return paginated result envelope
}

// processV1OutboundConfigsIDGet handles GET /v1/outbound_configs/<id>
// processV1OutboundConfigsIDPut handles PUT /v1/outbound_configs/<id>
```

**Step 3: Register routes in `main.go`**

Add to the URI dispatch table (following the existing pattern):

```go
case regexp.MustCompile(`^/v1/outbound_configs$`).MatchString(m.URI) && m.Method == "POST":
    response, err = h.processV1OutboundConfigsPost(ctx, m)
case regexp.MustCompile(`^/v1/outbound_configs$`).MatchString(m.URI) && m.Method == "GET":
    response, err = h.processV1OutboundConfigsGet(ctx, m)
case regexp.MustCompile(`^/v1/outbound_configs/[0-9a-f-]+$`).MatchString(m.URI) && m.Method == "GET":
    response, err = h.processV1OutboundConfigsIDGet(ctx, m)
case regexp.MustCompile(`^/v1/outbound_configs/[0-9a-f-]+$`).MatchString(m.URI) && m.Method == "PUT":
    response, err = h.processV1OutboundConfigsIDPut(ctx, m)
```

Also add `outboundConfigHandler` to `listenHandler` struct and `newListenHandler` constructor.

**Step 4: Verify**

```bash
cd bin-call-manager && go mod tidy && go mod vendor && go generate ./... && go test ./pkg/listenhandler/... && golangci-lint run ./pkg/listenhandler/...
```

**Step 5: Commit**

```bash
git add bin-call-manager/pkg/listenhandler/
git commit -m "NOJIRA-outbound-config

- bin-call-manager: Add OutboundConfig listenhandler routes (POST/GET/PUT)"
```

---

## Task 9: `bin-call-manager` — Prometheus metric + timeline event

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/validate.go` (emit metric on block)
- Modify: `bin-call-manager/pkg/callhandler/main.go` (register counter)

**Step 1: Register Prometheus counter in `main.go` `init()`**

```go
var outboundWhitelistRejectedTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "call_outbound_whitelist_rejected_total",
        Help: "Total number of outbound PSTN calls rejected by destination whitelist.",
    },
    []string{"destination_country"},
)

func init() {
    prometheus.MustRegister(outboundWhitelistRejectedTotal)
}
```

**Step 2: Emit counter in `ValidateDestination`** when returning false after country lookup (not for the short-circuit bypasses):

```go
outboundWhitelistRejectedTotal.WithLabelValues(country).Inc()
```

**Step 3: Emit timeline event in `CreateCallOutgoing`** when `ErrDestinationNotWhitelisted` is returned:

```go
if errors.Is(err, outboundconfig.ErrDestinationNotWhitelisted) {
    // fire-and-forget timeline event
    _ = h.reqHandler.TimelineV1EventCreate(ctx,
        commonoutline.ServiceNameCallManager,
        "call.outbound_whitelist_rejected",
        map[string]interface{}{
            "customer_id":         customerID,
            "destination_country": destinationCountry, // capture from scope
            "call_id":             id,
        },
    )
    return nil, err
}
```

**Step 4: Verify**

```bash
cd bin-call-manager && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
git add bin-call-manager/
git commit -m "NOJIRA-outbound-config

- bin-call-manager: Add whitelist rejection Prometheus counter and timeline event"
```

---

## Task 10: `bin-call-manager` — final verification

```bash
cd bin-call-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: all pass. Fix any remaining issues. Commit fixes as needed.

---

## Task 11: `bin-openapi-manager` — update OpenAPI schema

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

**Step 1: Remove `outbound_codecs` from the Customer/CustomerMetadata schema**

Find the `CustomerMetadata` (or `Metadata`) schema and remove the `outbound_codecs` property.

**Step 2: Add `OutboundConfig` schemas**

Add:
```yaml
OutboundConfig:
  type: object
  properties:
    id:
      type: string
      format: uuid
    customer_id:
      type: string
      format: uuid
    name:
      type: string
    detail:
      type: string
    destination_whitelist:
      type: array
      items:
        type: string
      description: "ISO 3166 alpha-2 country codes (lowercase). Empty = deny all PSTN calls."
    codecs:
      type: string
      description: "Comma-separated codec preference list (e.g. PCMU,PCMA,G729). Empty = server default."
    tm_create:
      type: string
      format: date-time
      nullable: true
    tm_update:
      type: string
      format: date-time
      nullable: true
    tm_delete:
      type: string
      format: date-time
      nullable: true

OutboundConfigUpdateRequest:
  type: object
  properties:
    name:
      type: string
    detail:
      type: string
    destination_whitelist:
      type: array
      items:
        type: string
      nullable: true
    codecs:
      type: string
      nullable: true
```

**Step 3: Add endpoint specs for `/v1/outbound_configs`**

```yaml
/v1/outbound_configs:
  post:
    operationId: PostOutboundConfigs
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/OutboundConfigUpdateRequest'
    responses:
      '200':
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/OutboundConfig'
  get:
    operationId: GetOutboundConfigs
    parameters:
      - name: customer_id
        in: query
        schema:
          type: string
      - name: page_size
        in: query
        schema:
          type: integer
      - name: page_token
        in: query
        schema:
          type: string
    responses:
      '200':
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/OutboundConfigList'

/v1/outbound_configs/{id}:
  get:
    operationId: GetOutboundConfigsId
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: string
    responses:
      '200':
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/OutboundConfig'
  put:
    operationId: PutOutboundConfigsId
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: string
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/OutboundConfigUpdateRequest'
    responses:
      '200':
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/OutboundConfig'
```

**Step 4: Regenerate types**

```bash
cd bin-openapi-manager && go generate ./... && go mod tidy && go mod vendor && go test ./...
```

**Step 5: Commit**

```bash
git add bin-openapi-manager/
git commit -m "NOJIRA-outbound-config

- bin-openapi-manager: Remove outbound_codecs from Customer schema
- bin-openapi-manager: Add OutboundConfig schema and endpoint specs"
```

---

## Task 12: `bin-api-manager` — server handlers + Customer fixture cleanup

**Files:**
- Create: `bin-api-manager/server/outbound_configs.go`
- Create: `bin-api-manager/server/outbound_configs_test.go`
- Modify: `bin-api-manager/pkg/servicehandler/outbound_config.go` (new)
- Modify: `bin-api-manager/server/customer_test.go` (remove `outbound_codecs`)
- Modify: `bin-api-manager/server/customers_test.go` (remove `outbound_codecs`)
- Modify: `bin-api-manager/server/service_agents_customer_test.go` (remove `outbound_codecs`)
- Run: `go generate ./...` to regenerate `gens/openapi_server/gen.go`

**Step 1: Regenerate openapi_server**

```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./...
```

**Step 2: Fix broken Customer test fixtures**

In all `*_test.go` files, remove `"outbound_codecs":""` from expected JSON strings. Use find-replace:
- Find: `,"outbound_codecs":""`
- Replace: (empty string)

Run: `cd bin-api-manager && go test ./server/...`
Expected: customer tests pass.

**Step 3: Add servicehandler methods**

Create `pkg/servicehandler/outbound_config.go`:

```go
package servicehandler

func (h *serviceHandler) OutboundConfigCreate(ctx context.Context, a *amagent.Agent, req *openapi_server.OutboundConfigUpdateRequest) (*cmoutboundconfig.WebhookMessage, error) {
    c, err := h.reqHandler.CallV1OutboundConfigCreate(ctx, a.CustomerID, convertToUpdateRequest(req))
    if err != nil {
        return nil, err
    }
    return c, nil
}

func (h *serviceHandler) OutboundConfigGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cmoutboundconfig.WebhookMessage, error) {
    c, err := h.outboundConfigGet(ctx, id)
    if err != nil {
        return nil, err
    }
    if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin) {
        return nil, fmt.Errorf("user has no permission")
    }
    return c, nil
}

// ... List, Update follow same two-level pattern
```

**Step 4: Create server handlers `outbound_configs.go`**

Follow the exact pattern of `calls.go`:

```go
func (h *server) PostOutboundConfigs(c *gin.Context) {
    a, ok := getAuthIdentity(c)
    // ...
    var req openapi_server.OutboundConfigUpdateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        abortWithError(c, cerrors.BadRequest(...))
        return
    }
    res, err := h.serviceHandler.OutboundConfigCreate(c.Request.Context(), a, &req)
    // ...
    c.JSON(200, res)
}
```

**IDOR note:** `OutboundConfigList` must use `a.CustomerID` from the JWT, not the `customer_id` query param:

```go
func (h *server) GetOutboundConfigs(c *gin.Context, params openapi_server.GetOutboundConfigsParams) {
    a, ok := getAuthIdentity(c)
    // ...
    // IDOR prevention: always use JWT customer ID, ignore params.CustomerId
    res, err := h.serviceHandler.OutboundConfigList(c.Request.Context(), a, pageSize, pageToken)
    // ...
}
```

**Step 5: Write handler tests**

Follow the exact pattern from `customers_test.go`:

```go
func TestPostOutboundConfigs(t *testing.T) {
    tests := []struct{
        name         string
        body         string
        mockRes      *cmoutboundconfig.WebhookMessage
        expectedCode int
        expectedRes  string
    }{
        {
            name:         "creates successfully",
            body:         `{"destination_whitelist":["us"],"codecs":"PCMU"}`,
            mockRes:      &cmoutboundconfig.WebhookMessage{ID: testID, CustomerID: testCustomerID, DestinationWhitelist: []string{"us"}, Codecs: "PCMU"},
            expectedCode: 200,
            expectedRes:  `{"id":"...","customer_id":"...","destination_whitelist":["us"],"codecs":"PCMU",...}`,
        },
    }
    // ...
}
```

**Step 6: Add `requesthandler` methods for OutboundConfig RPC calls**

In `bin-common-handler/pkg/requesthandler/` (if this is where RPC client calls live), add:
- `CallV1OutboundConfigCreate`
- `CallV1OutboundConfigGet`
- `CallV1OutboundConfigList`
- `CallV1OutboundConfigUpdate`

These follow the pattern of existing `CallV1*` methods — send a RabbitMQ request to call-manager's queue.

> Note: `bin-common-handler` receives new methods only because it's the shared requesthandler used by many services. This is the existing pattern — not a bin-common-handler model/domain package.

**Step 7: Verify**

```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 8: Commit**

```bash
git add bin-api-manager/ bin-common-handler/
git commit -m "NOJIRA-outbound-config

- bin-api-manager: Add OutboundConfig server handlers (POST/GET/PUT)
- bin-api-manager: Remove outbound_codecs from Customer test fixtures
- bin-api-manager: Enforce IDOR prevention on customer_id query param
- bin-common-handler: Add CallV1OutboundConfig* requesthandler methods"
```

---

## Task 13: RST docs + Sphinx rebuild

**Files:**
- Modify: `bin-api-manager/docsdev/source/customer_overview.rst`
- Modify: `bin-api-manager/docsdev/source/customer_struct_customer.rst`
- Create: `bin-api-manager/docsdev/source/outbound_config.rst`
- Create: `bin-api-manager/docsdev/source/outbound_config_overview.rst`
- Create: `bin-api-manager/docsdev/source/outbound_config_tutorial.rst`
- Create: `bin-api-manager/docsdev/source/outbound_config_struct_outbound_config.rst`
- Modify: `bin-api-manager/docsdev/source/index.rst` (add to toc)

**Step 1: Remove `outbound_codecs` from customer RST files**

In `customer_overview.rst`: remove the `outbound_codecs` field examples and the explanation paragraph (search for `outbound_codecs`).

In `customer_struct_customer.rst`: remove the `outbound_codecs` row from the field table.

**Step 2: Create `outbound_config_overview.rst`**

Follow the AI-Native RST guidelines from `bin-api-manager/CLAUDE.md`. Include:
- AI Context block (complexity: Low, cost: Free, async: No)
- Description of always-on enforcement and deploy-day behaviour
- Whitelist semantics (empty = deny all; must add countries explicitly)
- `outbound_codecs` migration note (field moved from Customer to OutboundConfig)
- AI Implementation Hint: "Before making an outbound PSTN call, ensure the destination country is in the customer's OutboundConfig.destination_whitelist. If not, the call will return 400."
- Error code table (400 when country not whitelisted)

**Step 3: Create `outbound_config_struct_outbound_config.rst`**

Document only the `WebhookMessage` fields:
- `id` (UUID) — server-generated
- `customer_id` (UUID) — source: `GET https://api.voipbin.net/v1.0/customer`
- `name` (string, optional)
- `detail` (string, optional)
- `destination_whitelist` (array of ISO 3166 alpha-2 strings) — empty = deny all PSTN calls
- `codecs` (string) — comma-separated (e.g. `PCMU,PCMA,G729`); empty = server default
- `tm_create`, `tm_update`, `tm_delete` (ISO 8601 timestamps, nullable)

**Step 4: Create `outbound_config_tutorial.rst`**

Include:
- Prerequisites block
- Step-by-step: create OutboundConfig → add countries → verify a call is allowed
- Complete request + response examples
- Note about deploy-day warning (new customers blocked until configured)
- Error handling (400 with fix guidance)

**Step 5: Clean rebuild and force-add**

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-outbound-config
git add -f bin-api-manager/docsdev/build/
git add bin-api-manager/docsdev/source/
```

**Step 6: Commit**

```bash
git commit -m "NOJIRA-outbound-config

- bin-api-manager: Add OutboundConfig RST docs (overview, tutorial, struct)
- bin-api-manager: Remove outbound_codecs from customer RST docs
- bin-api-manager: Rebuild Sphinx HTML"
```

---

## Task 14: `monitoring/api-validator` — E2E tests

**Files:**
- Create: `monitoring/api-validator/tests/outbound_configs_test.go` (or `.py` — follow the existing language/framework used in api-validator)

**Step 1: Inspect the existing api-validator test pattern**

```bash
ls ~/gitvoipbin/monorepo/monitoring/api-validator/
```

Follow the exact file/function naming, auth header pattern, and assertion style used in existing tests.

**Step 2: Write tests**

```
POST /v1/outbound_configs
  → 201 (or 200); assert id, customer_id, destination_whitelist == [], codecs == ""

GET /v1/outbound_configs/<id>
  → 200; assert same fields as create response

GET /v1/outbound_configs?customer_id=<jwt-customer-id>
  → 200; assert result is array with 1 item

PUT /v1/outbound_configs/<id>  body: {"destination_whitelist":["us","gb"]}
  → 200; assert destination_whitelist == ["us","gb"], codecs unchanged

PUT /v1/outbound_configs/<id>  body: {"codecs":"PCMU"}
  → 200; assert codecs == "PCMU", destination_whitelist unchanged

POST /v1/outbound_configs (duplicate — same customer)
  → 409

GET /v1/outbound_configs (using wrong customer's JWT)
  → 200 but returns that customer's own config (IDOR validation)
```

**Step 3: Run locally (if api-validator has a local run mode)**

```bash
cd monitoring/api-validator && <run command per existing test framework>
```

**Step 4: Commit**

```bash
git add monitoring/api-validator/
git commit -m "NOJIRA-outbound-config

- monitoring: Add api-validator E2E tests for OutboundConfig CRUD"
```

---

## Task 15: Final verification across all touched services

Run the full verification workflow in each service directory that was touched:

```bash
# bin-dbscheme-manager (Python — check migration chain only)
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-outbound-config/bin-dbscheme-manager/bin-manager
alembic -c alembic.ini heads  # expect exactly 1 head

# bin-call-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-outbound-config/bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-customer-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-outbound-config/bin-customer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-openapi-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-outbound-config/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-api-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-outbound-config/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-common-handler (if requesthandler was touched)
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-outbound-config/bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Fix any issues. Commit as `"NOJIRA-outbound-config: fix verification issues"`.

---

## Task 16: Pre-PR conflict check

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-outbound-config
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

If conflicts exist: rebase, resolve, re-run verification.
If clean: proceed to PR.
