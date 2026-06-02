# Listenhandler Error Mapping

> Backend `bin-*-manager` services map errors to RPC `sock.Response` status codes
> in their `pkg/listenhandler`. This is the canonical convention. Reference
> implementation: `bin-flow-manager/pkg/listenhandler`.

## Rule

Every endpoint function maps a **handler-call error** at the call site with the
shared `errorResponse` helper:

```go
tmp, err := h.xHandler.Method(ctx, ...)
if err != nil {
    return errorResponse(err), nil
}
```

`errorResponse(err)` maps:
- `*cerrors.VoipbinError` → its true HTTP status (via `cerrors.ToResponse`)
- `dbhandler.ErrNotFound` (matched through `errors.Wrap`/`%w`) → `404`
- anything else → `500`

Keep these **local** (do not route through `errorResponse`):
- request URI/`json.Unmarshal` parse failures → `simpleResponse(400)`
- response `json.Marshal` failures → `simpleResponse(500)`

## Do NOT

- Do **not** swallow handler errors with `return simpleResponse(500), nil` — that
  discards the typed not-found the handler layer produces and yields 500 for a
  missing resource (bug class of #955, #953).
- Do **not** flip the central dispatch tail default from `400` to `500`: several
  endpoints return bare `(nil, err)` for parse-class failures that rely on the
  `400` default; flipping regresses them. Map at the site instead.
- Do **not** wire an unwired central tail as part of a de-swallow change if it
  would silently reclassify unrelated bare-`(nil, err)` endpoints (e.g.
  call-manager). Track tail-wiring as its own enumerated + tested change.

## Helper location

`errorResponse` and `simpleResponse` live in each service's
`pkg/listenhandler/main.go`. Helper-level behavior is covered by
`pkg/listenhandler/error_response_test.go`.
