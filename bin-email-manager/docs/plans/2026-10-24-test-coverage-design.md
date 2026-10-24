# bin-email-manager unit test coverage to 61% (VOIP-1647)

## Scope and metric

- Add unit tests only. No non-test `.go` change, no `go.mod` or `go.sum` change.
- Metric: hand-written statements (excluding `gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` files; duplicate blocks merged by max count). Baseline 542/1123 = 48.3% (raw 44.0%). Target 61% or more (685+ statements, about +143).
- Measure: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in the service directory.

## Test rules

`Test_` prefixed names, gomock controller named `mc`, fixed UUID strings, synthetic values only (`example.com` addresses, fake keys). httptest handlers run in another goroutine, so they use `t.Errorf`, never `t.Fatalf`. Tests are table driven where cases share a shape.

## Plan

1. `pkg/emailhandler/engine_sendgrid_test.go`: build `engineSendgrid{reqHandler: mock, client: ...}` with `client.Request.BaseURL = server.URL + "/v3/mail/send"` (full path, loopback httptest). sendgrid uses the global `rest.DefaultClient`, so tests never replace it; the transport error case uses a closed loopback address. Cases: success with `X-Message-Id`, request body carries from, subject, content, recipients and the `voipbin_message_id` custom arg; recording attachment success (body has the `.zip` file name and base64 content); attachment lookup error, unknown reference type and download error (each: email still sent, body has no attachment); no `X-Message-Id`; closed server; non-2xx response without id.
2. `pkg/emailhandler/engine_mailgun_test.go`: `engineMailgun{reqHandler: mock, client: ...}` with `SetAPIBase(server.URL + "/v3")` and `SetClient` using a transport that rejects non-loopback hosts. Server replies `{"id":"<synthetic>","message":"Queued"}` and asserts the form fields (`from` with and without `TargetName`, `to`, `subject`, `text`, `h:X-Voipbin-Message-Id`). Cases: success, attachment success (multipart carries the file name), attachment lookup error, unknown reference type, download error (each fails the whole send), 1001 recipients (AddRecipient error), zero recipients, server error.
3. Residual error paths, written from the uncovered statements (not from existing test names): `emailhandler` `email.go` (about 7), `hook.go`, `send.go`, `util.go` (about 10); `dbhandler` (13); `listenhandler` (about 26, excluding `Run` which needs RabbitMQ). Existing cases such as `Test_List_DBError`, `Test_Hook_UnknownURI` are not duplicated. Expected contribution about +45.
4. `pkg/cachehandler`: struct literal handler with `redis.NewClient(&redis.Options{Addr: <closed loopback port from net.Listen then Close>, MaxRetries: -1, DialTimeout: short})` and a ctx timeout; `Connect`, `EmailGet`, `EmailSet` return errors; one `NewHandler` call. No miniredis.
5. Out of scope: `cmd/*` main wiring (needs DB and RabbitMQ), sendgrid `resp.Headers == nil` branch (unreachable with a real HTTP response).

Budget: engines about +180 statements (need +143), so steps 3 and 4 are margin. `Connect` pings with `context.Background()`, so it relies on the immediate refusal of the closed port (ctx timeout only affects `EmailGet` and `EmailSet`). An attachment download 500 from httptest also covers the non-200 branch of `download()` in `util.go`; step 3 does not repeat it.

## Verify (written before coding)

- V1 Hand total >= 61.0%; no touched package decreases; `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` pass; `git diff --exit-code go.mod go.sum`; `bash scripts/check-test-conventions.sh` pass.
- V2 No real host: every sendgrid test asserts the `BaseURL` host is `127.0.0.1`; mailgun uses the loopback-only transport; `download()` builds its own `http.Client`, so attachment URIs are httptest URLs only.
- V3 Each new test asserts an observable outcome (error text, returned id, request content, status code), not a value the test built itself.
- V4 Mutation sample in a scratch copy, each must fail a new test: sendgrid attachment `continue` turned into return (assert email sent and body has no attachment on failure; body has `.zip` and base64 on success); drop mailgun `TargetName` formatting; change the `X-Message-Id` lookup; change a listenhandler status code.
- V5 `git diff --stat origin/main...HEAD` lists only `bin-email-manager/` files.

## Defect candidates (reported only, not fixed)

- sendgrid ignores the response status code; a 4xx/5xx becomes a "could not get message id" error.
- Attachment failure is skipped in sendgrid but fails the whole send in mailgun.
