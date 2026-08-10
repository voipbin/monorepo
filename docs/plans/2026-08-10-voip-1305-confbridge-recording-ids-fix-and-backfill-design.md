# VOIP-1305: Confbridge recording_ids UUID encoding fix and backfill, Design

Date: 2026-08-10
Ticket: VOIP-1305
Services: bin-call-manager (code fix), bin-dbscheme-manager (data backfill)

## Problem

`buildConfbridgeAddRecordingIDs` in `bin-call-manager/pkg/dbhandler/confbridge.go`
bound `recordingID.Bytes()` (16 raw bytes) into `json_array_append`, while every
semantically identical site (`CallAddRecordingIDs`,
`ConfbridgeAddExternalMediaID`, conference-manager, queue-manager) binds
`.String()` (the 36 char UUID text). The write succeeds without error, so each
confbridge recording start appended a corrupted element to the
`call_confbridges.recording_ids` JSON column.

The read path unmarshals that column into `[]uuid.UUID`
(`bin-common-handler/pkg/databasehandler/mapping.go`, copyJSON). A corrupted
element makes `ScanRow` fail, so `ConfbridgeGetFromDB` fails permanently for
that row. The failure is delayed: the Redis cache (TTL 24h) keeps serving the
last good snapshot until expiry, eviction, or pod restart, after which the
confbridge becomes unreadable. From the moment of corruption, the ignored cache
refresh in `ConfbridgeAddRecordingIDs` leaves an error log signature
(confbridge cache update failure), usable to estimate historical occurrences.

The behavior was carried over verbatim by the VOIP-1078 squirrel migration with
an explicit "needs its own ticket" note. VOIP-1305 is that ticket.

## Two corruption forms

`.Bytes()` binding produces a different stored shape depending on the
go-sql-driver protocol path. Both were reproduced against MySQL 8.0 with the
real driver, and both must be handled because the production DSN (k8s secret)
cannot be inspected from the repository.

| Form | Driver path | Stored shape | JSON_TYPE | Detection (per element) |
|---|---|---|---|---|
| A | prepared statement (driver default, likely production path) | raw 16 bytes inside a JSON STRING (control bytes escaped, non ASCII bytes raw, possibly invalid UTF-8) | STRING | `LENGTH(JSON_UNQUOTE(elem)) = 16` (byte length; healthy UUID text is 36 bytes) |
| B | `interpolateParams=true` text protocol, or a `_binary` SQL literal | opaque JSON BLOB, rendered as `base64:type<N>:<payload>` | BLOB | `JSON_TYPE(elem) = 'BLOB'` |

A healthy 36 char UUID string matches neither condition. `CAST`/regexp based
detection was rejected: invalid UTF-8 makes it return NULL or misbehave
(measured), so detection uses only `JSON_TYPE` plus byte `LENGTH`.

## Code fix (bin-call-manager)

| File | Change |
|---|---|
| `pkg/dbhandler/confbridge.go` | `buildConfbridgeAddRecordingIDs` binds `recordingID.String()`; the carried-over-bug NOTE is replaced by a one line fix record |
| `pkg/dbhandler/call.go` | removed the comparison NOTE above `buildCallAddRecordingIDs` (obsolete after the fix) |
| `pkg/dbhandler/json_expr_golden_test.go` | ConfbridgeAddRecordingIDs golden case now asserts `.String()` argument |
| `pkg/dbhandler/recording_ids_scan_test.go` (new) | pins that Form A (invalid UTF-8 and all ASCII variants) and Form B fail `[]uuid.UUID` unmarshal on the ScanRow-equivalent path, and that the clean `.String()` shape round-trips; error strings are not asserted |

## Backfill migration (bin-dbscheme-manager)

Revision `ede50012c416_call_confbridges_backfill_recording_ids` (generated with
`alembic revision`, chained after `902325885953`, single head verified).
Repairs both forms element by element, in server side SQL only. Only BINARY(16)
row ids and integer counts reach the Python client, so corrupted bytes never
cross the driver boundary (no UTF-8 decode hazard).

Algorithm:

1. Phase 0, scope probe. `SELECT COALESCE(MAX(JSON_LENGTH(recording_ids)), 0)`
   as the per index loop bound; exit immediately when 0. This always executed
   path stays MariaDB compatible because the dbscheme Docker image build runs
   the whole migration chain against an empty temporary MariaDB (Phase 0 exits
   early there). A JSON null scalar row yields JSON_LENGTH 1, which is harmless
   (the loops match nothing for it).
2. Phase 1, candidate collection. For each array index, a plain `SELECT id`
   with the per element detection predicate collects corrupted row ids into a
   hex set. Plain SELECTs are lock free consistent reads under REPEATABLE READ
   (an earlier `INSERT ... SELECT` staging design was rejected because it takes
   shared next-key locks on every scanned row, measured as a real writer
   blocker). Exit with a zero report when the set is empty.
3. Phase 2, repair. The id set is processed in 500 id chunks as
   `IN (UNHEX('<hex>'), ...)` literals (hex digits only, no bind parameters),
   so exclusive locks touch only corrupted rows. Per index, two `JSON_REPLACE`
   UPDATEs restore the element to lowercase 8-4-4-4-12 UUID text: Form A feeds
   `HEX()` the unquoted 16 byte string; Form B feeds it the `FROM_BASE64`
   decoded payload after the last `CHAR(58)` separator, guarded by a decoded
   length of exactly 16. Array order is preserved; healthy elements are
   untouched. The SQL text contains no colon characters: `op.execute`/`text()`
   treat `:name` as a bind parameter, so the Form B separator is `CHAR(58)`.
4. Phase 3, verify and report. The Phase 1 scan is re-run restricted to the
   candidate chunks, then the migration prints
   `VOIP-1305 backfill: candidate_rows=N, repaired_rows=N, remaining_anomaly_rows=N`
   and a warning when remaining is greater than zero.

`downgrade()` is a no-op with a reason comment: reversing a data repair would
re-corrupt restored values (precedent: `9443d2d65ad8_add_join_bridge.py`).

Idempotency: repaired elements no longer match detection, so a re-run collects
only still-anomalous rows and changes nothing. Verified on a throwaway MySQL
8.0 with the full fixture matrix (both forms, mixed arrays with corruption at
index above 0, a partially repairable row, clean, SQL NULL, JSON null, and
empty array rows): exact UUID restoration, order preserved, untouched rows
byte identical, second run reported repaired 0 with zero data changes.

## Production runbook

1. Pre-check (read only, requires MySQL 8.0.4 or newer for JSON_TABLE; the
   migration itself does not use JSON_TABLE):

   ```sql
   SELECT COUNT(DISTINCT c.id) FROM call_confbridges c,
     JSON_TABLE(COALESCE(c.recording_ids, JSON_ARRAY()), '$[*]'
       COLUMNS (elem JSON PATH '$')) jt
   WHERE JSON_TYPE(jt.elem) = 'BLOB'
      OR (JSON_TYPE(jt.elem) = 'STRING' AND LENGTH(JSON_UNQUOTE(jt.elem)) = 16);
   ```

   The COALESCE guard keeps JSON_TABLE away from SQL NULL documents, which
   some older 8.0 patch levels reject; on current 8.0 the behavior is
   identical (verified: NULL, JSON null, and empty array rows are ignored,
   corrupted rows are counted). A zero count does not block applying the
   migration (it is a no-op then); apply anyway to keep the revision chain
   consistent.
2. Apply: a human runs `alembic upgrade head` over VPN (AI execution of
   upgrade against shared databases is prohibited). A low traffic window is
   recommended to soften the Phase 1 scan load; writers are only ever blocked
   on the corrupted rows themselves during Phase 2.
3. Post-check: re-run the query from step 1, expect 0. If the migration
   printed `remaining_anomaly_rows` greater than zero, those elements matched
   detection but could not be safely converted; they are preserved unchanged
   and need manual investigation.
4. Cache: stale confbridge entries in Redis converge naturally within the 24h
   TTL; no manual invalidation required.
5. Ordering: neither order (code fix deploy vs migration) can create new
   corruption. However, applying the migration before the fixed code is
   deployed leaves any corruption written after the Phase 1 snapshot
   unrepaired (the post-check query catches it), so deploying the code fix
   first and then applying the migration is recommended for a complete repair.
6. Optional forensics: confbridge cache update failure error logs mark past
   corruption events.

## Scope exclusion

`exprJSONArrayAppend` has no `coalesce(recording_ids, '[]')` guard, unlike the
conference-manager and queue-manager equivalents, so an append against a SQL
NULL column is silently lost. Current `ConfbridgeCreate` always writes `[]`,
so only legacy or manually created NULL rows are affected. This asymmetry is
deliberately out of scope for VOIP-1305; it is recorded as an observation and
tracked in a separate follow-up Jira ticket.

## RST docs impact: none

The fix restores the intended storage contract (a JSON array of UUID strings)
for an internal column; the API and webhook shapes are unchanged. Confbridge
is an internal resource with no WebhookMessage, so there is no user facing
struct documentation to update.

## Service docs impact: none

The change is confined to `pkg/dbhandler` internals plus a data migration. No
routing table, event set, configuration, domain model shape, or dependency
changed, so no `docs/*.md` extraction target in bin-call-manager is affected.
