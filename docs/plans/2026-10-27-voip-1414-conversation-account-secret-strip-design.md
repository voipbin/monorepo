# VOIP-1414: strip account secret/token from conversation-manager events and logs

## 1. Ticket review

Valid. On main, `accounthandler/db.go` passes the raw `*account.Account` to `PublishWebhookEvent` (Create, Update, Delete). `notifyhandler.PublishEvent` does `json.Marshal(data)`, so the internal event carries `secret` and `token`. The customer webhook path (`PublishWebhook` -> `CreateWebhookEvent`) is already stripped. Several logs print the account object, the request body or the hook URI (which holds `hub.verify_token`, compared with the account secret in `whatsapphandler/verify.go`). No service consumes account events functionally (`grep -rn subscribeTargets bin-*/pkg/subscribehandler` and the `EventTypeAccount*` constants show no other service subscribing to conversation-manager account events; bin-ai-manager subscribes to message events only). bin-timeline-manager subscribes to `#` and stores raw JSON in its events table, so earlier plaintext may remain there; that cleanup is out of scope. No earlier change fixes it.

## 2. Goal

R1: internal account events carry no secret/token. R2: service logs and error strings carry no account object, `ac.Secret`/`ac.Token`, account request body `m.Data`, or hook URI query. Scope is `bin-conversation-manager` only; `bin-common-handler` is unchanged.

Out of scope: WhatsApp `provider_data.app_secret` (still exposed by design), events already stored in timeline-manager, `conversation-control` stdout, conversation/message PII logs (hook raw `data`, LINE event payloads).

## 3. Design

R1. `Account.CreateWebhookEvent` stays (the type still satisfies `WebhookMessage`). Add `func (h *WebhookMessage) CreateWebhookEvent() ([]byte, error)` (pointer receiver, `json.Marshal(h)` only, no `ConvertWebhookMessage` call). `*WebhookMessage` then satisfies `WebhookEventMessage`: `EventSubscriptionID` is promoted from the embedded `Identity`, so the routing key keeps the account id. The three call sites pass `res.ConvertWebhookMessage()`. The customer webhook JSON is byte-identical to before (same struct, same marshal).

R2. Logs keep only ids, type, count, method, and the URI path.
- `listenhandler/main.go` and every `v1_*.go`: replace field `"request": m` with `"uri": m.URI, "method": m.Method`; drop `m.Data` from all messages (`data: %v/%s`); marshal-failure logs print `err` only.
- `v1_hooks.go`: log `req.ReceivedMethod` and the parsed path only; the `url.Parse` error is not logged (`*url.Error` embeds the URL).
- `conversationhandler/hook.go`: `Hook` and `HookVerify` log the URI path; `url.Parse` errors return a fixed message without the URI; the "could not parse account_id" error uses the path.
- `linehandler/hook.go`: `"customer_id": ac` becomes `"account_id": ac.ID` (4 places).
- `accounthandler/db.go`: Get logs `account_id`, List logs `count`, Update logs the field names only.

Line table (all changes drop the printed object, never add one):
- `accounthandler/db.go`: Get (`account` field), List (`accounts` field, and the `filters` field, which can carry secret/token filter values: only the filter key names are logged), Update (`fields` map -> key names only).
- Exact counts to remove (15 `request` fields and 10 `m.Data` outputs): `request`: main.go:176, v1_accounts.go 26/82/153/233, v1_conversations.go 25/93/173/241/284, v1_hooks.go 19/62, v1_messages.go 22/57/109 plus the second `WithField("request", m)` at 59; `m.Data`: main.go 298/315, v1_hooks.go 67, v1_accounts.go 87, v1_conversations.go 98/246/289/336, v1_messages.go 27/114. Line numbers are as of the base commit; the PR body lists the final grep result.
- `listenhandler/main.go`: processRequest `request` field; not-found and error `Errorf` `data: %s`.
- `v1_accounts.go`: `request` in 4 handlers (GET list, POST, PUT, DELETE); `m.Data` in the POST unmarshal log; marshal-failure logs at v1_accounts.go 64 (`tmps`), 99, 135, 215, 252 (`tmp`, a raw `*Account`) print `err` only; the same `message: %v` logs in v1_conversations.go (75, 119, 155, 216, 258, 301, 348) and v1_messages.go (39, 139) are conversation/message objects, left as is.
- `v1_hooks.go`: `request` (GET, POST), `m.Data` in the POST unmarshal log, `request: req` log, `url.Parse` error. The `HookVerify failed` (GET) and `Could not hook the message correctly` (POST) logs print the error returned by conversationhandler, which is why the returned errors are fixed messages.
- `v1_conversations.go`, `v1_messages.go`: `request` fields, `m.Data` in unmarshal-failure logs.
- `conversationhandler/hook.go`: `uri` log fields (Hook, HookVerify) become the path after a successful `url.Parse`; `url.Parse` failure returns a fixed error; the account_id error prints `u.Path`.
- `linehandler/hook.go`: 4 `customer_id: ac` fields.
- Left as is: raw hook `data` fields (`conversationhandler/hook.go`, `linehandler/hook.go`): customer message payloads, not account credentials (LINE events and WhatsApp payloads carry no account secret).

D1: update the Note in `docs/architecture.md` (account events no longer carry secret/token).

## 4. Complete list (command in PR body)

Commands (run in `bin-conversation-manager`; the PR body carries them with their output before and after the change). They are an aid; the log tests in section 5 are the primary evidence.
```
F='--include=*.go pkg cmd'; X='_test.go|mock_'
grep -rnE '"request": *(m|req)\b|"customer_id": *ac\b' $F | grep -vE "$X"
grep -rnE '(Debug|Info|Warn|Error)f?\(.*(m\.Data|ReceivedURI|, (tmps?|ac|res|req)[,)])' $F | grep -vE "$X"
grep -rnE '"(account|accounts|filters|fields|uri)": ' $F | grep -vE "$X"
grep -rnE 'WithField\("(request|account|accounts|filters|fields|uri)",' $F | grep -vE "$X"
```
Intentionally remaining after the change: `dbhandler/conversation.go` `filters` (conversation filters, no account data), `listenhandler/main.go` `"uri": m.URI` (internal RPC path), the `message: %v, tmp` marshal-failure logs of v1_conversations.go and v1_messages.go (conversation/message objects), and the raw hook `data` fields.

The sweep over `pkg` and `cmd` (tests and mocks excluded) for log/Errorf/panic output of `ac`, `res`, `tmp`, `tmps`, `m`, `m.Data`, `req`, `uri`, `ReceivedURI`, `ReceivedData`, `fields`, `filters` is covered by the lines in section 3; after the change the command above must print nothing except the intentionally kept ones (raw hook `data`, listed in section 3). Reviewed and left unchanged: `whatsapphandler` (errors carry no credential; token is only in the Authorization header), `linehandler/setup.go`, `teardown.go` (SDK errors), `dbhandler` (errors wrap driver messages without values).

## 5. Verify

- `account.WebhookMessage.CreateWebhookEvent`: JSON has no `secret`/`token` key, `EventSubscriptionID` equals the account id, and the test fails before the change (method missing: compile red).
- `db_test.go`: `PublishWebhookEvent` expectations use `tt.responseAccount.ConvertWebhookMessage()` (concrete value, not `gomock.Any()`).
- Behavior RED for R1: db_test asserts the `PublishWebhookEvent` argument is `*account.WebhookMessage` (concrete expected value) in Create, Update and Delete, and that its JSON has no sentinel secret/token; this fails on the unmodified code because the argument is `*Account`.
- Log tests use the logrus `test.NewGlobal` hook with `logrus.SetLevel(DebugLevel)` (restored afterwards, no `t.Parallel`, the logger is global). A plain substring check on `entry.Message` would pass falsely, because `sock.Request.Data` (`json.RawMessage`) and `[]byte` fields print as decimal lists in text format. So a shared test helper `internal/logtest` renders every entry with the JSON formatter (`[]byte` becomes base64), the text formatter and `fmt.Sprint` of each field, and fails if any rendering contains the sentinel in plain, base64 or decimal-list form. It also fails when a field value is of type `*sock.Request`, `*account.Account`, `[]*account.Account` or `request.V1DataHooksPost`.
- Cases (the sentinel is a fixed alphanumeric string such as `SENTINELabc123`, so no URL or JSON escaping changes it, and the percent-encoded form is also checked; the helper restores the logger level, formatter and hooks with `t.Cleanup`; the sentinel is placed in the account, in `m.Data` or in the URI query): accounthandler Get, List (also a sentinel in the filters), Update (and Delete, whose red comes from the `Get` log it calls); listenhandler `processRequest` error path and not-found path, accounts POST unmarshal failure and the marshal-failure paths, accounts PUT; hooks GET with an unparsable `received_uri` such as `http://h/%zz?hub.verify_token=SENTINELabc123` (400 and no leak); messages GET/POST (the second `request` log at v1_messages.go:59); hooks GET and POST with `hub.verify_token` in the URI; the error of an unparsable URI containing the sentinel: for `Hook` through the POST handler (the error is logged at `v1_hooks.go:77`), for `HookVerify` by calling `conversationhandler.HookVerify` directly, because the GET handler already answers 400 on a parse failure before reaching it; linehandler hook with a sentinel account. Each of these tests is run against the unmodified code (scratch copy) and must fail there.
- Marshal failure is induced with an invalid `json.RawMessage` `ProviderData` (e.g. `{bad`) on the account returned by the mocked handler: `json.Marshal` of the account fails and the unmodified code logs the whole `*Account`.
- PR body: the RED run output of the log tests on the unmodified code (scratch copy) is attached.
- Not a red target: the `Create` setup-failure log prints the error returned by the setup step, which the caller controls; it gets a regression guard test with a secret-free error only.

## 6. Impact

No schema change, no flag. Event consumers: no functional consumer found by grep; timeline-manager only stores the raw payload (it does not parse secret/token). Rollback: revert the commit.

Service docs sync: only `docs/architecture.md` is affected (the routing table is unchanged because only log lines change).
