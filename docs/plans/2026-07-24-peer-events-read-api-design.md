# peer_events read API: contact address-set search over the raw peer/local log

- Issue: NOJIRA
- Class: new read path across 4 services (bin-timeline-manager, bin-common-handler,
  bin-contact-manager, bin-api-manager) + bin-openapi-manager spec
- Date: 2026-07-24
- Author: Lux (CPO), decision confirmed with 대표님

## 1. Problem

`bin-timeline-manager` already ingests `peer_events` (design:
`bin-timeline-manager/docs/plans/2026-07-24-add-peer-events-table-design.md`,
merged as `NOJIRA-timeline-manager-peer-events` #1135). That design explicitly
scoped OUT any read/query API (§8): "A read/query API for peer_events (RPC +
REST). Not requested yet; this design covers ingestion only."

This design adds that read path, so a customer's `contact_interactions` screen
can also surface `peer_events` for a contact's registered addresses.

**Explicit product decision (confirmed with 대표님, this session):**
- `contact_interactions` (identity-resolved, CRM-eligibility-filtered) is
  left completely untouched. This is an ADDITIVE new capability, not a
  replacement or a merge of the two data sources into one query.
- The new API returns `peer_events` rows AS-IS, including internal-resource
  noise (agent/AI/conference/SIP legs) that `contact_interactions`
  deliberately excludes. **No server-side eligibility filter is applied.**
  The client (square-admin / square-talk) is responsible for any
  presentation-layer filtering or grouping it wants to do with the noise.
- Filter shape: a NEW resource, `GET /contact_peer_events`, accepting
  `contact_id` (bin-api-manager resolves the contact's registered addresses
  server-side and passes the resulting `(type, target)` set down to
  timeline-manager) OR `peer_type` + `peer_target` (single-pair passthrough,
  power-user/debugging path, mirrors `contact_interactions`' own filter
  shape). This mirrors `contact_interactions`' precedent of doing identity
  resolution in the calling service, not pushing raw contact_id into
  timeline-manager (which has zero concept of `contact_id` by design, see
  the ingestion design's §2.5 "No contact_id resolution").

## 2. What already exists (verified) and what is new

**Already exists, untouched by this design:**
- `peer_events` ClickHouse table + ingestion path (§1 above).
- `contact_interactions` MySQL table + full read/write path
  (`bin-contact-manager/pkg/contacthandler/interaction*.go`,
  `bin-api-manager/server/contact_interactions.go`,
  `pkg/servicehandler/{interaction,serviceagent_interaction}*.go`).
- `bin-api-manager`'s existing `contactGet` helper
  (`pkg/servicehandler/contact.go:19`) already returns `cmcontact.Contact`
  with `.Addresses []cmcontact.Address` populated (each has an embedded
  `commonaddress.Address` with `Type`/`Target`) — this is the exact
  address-set source needed for the `contact_id` filter mode, with NO new
  contact-manager endpoint required.

**New in this design (4 services + 1 spec repo):**

| Service | New pieces |
|---|---|
| `bin-timeline-manager` | `dbhandler.PeerEventList` (ClickHouse query), `peereventhandler.PeerEventHandler` (business logic + pagination), `listenhandler` route `/v1/peer-events` (GET — `customer_id`/`page_token`/`page_size` as query params, `peer_pairs` array in the body, mirroring `v1AnalysesGet`'s existing GET-with-body-filter precedent in this same service; see §5.4) |
| `bin-common-handler` | `requesthandler.TimelineV1PeerEventList` (mirrors `TimelineV1AnalysisList`'s GET-with-query-authority-plus-body-filter shape, not `TimelineV1EventList`'s POST shape) |
| `bin-contact-manager` | none (deliberately — see §1's "left completely untouched") |
| `bin-api-manager` | new `serviceHandler.PeerEventList` / `ServiceAgentPeerEventList`, new `server/contact_peer_events.go`, new OpenAPI-generated types |
| `bin-openapi-manager` | new `openapi/paths/contact_peer_events/main.yaml` (+ `service_agents/contact_peer_events.yaml`), new response schema |

## 3. Filter contract

Exactly one of the following is required (same "exactly one filter" discipline
as `contact_interactions`, `server/contact_interactions.go:66-81`):

- `contact_id` (uuid) — bin-api-manager resolves via the existing
  `contactGet` call, builds the address set from `contact.Addresses`
  (`[]{Type, Target}`, deduplicated), and passes it to timeline-manager as
  `peer_pairs: [{peer_type, peer_target}, ...]`. If the contact has zero
  addresses, short-circuits to an empty result (no RPC call to
  timeline-manager) — mirrors `contact_interactions`' contract of never
  producing an unbounded/unfiltered query by accident.
- `peer_type` + `peer_target` (single pair) — passed through directly as a
  one-element `peer_pairs` array. This is the same shape
  `contact_interactions` already exposes today, kept for parity/debugging
  (a caller who already knows the exact peer identity, e.g. square-admin's
  "raw activity for this phone number" debug view, without needing a
  resolved `contact_id`).

`address_id` (as `contact_interactions` supports) is explicitly NOT
supported in v1 here — resolving a single `address_id` to `(type, target)`
would require a new contact-manager RPC (`AddressGet` is already used
elsewhere in `bin-api-manager` for a DIFFERENT purpose — tenant-scoped
address CRUD — but wiring it into this new handler is extra surface for a
mode nobody asked for yet). `contact_id` already covers the primary use
case (\"show me everything for this contact, noise included\"); `address_id`
can be added later as a trivial extension if a caller needs
address-level (not contact-level) scoping.

## 4. Schema (ClickHouse query, bin-timeline-manager)

```sql
SELECT timestamp, customer_id, publisher, event_type, reference_id, direction,
       peer_type, peer_target, local_type, local_target, data
FROM peer_events
WHERE customer_id = ?
  AND (peer_type, peer_target) IN ( (?,?), (?,?), ... )
  AND timestamp < ?   -- page token, omitted on first page
ORDER BY timestamp DESC
LIMIT ?
```

This matches the ingestion design's §4 `ORDER BY (customer_id, peer_type,
peer_target, timestamp)` primary-index rationale — a multi-value `(peer_type,
peer_target) IN (...)` predicate is exactly the access pattern that ORDER BY
was chosen for.

Pagination: same cursor convention as `events`
(`bin-timeline-manager/pkg/dbhandler/event.go`'s `buildEventQuery` — `timestamp
< ?` token, `ORDER BY timestamp DESC LIMIT ?`, request `pageSize+1` to probe
`hasMore`). `NextPageToken` = last row's `timestamp` formatted with
`commonutil.ISO8601Layout` (matches `eventhandler.List`'s convention exactly).

## 5. New code (bin-timeline-manager)

### 5.1 `models/peerevent/` (new package, mirrors `models/event/`)

```go
// models/peerevent/peerevent.go
package peerevent

type PeerEvent struct {
    Timestamp   time.Time       `json:"timestamp"`
    CustomerID  uuid.UUID       `json:"customer_id"`
    Publisher   string          `json:"publisher"`    // synthetic label: "call" / "conversation_message" / "conversation"
    EventType   string          `json:"event_type"`
    ReferenceID uuid.UUID       `json:"reference_id"`
    Direction   string          `json:"direction"`
    PeerType    string          `json:"peer_type"`
    PeerTarget  string          `json:"peer_target"`
    LocalType   string          `json:"local_type"`
    LocalTarget string          `json:"local_target"`
    Data        json.RawMessage `json:"data"`
}

// models/peerevent/request.go
type PeerPair struct {
    PeerType   string `json:"peer_type"`
    PeerTarget string `json:"peer_target"`
}

type PeerEventListRequest struct {
    CustomerID uuid.UUID  `json:"customer_id"`
    PeerPairs  []PeerPair `json:"peer_pairs"`

    PageToken string `json:"page_token,omitempty"`
    PageSize  int    `json:"page_size,omitempty"`
}

// models/peerevent/response.go
type PeerEventListResponse struct {
    Result        []*PeerEvent `json:"result"`
    NextPageToken string       `json:"next_page_token,omitempty"`
}
```

Deliberately a NEW model package, not appended to `models/event`: `Event` and
`PeerEvent` are different ClickHouse tables with non-overlapping column sets
(`Event` has no `reference_id`/`peer_type`/etc.; `PeerEvent` has no
`resource_id`/`data_type`), and the two request/response shapes diverge
(`EventListRequest` keys on `publisher`+`resource_id`, `PeerEventListRequest`
keys on a `peer_pairs` array). Mirrors the ingestion design's already-created
`dbhandler.PeerEventRow` being separate from `EventRow` for the same reason.

### 5.2 `pkg/dbhandler/peer_event_read.go` (new)

**Round 1 fix:** `PeerPairFilter` is defined and used ONLY inside `pkg/dbhandler`
(this file). It must NOT leak into `peereventhandler`'s public interface —
see §5.3's corrected signature, which takes primitives instead
(`eventhandler.EventHandler.List` never imports a `dbhandler.*` type into its
own interface either; `pkg/listenhandler` must not need to import
`pkg/dbhandler` just to call `peerEventHandler.List(...)`, matching how
`v1_events.go` never imports `dbhandler` for that purpose today).

**Round 1 fix:** `PeerEventList` must also be added to the `DBHandler`
interface in `pkg/dbhandler/main.go` (alongside the existing
`EventBatchInsert`/`EventList`/`AggregatedEventList`/`PeerEventBatchInsert`/etc.
entries) — every other list method in that file is declared on the interface,
not just the concrete `*dbHandler` type; omitting it here would break both
`mockgen -source main.go`'s ability to produce a usable mock and
`peereventhandler`'s convention of depending on the interface, not the
concrete type.

```go
package dbhandler

// PeerPairFilter is a (peer_type, peer_target) pair used for the ClickHouse
// multi-value match. dbhandler-local only — see the Round 1 fix note above.
type PeerPairFilter struct {
    PeerType   string
    PeerTarget string
}

// buildPeerEventQuery constructs the SQL query and args for listing peer_events.
func buildPeerEventQuery(
    customerID uuid.UUID,
    pairs []PeerPairFilter,
    pageToken string,
    pageSize int,
) (string, []interface{}) {
    query := `
        SELECT timestamp, customer_id, publisher, event_type, reference_id,
               direction, peer_type, peer_target, local_type, local_target, data
        FROM peer_events
        WHERE customer_id = ?
    `
    args := []interface{}{customerID.String()}

    // Multi-value (peer_type, peer_target) match, OR-expanded for portability
    // with the existing buildEventConditions style in this package (event.go
    // already prefers explicit OR-joins over driver-specific tuple-IN syntax).
    if len(pairs) > 0 {
        var ors []string
        for _, p := range pairs {
            ors = append(ors, "(peer_type = ? AND peer_target = ?)")
            args = append(args, p.PeerType, p.PeerTarget)
        }
        query += " AND (" + strings.Join(ors, " OR ") + ")"
    }

    if pageToken != "" {
        query += " AND timestamp < ?"
        args = append(args, pageToken)
    }

    query += " ORDER BY timestamp DESC LIMIT ?"
    args = append(args, pageSize)

    return query, args
}

// PeerEventList queries peer_events from ClickHouse.
func (h *dbHandler) PeerEventList(
    ctx context.Context,
    customerID uuid.UUID,
    pairs []PeerPairFilter,
    pageToken string,
    pageSize int,
) ([]*peerevent.PeerEvent, error) {
    if h.conn == nil {
        return nil, errors.New("clickhouse connection not established")
    }

    query, args := buildPeerEventQuery(customerID, pairs, pageToken, pageSize)

    rows, err := h.conn.Query(ctx, query, args...)
    if err != nil {
        return nil, errors.Wrap(err, "could not query peer_events")
    }
    defer func() { _ = rows.Close() }()

    result := []*peerevent.PeerEvent{}
    for rows.Next() {
        var e peerevent.PeerEvent
        var custIDStr, refIDStr, data string
        if err := rows.Scan(
            &e.Timestamp, &custIDStr, &e.Publisher, &e.EventType, &refIDStr,
            &e.Direction, &e.PeerType, &e.PeerTarget, &e.LocalType, &e.LocalTarget, &data,
        ); err != nil {
            return nil, errors.Wrap(err, "could not scan peer_event row")
        }
        e.CustomerID = uuid.FromStringOrNil(custIDStr)
        e.ReferenceID = uuid.FromStringOrNil(refIDStr)
        e.Data = json.RawMessage(data)
        result = append(result, &e)
    }
    if err := rows.Err(); err != nil {
        return nil, errors.Wrap(err, "error iterating rows")
    }
    return result, nil
}
```

`PeerPairFilter` is a small dbhandler-local type — kept OUT of
`peereventhandler`'s public interface per the Round 1 fix above (§5.3 takes
primitives, converts to `[]PeerPairFilter` only at the dbhandler call site
inside `peereventhandler.List`'s own body).

**Same ClickHouse scan-type constraint as `events` (verified from this
service's CLAUDE.md "Critical Implementation Notes"): scan `String` columns
into Go `string`, not custom types — `CustomerID`/`ReferenceID` scanned as
string then parsed via `uuid.FromStringOrNil`, matching `EventList`'s
existing `publisherStr` pattern exactly.**

### 5.3 `pkg/peereventhandler/` (new package, mirrors `pkg/eventhandler/`)

**Round 1 fix:** signature now takes a locally-defined primitive-pair type
(no `dbhandler.*` import), matching `eventhandler.EventHandler.List`'s
real "primitives only" interface exactly. The wire-DTO
(`models/peerevent.PeerPair`) → handler-param
(`peereventhandler.PeerPair`) conversion is a trivial 1:1 field copy
performed in `v1_peer_events.go` (§5.4) — named `toPeerEventHandlerPairs`
below, closing Round 1 finding #2 (previously-unnamed conversion step).

```go
package peereventhandler

//go:generate mockgen -package peereventhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

// PeerPair is peereventhandler's own primitive pair type — intentionally NOT
// dbhandler.PeerPairFilter (Round 1 fix: no dbhandler.* leaks into this
// package's public interface) and NOT models/peerevent.PeerPair (the wire
// DTO belongs to the listenhandler/transport boundary, not the business
// logic layer) -- three distinct, identically-shaped types at three layers,
// same pattern the rest of this codebase uses at handler boundaries.
type PeerPair struct {
    PeerType   string
    PeerTarget string
}

type PeerEventHandler interface {
    List(ctx context.Context, customerID uuid.UUID, pairs []PeerPair, pageToken string, pageSize int) (*peerevent.PeerEventListResponse, error)
}

const (
    DefaultPageSize = 100
    MaxPageSize     = 1000  // same clamp policy as eventhandler.List
)

func (h *peerEventHandler) List(ctx context.Context, customerID uuid.UUID, pairs []PeerPair, pageToken string, pageSize int) (*peerevent.PeerEventListResponse, error) {
    if customerID == uuid.Nil {
        return nil, errors.New("customer_id is required")
    }
    if len(pairs) == 0 {
        return nil, errors.New("at least one peer_type+peer_target pair is required")
    }

    if pageSize <= 0 {
        pageSize = DefaultPageSize
    }
    if pageSize > MaxPageSize {
        pageSize = MaxPageSize
    }

    dbPairs := make([]dbhandler.PeerPairFilter, len(pairs))
    for i, p := range pairs {
        dbPairs[i] = dbhandler.PeerPairFilter{PeerType: p.PeerType, PeerTarget: p.PeerTarget}
    }

    rows, err := h.db.PeerEventList(ctx, customerID, dbPairs, pageToken, pageSize+1)
    if err != nil {
        return nil, errors.Wrap(err, "could not list peer events")
    }

    res := &peerevent.PeerEventListResponse{Result: rows}
    if len(rows) > pageSize {
        res.Result = rows[:pageSize]
        res.NextPageToken = rows[pageSize-1].Timestamp.Format(commonutil.ISO8601Layout)
    }
    return res, nil
}
```

Note: `peereventhandler`'s own `.go` file DOES import `pkg/dbhandler` (it
already imports the `DBHandler` interface type for its `db` field, exactly
like `eventhandler` does) — the Round 1 fix is specifically that
`dbhandler.PeerPairFilter` does not appear in `PeerEventHandler`'s exported
interface signature, so `pkg/listenhandler` (which only talks to
`peereventhandler.PeerEventHandler`, never to `dbhandler` directly) never
needs a `dbhandler` import. The `dbPairs` conversion above is entirely
internal to this package's method body.

### 5.4 `pkg/listenhandler/v1_peer_events.go` (new) + `main.go` routing

**GET, not POST** (corrected per 대표님's review — see rationale below).
New route `regV1PeerEvents = regexp.MustCompile(`/v1/peer-events(\?|$)`)`,
`GET`, added to the `switch` in `processRequest` alongside the existing
`regV1Correlations`/`regV1AnalysesGet` GET cases.

**Round 3 fix (BLOCKER):** the regex MUST be `(\?|$)`-terminated, NOT
`$`-only. Every real caller (§6.1's `TimelineV1PeerEventList`) always
builds a URI with a trailing query string
(`/v1/peer-events?customer_id=...&page_token=...&page_size=...`), and a
plain `/v1/peer-events$` regex does NOT match a string with a `?...` suffix
— every production request would fall through to `processRequest`'s
`default: 404` case, making the entire new endpoint permanently
unreachable. The correct precedent to copy is `regV1Analyses =
regexp.MustCompile(`/v1/analyses(\?|$)`)` (`pkg/listenhandler/main.go:37`)
— NOT `regV1Correlations` (`/v1/correlations/` + regUUID + `$`), which is
`$`-anchored only because that route is a path-only single-resource GET
with no query string ever appended, a different shape than this list
endpoint. (An earlier draft of this section copied the wrong sibling
regex; corrected here.)

This mirrors the REAL existing precedent in THIS SAME SERVICE for "list
with an array/complex filter": `v1AnalysesGet` (`pkg/listenhandler/v1_analyses.go:78-129`,
`GET /v1/analyses`) takes `customer_id`/`page_token`/`page_size` as query
params (the "requesthandler authority", per that file's own comment) and
an arbitrary JSON filter map in the body — GET with a body is not a
contradiction here because this is an internal RabbitMQ RPC transport
(`sock.Request{URI, Method, Data}`), not literal HTTP; `sock.RequestMethod`
is just a routing label the `listenhandler`'s regex+switch dispatches on,
with no REST-framework body-on-GET restriction to violate.

(An earlier draft of this doc proposed POST here, reasoning from
`/v1/events`/`/v1/aggregated-events`'s POST-for-list precedent alone. That
reasoning under-weighted the more on-point precedent: `v1_analyses.go`
already does exactly "GET + query params for authority fields + body for
the array/complex filter" for the same "list filtered by something
richer than a single ID" shape this design needs. Corrected here.)

Request/response DTOs in `pkg/listenhandler/models/request/peer_event.go` /
`models/response/peer_event.go`, following the `v1AnalysesGet` pattern
(`pkg/listenhandler/v1_analyses.go:82-129`): `customer_id` parsed from the
query string via `queryValues(m.URI)` (the existing helper in
`v1_analyses.go`, reused here — no new URI-parsing helper needed), the
`peer_pairs` array (plus optional `page_token`/`page_size` if not passed as
query params — this design keeps them in the query string alongside
`customer_id`, matching `v1AnalysesGet`'s split exactly) parsed from
`m.Data` via `json.Unmarshal`, convert wire pairs to handler pairs, call
`h.peerEventHandler.List(...)`, map to response DTO, marshal.

```go
// v1PeerEventsGet handles GET /v1/peer-events — list peer_events rows
// matching the given (peer_type, peer_target) pairs, scoped to customer_id.
// customer_id/page_token/page_size arrive as query params (the
// requesthandler authority, same split v1AnalysesGet uses); peer_pairs
// arrives as a JSON body (an array, same reason /v1/events keeps its
// `events []string` filter in the body rather than the query string).
func (h *listenHandler) v1PeerEventsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    log := logrus.WithField("func", "v1PeerEventsGet")

    // Round 3 fix (MINOR #2): nil-guard, matching every sibling handler in
    // this package that depends on an optionally-nil-until-fully-wired
    // handler (v1AnalysesGet/v1AnalysesPost/etc. all start with this same
    // check against h.analysisHandler). Prevents a nil-pointer panic if
    // NewListenHandler is ever called with peerEventHandler left nil
    // (e.g. an incomplete future refactor of the constructor wiring).
    if h.peerEventHandler == nil {
        return simpleResponse(http.StatusServiceUnavailable), nil
    }

    q := queryValues(m.URI)
    customerID := uuid.FromStringOrNil(q.Get("customer_id"))
    if customerID == uuid.Nil {
        return simpleResponse(http.StatusBadRequest), nil
    }
    pageToken := q.Get("page_token")
    pageSize := parsePageSize(q.Get("page_size"))

    var req request.V1DataPeerEventsGet
    if len(m.Data) > 0 {
        if err := json.Unmarshal(m.Data, &req); err != nil {
            log.Errorf("Could not unmarshal request. err: %v", err)
            return simpleResponse(http.StatusBadRequest), nil
        }
    }

    res, err := h.peerEventHandler.List(ctx, customerID, toPeerEventHandlerPairs(req.PeerPairs), pageToken, int(pageSize))
    if err != nil {
        log.Errorf("Could not list peer events. err: %v", err)
        return errorResponse(err), nil
    }

    result := &response.V1DataPeerEventsGet{
        Result:        res.Result,
        NextPageToken: res.NextPageToken,
    }
    data, err := json.Marshal(result)
    if err != nil {
        return nil, errors.Wrap(err, "could not marshal response")
    }
    return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}
```

**Round 3 fix (MINOR #3):** the request/response DTO struct bodies,
previously only described in prose, shown explicitly:

```go
// pkg/listenhandler/models/request/peer_event.go
package request

// V1DataPeerEventsGet is the body-carried filter for GET /v1/peer-events.
// customer_id/page_token/page_size are NOT here — they arrive as query
// params (parsed directly from m.URI in v1PeerEventsGet), matching
// v1AnalysesGet's own customer_id/page_token/page_size-via-query-param
// split. Only the array filter that doesn't fit cleanly in a query string
// lives in the body, same reason /v1/events keeps `events []string` there.
type V1DataPeerEventsGet struct {
    PeerPairs []PeerPair `json:"peer_pairs"`
}

type PeerPair struct {
    PeerType   string `json:"peer_type"`
    PeerTarget string `json:"peer_target"`
}

// pkg/listenhandler/models/response/peer_event.go
package response

import "monorepo/bin-timeline-manager/models/peerevent"

type V1DataPeerEventsGet struct {
    Result        []*peerevent.PeerEvent `json:"result"`
    NextPageToken string                 `json:"next_page_token,omitempty"`
}
```

**Round 1 fix (finding #2):** the `[]request.PeerPair → []peereventhandler.PeerPair`
conversion is a named function in this file:

```go
// toPeerEventHandlerPairs converts the wire-DTO pair slice
// (request.V1DataPeerEventsGet.PeerPairs) into peereventhandler's own
// primitive pair type. Trivial 1:1 field copy — kept as a named function
// (not inlined) so it has one obvious place to extend if the wire shape
// and handler shape ever diverge.
func toPeerEventHandlerPairs(pairs []request.PeerPair) []peereventhandler.PeerPair {
    res := make([]peereventhandler.PeerPair, len(pairs))
    for i, p := range pairs {
        res[i] = peereventhandler.PeerPair{PeerType: p.PeerType, PeerTarget: p.PeerTarget}
    }
    return res
}
```

`NewListenHandler` constructor gains a `peerEventHandler
peereventhandler.PeerEventHandler` parameter; `cmd/timeline-manager/main.go`
wires it up alongside the existing `eventHandler`.

## 6. New code (bin-common-handler)

### 6.1 `pkg/requesthandler/timeline_peer_events.go` (new)

**GET, not POST** — mirrors `TimelineV1AnalysisList`
(`pkg/requesthandler/timeline_analyses.go:70-89`) exactly: `customer_id`/
`page_token`/`page_size` in the query string, the array filter
(`peer_pairs`) JSON-marshaled into the body, sent with
`sock.RequestMethodGet`.

```go
package requesthandler

func (r *requestHandler) TimelineV1PeerEventList(ctx context.Context, req *tmpeerevent.PeerEventListRequest) (*tmpeerevent.PeerEventListResponse, error) {
    uri := fmt.Sprintf(
        "/v1/peer-events?customer_id=%s&page_token=%s&page_size=%d",
        req.CustomerID.String(), url.QueryEscape(req.PageToken), req.PageSize,
    )

    m, err := json.Marshal(struct {
        PeerPairs []tmpeerevent.PeerPair `json:"peer_pairs"`
    }{PeerPairs: req.PeerPairs})
    if err != nil {
        return nil, err
    }

    tmp, err := r.sendRequestTimeline(ctx, uri, sock.RequestMethodGet, "timeline/peer-events", requestTimeoutDefault, 0, ContentTypeJSON, m)
    if err != nil {
        return nil, err
    }

    var res tmpeerevent.PeerEventListResponse
    if errParse := parseResponse(tmp, &res); errParse != nil {
        return nil, errParse
    }
    return &res, nil
}
```

Add the method signature to the `RequestHandler` interface in `main.go`
(next to the existing `// timeline-manager events` block), regenerate
`pkg/requesthandler/mock_main.go`.

## 7. New code (bin-api-manager)

### 7.1 `pkg/servicehandler/contact_peer_event.go` (new)

```go
package servicehandler

// PeerEventList resolves the contact_id OR peer_type+peer_target filter into
// a timeline-manager peer_pairs query and returns the raw (unfiltered)
// peer_events rows. Unlike InteractionList, this NEVER applies CRM
// eligibility filtering — the caller (square-admin/square-talk) is expected
// to do any presentation-layer grouping/filtering of noise itself.
func (h *serviceHandler) PeerEventList(
    ctx context.Context,
    a *auth.AuthIdentity,
    contactID uuid.UUID,
    peerType, peerTarget string,
    pageToken string,
    pageSize uint64,
) ([]*tmpeerevent.PeerEvent, string, error) {
    if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
        return nil, "", serviceerrors.ErrPermissionDenied
    }

    pairs, err := h.resolvePeerPairs(ctx, a.CustomerID, contactID, peerType, peerTarget)
    if err != nil {
        return nil, "", err
    }
    if len(pairs) == 0 {
        return nil, "", nil // no addresses on this contact -> empty result, no RPC call
    }

    req := &tmpeerevent.PeerEventListRequest{
        CustomerID: a.CustomerID,
        PeerPairs:  pairs,
        PageToken:  pageToken,
        PageSize:   int(pageSize),
    }
    res, err := h.reqHandler.TimelineV1PeerEventList(ctx, req)
    if err != nil {
        return nil, "", err
    }
    return res.Result, res.NextPageToken, nil
}

// resolvePeerPairs implements the "exactly one filter" contract (§3):
// contact_id -> resolve via contactGet + tenant check, dedupe Contact.Addresses
// into peer_pairs; OR peer_type+peer_target -> single-pair passthrough.
func (h *serviceHandler) resolvePeerPairs(
    ctx context.Context,
    customerID uuid.UUID,
    contactID uuid.UUID,
    peerType, peerTarget string,
) ([]tmpeerevent.PeerPair, error) {
    switch {
    case contactID != uuid.Nil:
        ct, err := h.contactGet(ctx, contactID)
        if err != nil {
            return nil, err
        }
        // Tenant guard: never resolve another customer's contact (mirrors
        // interactionListByContact's STEP 0 in bin-contact-manager).
        if ct.CustomerID != customerID {
            return nil, serviceerrors.ErrNotFound
        }

        seen := make(map[tmpeerevent.PeerPair]struct{})
        var pairs []tmpeerevent.PeerPair
        for _, addr := range ct.Addresses {
            p := tmpeerevent.PeerPair{PeerType: string(addr.Type), PeerTarget: addr.Target}
            if _, ok := seen[p]; ok {
                continue
            }
            seen[p] = struct{}{}
            pairs = append(pairs, p)
        }
        return pairs, nil

    case peerType != "" && peerTarget != "":
        return []tmpeerevent.PeerPair{{PeerType: peerType, PeerTarget: peerTarget}}, nil

    default:
        return nil, cerrors.InvalidArgument(
            commonoutline.ServiceNameAPIManager, "INVALID_FILTER",
            "Exactly one filter is required: contact_id, or peer_type+peer_target.",
        )
    }
}
```

**Round 2 fix (MINOR):** the `default` branch above raises
`cerrors.InvalidArgument` directly inside `pkg/servicehandler` — this is a
SECOND enforcement point for the same "exactly one filter" contract that
§7.1's Round 1 fix note already says is "enforced ONCE, at the HTTP layer"
(`server/contact_peer_events.go`'s `filterCount` check). Reconciling this
explicitly rather than leaving it as an apparent contradiction: the
HTTP-layer `filterCount` check is the PRIMARY, always-hit guard for any
request coming through `GetContactPeerEvents`/`GetServiceAgentsContactPeerEvents`.
`resolvePeerPairs`'s own `default` case is defense-in-depth for the
`neither-provided` sub-case only, reachable in practice only if a future
caller invokes `serviceHandler.PeerEventList` directly (bypassing the
`server/` HTTP layer — e.g. a future internal caller, or a test that calls
the servicehandler method directly without going through
`GetContactPeerEvents`). Using `cerrors.InvalidArgument` here (rather than
a plain `errors.New`) is intentional, not an accidental layering violation:
`cerrors` is a shared package with no `server/`-only import restriction
(unlike, say, `gin.Context`), and returning the same typed, structured error
here that the HTTP layer would have produced keeps the error response
identical regardless of which entry point rejected the request. The
"both-provided" sub-case remains HTTP-layer-only (§7.1's Round 1 fix,
§11's test-plan note) since `resolvePeerPairs`'s `switch` order
(`contactID != uuid.Nil` checked first) would silently prefer `contact_id`
if both were somehow passed through — a defense-in-depth default for
"neither" is safe and cheap; defense-in-depth for "both" would require an
explicit both-provided check this design deliberately does not duplicate.

Also add `ServiceAgentPeerEventList` (same body, `PermissionAll` gate
instead of `PermissionCustomerAdmin|PermissionCustomerManager`, `a.CustomerID`
used directly) — mirrors `ServiceAgentInteractionList`'s real relationship to
`InteractionList` (`pkg/servicehandler/serviceagent_interaction.go:22-61`):
**Round 1 fix** — the real `ServiceAgentInteractionList` does NOT call
`agentGet`; it checks `a.IsAgent()`/`a.IsDirect()` and scopes directly by
`a.CustomerID`, exactly like the non-agent path, differing only in the
permission bitmask (`PermissionAll` vs
`PermissionCustomerAdmin|PermissionCustomerManager`). The prior draft of
this doc incorrectly said "agentGet instead of the plain-customer path" —
corrected here; `ServiceAgentPeerEventList` must NOT introduce an
`agentGet` call that has no precedent in the function it mirrors.

**Round 1 fix (MINOR finding #7):** `resolvePeerPairs`'s tenant check
(`if ct.CustomerID != customerID { return nil, serviceerrors.ErrNotFound }`)
intentionally does NOT use the `h.hasPermission(ctx, a, ct.CustomerID,
amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) →
ErrPermissionDenied` pattern used by every other contactID-touching method
in `pkg/servicehandler/contact.go` (`ContactGet`, `ContactUpdate`,
`ContactAddressCreate/Update/Delete`, etc.). This is a deliberate
anti-enumeration choice, same rationale already established in this
service for cross-tenant address claims
(`ContactAddressClaim`/`ServiceAgentContactAddressClaim`, both in
`contact_address.go`, return `ErrNotFound` rather than `ErrPermissionDenied`
for a cross-tenant address/contact — "avoid leaking the existence of
another tenant's [resource]"). `resolvePeerPairs` follows that established
precedent, not the plain-CRUD precedent, because knowing "a contact_id you
don't own exists" is exactly the enumeration risk those claim-path
functions were written to avoid.

**Round 1 fix (MINOR finding #6):** §3's filter contract is enforced ONCE,
at the HTTP layer (`server/contact_peer_events.go`'s `filterCount`
validation, mirroring `contact_interactions.go:66-81` exactly) — a request
with BOTH `contact_id` and `peer_type`+`peer_target` set is rejected with
`400 INVALID_FILTER` before ever reaching `serviceHandler.PeerEventList`/
`resolvePeerPairs`. `resolvePeerPairs`'s internal `switch` above checks
`contactID != uuid.Nil` first purely as an implementation detail of a
function that, by the time it runs, is only ever called with exactly one
of the two filters populated — this is not a defense-in-depth precedence
rule to test, and no "both provided" test case belongs in
`resolvePeerPairs`'s own unit tests (§11 corrected accordingly). Only the
HTTP-layer `filterCount` test needs a "both provided → 400" case, mirroring
`contact_interactions_test.go`'s existing equivalent.

### 7.2 `server/contact_peer_events.go` (new)

`GetContactPeerEvents` handler, mirrors `GetContactInteractions`
(`server/contact_interactions.go:19-91`): parse `contact_id` OR
`peer_type`+`peer_target` query params, exactly-one-filter validation,
`page_size`/`page_token`, call `h.serviceHandler.PeerEventList(...)`,
`c.JSON(200, GenerateListResponse(items, nextToken))`.

**Round 1 fix (MINOR finding #5):** `page_size` default/max at this
HTTP layer is `100`, clamped to `[1,100]` — identical to
`GetContactInteractions`'s own clamp
(`server/contact_interactions.go:33-39`), NOT `peereventhandler`'s internal
`DefaultPageSize=100`/`MaxPageSize=1000` (§5.3). This is intentional, not
an inconsistency to resolve: `bin-api-manager`'s HTTP layer is the
customer-facing contract and stays at the platform's existing external
page-size ceiling (100, matching every other list endpoint in this
service); `peereventhandler`'s internal `1000` ceiling is a defensive
inner bound mirroring `eventhandler.List`'s own internal RPC-layer clamp,
never actually reachable from this HTTP path since bin-api-manager's `100`
dominates first.

Plus
`server/service_agents_contact_peer_events.go` for the
`/service_agents/contact_peer_events` mirror (per
`voipbin-service-agent-endpoint-implementation` convention — new file, new
functions, does not touch the existing Admin/Manager handler).

### 7.3 Interface + mocks

Add `PeerEventList` / `ServiceAgentPeerEventList` signatures to the
`ServiceHandler` interface in `pkg/servicehandler/main.go` (next to the
existing `InteractionList`/`ServiceAgentInteractionList` block), regenerate
`pkg/servicehandler/mock_main.go`.

## 8. New code (bin-openapi-manager)

- `openapi/paths/contact_peer_events/main.yaml` — `GET /contact_peer_events`,
  query params `contact_id` (uuid) OR `peer_type`+`peer_target` (strings),
  `page_size`/`page_token`, response
  `$ref: '#/components/schemas/TimelineManagerPeerEventListResponse'`.
- `openapi/paths/service_agents/contact_peer_events.yaml` — same shape,
  `tags: [Service Agent]`.
- New schema `TimelineManagerPeerEvent` (mirrors the `peerevent.PeerEvent`
  Go struct fields 1:1 — this table has no `WebhookMessage`
  external/internal split the way channel resources do, since `data` IS
  already the original webhook payload verbatim; the RST/OpenAPI schema is
  the Go struct as-is) and `TimelineManagerPeerEventListResponse` (`result`
  array + `next_page_token`).
- Register both new path files under `paths:` in `openapi/openapi.yaml`.
- `go generate ./...` in bin-openapi-manager, then in bin-api-manager (per
  the standard OpenAPI-first workflow, `bin-api-manager/CLAUDE.md`).

## 9. RST docs (bin-api-manager/docsdev)

New user-visible endpoint → mandatory per `bin-api-manager/CLAUDE.md`'s "RST
Docs Sync" rule. New files
`docsdev/source/contact_peer_event_overview.rst` and
`docsdev/source/contact_peer_event_struct.rst`, registered in the docs
index, explicitly documenting the "raw, unfiltered, includes internal noise
— client must filter for presentation" contract so external developers are
not surprised by agent/AI/conference rows appearing alongside customer
rows. Rebuild per the mandatory steps (`rm -rf build && sphinx-build`,
`git add -f build/`).

## 10. Explicitly out of scope (carried from the ingestion design + this
session's decision)

- Any change to `contact_interactions`, its dbhandler, its read algorithm
  (set-MINUS, ownership periods), or its existing REST surface. Fully
  untouched.
- Any server-side eligibility/noise filter on the new endpoint. By explicit
  product decision this session: "노이즈도 포함해서 보여줄꺼야. 클라이언트에서
  알아서 처리하도록 할 예정이야." (include noise; client handles it).
- `address_id` filter mode (§3 — deferred, trivial future extension).
- Any write path. `peer_events` remains ingestion-only from the
  subscribehandler side; this design is 100% read-only additions.
- Any change to `peer_events` schema, ClickHouse migration, or the
  eligiblePeerEvents allowlist from the ingestion design.

## 11. Test plan

- Unit: `buildPeerEventQuery` — single pair, multi-pair OR-expansion,
  page-token presence/absence, matches the existing `buildEventQuery` test
  pattern (`pkg/dbhandler/event_test.go`).
- Unit: `PeerEventList` (dbhandler) — nil-conn guard, empty-pairs
  passthrough-to-SQL (handler layer is the one that rejects empty pairs,
  dbhandler itself stays a thin query executor consistent with `EventList`).
- Unit: `peereventhandler.List` — customer_id required, pairs required,
  page-size clamping (mirrors `eventhandler.List`'s existing test table).
- Unit: `toPeerEventHandlerPairs` (listenhandler) — trivial 1:1 field-copy
  conversion, single test case confirming shape parity (Round 1 fix #2/#8).
- Unit: `resolvePeerPairs` — contact_id path (dedupe, tenant-mismatch ->
  NotFound, zero-address contact -> empty pairs no RPC), peer_type+peer_target
  passthrough path, neither-provided -> InvalidArgument. NO "both-provided"
  case here (Round 1 fix #6 — that validation lives at the HTTP layer only,
  see below; `resolvePeerPairs` is never reachable with both filters set).
- Unit: `serviceHandler.PeerEventList` / `ServiceAgentPeerEventList` — mirrors
  `Test_InteractionList`/`Test_ServiceAgentInteractionList` structure.
- Unit: `server.GetContactPeerEvents` — HTTP-level filter validation mirrors
  `contact_interactions_test.go`, INCLUDING the "both contact_id AND
  peer_type+peer_target provided → 400 INVALID_FILTER" case (this is where
  that precedence/rejection behavior is actually enforced and tested,
  correcting the prior draft's misplacement of this case under
  `resolvePeerPairs`).
- Full verification workflow (`go mod tidy && go mod vendor && go generate
  ./... && go test ./... && golangci-lint run`) in bin-timeline-manager,
  bin-common-handler, AND bin-api-manager (three Go services touched);
  `go generate ./...` in bin-openapi-manager first.
- Manual/integration: confirm a contact with a mix of `tel`+`email`
  addresses returns rows across both address types, and that
  agent/AI/conference peer rows for the SAME phone number (if any exist in
  test fixtures) are NOT filtered out — the defining behavior of this
  endpoint vs. `contact_interactions`.

## 12. Round 1 review disposition

Round 1 adversarial review (independent subagent, cross-checked every
factual claim against source): **CHANGES_REQUESTED**, no BLOCKER-class
defects found. 4 MAJOR + 4 MINOR findings, all addressed inline above:

| # | Class | Finding | Fix location |
|---|---|---|---|
| 1 | MAJOR | `dbhandler.PeerPairFilter` leaked into `peereventhandler`'s public interface, unlike the real `eventhandler.List` (primitives only) | §5.3 (own `PeerPair` type) |
| 2 | MAJOR | Wire-DTO → handler-param conversion function was never named/spec'd | §5.4 (`toPeerEventHandlerPairs`) |
| 3 | MAJOR | `PeerEventList` omitted from `DBHandler` interface in `pkg/dbhandler/main.go` | §5.2 |
| 4 | MAJOR | `ServiceAgentPeerEventList` incorrectly claimed to use `agentGet`; real `ServiceAgentInteractionList` uses `a.CustomerID` directly | §7.1 |
| 5 | MINOR | `page_size` default/max at the bin-api-manager HTTP layer left unstated | §7.2 |
| 6 | MINOR | "both filters provided" precedence case misattributed to `resolvePeerPairs` instead of the HTTP-layer `filterCount` check | §7.1, §11 |
| 7 | MINOR | `resolvePeerPairs`'s `ErrNotFound`-not-`ErrPermissionDenied` tenant-check style left unjustified | §7.1 (anti-enumeration precedent cited) |
| 8 | MINOR | No test named for the wire-DTO conversion function | §11 |

Round 2 review is required next (3-round floor per VoIPBin design-first
convention), even though Round 1 already reached CHANGES_REQUESTED with only
MAJOR/MINOR (no BLOCKER) findings.

## 13. Round 2 review disposition

Round 2 adversarial review (independent subagent, verified each of Round 1's
8 fixes against the doc text AND against source; also did a fresh pass for
new issues): **CHANGES_REQUESTED**, no BLOCKER/MAJOR found. All 8 Round 1
fixes verified genuinely correct and complete (not just claimed) — including
re-confirming the three-pair-type data flow compiles/imports correctly, the
`page_size` [1,100] vs [100,1000] split matches real
`server/contact_interactions.go:33-39`, and the `ErrNotFound`
anti-enumeration precedent matches real `contact_address.go`. Also
independently re-verified the GET-not-POST correction (made between Round 1
and Round 2, per 대표님's review) is structurally sound: §5.4/§6.1's GET +
query-authority + body-filter shape was not part of Round 2's dispatched
scope but is internally consistent with the rest of the doc as read.

One NEW MINOR finding: `resolvePeerPairs`'s `default` (neither-filter-provided)
branch raises `cerrors.InvalidArgument` directly inside `pkg/servicehandler`,
a second enforcement point beyond the HTTP-layer `filterCount` check the
doc's Round 1 fix already describes as the sole enforcement point. Reconciled
in §7.1 above: this is intentional defense-in-depth for direct
`serviceHandler.PeerEventList` callers that bypass `server/`, not a
contradiction — the doc previously stated "enforced once" without
acknowledging this secondary guard existed; now explicit.

Structural check: all section headers (`## 1`–`## 13`, `### 5.1`–`5.4`,
`### 6.1`, `### 7.1`–`7.3`) confirmed sequential and non-duplicated, no
leftover fragments from iterative patching.

No BLOCKER/MAJOR in Round 1 or Round 2. Round 3 (final, minimum-floor round)
is required next per the design-first convention's 3-round floor, followed
by 2 consecutive APPROVEs to close the loop.

## 14. Round 3 review disposition

Round 3 adversarial review (independent subagent, focused primarily on the
GET-vs-POST change made after Round 1/before Round 2, which had not yet
received independent scrutiny): **CHANGES_REQUESTED**, 1 BLOCKER + 2 MINOR.

- **BLOCKER (fixed in §5.4):** `regV1PeerEvents`'s regex was
  `/v1/peer-events$` ($-anchored only), which does NOT match any real
  request URI — every actual caller (§6.1's `TimelineV1PeerEventList`)
  always appends a query string (`?customer_id=...`), so every production
  request would have 404'd, permanently. The doc had copied the wrong
  sibling precedent (`regV1Correlations`, a path-only route with no query
  string) instead of the correct one (`regV1Analyses =
  regexp.MustCompile(`/v1/analyses(\?|$)`)`). Fixed: `regV1PeerEvents =
  regexp.MustCompile(`/v1/peer-events(\?|$)`)`.
- **MINOR (fixed in §5.4):** `v1PeerEventsGet` was missing the
  `h.peerEventHandler == nil` guard every sibling GET handler in this
  package has. Added.
- **MINOR (fixed in §5.4):** the `request.V1DataPeerEventsGet` /
  `response.V1DataPeerEventsGet` struct bodies were referenced in prose but
  never shown as code (unlike every other new-model section in this doc).
  Added explicit struct definitions.

Round 3 also re-verified (no regressions found): the `v1AnalysesGet`/
`TimelineV1AnalysisList` GET-with-body precedent is real and accurate;
`queryValues`/`parsePageSize` helpers already exist in
`pkg/listenhandler/v1_analyses.go` and are callable without new imports;
GET-with-non-empty-body is an established platform-wide pattern (not
special-cased/stripped anywhere in `bin-common-handler`); no stale
`PeerEventsPost`-era naming remains anywhere in the doc; §7.1's
`a.CustomerID`-not-`agentGet` fix and §5.2's `DBHandler` interface addition
both remain intact; all section headers remain sequential and
non-duplicated.

This was a genuine BLOCKER caught only because Round 3 gave the
GET-vs-POST change (introduced between Round 1 and Round 2) its first
independent adversarial pass — confirming the value of the 3-round floor
even when Rounds 1/2 were otherwise clean. Round 4 is required next: this
was NOT an APPROVE, so the "2 consecutive APPROVEs" counter has not yet
started.

## 15. Round 4 and Round 5 review disposition — loop closed

- **Round 4:** APPROVE, 0 findings. Independently re-verified the Round 3
  regex fix against both real input shapes (`/v1/peer-events` and
  `/v1/peer-events?customer_id=...&page_token=...&page_size=...`), checked
  for collisions against every other route in
  `bin-timeline-manager/pkg/listenhandler/main.go`, and confirmed the
  `nil`-guard and DTO struct fixes were genuinely present in code (not just
  re-asserted in prose). This was APPROVE #1 of the required 2 consecutive
  APPROVEs.
- **Round 5:** APPROVE, 0 findings. Independent fresh pass over the entire
  document (all 14 sections at the time), with explicit spot-checks not
  covered by Rounds 1–4: the OR-expansion SQL pattern (§4/§5.2) against real
  precedent in `bin-contact-manager/pkg/dbhandler/interaction.go` and
  `bin-timeline-manager/pkg/dbhandler/event.go`; the `models/peerevent`
  package-split rationale (§5.1) against real `models/event/*.go` shapes;
  the `DBHandler` interface insertion point (§5.2) against the live
  interface in `pkg/dbhandler/main.go`. This was APPROVE #2 — the loop
  closed (minimum 3-round floor exceeded, 2 consecutive APPROVEs reached).
  The design was implementation-ready as of this point.

## 16. Post-approval redesign — `peer`/`local` response as `commonaddress.Address`; internal filter plumbing unified (external `peer_type`/`peer_target` query params UNCHANGED)

During implementation (after the design loop closed), 대표님 raised three
follow-up questions that led to a real redesign of the response shape and
filter contract described in §3/§4/§5.1–§5.3 above. **This section is the
authoritative record of what actually shipped; §3/§4/§5.1–§5.3's flat
`peer_type`/`peer_target` response fields and `PeerPairFilter`/`PeerPair`
plumbing are superseded by what follows.**

**What changed and why:**

1. **Response shape:** `PeerEvent.Peer` / `.Local` are now the full
   `commonaddress.Address` struct (`Type`/`Target`/`TargetName`/`Name`/
   `Detail`), JSON-serialized, exactly mirroring `contact_interactions`'
   own `Interaction.Peer`/`.Local` shape — not flat `peer_type`/
   `peer_target` strings. Rationale: this is a JSON+generated-column split
   identical in spirit to `contact_interactions`' MySQL STORED GENERATED
   COLUMN pattern (`bin-contact-manager/scripts/database_scripts_test/
   contacts.sql`), just implemented in Go at insert time since ClickHouse
   has no generated-column equivalent. Consumers get a structurally
   consistent `Address` object across both `contact_interactions` and this
   new endpoint, instead of two different filter/response shapes for
   conceptually the same "who is this" data.
2. **Storage:** ClickHouse `peer_events` gained two new physical `String`
   columns, `peer` and `local` (migration
   `000006_add_peer_events_address_json_columns`), holding the
   JSON-marshaled `Address`. The original `peer_type`/`peer_target`/
   `local_type`/`local_target` columns are UNCHANGED and remain — they stay
   internal-only, used solely for the `ORDER BY`/`WHERE` index (see
   `dbhandler.PeerEventRow`'s doc comment); they are populated by the same
   `buildPeerEventRows`/`newPeerEventRow` insert-time logic (added by this
   migration) that also marshals the JSON columns, from the exact same
   `commonaddress.Address` value — no drift possible between the search
   columns and the response JSON since both are derived from one value in
   one place.
3. **Filter contract simplified:** the 3-layer `PeerPair`/`PeerPairFilter`
   primitive-type plumbing described in §5.2/§5.3 (`peer_type string` +
   `peer_target string` at every layer) is gone. Every layer now passes
   `commonaddress.Address` (or `[]commonaddress.Address`) end-to-end:
   `bin-api-manager`'s `resolvePeerAddresses` (renamed from
   `resolvePeerPairs`) returns `[]commonaddress.Address` directly from
   `Contact.Addresses` or the single-address filter;
   `PeerEventListRequest.PeerAddresses` (renamed from `PeerPairs`) carries
   `[]commonaddress.Address`; `peereventhandler.List` and
   `dbhandler.PeerEventList` both take `addrs []commonaddress.Address`
   and read `.Type`/`.Target` only where the SQL `WHERE` clause needs flat
   values. This removes a full type-conversion layer (wire DTO →
   handler-local primitive → dbhandler-local filter) that existed only to
   avoid leaking `dbhandler.PeerPairFilter` up the stack — moot now that no
   dbhandler-local type exists to leak; `commonaddress.Address` is already
   the platform's standard cross-service filter/response type (used
   unchanged by `contact_interactions` today).
4. **API filter parameter NOT renamed — corrected from an earlier draft of
   this section.** The external HTTP query contract is UNCHANGED: the
   single-pair filter mode is still two separate `peer_type`+`peer_target`
   query params (`bin-api-manager/server/contact_peer_events.go`,
   `service_agents_contact_peer_events.go` — `params.PeerType`/
   `params.PeerTarget`), NOT a single `peer_address`-shaped parameter. The
   OpenAPI spec (`openapi/paths/contact_peer_events/main.yaml`,
   `openapi/paths/service_agents/contact_peer_events.yaml`) and generated
   types (`GetContactPeerEventsParams.PeerType`/`.PeerTarget`) reflect this
   — no `peer_address` param exists anywhere in the spec or generated code.
   Only the code ONE LAYER IN from the HTTP boundary changed: the handler
   still parses two query params, then immediately constructs a single
   `*commonaddress.Address{Type, Target}` from them (line ~78-81 of
   `contact_peer_events.go`) before calling `resolvePeerAddresses`/
   `PeerEventList` — i.e. the `commonaddress.Address` unification described
   in point 3 starts at the servicehandler boundary, not the HTTP query
   string. `contact_id` filter mode is unchanged in behavior (§3).
   (An earlier draft of this point incorrectly claimed the query param
   itself was renamed to `peer_address` — caught and corrected in Round 1
   PR review, see §17.)

**Why this was not caught in Rounds 1–5:** those rounds reviewed the
design as originally drafted (flat `peer_type`/`peer_target` throughout,
per §3/§4/§5.1–§5.3), which was internally consistent and passed 2
consecutive APPROVEs on its own terms. The redesign is a scope change
made during implementation in response to 대표님's direct review of the
shipped response/request shapes, not a defect the review loop missed. No
new review round was run for this section; the actual code (cross-checked
against this section during r13a/r13b) is the source of truth going
forward. §3/§4/§5.1–§5.3 are left as-is above for historical record of the
original design rationale (filter contract structure, ClickHouse ORDER BY
choice, pagination convention all remain valid and unchanged); readers
implementing against this doc should treat §16 as overriding the specific
field names and type signatures in those earlier sections.

## 17. Round 1 PR review disposition (commit 44501b298, the §16 redesign)

Round 1 adversarial PR review (independent subagent, run retroactively
after the commit was already pushed to open PR #1136 — see the process
note below): **CHANGES_REQUESTED**, 0 BLOCKER, 2 MAJOR, 2 MINOR.

- **MAJOR #1 (fixed above in §16 point 4):** the commit message and this
  doc's original §16 point 4 both claimed the external API filter param
  was renamed to `peer_address`. This was false — the shipped code keeps
  `peer_type`+`peer_target` as two separate query params at the HTTP
  boundary; only the servicehandler layer, one hop in, unifies them into
  `commonaddress.Address`. §16 point 4 corrected to state this accurately.
- **MAJOR #2 (fixed below):** `contact_peer_event_overview.rst` was only
  reformatted (table border widths) in the r13a pass, not actually
  updated for the redesign — it happened to still read correctly for the
  request/filter side (since, per MAJOR #1, that side didn't change), but
  the doc never explained why the response side (`peer`/`local`, now
  Address objects) and the request side (`peer_type`/`peer_target`, still
  flat strings) are asymmetric. Fixed by adding an explicit note.
- **MINOR (fixed above):** `gofmt` was not clean on
  `bin-timeline-manager/models/peerevent/peerevent.go` (struct tag
  alignment) and `bin-timeline-manager/pkg/dbhandler/main.go` (import
  grouping). Both fixed with `gofmt -w`; `golangci-lint` was already 0
  issues on both (gofmt isn't in the configured linter set here), but repo
  hygiene expects clean `gofmt` regardless.
- **MINOR (no action needed):** the error message string in
  `contact_peer_events.go`/`service_agents_contact_peer_events.go`
  ("Exactly one filter is required: contact_id, or
  peer_type+peer_target.") is accurate given MAJOR #1's resolution — no
  change needed.

Round 1 also independently re-verified (no regressions found): ClickHouse
migration `000006`'s column order matches `peer_event_insert.go`'s INSERT
list and `peer_event_read.go`'s SELECT/Scan exactly; `newPeerEventRow()`'s
3 call sites in `subscribehandler/peer_event.go` all pass arguments in
identical order with correct JSON-marshal-failure fallback; `DBHandler`
interface + mock, `peereventhandler.List` + mock, and the
`bin-common-handler` RPC wire shape are all internally consistent;
`resolvePeerPairs → resolvePeerAddresses` rename is complete with zero
live references to the old name remaining anywhere in the 4 services;
`go build`/`go test`/`golangci-lint` pass clean on all 4 touched services.

**Process note:** this Round 1 review should have run BEFORE the commit
was pushed to PR #1136, per the design-first-with-review-loops convention
("every push to an open PR branch is a review-loop trigger, no exception
for small/follow-up commits"). It was skipped and only caught when 대표님
asked directly "리뷰 루프 돌렸어?" — run retroactively here. Round 2 is
required next (minimum 3-round floor), followed by 2 consecutive
APPROVEs to close.

## 18. Round 2 PR review disposition (commit 1721b9bae, the Round 1 fixes)

Round 2 adversarial PR review (independent subagent): **CHANGES_REQUESTED**,
0 BLOCKER, 1 new MAJOR, 0 new MINOR. All of Round 1's fixes were
independently re-verified as genuine (query-param claim, gofmt/goimports,
`bin-timeline-manager` build/test/lint, Sphinx rebuild, `:ref:` label
resolution) — no regressions.

- **MAJOR (fixed below):** the "Request vs. Response Shape" RST note added
  in commit 1721b9bae (and the pre-existing prose in
  `contact_peer_event_struct.rst`, carried over unscrutinized from commit
  44501b298) claimed `peer`/`local` match "Contact Interactions' own
  peer/local shape" and "Contact's Address" shape. Neither claim holds up:
  there is no dedicated "Contact Interactions" struct page anywhere in
  `docsdev/source/` for a reader to actually check, and `Contact.Address`
  (`contact-struct-contact-address`, fields `id`/`type`/`target`/
  `is_primary`/`tm_create`) is a genuinely different shape from
  `commonaddress.Address` (`type`/`target`/`target_name`/`name`/`detail`)
  — they share only `type`/`target`. The `:ref:` links technically resolve
  (Sphinx doesn't error), which is why this survived Round 1's own build
  check — but they point to unrelated/nonexistent content, making the
  claim misleading despite building cleanly. Fixed in both
  `contact_peer_event_struct.rst` and `contact_peer_event_overview.rst`:
  reworded to state `peer`/`local` are `commonaddress.Address`, the
  platform's standard cross-service address type, explicitly distinct
  from Contact's Address shape — no claim of matching an undocumented
  Interaction struct.

Round 2 also independently re-derived (no new issues): ClickHouse column
order across migrations 000005+000006 vs. `peer_event_insert.go`'s INSERT
list and `peer_event_read.go`'s SELECT/Scan (exact match, re-verified
digit-by-digit, not re-trusting Round 1's claim); the OR-expansion query
in `peer_event_read.go` uses parameterized placeholders throughout (no
injection risk); the "exactly one filter" HTTP-layer gate in
`contact_peer_events.go` is the sole enforcement point, with
`resolvePeerAddresses`'s `default` branch confirmed unreachable in
practice as documented; §17's narrative checked for internal consistency
against its own claims (specific, non-overclaiming, self-critical about
the missed review-loop trigger) — no issues.

This confirms the value of running Round 2 independently rather than
re-trusting Round 1's "verified clean" list: the RST accuracy problem was
introduced in the ORIGINAL commit (44501b298) and only surfaced because
Round 2 re-derived facts instead of accepting Round 1's summary at face
value. Round 3 is required next (minimum 3-round floor).

## 19. Round 3 PR review disposition (final minimum-floor round) — APPROVE #1

Round 3 adversarial PR review (independent subagent): **APPROVE**, 0
BLOCKER, 0 MAJOR, 1 MINOR (non-blocking). This is APPROVE #1 of the
required 2 consecutive APPROVEs — the loop does not close on this round
alone.

Independently re-verified (not re-trusting Rounds 1–2's summaries): the
Round 2 RST wording fix is genuinely correct and no other `.rst` file in
`docsdev/source/` repeats the stale "Interaction struct" claim; a fresh
Sphinx rebuild is clean and the rendered HTML's only "Interaction"
mentions are legitimate links to the real Contact Interactions feature;
`commonaddress.Address`'s actual Go struct (read directly from
`bin-common-handler/models/address/main.go`) matches both RST files'
claimed field list exactly; all 4 touched services (`bin-timeline-manager`,
`bin-common-handler`, `bin-openapi-manager`, `bin-api-manager`) pass
build/test/lint fresh; §16–18 are internally consistent with no
contradictions and §18's disposition record matches the current RST text;
the migration's `down.sql` correctly reverses the `up.sql` (drops `local`
then `peer`, mirroring add order in reverse); the OpenAPI-generated
`CommonAddress` type used by `TimelineManagerPeerEvent.Peer`/`.Local` has
identical fields/json tags to the Go source `commonaddress.Address` — no
generated-vs-source drift.

- **MINOR (fixed below):** `contact_peer_events_test.go`'s and
  `service_agents_contact_peer_events_test.go`'s "filter by
  peer_type+peer_target" test cases asserted `gomock.Any()` for the
  constructed `commonaddress.Address` argument rather than asserting its
  actual value, leaving the two-query-param → `commonaddress.Address`
  construction in `contact_peer_events.go:78-81` /
  `service_agents_contact_peer_events.go` unverified by the test suite
  (only the HTTP status code was checked). Fixed: both test tables now
  carry an `expectContactID`/`expectPeerAddress` field per case and assert
  the exact `&commonaddress.Address{Type: "tel", Target: "+155****1111"}`
  value on the mock expectation, for both the `contact_id` case
  (`expectPeerAddress: nil`, `expectContactID: <uuid>`) and the
  `peer_type`+`peer_target` case (`expectContactID: uuid.Nil`,
  `expectPeerAddress: &commonaddress.Address{...}`). Verified passing with
  `go test ./server/... -run PeerEvents -v` and the full `bin-api-manager`
  verification workflow (`go mod tidy && go mod vendor && go test ./...
  && golangci-lint run`), all clean.

Round 4 is next: since Round 3 was a clean APPROVE, Round 4 needs to also
be a clean APPROVE to reach 2 consecutive APPROVEs and close the loop. If
Round 4 finds anything, the consecutive-APPROVE counter resets.

## 20. Round 4 PR review disposition — APPROVE #1 of 2 consecutive

Round 4 adversarial PR review (independent subagent, verifying commit
50f609ffc's fix of Round 3's MINOR): **APPROVE**, 0 BLOCKER, 0 MAJOR, 0
MINOR. This is genuinely clean — Round 3's own APPROVE doesn't count
toward the consecutive-pair requirement because it carried a MINOR that
needed fixing; Round 4 is the first fully clean round and becomes
**APPROVE #1 of the required 2 consecutive**.

Independently re-verified: both test cases' `expectContactID`/
`expectPeerAddress` values are correct, including an independent
`urllib.parse.unquote('%2B155****1111')` check confirming the URL-decoded
target literal (`+155****1111`) used in the test matches what gin's
query-param decoding actually produces; `gofmt -l`/`goimports -l` clean;
a fresh `go clean -testcache && go test ./server/... -run PeerEvents -v`
run shows all subtests genuinely PASS (gomock's strict argument matching
means a wrong Address construction would produce a real test failure, not
just a compile pass); the full `bin-api-manager` verification workflow
passes clean (one transient `pkg/zmqsubhandler` port-race failure was
investigated and confirmed pre-existing test-environment flakiness
unrelated to this PR, reproduced to pass on retry and in isolation); §19
accurately describes commit 50f609ffc; document headers `## 1`–`## 19`
sequential with no gaps/duplicates; confirmed `Test_GetContactPeerEvents`
and `Test_GetServiceAgentsContactPeerEvents` live in the single
`contact_peer_events_test.go` file (no missed sibling test file).

Round 5 is next: needs to also be a clean APPROVE to reach 2 consecutive
and close the loop. If Round 5 finds anything, the counter resets to 0.
