# Access Key Token Hashing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace plain-text access key token storage with SHA-256 hashing, add `vb_` prefix to tokens, and show tokens only once at creation time.

**Architecture:** Hash tokens with SHA-256 before DB storage. Auth flow hashes incoming token and queries by hash. New columns `token_hash` (CHAR(64)) and `token_prefix` (VARCHAR(16)) replace old `token` column. Token format: `vb_<32-char base64url random>`.

**Tech Stack:** Go, SHA-256 (`crypto/sha256`), MySQL (Alembic migration), Squirrel query builder

**Worktree:** `/home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-hash-accesskey-tokens`

---

### Task 1: Add SHA-256 hex hash utility to bin-common-handler

**Files:**
- Create: `bin-common-handler/pkg/utilhandler/hash_sha256.go`
- Create: `bin-common-handler/pkg/utilhandler/hash_sha256_test.go`
- Modify: `bin-common-handler/pkg/utilhandler/main.go:13-46` (add to UtilHandler interface)

**Step 1: Write the failing test**

Create `bin-common-handler/pkg/utilhandler/hash_sha256_test.go`:

```go
package utilhandler

import (
	"testing"
)

func Test_HashSHA256Hex(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "known vector",
			input:  "hello",
			expect: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
		{
			name:   "empty string",
			input:  "",
			expect: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:   "token format",
			input:  "vb_a3Bf9xKmPq2nR7sT4wYzLp8mN5qR1xW",
			expect: "", // will compute and verify length is 64
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := HashSHA256Hex(tt.input)

			if len(res) != 64 {
				t.Errorf("Expected 64 char hex string, got %d chars", len(res))
			}

			if tt.expect != "" && res != tt.expect {
				t.Errorf("Wrong hash.\nexpect: %s\ngot:    %s", tt.expect, res)
			}
		})
	}
}

func Test_HashSHA256Hex_Deterministic(t *testing.T) {
	input := "vb_testtoken123456789"
	hash1 := HashSHA256Hex(input)
	hash2 := HashSHA256Hex(input)

	if hash1 != hash2 {
		t.Errorf("Hash is not deterministic. got %s and %s", hash1, hash2)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd bin-common-handler && go test -v ./pkg/utilhandler/ -run Test_HashSHA256`
Expected: FAIL with "undefined: HashSHA256Hex"

**Step 3: Write minimal implementation**

Create `bin-common-handler/pkg/utilhandler/hash_sha256.go`:

```go
package utilhandler

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashSHA256Hex returns the lowercase hex-encoded SHA-256 hash of the input string.
func (h *utilHandler) HashSHA256Hex(input string) string {
	return HashSHA256Hex(input)
}

// HashSHA256Hex returns the lowercase hex-encoded SHA-256 hash of the input string.
func HashSHA256Hex(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
```

**Step 4: Add to UtilHandler interface**

In `bin-common-handler/pkg/utilhandler/main.go`, add to the `UtilHandler` interface:

```go
// hash
HashCheckPassword(password, hashString string) bool
HashGenerate(org string, cost int) (string, error)
HashSHA256Hex(input string) string  // <-- add this line
```

**Step 5: Run tests to verify they pass**

Run: `cd bin-common-handler && go test -v ./pkg/utilhandler/ -run Test_HashSHA256`
Expected: PASS

**Step 6: Regenerate mocks and run full verification**

Run: `cd bin-common-handler && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: PASS

**Step 7: Commit**

```bash
git add bin-common-handler/pkg/utilhandler/hash_sha256.go bin-common-handler/pkg/utilhandler/hash_sha256_test.go bin-common-handler/pkg/utilhandler/main.go bin-common-handler/pkg/utilhandler/mock_main.go
git commit -m "NOJIRA-hash-accesskey-tokens

- bin-common-handler: Add HashSHA256Hex utility for access key token hashing"
```

---

### Task 2: Update accesskey model with token_hash and token_prefix fields

**Files:**
- Modify: `bin-customer-manager/models/accesskey/accesskey.go:10-24`
- Modify: `bin-customer-manager/models/accesskey/field.go:6-23`
- Modify: `bin-customer-manager/models/accesskey/webhook.go:11-44`

**Step 1: Update the Accesskey struct**

In `bin-customer-manager/models/accesskey/accesskey.go`, replace the struct:

```go
type Accesskey struct {
	ID         uuid.UUID `json:"id" db:"id,uuid"`
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"`

	Name   string `json:"name,omitempty" db:"name"`
	Detail string `json:"detail,omitempty" db:"detail"`

	TokenHash   string `json:"-" db:"token_hash"`
	TokenPrefix string `json:"token_prefix" db:"token_prefix"`

	TMExpire *time.Time `json:"tm_expire" db:"tm_expire"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
```

Key changes:
- Remove `Token string` field
- Add `TokenHash string` with `json:"-"` (never serialized)
- Add `TokenPrefix string` with `json:"token_prefix"`

**Step 2: Update the Field constants**

In `bin-customer-manager/models/accesskey/field.go`, replace:

```go
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldName   Field = "name"
	FieldDetail Field = "detail"

	FieldTokenHash   Field = "token_hash"
	FieldTokenPrefix Field = "token_prefix"

	FieldTMExpire Field = "tm_expire"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
```

Key changes:
- Remove `FieldToken`
- Add `FieldTokenHash` and `FieldTokenPrefix`

**Step 3: Update the WebhookMessage struct**

In `bin-customer-manager/models/accesskey/webhook.go`, update:

```go
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	Token       string `json:"token,omitempty"`
	TokenPrefix string `json:"token_prefix,omitempty"`

	TMExpire *time.Time `json:"tm_expire"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}
```

Update `ConvertWebhookMessage`:

```go
func (h *Accesskey) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,

		Name:   h.Name,
		Detail: h.Detail,

		TokenPrefix: h.TokenPrefix,

		TMExpire: h.TMExpire,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}
```

Key changes:
- `Token` field kept in WebhookMessage as `omitempty` — only populated on Create
- `TokenPrefix` added
- `ConvertWebhookMessage` no longer copies Token (it's not in the DB struct)

**Step 4: Update field_test.go**

In `bin-customer-manager/models/accesskey/field_test.go`, update the test case:

Replace `{"field_token", FieldToken, "token"}` with:
```go
{"field_token_hash", FieldTokenHash, "token_hash"},
{"field_token_prefix", FieldTokenPrefix, "token_prefix"},
```

**Step 5: Verify compilation**

Run: `cd bin-customer-manager && go build ./...`
Expected: Compilation errors in files that reference `Token` or `FieldToken` — that's expected, we'll fix them in subsequent tasks.

**Step 6: Commit model changes only**

```bash
git add bin-customer-manager/models/accesskey/
git commit -m "NOJIRA-hash-accesskey-tokens

- bin-customer-manager: Update accesskey model with token_hash and token_prefix fields"
```

---

### Task 3: Update customer-manager accesskeyhandler to hash tokens on create

**Files:**
- Modify: `bin-customer-manager/pkg/accesskeyhandler/main.go:17-19` (constant)
- Modify: `bin-customer-manager/pkg/accesskeyhandler/db.go:57-73,75-126` (GetByToken + Create)
- Modify: `bin-customer-manager/pkg/accesskeyhandler/db_test.go`

**Step 1: Update constants and imports**

In `bin-customer-manager/pkg/accesskeyhandler/main.go`, change:

```go
const (
	defaultLenToken       = 32
	defaultTokenPrefix    = "vb_"
	defaultTokenPrefixLen = 11 // len("vb_") + 8 random chars
)
```

**Step 2: Update the Create function**

In `bin-customer-manager/pkg/accesskeyhandler/db.go`, update the `Create` method:

```go
func (h *accesskeyHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	expire time.Duration,
) (*accesskey.Accesskey, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "Create",
		"name":   name,
		"detail": detail,
		"expire": expire,
	})
	log.Debug("Creating a new accesskey.")

	id := h.utilHandler.UUIDCreate()
	tmExpire := h.utilHandler.TimeNowAdd(expire)

	// Generate token with vb_ prefix
	randomPart, err := h.utilHandler.StringGenerateRandom(defaultLenToken)
	if err != nil {
		log.Errorf("Could not generate the token. err: %v", err)
		return nil, fmt.Errorf("could not generate token: %w", err)
	}
	token := defaultTokenPrefix + randomPart

	// Hash the token for storage
	tokenHash := h.utilHandler.HashSHA256Hex(token)

	// Extract prefix for display
	tokenPrefix := token
	if len(tokenPrefix) > defaultTokenPrefixLen {
		tokenPrefix = tokenPrefix[:defaultTokenPrefixLen]
	}

	a := &accesskey.Accesskey{
		ID:         id,
		CustomerID: customerID,

		Name:   name,
		Detail: detail,

		TokenHash:   tokenHash,
		TokenPrefix: tokenPrefix,

		TMExpire: tmExpire,
	}

	if err := h.db.AccesskeyCreate(ctx, a); err != nil {
		log.Errorf("Could not create a new accesskey. err: %v", err)
		return nil, err
	}

	res, err := h.db.AccesskeyGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created accesskey info. err: %v", err)
		return nil, err
	}

	// notify
	h.notifyHandler.PublishEvent(ctx, accesskey.EventTypeAccesskeyCreated, res)

	return res, nil
}
```

Note: The raw `token` is NOT stored in the returned `Accesskey` struct (since `Token` field no longer exists in the struct). The caller (api-manager) will need a separate mechanism to get the raw token. We'll handle this by returning the raw token alongside the Accesskey. See Task 5.

**Wait — design revision needed.** The `Create` function returns `*accesskey.Accesskey`, which no longer has a `Token` field. The raw token must be returned to the API layer. Two options:

**Option A:** Add a transient `RawToken` field to the struct (not persisted, not in DB).
**Option B:** Change the return signature to return both.

**Option A is simpler** since it doesn't change the interface that `requesthandler` uses across RPC:

In `bin-customer-manager/models/accesskey/accesskey.go`, add a transient field:

```go
// RawToken holds the plain-text token temporarily during creation.
// It is NOT stored in the database (db:"-") and NOT serialized to JSON by default (json:"-").
// The webhook conversion explicitly copies it when present.
RawToken string `json:"-" db:"-"`
```

Then in `Create`:
```go
// Set the raw token on the result for one-time return
res.RawToken = token
```

And in `ConvertWebhookMessage`:
```go
func (h *Accesskey) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		// ... existing fields ...
		Token:       h.RawToken, // only populated during creation
		TokenPrefix: h.TokenPrefix,
		// ...
	}
}
```

**Step 3: Update GetByToken to use hash lookup**

In `bin-customer-manager/pkg/accesskeyhandler/db.go`, rename and update:

```go
// GetByToken returns accesskey info by hashing the provided token and looking up by hash.
func (h *accesskeyHandler) GetByToken(ctx context.Context, token string) (*accesskey.Accesskey, error) {
	log := logrus.WithField("func", "GetByToken")

	tokenHash := h.utilHandler.HashSHA256Hex(token)

	filter := map[accesskey.Field]any{
		accesskey.FieldTokenHash: tokenHash,
	}

	tmp, err := h.db.AccesskeyList(ctx, 100, "", filter)
	if err != nil || len(tmp) == 0 || len(tmp) > 1 {
		log.Errorf("Could not get accesskeys info. err: %v", err)
		return nil, err
	}

	res := tmp[0]
	return res, nil
}
```

**Step 4: Update the Create test**

In `bin-customer-manager/pkg/accesskeyhandler/db_test.go`, update `Test_Create`:

```go
func Test_Create(t *testing.T) {
	expireTime := time.Date(2024, 4, 4, 7, 15, 59, 233415000, time.UTC)

	tests := []struct {
		name string

		customerID uuid.UUID
		userName   string
		detail     string
		expire     time.Duration

		responseUUID       uuid.UUID
		responseRandomPart string
		responseExpire     *time.Time
		expectTokenHash    string
		expectTokenPrefix  string
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("58d43704-a75e-11ef-b9b7-279abaf5dda3"),
			userName:   "test1",
			detail:     "detail1",
			expire:     time.Duration(time.Hour * 24 * 365),

			responseUUID:       uuid.FromStringOrNil("5947fe5a-a75e-11ef-8595-878f92d49c95"),
			responseRandomPart: "a3Bf9xKmPq2nR7sT4wYzLp8mN5qR1xW",
			responseExpire:     &expireTime,
			expectTokenHash:    utilhandler.HashSHA256Hex("vb_a3Bf9xKmPq2nR7sT4wYzLp8mN5qR1xW"),
			expectTokenPrefix:  "vb_a3Bf9xKm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &accesskeyHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockUtil.EXPECT().TimeNowAdd(tt.expire).Return(tt.responseExpire)
			mockUtil.EXPECT().StringGenerateRandom(defaultLenToken).Return(tt.responseRandomPart, nil)
			mockUtil.EXPECT().HashSHA256Hex("vb_" + tt.responseRandomPart).Return(tt.expectTokenHash)

			expectAccesskey := &accesskey.Accesskey{
				ID:         tt.responseUUID,
				CustomerID: tt.customerID,
				Name:       tt.userName,
				Detail:     tt.detail,
				TokenHash:   tt.expectTokenHash,
				TokenPrefix: tt.expectTokenPrefix,
				TMExpire:   tt.responseExpire,
			}

			mockDB.EXPECT().AccesskeyCreate(ctx, expectAccesskey).Return(nil)

			returnedAccesskey := &accesskey.Accesskey{
				ID:          tt.responseUUID,
				TokenPrefix: tt.expectTokenPrefix,
			}
			mockDB.EXPECT().AccesskeyGet(ctx, tt.responseUUID).Return(returnedAccesskey, nil)
			mockNotify.EXPECT().PublishEvent(ctx, accesskey.EventTypeAccesskeyCreated, gomock.Any()).Return()

			res, err := h.Create(ctx, tt.customerID, tt.userName, tt.detail, tt.expire)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}

			// Verify the raw token is set for one-time return
			expectedRawToken := "vb_" + tt.responseRandomPart
			if res.RawToken != expectedRawToken {
				t.Errorf("RawToken not set.\nexpect: %s\ngot:    %s", expectedRawToken, res.RawToken)
			}
		})
	}
}
```

**Step 5: Update the GetByToken test**

In `bin-customer-manager/pkg/accesskeyhandler/db_test.go`, update `Test_GetByToken`:

```go
func Test_GetByToken(t *testing.T) {
	tests := []struct {
		name string

		token string

		responseAccesskeys []*accesskey.Accesskey
		expectFilter       map[accesskey.Field]any
		expectRes          *accesskey.Accesskey
	}{
		{
			name: "normal",

			token: "vb_a3Bf9xKmPq2nR7sT4wYzLp8mN5qR1xW",

			responseAccesskeys: []*accesskey.Accesskey{
				{
					ID: uuid.FromStringOrNil("8061b60a-ab11-11ef-8cd0-4721783d6664"),
				},
			},
			expectFilter: map[accesskey.Field]any{
				accesskey.FieldTokenHash: utilhandler.HashSHA256Hex("vb_a3Bf9xKmPq2nR7sT4wYzLp8mN5qR1xW"),
			},
			expectRes: &accesskey.Accesskey{
				ID: uuid.FromStringOrNil("8061b60a-ab11-11ef-8cd0-4721783d6664"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &accesskeyHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			tokenHash := utilhandler.HashSHA256Hex(tt.token)
			mockUtil.EXPECT().HashSHA256Hex(tt.token).Return(tokenHash)
			mockDB.EXPECT().AccesskeyList(ctx, gomock.Any(), "", tt.expectFilter).Return(tt.responseAccesskeys, nil)

			res, err := h.GetByToken(ctx, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got:%v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
```

**Step 6: Update error tests that reference Token/FieldToken**

Update `Test_Create_Error` to mock `HashSHA256Hex` and use `defaultLenToken = 32`.

**Step 7: Run tests**

Run: `cd bin-customer-manager && go test -v ./pkg/accesskeyhandler/...`
Expected: PASS (may need to fix compilation errors from FieldToken references in dbhandler first — see Task 4)

**Step 8: Commit**

```bash
git add bin-customer-manager/models/accesskey/ bin-customer-manager/pkg/accesskeyhandler/
git commit -m "NOJIRA-hash-accesskey-tokens

- bin-customer-manager: Hash tokens with SHA-256 on create, lookup by hash"
```

---

### Task 4: Update customer-manager dbhandler for token_hash column

**Files:**
- Modify: `bin-customer-manager/pkg/dbhandler/accesskey_test.go`

The DB handler uses `commondatabasehandler.GetDBFields()` and `commondatabasehandler.ScanRow()` which rely on the struct's `db:` tags. Since we changed the struct tags in Task 2 (removed `db:"token"`, added `db:"token_hash"` and `db:"token_prefix"`), the DB handler automatically uses the new columns. No code changes needed in `accesskey.go` — just test updates.

**Step 1: Update dbhandler test**

In `bin-customer-manager/pkg/dbhandler/accesskey_test.go`, update any test that uses `FieldToken`:

Replace:
```go
accesskey.FieldToken: "d09df996-ab0f-11ef-862c-e3a5ac697296",
```
With:
```go
accesskey.FieldTokenHash: "d09df996ab0f11ef862ce3a5ac697296abcdef1234567890abcdef1234567890",
```

**Step 2: Run tests**

Run: `cd bin-customer-manager && go generate ./... && go test -v ./pkg/dbhandler/...`
Expected: PASS

**Step 3: Run full verification**

Run: `cd bin-customer-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: PASS

**Step 4: Commit**

```bash
git add bin-customer-manager/
git commit -m "NOJIRA-hash-accesskey-tokens

- bin-customer-manager: Update dbhandler tests for token_hash column"
```

---

### Task 5: Update api-manager auth and accesskey handlers

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/accesskeys.go:87-113` (AccesskeyRawGetByToken)
- Modify: `bin-api-manager/pkg/servicehandler/accesskeys.go:31-61` (AccesskeyCreate)

**Step 1: Update AccesskeyRawGetByToken to use token_hash**

In `bin-api-manager/pkg/servicehandler/accesskeys.go`, update `AccesskeyRawGetByToken`:

```go
func (h *serviceHandler) AccesskeyRawGetByToken(ctx context.Context, token string) (*csaccesskey.Accesskey, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "AccesskeyGetByToken",
	})

	// Hash the token before lookup
	tokenHash := h.utilHandler.HashSHA256Hex(token)

	// filters
	filters := map[csaccesskey.Field]any{
		csaccesskey.FieldTokenHash: tokenHash,
		csaccesskey.FieldDeleted:   false,
	}

	tmps, err := h.reqHandler.CustomerV1AccesskeyList(ctx, "", 10, filters)
	if err != nil {
		log.Infof("Could not get accesskeys info. err: %v", err)
		return nil, err
	}

	if len(tmps) == 0 {
		return nil, fmt.Errorf("not found")
	}

	res := tmps[0]
	return &res, nil
}
```

**Step 2: Update AccesskeyCreate to return token in webhook message**

In `bin-api-manager/pkg/servicehandler/accesskeys.go`, update `AccesskeyCreate`:

```go
func (h *serviceHandler) AccesskeyCreate(ctx context.Context, a *amagent.Agent, name string, detail string, expire int32) (*csaccesskey.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "AccesskeyCreate",
		"agent":  a,
		"name":   name,
		"detail": detail,
		"expire": expire,
	})
	log.Debug("Creating a new accesskey.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	if expire < 86400 {
		return nil, fmt.Errorf("wrong expiration")
	}

	tmp, err := h.reqHandler.CustomerV1AccesskeyCreate(ctx, a.CustomerID, name, detail, expire)
	if err != nil {
		log.Errorf("Could not create accesskey. err: %v", err)
		return nil, err
	}
	log.WithField("accesskey", tmp).Debugf("Created accesskey. accesskey_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
```

Note: The `ConvertWebhookMessage()` already handles the `RawToken` → `Token` mapping (from Task 3). The raw token travels via RPC in the `RawToken` field. But wait — RPC serializes via JSON, and `RawToken` has `json:"-"`. This is a problem.

**Design correction:** The `RawToken` field needs to be serializable over RPC. Change the tag to `json:"raw_token,omitempty"` in the struct. It still won't appear in API responses because the API returns `WebhookMessage`, not `Accesskey`.

In `bin-customer-manager/models/accesskey/accesskey.go`:
```go
RawToken string `json:"raw_token,omitempty" db:"-"`
```

**Step 3: Run tests**

Run: `cd bin-api-manager && go mod tidy && go mod vendor && go test -v ./pkg/servicehandler/... -run TestAccesskey`
Expected: PASS (after updating test mocks to use new field names)

**Step 4: Update api-manager accesskey tests**

Update any tests in `bin-api-manager/pkg/servicehandler/` that reference `csaccesskey.FieldToken` to use `csaccesskey.FieldTokenHash` with a hashed value.

**Step 5: Run full verification**

Run: `cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: PASS

**Step 6: Commit**

```bash
git add bin-api-manager/pkg/servicehandler/
git commit -m "NOJIRA-hash-accesskey-tokens

- bin-api-manager: Hash tokens before lookup in auth flow
- bin-api-manager: Return token only on creation via WebhookMessage"
```

---

### Task 6: Update OpenAPI schema

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml:2861-2910` (CustomerManagerAccesskey schema)

**Step 1: Update the schema**

In `bin-openapi-manager/openapi/openapi.yaml`, update the `CustomerManagerAccesskey` schema:

```yaml
    CustomerManagerAccesskey:
      type: object
      properties:
        id:
          type: string
          format: uuid
          x-go-type: string
          description: The unique identifier of the access key.
          example: "550e8400-e29b-41d4-a716-446655440000"
        customer_id:
          type: string
          format: uuid
          x-go-type: string
          description: "The unique identifier of the customer. Returned from the `GET /customers` response."
          example: "7c4d2f3a-1b8e-4f5c-9a6d-3e2f1a0b4c5d"
        name:
          type: string
          description: Name of the access key.
          example: "Production API Key"
        detail:
          type: string
          description: Additional details about the access key.
          example: "API key for production environment"
        token:
          type: string
          description: "The access key token. Only returned once at creation time via `POST /accesskeys`. Subsequent `GET` requests will not include this field. Store it securely immediately after creation."
          example: "vb_a3Bf9xKmPq2nR7sT4wYzLp8mN5qR1xW"
        token_prefix:
          type: string
          description: "A short prefix of the access key token for identification purposes. Always returned in `GET` responses. Example: `vb_a3Bf9xKm`."
          example: "vb_a3Bf9xKm"
        tm_expire:
          type: string
          format: date-time
          x-go-type: string
          description: Timestamp when the access key expires.
          example: "2027-01-15T09:30:00.000000Z"
        tm_create:
          type: string
          format: date-time
          x-go-type: string
          description: Timestamp when the access key was created.
          example: "2026-01-15T09:30:00.000000Z"
        tm_update:
          type: string
          format: date-time
          x-go-type: string
          description: Timestamp when the access key was last updated.
          example: "2026-01-15T09:30:00.000000Z"
        tm_delete:
          type: string
          format: date-time
          x-go-type: string
          description: Timestamp when the access key was deleted.
```

**Step 2: Regenerate OpenAPI types**

Run: `cd bin-openapi-manager && go generate ./...`

**Step 3: Regenerate api-manager server code**

Run: `cd bin-api-manager && go mod tidy && go mod vendor && go generate ./...`

**Step 4: Run verification for both**

Run:
```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
git add bin-openapi-manager/openapi/ bin-openapi-manager/gens/ bin-api-manager/gens/
git commit -m "NOJIRA-hash-accesskey-tokens

- bin-openapi-manager: Add token_prefix field to CustomerManagerAccesskey schema
- bin-openapi-manager: Update token field description for one-time visibility
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

### Task 7: Create Alembic migration

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<revision>_customer_accesskeys_hash_token.py`

**Step 1: Generate migration file**

Run: `cd bin-dbscheme-manager/bin-manager/main && alembic -c alembic.ini revision -m "customer_accesskeys_hash_token"`

**Step 2: Edit the migration**

```python
"""customer_accesskeys_hash_token

Revision ID: <auto-generated>
Revises: <auto-determined>
Create Date: <auto-generated>

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = '<auto-generated>'
down_revision = '<auto-determined>'
branch_labels = None
depends_on = None


def upgrade():
    # Add new columns
    op.execute("""ALTER TABLE customer_accesskeys ADD COLUMN token_hash CHAR(64);""")
    op.execute("""ALTER TABLE customer_accesskeys ADD COLUMN token_prefix VARCHAR(16);""")

    # Backfill: hash existing plain-text tokens
    op.execute("""
        UPDATE customer_accesskeys
        SET token_hash = SHA2(token, 256),
            token_prefix = LEFT(token, 8)
        WHERE token IS NOT NULL AND token != '';
    """)

    # Add index on token_hash
    op.execute("""CREATE INDEX idx_customer_accesskeys_token_hash ON customer_accesskeys(token_hash);""")

    # Drop old token index and column
    op.execute("""DROP INDEX idx_customer_accesskeys_token ON customer_accesskeys;""")
    op.execute("""ALTER TABLE customer_accesskeys DROP COLUMN token;""")


def downgrade():
    # Add back token column
    op.execute("""ALTER TABLE customer_accesskeys ADD COLUMN token VARCHAR(1023);""")
    op.execute("""CREATE INDEX idx_customer_accesskeys_token ON customer_accesskeys(token);""")

    # Drop new columns
    op.execute("""DROP INDEX idx_customer_accesskeys_token_hash ON customer_accesskeys;""")
    op.execute("""ALTER TABLE customer_accesskeys DROP COLUMN token_hash;""")
    op.execute("""ALTER TABLE customer_accesskeys DROP COLUMN token_prefix;""")
```

**IMPORTANT:** Do NOT run `alembic upgrade`. Only create the migration file.

**Step 3: Commit**

```bash
git add bin-dbscheme-manager/
git commit -m "NOJIRA-hash-accesskey-tokens

- bin-dbscheme-manager: Add migration to hash accesskey tokens with SHA-256"
```

---

### Task 8: Update RST documentation

**Files:**
- Modify: `bin-api-manager/docsdev/source/accesskey_struct.rst`
- Modify: `bin-api-manager/docsdev/source/accesskey_tutorial.rst`
- Modify: `bin-api-manager/docsdev/source/accesskey_overview.rst`

**Step 1: Update accesskey_struct.rst**

Replace the example JSON and field descriptions:

```rst
.. _accesskey-struct:

Struct
======

.. _accesskey-struct-accesskey:

Accesskey
---------

.. code::

   {
      "id": "5f1f8f7e-9b3d-4c60-8465-b69e9f28b6db",
      "customer_id": "a1d9b2cd-4578-4b23-91b6-5f5ec4a2f840",
      "name": "My API Key",
      "detail": "For accessing reporting APIs",
      "token_prefix": "vb_a3Bf9xKm",
      "tm_expire": "2027-04-28T01:41:40.503790Z",
      "tm_create": "2026-04-28T01:41:40.503790Z",
      "tm_update": "2026-04-28T01:41:40.503790Z",
      "tm_delete": "9999-01-01T00:00:00.000000Z"
   }

* ``id`` (UUID): The accesskey's unique identifier. Returned when creating an accesskey via ``POST /accesskeys`` or when listing accesskeys via ``GET /accesskeys``.
* ``customer_id`` (UUID): The customer that owns this accesskey. Obtained from the ``id`` field of ``GET /customers``.
* ``name`` (String, Optional): An optional human-readable name for the accesskey. Useful for identification in multi-key environments.
* ``detail`` (String, Optional): An optional description of the accesskey's intended use or purpose.
* ``token`` (String, Optional): The full API token credential. **Only returned once at creation time** via ``POST /accesskeys``. Store it securely and immediately. If lost, delete the key and create a new one.
* ``token_prefix`` (String): A short prefix of the token (e.g., ``vb_a3Bf9xKm``) for identification. Always returned in ``GET`` responses.
* ``tm_expire`` (String, ISO 8601): Timestamp when the accesskey will expire. After this time, the key will no longer be valid for authentication.
* ``tm_create`` (String, ISO 8601): Timestamp when the accesskey was created.
* ``tm_update`` (String, ISO 8601): Timestamp when the accesskey was last updated.
* ``tm_delete`` (String, ISO 8601): Timestamp when the accesskey was deleted, if applicable.

.. note:: **AI Implementation Hint**

   A ``tm_delete`` value of ``9999-01-01T00:00:00.000000Z`` indicates the accesskey has not been deleted and is still active. This sentinel value is used across all VoIPBIN resources to represent "not yet occurred."

.. note:: **AI Implementation Hint**

   The ``token`` field is only present in the response to ``POST /accesskeys`` (creation). All subsequent ``GET /accesskeys`` and ``GET /accesskeys/{id}`` responses will NOT include the ``token`` field. Use ``token_prefix`` to identify which key is which. If the token is lost, delete the key via ``DELETE /accesskeys/{id}`` and create a new one.
```

**Step 2: Update accesskey_tutorial.rst**

Update all example responses:
- Create response: show `token` with `vb_` prefix AND `token_prefix`
- List/Get responses: show `token_prefix` only, no `token`

Add AI Implementation Hint about one-time visibility.

**Step 3: Update accesskey_overview.rst**

- Update example query parameter: `accesskey=vb_a3Bf9xKmPq2nR7sT4wYzLp8mN5qR1xW`
- Add security note about server-side hashing
- Update Authentication section

**Step 4: Rebuild HTML**

Run: `cd bin-api-manager/docsdev && python3 -m sphinx -M html source build`

**Step 5: Commit**

```bash
git add bin-api-manager/docsdev/source/accesskey_struct.rst
git add bin-api-manager/docsdev/source/accesskey_tutorial.rst
git add bin-api-manager/docsdev/source/accesskey_overview.rst
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-hash-accesskey-tokens

- bin-api-manager: Update accesskey RST docs for token hashing
- bin-api-manager: Document one-time token visibility and token_prefix field
- bin-api-manager: Rebuild HTML documentation"
```

---

### Task 9: Final verification and PR

**Step 1: Run full verification for all changed services**

```bash
# bin-common-handler
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-customer-manager
cd bin-customer-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-api-manager
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# bin-openapi-manager
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Check for conflicts with main**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-hash-accesskey-tokens
```

Create PR with title: `NOJIRA-hash-accesskey-tokens`

Body:
```
Hash access key tokens with SHA-256 before database storage. Tokens are
shown once at creation time and cannot be retrieved again. Adds vb_ prefix
for secret scanning identification.

- bin-common-handler: Add HashSHA256Hex utility function
- bin-customer-manager: Replace plain-text token with token_hash (SHA-256) and token_prefix
- bin-customer-manager: Update Create to generate vb_-prefixed tokens and hash before storage
- bin-customer-manager: Update GetByToken to hash input before DB lookup
- bin-api-manager: Hash tokens in auth flow before lookup
- bin-api-manager: Return full token only on creation, token_prefix on GET/List
- bin-api-manager: Update RST documentation for token hashing and one-time visibility
- bin-openapi-manager: Add token_prefix field to CustomerManagerAccesskey schema
- bin-dbscheme-manager: Add Alembic migration to hash existing tokens and replace token column
```
