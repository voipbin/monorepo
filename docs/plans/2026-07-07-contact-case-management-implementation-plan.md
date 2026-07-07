# Contact Case Management — Implementation Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task, in phase order. Each phase is independently mergeable/testable; do not skip ahead across phases even though later phases depend on earlier ones.

**Goal:** Implement the `Case` entity and its supporting mechanisms (get-or-create, lifecycle, cross-channel linking, tagging, notes) as specified in `docs/plans/2026-07-07-contact-case-management-design.md` (final, 24-round-reviewed).

**Architecture:** `Case` is a new, thin grouping entity owned by `bin-contact-manager`, sitting alongside the existing `Interaction`/`Resolution` tables (VOIP-1208/1209 foundation) without replacing their write paths. Cross-channel case linking is carried by a single new `Conversation.Metadata.ContactCaseID` field in `bin-conversation-manager`, written from two places and read uniformly. A new `POST /v1.0/cases/{id}/messages` endpoint in `bin-api-manager` lets agents send case-linked messages with explicit source-ownership and destination-binding validation.

**Tech Stack:** Go (all `bin-*` services), MySQL/MariaDB (via `bin-dbscheme-manager` Alembic-style migrations), RabbitMQ RPC (`bin-common-handler/pkg/requesthandler`), OpenAPI-generated REST layer (`bin-api-manager`).

**Reference:** `docs/plans/2026-07-07-contact-case-management-design.md` — section numbers (`§n`) below refer to this document. PR #1059, JIRA VOIP-1228.

---

## Phase ordering and rationale

Phases are ordered so each one is buildable and testable in isolation, and later phases only depend on earlier ones (never the reverse):

1. **Phase 0 — Schema migrations** (`bin-dbscheme-manager`): nothing else compiles/tests meaningfully without the tables existing.
2. **Phase 1 — `bin-conversation-manager` Metadata + RPCs**: needed before contact-manager's Case get-or-create can call it (§4.4/§4.5), and is independently testable in isolation (no Case concept needed inside conversation-manager itself).
3. **Phase 2 — `bin-contact-manager` Resolution nullable-`InteractionID` refactor** (§3.3): a real breaking change; isolate it from Case work so it can be reviewed/merged on its own and doesn't block or get tangled with Phase 3.
4. **Phase 3 — `bin-contact-manager` Case core**: entity, get-or-create (§4), lifecycle (§5), contact attribution (§6).
5. **Phase 4 — `bin-contact-manager` Case-adjacent features**: `CaseNote` (§3.5), tag assignments (§7), `case-control` CLI.
6. **Phase 5 — `bin-api-manager` REST surface**: `/v1.0/cases/*` routes, including `POST /v1.0/cases/{id}/messages` (§4.5) which depends on Phase 1's new conversation-manager RPCs and Phase 3's Case handlers.
7. **Phase 6 — `bin-openapi-manager` spec + docs**.

Each phase below is broken into tasks. Tasks are sized for a single subagent dispatch (not literal 2-5-minute atomic steps, given the scale of this feature — but each task is self-contained, has an explicit TDD loop, and ends in a commit).

---

## Phase 0: Schema migrations (`bin-dbscheme-manager`)

### Task 0.1: Create `contact_cases` table

**Objective:** Add the `Case` table with the generated-column unique-open-peer index and supporting indexes (§3.1).

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<new_revision>_contact_case_management_create_cases.py`

**Step 1: Generate revision skeleton**

Run: `cd bin-dbscheme-manager/bin-manager/main && alembic revision -m "contact_case_management_create_cases"`

**Step 2: Write the migration**

```python
"""contact_case_management_create_cases

Revision ID: <generated>
Revises: <current head — run `alembic heads` to confirm>
Create Date: <generated>
"""
from alembic import op
import sqlalchemy as sa

revision = "<generated>"
down_revision = "<current head>"

def upgrade():
    op.create_table(
        "contact_cases",
        sa.Column("id", sa.BINARY(16), primary_key=True),
        sa.Column("customer_id", sa.BINARY(16), nullable=False),
        sa.Column("peer_type", sa.String(255), nullable=False),
        sa.Column("peer_target", sa.String(255), nullable=False),
        sa.Column("reference_type", sa.String(255), nullable=False),
        sa.Column("contact_id", sa.BINARY(16), nullable=True),
        sa.Column("owner_type", sa.String(255), nullable=True),
        sa.Column("owner_id", sa.BINARY(16), nullable=True),
        sa.Column("status", sa.String(32), nullable=False),
        sa.Column("opened_at", sa.DateTime(6), nullable=True),
        sa.Column("closed_at", sa.DateTime(6), nullable=True),
        sa.Column("closed_reason", sa.String(32), nullable=True),
        sa.Column("closed_by_type", sa.String(32), nullable=True),
        sa.Column("closed_by_id", sa.BINARY(16), nullable=True),
        sa.Column("previous_case_id", sa.BINARY(16), nullable=True),
        sa.Column("tm_create", sa.DateTime(6), nullable=True),
        sa.Column("tm_update", sa.DateTime(6), nullable=True),
    )

    # Generated column + unique index (§3.1) -- MySQL has no native partial/filtered
    # unique index; raw SQL is required for GENERATED ALWAYS AS + STORED.
    op.execute("""
        ALTER TABLE contact_cases
        ADD COLUMN open_peer_uk BINARY(32) GENERATED ALWAYS AS (
            IF(status = 'open',
               UNHEX(SHA2(CONCAT_WS('|', customer_id, peer_type, peer_target, reference_type), 256)),
               NULL)
        ) STORED
    """)
    op.execute("ALTER TABLE contact_cases ADD UNIQUE INDEX uq_case_open_peer (open_peer_uk)")

    op.create_index("idx_case_unresolved", "contact_cases", ["customer_id", "status", "contact_id"])
    op.create_index("idx_case_owner", "contact_cases", ["customer_id", "owner_type", "owner_id"])
    op.create_index("idx_case_customer_reftype", "contact_cases", ["customer_id", "reference_type"])

def downgrade():
    op.drop_table("contact_cases")
```

**Step 3: Run migration against local/dev DB**

Run: `alembic upgrade head`
Expected: no errors; `DESCRIBE contact_cases;` in MySQL shows all columns plus `open_peer_uk`.

**Step 4: Verify generated column behavior manually**

```sql
INSERT INTO contact_cases (id, customer_id, peer_type, peer_target, reference_type, status, opened_at, tm_create, tm_update)
VALUES (UUID_TO_BIN(UUID()), UUID_TO_BIN(UUID()), 'tel', '+15551234567', 'call', 'open', NOW(6), NOW(6), NOW(6));
-- Second insert with the SAME customer_id/peer_type/peer_target/reference_type and status='open' MUST fail:
-- (repeat above with same customer_id) -> expect ER_DUP_ENTRY on uq_case_open_peer
```

**Step 5: Commit**

```bash
cd bin-dbscheme-manager
git add bin-manager/main/versions/<new_revision>_contact_case_management_create_cases.py
git commit -m "NOJIRA-contact-case-management

- bin-dbscheme-manager: Create contact_cases table with generated-column
  unique-open-peer index and supporting indexes"
```

### Task 0.2: Add `case_id` to `contact_interactions`

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<new_revision>_contact_case_management_add_case_id_to_interactions.py`

```python
def upgrade():
    op.add_column("contact_interactions", sa.Column("case_id", sa.BINARY(16), nullable=True))
    op.create_index("idx_interaction_case_id", "contact_interactions", ["case_id"])

def downgrade():
    op.drop_index("idx_interaction_case_id", table_name="contact_interactions")
    op.drop_column("contact_interactions", "case_id")
```

Verify: `alembic upgrade head`, then `DESCRIBE contact_interactions;` shows nullable `case_id`. Commit as in Task 0.1's pattern.

### Task 0.3: Add `case_id` to `contact_resolutions`; relax `interaction_id` to nullable; add CHECK + generated-column unique index (§3.3)

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/<new_revision>_contact_case_management_resolutions_case_support.py`

```python
def upgrade():
    op.add_column("contact_resolutions", sa.Column("case_id", sa.BINARY(16), nullable=True))
    op.alter_column("contact_resolutions", "interaction_id", existing_type=sa.BINARY(16), nullable=True)
    op.execute("""
        ALTER TABLE contact_resolutions
        ADD CONSTRAINT chk_resolution_case_or_interaction
        CHECK (interaction_id IS NOT NULL OR case_id IS NOT NULL)
    """)
    op.execute("""
        ALTER TABLE contact_resolutions
        ADD COLUMN case_positive_uk BINARY(16) GENERATED ALWAYS AS (
            IF(resolution_type = 'positive' AND interaction_id IS NULL AND tm_delete IS NULL,
               case_id, NULL)
        ) STORED
    """)
    op.execute("ALTER TABLE contact_resolutions ADD UNIQUE INDEX uq_resolution_case_positive (case_positive_uk)")
    op.create_index("idx_resolution_case_id", "contact_resolutions", ["case_id"])

def downgrade():
    op.drop_index("idx_resolution_case_id", table_name="contact_resolutions")
    op.execute("ALTER TABLE contact_resolutions DROP INDEX uq_resolution_case_positive")
    op.execute("ALTER TABLE contact_resolutions DROP COLUMN case_positive_uk")
    op.execute("ALTER TABLE contact_resolutions DROP CONSTRAINT chk_resolution_case_or_interaction")
    op.alter_column("contact_resolutions", "interaction_id", existing_type=sa.BINARY(16), nullable=False)
    op.drop_column("contact_resolutions", "case_id")
```

**Note (per design §3.3):** this migration is DB-additive-safe, but Phase 2 below handles the required Go-side refactor separately — do not assume this migration alone is sufficient.

Verify + commit as above.

### Task 0.4: Create `contact_case_notes` table (§3.5)

```python
def upgrade():
    op.create_table(
        "contact_case_notes",
        sa.Column("id", sa.BINARY(16), primary_key=True),
        sa.Column("customer_id", sa.BINARY(16), nullable=False),
        sa.Column("case_id", sa.BINARY(16), nullable=False),
        sa.Column("author_type", sa.String(32), nullable=False),
        sa.Column("author_id", sa.BINARY(16), nullable=True),
        sa.Column("text", sa.Text(), nullable=False),
        sa.Column("tm_create", sa.DateTime(6), nullable=True),
        sa.Column("tm_update", sa.DateTime(6), nullable=True),
        sa.Column("tm_delete", sa.DateTime(6), nullable=True),
    )
    op.create_index("idx_case_note_case_id", "contact_case_notes", ["case_id"])

def downgrade():
    op.drop_table("contact_case_notes")
```

Verify + commit.

### Task 0.5: Create `contact_case_tag_assignments` table (§7 round-22 correction)

**Objective:** Mirror the existing `contact_tag_assignments` schema exactly (verify against `bin-dbscheme-manager/.../a1b2c3d4e5f6_contact_create_tables.py:141-149` before writing this — do not guess the column shape).

```python
def upgrade():
    op.create_table(
        "contact_case_tag_assignments",
        sa.Column("case_id", sa.BINARY(16), primary_key=True),
        sa.Column("tag_id", sa.BINARY(16), primary_key=True),
        sa.Column("tm_create", sa.DateTime(6), nullable=True),
    )
    op.create_index("idx_case_tag_assignment_tag_id", "contact_case_tag_assignments", ["tag_id"])

def downgrade():
    op.drop_table("contact_case_tag_assignments")
```

Verify + commit. **After this task**, run the full `bin-dbscheme-manager` verification workflow (per root `CLAUDE.md`) once for all 5 migrations together, then open a single PR for Phase 0 (or fold into the overall feature PR per current workflow — confirm with pchero which convention applies here since this repo already has PR #1059 open for the design doc).

---

## Phase 1: `bin-conversation-manager` — `Metadata` field + new RPCs (§4.3/§4.4/§4.5)

### Task 1.1: Add `Metadata` model (mirrors `bin-customer-manager`'s pattern exactly)

**Files:**
- Create: `bin-conversation-manager/models/conversation/metadata.go`
- Modify: `bin-conversation-manager/models/conversation/conversation.go` (add `Metadata` field to `Conversation` struct)

**Step 1: Write failing test**

`bin-conversation-manager/models/conversation/metadata_test.go`:
```go
package conversation_test

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"monorepo/bin-conversation-manager/models/conversation"
)

func TestMetadata_JSONRoundtrip(t *testing.T) {
	caseID := uuid.Must(uuid.NewV4())
	m := conversation.Metadata{ContactCaseID: &caseID}

	b, err := json.Marshal(m)
	assert.NoError(t, err)

	var got conversation.Metadata
	assert.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, caseID, *got.ContactCaseID)
}

func TestMetadata_NilContactCaseID_OmitsField(t *testing.T) {
	m := conversation.Metadata{}
	b, err := json.Marshal(m)
	assert.NoError(t, err)
	assert.JSONEq(t, `{}`, string(b))
}
```

**Step 2: Run to verify failure**

Run: `cd bin-conversation-manager && go test ./models/conversation/... -run TestMetadata -v`
Expected: FAIL — `conversation.Metadata` undefined.

**Step 3: Write minimal implementation**

`bin-conversation-manager/models/conversation/metadata.go`:
```go
package conversation

import "github.com/gofrs/uuid"

// Metadata is a generic, extensible per-Conversation annotation payload.
// Follows the bin-customer-manager Metadata precedent: a single typed
// struct in one nullable JSON column, with its own dedicated update RPC
// rather than the general partial-update allowlist.
type Metadata struct {
	// ContactCaseID is set by bin-contact-manager, from either write path
	// in the design doc's §4.3, to claim this Conversation for a Case.
	// Read-only from conversation-manager's own perspective: never read
	// by getExecuteMode or any flow/agent-routing decision.
	ContactCaseID *uuid.UUID `json:"contact_case_id,omitempty"`
}
```

Add to `conversation.go`'s `Conversation` struct:
```go
Metadata Metadata `json:"metadata,omitempty" db:"metadata,json"`
```

**Step 4: Run to verify pass**

Run: `go test ./models/conversation/... -run TestMetadata -v`
Expected: PASS (2 tests)

**Step 5: Commit**

```bash
git add bin-conversation-manager/models/conversation/metadata.go bin-conversation-manager/models/conversation/metadata_test.go bin-conversation-manager/models/conversation/conversation.go
git commit -m "NOJIRA-contact-case-management

- bin-conversation-manager: Add Metadata field to Conversation, mirroring
  bin-customer-manager's Metadata pattern"
```

### Task 1.2: DB migration — add `metadata` column to `conversation_conversations`

**Files:** Create migration in `bin-dbscheme-manager` (same pattern as Phase 0; nullable JSON column, mirroring `bin-customer-manager`'s existing `customer.Metadata` migration precedent — read `c78cf0c45f54_customer_customers_add_column_metadata.py` first and copy its shape exactly).

```python
def upgrade():
    op.add_column("conversation_conversations", sa.Column("metadata", sa.JSON(), nullable=True))

def downgrade():
    op.drop_column("conversation_conversations", "metadata")
```

Verify + commit under `bin-dbscheme-manager`.

### Task 1.3: `dbhandler` — read/write `Metadata` on Conversation rows

**Files:**
- Modify: `bin-conversation-manager/pkg/dbhandler/conversation.go` (scan/marshal `metadata` column in existing `ConversationGet`/`ConversationGetBySelfAndPeer`/insert/update row-mapping functions)
- Test: `bin-conversation-manager/pkg/dbhandler/conversation_test.go`

**Step 1: Write failing test** — extend an existing `Test_ConversationGetBySelfAndPeer_*` test (or add a new one) asserting a row with a non-null `metadata` JSON blob correctly unmarshal into `Conversation.Metadata.ContactCaseID`.

**Step 2: Run to verify failure** (column not yet read).

**Step 3: Implement** — add `metadata` to the `SELECT` column list and `Scan()` target (JSON-to-struct, following the exact pattern already used for any other JSON column in this file — grep for `db:"...,json"` usage elsewhere in this package first and copy it verbatim, do not invent a new JSON-scanning helper).

**Step 4: Run to verify pass.**

**Step 5: Commit.**

### Task 1.4: New RPC — `ConversationGetBySelfAndPeer` (get-only, cross-service) — §4.4

**Objective:** Expose the existing internal `GetBySelfAndPeer` lookup (`bin-conversation-manager/pkg/dbhandler/conversation.go:155-189`, per the design doc's citation) as a cross-service RPC. **Get-only — must never create.**

**Files:**
- Modify: `bin-conversation-manager/pkg/listenhandler/main.go` (route registration)
- Create/Modify: `bin-conversation-manager/pkg/listenhandler/v1_conversations.go` (handler)
- Create: `bin-common-handler/pkg/requesthandler/conversation_conversations.go` (add `ConversationV1ConversationGetBySelfAndPeer` client method — or extend the existing file if a `conversation_conversations.go` already exists; check first)

**Step 1: Write failing test** (listenhandler level) — request with a known `(self, peer)` pair that exists returns the Conversation; a pair that doesn't exist returns 404, **and asserts no row was created** (re-query the DB in the test to confirm).

**Step 2: Run to verify failure.**

**Step 3: Implement:**
- listenhandler route: `GET /v1/conversations/self_and_peer?self_type=...&self_target=...&peer_type=...&peer_target=...` (or RPC-style request/response matching this codebase's existing listenhandler conventions — check `v1_conversations.go`'s existing routes for the exact convention before inventing a new one).
- Handler calls `conversationHandler.GetBySelfAndPeer(ctx, self, peer)` (the existing internal method) directly — **do not** call `GetOrCreateBySelfAndPeer`.
- `bin-common-handler` client method: `ConversationV1ConversationGetBySelfAndPeer(ctx, self, peer commonaddress.Address) (*conversation.Conversation, error)`.

**Step 4: Run to verify pass.**

**Step 5: Commit.**

### Task 1.5: New RPC — `ConversationGetOrCreateBySelfAndPeer` (get-or-create, cross-service) — §4.5 round-12 correction

**Objective:** A **separate, distinct** RPC from Task 1.4 exposing the existing internal `GetOrCreateBySelfAndPeer` (`db.go:43-86`). This one **does** create on miss — correct here because §4.5 only calls it when a real message is genuinely about to be sent.

**Files:** Same three files as Task 1.4, additive.

**Step 1: Write failing test** — a pair that doesn't exist creates a new Conversation and returns it (assert the row now exists and, separately, assert the design's expected webhook fires — since `Create()` unconditionally calls `PublishWebhookEvent`, per the design doc, verify this test does NOT assert "no webhook," unlike Task 1.4's negative test).

**Step 2-5:** Same TDD/commit pattern as Task 1.4.

**Pitfall to avoid:** do not accidentally reuse Task 1.4's route/handler for this — the design doc's round-12 correction is explicit that these are two distinct RPCs with different semantics on miss.

### Task 1.6: New RPC — `ConversationUpdateMetadata`

**Objective:** Whole-struct-replace `Metadata` on a Conversation, mirroring `bin-customer-manager`'s `UpdateMetadata` RPC/handler shape exactly (read `bin-customer-manager/pkg/customerhandler/db.go`'s `UpdateMetadata`, lines ~229-256 per prior session notes, before writing this).

**Files:**
- Modify: `bin-conversation-manager/pkg/conversationhandler/db.go` (or a new file, following whatever convention `bin-customer-manager` used) — add `UpdateMetadata(ctx, conversationID, metadata)`.
- Modify: `bin-conversation-manager/pkg/listenhandler/v1_conversations.go` — **do not** add `Metadata` to the existing generic PUT allowlist (`FieldOwnerType, FieldOwnerID, FieldName, FieldDetail, FieldAccountID` at `v1_conversations.go:184-190`) — add a dedicated route instead, per the design doc's explicit rationale for why `Metadata` is namespaced separately.
- `bin-common-handler`: add `ConversationV1ConversationUpdateMetadata`.

TDD/commit pattern as above. **Explicit negative test required:** assert this RPC's handler is NOT reachable via the generic conversation PUT endpoint's field allowlist (i.e., PUTing `metadata` through the general update route has no effect) — this is the exact invariant the design doc calls out.

### Task 1.7: Extend `MessageEventReceived` and `MessageEventSent` to echo `Metadata.ContactCaseID`

**Objective:** Both the inbound (`pkg/conversationhandler/message.go:109-142`, `MessageEventReceived`) and outbound (`MessageEventSent`) paths already load the `Conversation` row. Extend the `conversation_message_created` event payload (`message.EventTypeMessageCreated`) to include the `case_id` hint field, sourced from `Metadata.ContactCaseID`, when present.

**Files:**
- Modify: `bin-conversation-manager/pkg/conversationhandler/message.go`
- Modify: `bin-conversation-manager/models/message/event.go` (or wherever the event payload struct is defined — add a `CaseID *uuid.UUID` field, `omitempty`)

**Step 1: Write failing test** — a Conversation with `Metadata.ContactCaseID` set, on both an inbound and an outbound message, produces an event payload with the matching `case_id` field populated. A Conversation with no metadata produces a payload with the field absent/nil.

**Step 2-5:** Standard TDD/commit.

**Explicit negative test required (per design doc):** assert `getExecuteMode` (or wherever flow/agent-routing dispatch is decided) never reads `Metadata.ContactCaseID` — grep the dispatch function's inputs in the test and assert the field isn't referenced.

**After Task 1.7:** run full `bin-conversation-manager` verification workflow (`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`), then commit/push and open Phase 1's PR (or continue on the same branch per current workflow, confirm with pchero).

---

## Phase 2: `bin-contact-manager` — `Resolution.InteractionID` nullable refactor (§3.3)

**This phase is isolated on purpose** — it's a real breaking change unrelated to Case itself, and should be reviewable independently.

### Task 2.1: Update `Resolution` model

**Files:**
- Modify: `bin-contact-manager/models/resolution/resolution.go:18` — change `InteractionID uuid.UUID` to `InteractionID *uuid.UUID`; add `CaseID *uuid.UUID`.

**Step 1: Write failing test** — a `Resolution` value with `InteractionID: nil, CaseID: &someID` constructs and JSON-marshals without error.

**Step 2-4:** standard TDD loop; this will also break existing callers — expect widespread compile failures across `dbhandler`/`contacthandler` until Tasks 2.2-2.4 land. This is expected and matches the design doc's explicit warning that this is not a trivial widen.

### Task 2.2: Add `ResolutionListByCase` / dual-scope `ResolutionDelete`

**Files:**
- Modify: `bin-contact-manager/pkg/dbhandler/resolution.go` — add `ResolutionListByCase(ctx, caseID)`; change `ResolutionDelete`/`ResolutionListByInteraction` call sites to handle the now-nullable `interaction_id` scoping parameter (per design doc: these need a case-scoped counterpart, not just a widened signature).
- Test: `bin-contact-manager/pkg/dbhandler/resolution_test.go` — add nil-`InteractionID` cases per the design doc's explicit call-out.

Standard TDD loop per new/changed function.

### Task 2.3: Fix unguarded `map[uuid.UUID]bool` keying in `interaction_read.go`

**Files:**
- Modify: `bin-contact-manager/pkg/contacthandler/interaction_read.go` (the map keyed directly off `r.InteractionID` with no nil-check, per design doc's explicit citation)

**Step 1: Write failing test** — a Resolution list containing a case-level (nil-`InteractionID`) Resolution does not panic and does not incorrectly key/dedupe against a zero-UUID.

**Step 2-4:** implement the nil-guard (e.g., skip case-level Resolutions in this particular map, or key by `*r.InteractionID` only when non-nil — follow whatever this function's actual purpose requires; read the surrounding code before deciding).

### Task 2.4: Propagate through `contacthandler/resolution.go`

Update this layer to match the new nullable signature; update its existing tests. Standard TDD loop.

**After Task 2.4:** run full `bin-contact-manager` verification workflow. This should now compile and all existing tests pass with the nullable `InteractionID`. Commit, and consider this phase mergeable on its own.

---

## Phase 3: `bin-contact-manager` — Case core (§3.1, §3.4, §4, §5, §6)

### Task 3.1: `Case` model

**Files:**
- Create: `bin-contact-manager/models/kase/kase.go` (Go reserves `case` as a keyword — follow this monorepo's existing convention for such collisions; check if a similar naming precedent exists elsewhere in the codebase before choosing `kase`/`contactcase`/other, and use whatever convention already exists)

Define the full `Case` struct per design doc §3.1 (ID, CustomerID, PeerType, PeerTarget, ReferenceType, ContactID, embedded `commonidentity.Owner`, Status, OpenedAt, ClosedAt, ClosedReason, ClosedByType, ClosedByID, PreviousCaseID, TMCreate, TMUpdate).

Standard TDD loop (struct construction + JSON marshal test), commit.

### Task 3.2: `dbhandler` — `CaseGetByPeer` (locked read for get-or-create step 1)

**Files:**
- Create: `bin-contact-manager/pkg/dbhandler/kase.go`
- Test: `bin-contact-manager/pkg/dbhandler/kase_test.go`

Implement `CaseGetOpenByPeer(ctx, tx, customerID, peerType, peerTarget, referenceType) (*Case, error)` using `SELECT ... FOR UPDATE`, matching design doc §4 step 1's pseudocode exactly. Standard TDD loop per function — write each of the following as its own task-sized TDD cycle:

- `CaseGetOpenByPeer` (locked select)
- `CaseGetByID` (unlocked, for read APIs)
- `CaseInsert` (the INSERT branch, expects a duplicate-key error type the caller can detect)
- `CaseUpdateStatusClosed` (used both by timeout-close and `/close`, with the `WHERE status='open'` optimistic-guard per §5.1)
- `CaseUpdateTMUpdate`
- `CaseUpdateContactID`
- `CaseListUnresolved` (backed by `idx_case_unresolved`)
- `CaseListByOwner` (backed by `idx_case_owner`)
- `CaseGetLastClosedByPeer` (for `previous_case_id` chaining on fresh-insert path)

### Task 3.3: `caseHandler.GetOrCreate` — the full §4 algorithm

**Files:**
- Create: `bin-contact-manager/pkg/casehandler/db.go` (or `getorcreate.go`)
- Test: `bin-contact-manager/pkg/casehandler/db_test.go`

Implement exactly the pseudocode in design doc §4, including:
- explicit `case_id` hint validation (customer_id match + `status='open'`) as step 1
- reuse-if-open, timeout-close-then-reopen, and fresh-insert branches
- the **bounded retry loop with locked re-select** on `ON DUPLICATE KEY` (§4.2's round-2 correction — this is the most safety-critical part of this task; write a dedicated concurrency test using two goroutines racing the same peer, per the existing test conventions elsewhere in this codebase for similar races)
- contact auto-match via the existing address-set lookup (skipped when case_id came from the hint path)
- the Interaction insert with `case_id` set, and `tm_update` bump, all in one transaction

Break this into multiple TDD sub-tasks in this order (each a full red-green-commit cycle):
1. Reuse-on-open-match (simplest branch first)
2. Explicit-hint validation (valid / stale / wrong-tenant / closed hint, per §4.3)
3. Timeout-triggered close-and-reopen chain
4. Fresh-insert (no prior case for this peer)
5. Concurrent-insert duplicate-key retry (the race test)
6. The specific round-2 TOCTOU: a case closes between a losing transaction's duplicate-key retry-select and its Interaction insert — must NOT attach the Interaction to the now-closed case.
7. Loop-exhaustion path — surfaces a transient 5xx, does not silently drop.

### Task 3.4: Wire `GetOrCreate` into the existing `EventCallCreated` / `EventConversationMessageCreated` projection points

**Files:**
- Modify: `bin-contact-manager/pkg/contacthandler/interaction.go` (the existing `EventCallCreated`/`EventConversationMessageCreated` handlers, per design doc §4's "triggered from the same projection points" and the earlier session's confirmed line numbers ~36-119)

**Step 1: Write failing test** — an inbound call event now results in an Interaction row with a non-nil `case_id`.

**Step 2-5:** standard loop. Verify no regression to existing Interaction-creation behavior (these handlers' existing tests must still pass unmodified in their non-Case-related assertions).

### Task 3.5: Proactive linking write (§4.4) — `ConversationGetBySelfAndPeer` + `ConversationUpdateMetadata` calls from Case-open

**Files:**
- Modify: `bin-contact-manager/pkg/casehandler/db.go` (or a new file) — after a *new* Case opens for a non-`conversation_message` `reference_type`, call the two RPCs from Phase 1 (Task 1.4 get-only lookup, then Task 1.6 metadata update if found).
- `bin-common-handler` client: use the methods added in Phase 1.

**Step 1: Write failing test (with mocked RPC client)** — new call-case-open triggers exactly one `ConversationGetBySelfAndPeer` call; if it returns a Conversation, exactly one `ConversationUpdateMetadata` call follows with the right `case_id`; if not found, no further RPC.

**Step 2-5:** standard loop.

**Explicit tests required (per design doc §10):**
- Both RPCs happen **after** the Case DB transaction commits (assert via mock call ordering/timing, not inside the transaction).
- Either RPC failing does not roll back or fail the Case-open operation (Case still opens; assert this with a mock that returns an error from each RPC in turn).
- This trigger fires only on Case-**open** (new INSERT), not on every event that resolves to an existing already-linked Case (assert no RPC call on cache-hit reuse).

### Task 3.6: Case lifecycle — close (§5.1), timeout (§5.2), `/continue` (§5.3)

**Files:**
- Modify/Create: `bin-contact-manager/pkg/casehandler/lifecycle.go`
- Test: corresponding `_test.go`

Sub-tasks (each its own TDD cycle):
1. `Close` — idempotent double-close via `WHERE status='open'` guard; **must return the actually-persisted `closed_reason`/`closed_by`**, not assume the caller's own action won (§5.1's explicit correction).
2. Timeout evaluation — lazy, at next inbound event (already wired via Task 3.3's timeout branch; this task is about the `CASE_TIMEOUT_HOURS` env var itself: platform-wide, default 24, following `bin-ai-manager`'s `AIcallConversationIdleTimeoutHours` config-shape and `SetXXXForTest` convention exactly — read that file first).
3. `Continue` — reuses the **same** `GetOrCreate`/insert-retry primitive from Task 3.3 (not a separate implementation), parameterized with `previous_case_id = id`; requires source case `status='closed'`; requires caller is owning agent or admin/manager (relies on the owner-survives-closing invariant).

### Task 3.7: Contact attribution derivation (§3.4) and unresolved queue (§6)

**Files:**
- Create: `bin-contact-manager/pkg/casehandler/contact_attribution.go`

Implement the single `deriveCaseContactID` function per design doc §3.4's pseudocode exactly, called from:
- Resolution create (case-level)
- Resolution soft-delete
- Resolution replace (soft-delete + insert + derivation, **single transaction** — explicit test that no transient NULL is externally observable mid-transaction)
- `case-control reconcile-contact` CLI (Task 4.3)

Plus `CaseListUnresolved` API-facing wrapper (already have the dbhandler function from Task 3.2; this task wires it through `casehandler`).

**After Task 3.7:** run full `bin-contact-manager` verification workflow. Commit, consider Phase 3 complete.

---

## Phase 4: `bin-contact-manager` — Case-adjacent features (§3.5, §7)

### Task 4.1: `CaseNote` model + handler + `case_note_created` event (§3.5)

**Files:**
- Create: `bin-contact-manager/models/casenote/casenote.go`
- Create: `bin-contact-manager/pkg/dbhandler/casenote.go` (create/list/delete, soft-delete via `tm_delete`)
- Create: `bin-contact-manager/pkg/casenotehandler/` (or fold into `casehandler`, following existing package-naming convention in this codebase for sibling entities)

**Critical requirement (per design doc, corrected across 3 drafts):** `case_note_created` MUST be published via the plain `notifyHandler.PublishEvent()` primitive — **never** `PublishWebhookEvent()`. Write this as an explicit negative test:

```go
func Test_CaseNoteCreate_NeverUsesPublishWebhookEvent(t *testing.T) {
	mockNotify := notifyhandlermock.NewMockNotifyHandler(mc)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	// ... create a CaseNote, assert only PublishEvent was called
}
```

Also required: an explicit negative test asserting `CaseNote` never appears in the customer-facing Interaction list or any webhook payload (query the Interaction list API path with a case that has notes, assert none leak in).

Standard TDD loop for create/list/delete.

### Task 4.2: Case tag assignments (§7 round-22 correction)

**Files:**
- Create: `bin-contact-manager/pkg/dbhandler/case_tag_assignment.go` — mirror `pkg/dbhandler/tag_assignment.go`'s `TagAssignmentCreate/Delete/ListByContactID` exactly, renamed to `CaseTagAssignmentCreate/Delete/ListByCaseID`, against the new `contact_case_tag_assignments` table from Phase 0 Task 0.5.
- Modify: `bin-contact-manager/pkg/contacthandler/` (or `casehandler/`) — expose case-scoped tag assign/unassign/list, calling `bin-tag-manager`'s existing `TagV1TagGet`/`TagV1TagList` only to validate `tag_id` existence (no other tag-manager interaction needed — tag-manager itself is unchanged per the design doc).

Standard TDD loop, directly copying the existing Contact tag-assignment tests' shape for the Case-scoped equivalent.

### Task 4.3: `case-control` CLI (including `reconcile-contact`)

**Files:**
- Create: `bin-contact-manager/cmd/case-control/main.go` (follow the existing `cmd/*-control/main.go` convention in this repo — e.g. `bin-tag-manager/cmd/tag-control/main.go` — for command structure)

Implement `case-control reconcile-contact <case_id | --all>` calling Task 3.7's `deriveCaseContactID` and overwriting `Case.contact_id`. Idempotent — test running it twice in a row produces the same result with no side effects on the second run.

**After Task 4.3:** run full `bin-contact-manager` verification workflow (this is likely the largest single verification run in this plan — expect it to take longer). Commit.

---

## Phase 5: `bin-api-manager` — REST surface (§9)

### Task 5.1: `/v1.0/cases` CRUD + list + list-unresolved

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/case.go` (new file) — thin wrapper calling `bin-contact-manager`'s new RPCs via `bin-common-handler`.
- Modify: `bin-api-manager/server/cases.go` (new file, OpenAPI-generated handler glue — follow the exact pattern of an existing similar resource, e.g. `server/outbound_configs.go`, for request/response conversion).
- `bin-common-handler`: add `ContactV1CaseGet/List/ListUnresolved/Close/Continue/Assign/Unassign` client methods (mirroring the existing `ContactV1InteractionGet/List` pattern in `bin-common-handler/pkg/requesthandler/contact_interactions.go`).

`GET /v1.0/cases` supports `status`, `owner_type`/`owner_id`, and unresolved (`contact_id IS NULL` + `status=open`) filter query params (§9). Confirm this inherits `PermissionProjectSuperAdmin`'s cross-customer bypass automatically (§7's explicit call-out) — write a test asserting a superadmin caller can list cases across customers without extra code.

Standard TDD loop per endpoint.

### Task 5.2: `POST /v1.0/cases/{id}/close` and `POST /v1.0/cases/{id}/continue`

Standard TDD loop, thin passthrough to Phase 3 Task 3.6's handlers. Verify response shape matches §5.1's "return the actually-persisted state" contract exactly (not the caller's assumed outcome).

### Task 5.3: `/v1.0/cases/{id}/notes` CRUD

Standard TDD loop, thin passthrough to Phase 4 Task 4.1.

### Task 5.4: `POST /v1.0/cases/{id}/messages` (§4.5) — the most involved endpoint in this plan

**Files:**
- Create: `bin-api-manager/pkg/servicehandler/case_message.go`
- Modify: `bin-api-manager/server/cases.go`

Implement in this exact order (each its own TDD sub-task):

1. **Case validation**: `case_id` belongs to calling customer, `status='open'` — reject closed cases with an actionable error pointing at `/continue`.
2. **Destination-to-case binding**: if `case.contact_id` set, `destination` must be in `AddressListByContactID(contact_id)`; else `destination` must equal `case.peer_target` exactly; else reject with a **single generic 4xx message** that does not distinguish which branch failed (explicit test: assert the error message/code is identical for both failure sub-cases — this is the anti-oracle property from the round-11 security review, do not regress it).
3. **Source-ownership validation (round-17 correction — new, required code, not inherited from anywhere)**: query `NumberV1NumberList` with `FieldCustomerID=<case.customer_id>`, `FieldNumber=source`, `FieldType=Normal`, `FieldStatus=Active`, `FieldDeleted=false`; reject with 4xx if no match. Write explicit test cases for: not-a-real-number, wrong-customer's-number, inactive-number, non-normal-type-number — all rejected.
4. **Resolve `conversationID`** via the new `ConversationGetOrCreateBySelfAndPeer` RPC (Phase 1 Task 1.5) — `self=source, peer=destination`.
5. **Metadata write** via `ConversationUpdateMetadata` (Phase 1 Task 1.6) — **fail open**: if this RPC errors, log and proceed to step 6 anyway (explicit test: mock this RPC to fail, assert the send in step 6 still happens and the overall request still succeeds).
6. **Send** via the existing `ConversationMessageSend` (`pkg/servicehandler/conversation_message.go:107-144`, unchanged) with the resolved `conversationID`.

**End-to-end test required:** after a successful send, assert the resulting `MessageEventSent` payload correctly carries the just-written `case_id` hint (i.e. actually verify Phase 1 Task 1.7's echo-read path picks up what this task just wrote — this is the one test that proves the write-then-read ordering across the two phases is correct in practice, not just in isolated unit tests).

**After Task 5.4:** run the full `bin-api-manager` verification workflow. Commit.

---

## Phase 6: `bin-openapi-manager` spec + docs sync

### Task 6.1: Add OpenAPI spec entries for all `/v1.0/cases/*` routes

Follow the `voipbin-openapi-spec-handler-parity` skill/convention referenced in the design doc §9. Verify `bin-api-manager`'s generated server code (`gens/openapi_server/gen.go`) regenerates cleanly against the new spec (`go generate ./...` per the root `CLAUDE.md` verification workflow) and that the hand-written handlers from Phase 5 satisfy the generated interface.

### Task 6.2: Sync any relevant RST developer docs

Check `bin-api-manager/docsdev/source/` for an existing pattern (e.g. `conversation_tutorial.rst`) and add an equivalent `case_tutorial.rst` if this repo's convention expects one for every major new resource — confirm with pchero whether this is required for this feature before doing the work (skip if not established practice for backend-only PRs).

---

## Cross-cutting notes for every phase

- **Verification workflow is mandatory after every task's code changes**, not just at phase end, per root `CLAUDE.md`: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` in the relevant service directory. Phase-end full-service runs (called out above) are in addition to, not instead of, per-task verification.
- **Commit message format**: `NOJIRA-contact-case-management` title (matches the existing branch already open for PR #1059), with `bin-<service>:` prefixed bullets per root `CLAUDE.md` convention.
- **Do not merge to `main` or open new PRs without explicit instruction** — this plan assumes continued work on the existing branch/PR #1059 unless pchero directs otherwise.
- **Review loop**: per this workspace's established convention, run the adversarial review loop (parallel fresh subagents, 2+ rounds, consensus required) on each phase's changes before considering that phase done — this mirrors the process already applied to the design doc itself (24 rounds, multiple real defects caught each time). Do not skip review because "it's just implementation, not design" — the design review loop caught fabricated function names, missing RPCs, and a real security gap; the same rigor applies to code.
