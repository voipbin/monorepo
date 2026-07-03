# service_agents/transcripts API design (SQUARE-7 backend dependency)

## Problem

SQUARE-7 (square-talk: "Contacts → Interaction click auto-shows the call's
transcribe") needs to show the *actual conversation text* of a transcribe, not
just its metadata. `service_agents/transcribes` (added in PR #1049) already
returns `TranscribeManagerTranscribe` — status/language/provider/timestamps —
but carries no message content. The individual spoken lines live in a
separate resource, `TranscribeManagerTranscript` (plural is misleading; one
`Transcript` row = one utterance), queried via `GET /transcripts?transcribe_id=`.

That endpoint exists only as a top-level, Admin/Manager-gated path
(`bin-api-manager/pkg/servicehandler/transcript.go`'s `TranscriptList`, gated
`PermissionCustomerAdmin|PermissionCustomerManager`). There is no
`service_agents/transcripts` twin. Per this repo's established rule (see
`bin-openapi-manager/CLAUDE.md`'s "Adding a Service Agent resource" section,
and the `voipbin-square-talk-feature` skill's "missing capability = missing
endpoint first" section), the fix is to add the missing Agent-facing
endpoint — never to relax the top-level endpoint's permission.

## Shape of the gap

This mirrors the exact precedent that shipped `service_agents/transcribes`
(PR #1049) one JIRA cycle ago: the Agent-facing need is the resource's core
READ shape (list all transcript lines for one transcribe), not a narrow
write verb. Per the skill's rule of thumb, this is a "stand up the parallel
`service_agents/<resource>`" case, not a "relax two write verbs" case.

## Design

### 1. OpenAPI (`bin-openapi-manager`)

New file `openapi/paths/service_agents/transcripts.yaml`:

```yaml
get:
  summary: List transcript lines
  description: >-
    Retrieves the individual spoken/written lines (transcript entries) for
    one transcribe session, scoped to the service agent's own customer.
    Ownership is authorized against the target transcribe's own customer_id
    (fetched by transcribe_id), not a re-derived parent resource — see
    servicehandler implementation notes.
  tags:
    - Service Agent
  parameters:
    - name: transcribe_id
      in: query
      required: true
      description: >-
        The transcribe session ID returned from `GET /service_agents/transcribes`
        or `POST /service_agents/transcribes`.
      schema:
        type: string
        format: uuid
      example: "550e8400-e29b-41d4-a716-446655440000"
    - $ref: '#/components/parameters/PageSize'
    - $ref: '#/components/parameters/PageToken'
  responses:
    '200':
      description: A list of transcript lines, in creation order.
      content:
        application/json:
          schema:
            allOf:
              - $ref: '#/components/schemas/CommonPagination'
              - type: object
                properties:
                  result:
                    type: array
                    items:
                      $ref: '#/components/schemas/TranscribeManagerTranscript'
    '400':
      $ref: '#/components/responses/BadRequest'
    '401':
      $ref: '#/components/responses/Unauthenticated'
    '403':
      $ref: '#/components/responses/PermissionDenied'
    '404':
      $ref: '#/components/responses/NotFound'
    '500':
      $ref: '#/components/responses/InternalError'
```

Register under `paths: /service_agents/transcripts:` in `openapi.yaml`.
Reuses `TranscribeManagerTranscript` schema (already defined, no changes
needed there).

### 2. servicehandler (`bin-api-manager/pkg/servicehandler/serviceagent_transcript.go`, new file)

```go
func (h *serviceHandler) ServiceAgentTranscriptList(
	ctx context.Context, a *auth.AuthIdentity, size uint64, token string, transcribeID uuid.UUID,
) ([]*tmtranscript.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":          "ServiceAgentTranscriptList",
		"customer_id":   a.CustomerID,
		"username":      a.DisplayName(),
		"transcribe_id": transcribeID,
		"size":          size,
		"token":         token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// Fetch the target transcribe first — its own CustomerID field is the
	// authoritative tenant boundary. No second RPC to a parent resource
	// (call/conference/recording) is needed: this is a pure Get/Read path,
	// not a Create/Start path validating a caller-supplied reference before
	// the row exists. Mirrors TranscriptList's own existing shape (fetch,
	// then hasPermission on the fetched struct's field) — only the
	// permission bitmask changes (PermissionAll instead of Admin|Manager).
	t, err := h.transcribeGet(ctx, transcribeID)
	if err != nil {
		log.Infof("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, t.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	filters := map[string]string{
		"transcribe_id": transcribeID.String(),
		"deleted":       "false",
	}
	typedFilters, err := h.convertTranscriptFilters(filters)
	if err != nil {
		return nil, err
	}

	tmps, err := h.reqHandler.TranscribeV1TranscriptList(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get transcripts. err: %v", err)
		return nil, err
	}

	res := []*tmtranscript.WebhookMessage{}
	for _, tmp := range tmps {
		res = append(res, tmp.ConvertWebhookMessage())
	}
	return res, nil
}
```

Reuses the existing private `convertTranscriptFilters` helper (already
defined in `transcript.go`, no duplication needed — same file/package).

**Pagination fix (added after round-1 review — IMPORTANT fix):** an earlier
draft of this design hardcoded `token=""`/`size=100` inside the
servicehandler function, silently ignoring the OpenAPI spec's advertised
`page_size`/`page_token` query parameters — the exact same "advertises
pagination, doesn't implement it" defect the reviewer found already present
in the top-level `TranscriptList`/`GetTranscripts` pair. Rather than
propagate that defect into a brand-new endpoint, `size`/`token` are now real
parameters threaded from the HTTP layer through to
`TranscribeV1TranscriptList`, matching `ServiceAgentTranscribeList`'s own
existing pagination handling exactly (same `token == "" → TimeGetCurTime()`
default-token pattern). A call with more than one page of utterances now
gets a genuinely usable `next_page_token` in the response instead of a
silent truncation at 100 rows — see server handler section below for the
param wiring. (Fixing the OLDER top-level `TranscriptList`'s identical defect
is explicitly out of scope for this PR — flagged as a separate, pre-existing
issue not introduced here.)

### 3. Interface + mock

Add `ServiceAgentTranscriptList(ctx, a, size, token, transcribeID) ([]*tmtranscript.WebhookMessage, error)`
to the `ServiceHandler` interface in `pkg/servicehandler/main.go` (signature
updated to match the pagination fix above), regenerate mock via `mockgen
-package servicehandler -destination ./mock_main.go -source main.go
-build_flags=-mod=mod`.

### 4. server handler (`bin-api-manager/server/service_agents_transcripts.go`, new file)

```go
func (h *server) GetServiceAgentsTranscripts(c *gin.Context, params openapi_server.GetServiceAgentsTranscriptsParams) {
	a, ok := getAuthIdentity(c)
	if !ok {
		abortWithError(c, cerrors.Unauthenticated(...))
		return
	}

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	transcribeID := uuid.FromStringOrNil(params.TranscribeId)

	tmps, err := h.serviceHandler.ServiceAgentTranscriptList(c.Request.Context(), a, pageSize, pageToken, transcribeID)
	if err != nil {
		abortWithServiceError(c, err)
		return
	}

	nextToken := ""
	if len(tmps) > 0 && tmps[len(tmps)-1].TMCreate != nil {
		nextToken = tmps[len(tmps)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}
```

Mirrors `server/transcribes.go`'s `GetTranscribes` param-parsing shape (not
`server/transcripts.go`'s `GetTranscripts`, which is the one with the
hardcoded-pagination defect flagged in round-1 review) — this file now
threads `page_size`/`page_token` all the way through, matching the
OpenAPI spec's advertised contract.

### 5. Edge cases (added after round-1 review)

- **Missing/malformed `transcribe_id`**: OpenAPI marks `transcribe_id` as
  `required: true`, so oapi-codegen's generated `ServerInterfaceWrapper`
  rejects a request with the parameter entirely absent with an automatic 400
  before this handler ever runs (confirmed against `GetTranscribes`' actual
  generated wrapper code in round-1 review). A syntactically-present but
  non-UUID string (e.g. `transcribe_id=not-a-uuid`) is NOT caught by that
  wrapper — it reaches `uuid.FromStringOrNil`, which silently converts it to
  `uuid.Nil`, exactly the same as the pre-existing top-level `GetTranscripts`
  handler's own behavior. This is an intentional match to the existing
  pattern, not a new gap introduced by this endpoint: `uuid.Nil` flows into
  `transcribeGet(ctx, uuid.Nil)`, which the RPC chain resolves to a
  not-found response (404), so the net behavior for a malformed ID is
  "404 Not Found" rather than "400 Bad Request." Documented here explicitly
  as the accepted behavior — improving this (returning 400 for a
  syntactically invalid UUID) would be a good small follow-up but is a
  pre-existing platform-wide pattern, not something to fix ad hoc inside
  this one new endpoint.
- **Pagination truncation signal**: with the fix above, `next_page_token`
  in the response is now a real, usable value (not always empty) — a client
  reading an empty `next_page_token` after a full page of exactly `pageSize`
  results should treat that as potentially more data available and can
  re-request with the token; an empty token AND fewer than `pageSize`
  results returned means "no more data." (Same convention `ServiceAgent
  TranscribeList` already establishes — no new convention introduced.)

### 6. Tests

- `serviceagent_transcript_test.go`: assert a plain-Agent-permission identity
  succeeds (the whole point), assert cross-customer denial (agent's
  CustomerID differs from the fetched transcribe's CustomerID → PermissionDenied),
  assert `transcribeGet` error (e.g. not-found transcribe_id) propagates,
  assert a malformed/`uuid.Nil` `transcribe_id` resolves to the not-found
  path per the edge-case section above (not a panic or a 400).
- `service_agents_transcripts_test.go`: HTTP-level test hitting the new path,
  mirroring `transcribes_test.go`'s existing style, including a case
  asserting `page_size`/`page_token` are actually forwarded (not silently
  dropped — this is the regression test for the round-1 pagination fix).

### 7. Verification

Standard workflow: `bin-openapi-manager` (`go generate ./...`) first, then
`bin-api-manager` (`go mod tidy && go mod vendor && go generate ./... && go
test ./... && golangci-lint run -v --timeout 5m`).

## Out of scope

- No changes to the top-level `/transcripts` endpoint (stays Admin/Manager;
  its own pre-existing hardcoded-pagination defect, found during this
  design's review, is a separate follow-up issue, not fixed here).
- No write/delete on transcript lines (read-only resource by design —
  transcript lines are produced by the STT pipeline, never edited/created
  via this API).
- Returning 400 (instead of the current 404) for a syntactically invalid
  (non-UUID) `transcribe_id` string — matches the pre-existing top-level
  endpoint's behavior; a platform-wide fix, not scoped to this one PR.
