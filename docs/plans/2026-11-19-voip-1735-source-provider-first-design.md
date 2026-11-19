# VOIP-1735: send SMS and MMS through the provider that owns the source number first, and add Twilio sending

Services: `bin-message-manager` (main), `bin-hook-manager` (forwards the Twilio signature header), docs: the `docs/` and `CLAUDE.md` of both services, `bin-api-manager/docsdev/source/self_hosting_providers.rst` and `number_provider_overview.rst` (clean rebuild, `git add -f build/`).

## 1. Ticket review (facts re-checked on origin/main e93ef44a4)

All facts F1 to F15 of the analysis hold. The code facts that the design depends on:

- The provider order is fixed: `messageHandler.smsProviders` (telnyx, messagebird, `main.go`), and `dispatchMMS` has two fixed stages (`mmsTelnyx`, then `mmsMessagebird` for the targets telnyx refused and that are +1 to +1).
- `resolveSource` already receives the number-manager row (`nmnumber.Number`, field `ProviderName`) and drops it. It has two callers: `Send` and `SendMMS`.
- Number-manager only creates telnyx and twilio owned normal numbers. Twilio sending does not exist in message-manager.
- Only telnyx receipts exist: hook-manager forwards `Host+Path` (no scheme, no query) and the body; message-manager `Hook` accepts only a `telnyx` suffix. The wire model `hmhook.Hook` already has `ReceivedMethod` and `ReceivedSignature` (filled today by the billing and conversation handlers), so no model or request struct change is needed; the Twilio signature must be checked against the configured callback URL, not against `ReceivedURI`.
- The provider name is hard coded in four more places: `send.go` and `send_mms.go` (record creation), `hook.go` (inbound telnyx message, stays telnyx), `providerName()` and the `accept` calls and metric labels inside `dispatchMMS` in `dispatch.go`.

Twilio documentation (checked for this change):

- Create message: `POST https://api.twilio.com/2010-04-01/Accounts/{AccountSid}/Messages.json`, HTTP Basic (Account SID, Auth Token), form body `To`, `From`, `Body`, repeated `MediaUrl` (up to 10), `StatusCallback`, `SendAsMms`. The answer holds `sid`, `num_segments` (string), `status`, `error_code`. Errors are 4xx with JSON `code` and `message`.
- Status callback: `POST` (form) to `StatusCallback` with `MessageSid`, `MessageStatus` (`queued`, `sending`, `sent`, `delivered`, `undelivered`, `failed`, ...), `ErrorCode`. No ordering guarantee. New parameters may appear.
- Signature: header `X-Twilio-Signature` = base64(HMAC-SHA1(AuthToken, exact callback URL + every form key and value sorted by key)).
- MMS reach: the United States and Canada are confirmed (help.twilio.com "Twilio Global MMS"); Australia is listed there for Australian numbers only and other pages say US and Canada only. The design relies on the US and Canada part.

## 2. Design (behavior)

Terms: owner = `provider_name` of the source number. Default order = (telnyx, messagebird).

1. The order of a message is computed once, in `Send` and `SendMMS`, from the owner:
   - owner is telnyx or messagebird: the owner first, then the rest of the default order.
   - owner is twilio and Twilio credentials are set: (twilio, telnyx, messagebird).
   - anything else (empty, twilio without credentials): the default order. Twilio is never added to a default order.
2. Every stage of the message uses that order: SMS chain, media MMS, promoted MMS and its SMS fallback. The order is a value of the message. The handler's provider registry is never sorted or changed.
3. The fallback rule is unchanged: only a definitive rejection (4xx except 408) goes to the next provider.
4. MMS eligibility per provider: telnyx all targets; messagebird and twilio only when the source and the destination both start with `+1`. Twilio documents MMS for the United States and Canada (and Australia for Australian numbers only); the rule uses the US and Canada part, the same rule as messagebird. Extending it to Australia is a later change.
5. The message record starts with the first provider of its order. At the end the provider that accepted the most targets wins; a tie goes to the provider that is earlier in the order of that message.
6. Twilio credentials: `TWILIO_ACCOUNT_SID` and `TWILIO_AUTH_TOKEN`, read from the environment at send time (envsecret rules: empty or placeholder is not set). Both are needed. The callback URL setting is `TWILIO_STATUS_CALLBACK_URL`: when it is not set, a placeholder or not an http(s) URL, the request is sent without `StatusCallback` and the sending goes on. A valid value is an http(s) URL with a host, no user information, no query string, and a path that ends with `/messages/twilio` (the hook route; any other path would be dropped as an unknown hook). `LogStates` warns when the value is set but not valid; accepted targets then stay in the status of the acceptance (the same limit as messagebird). No global on/off flag is added.
7. Twilio delivery receipts arrive at hook-manager `POST /v1.0/messages/twilio`. Hook-manager forwards the `X-Twilio-Signature` header for the `twilio` target only (the telnyx forwarding is unchanged) and holds no secret. Message-manager verifies the signature and refuses (fail-closed) when the signature is missing or wrong, or when the Auth Token or the callback URL is not set. Refusal answers 401, changes nothing and is logged and counted. It is never `ErrReceiptRetry`.
8. The signed URL is the configured `TWILIO_STATUS_CALLBACK_URL` exactly as it was given to Twilio in the send request, not a URL rebuilt from the request headers. A body that is not a status callback (no `MessageStatus`) is ignored after the signature check and never creates a message.
9. The Twilio Account SID is read only from the environment. A deployment that keeps the SID in the number provider row must put the same account value into `TWILIO_ACCOUNT_SID`; a different account is answered 401 by Twilio and the target falls back.
10. No table, admin API, admin screen or customer API contract changes. New settings: `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_STATUS_CALLBACK_URL`.

## 3. Design (technical)

### 3.1 message-manager, ordering (`pkg/messagehandler/order.go`, new)

- `defaultProviderOrder` is a package value; `providerOrder(ctx, owner nmnumber.ProviderName) []message.ProviderName` returns a new slice every call.
- `messageHandler.twilioAvailable func(ctx) bool` is set from `requestExternal.TwilioAvailable`; nil means false (existing test fixtures stay valid).
- `mmsEligible(name, source, target)` replaces the two stage conditions; `mmsFallbackEligible` stays the +1 to +1 test.
- Registries (needed by the order loop; the shared list must not be sorted; fixtures in `send_test.go`, `send_mms_test.go`, `message_test.go` change with it): `smsProviders map[ProviderName]SMSProvider` and `mmsProviders map[ProviderName]MMSProvider`, written once in `NewMessageHandler`, read only afterwards.

### 3.2 `resolveSource`

Returns `(*Address, nmnumber.ProviderName, error)`; the provider is `num.ProviderName` of the row it already read, in both paths. The lookup count does not change.

### 3.3 `Send` and `SendMMS`

After `resolveSource` compute `order := h.providerOrder(...)`; `create`/`createMMS` take `order[0]`; `dispatchSMS` and `dispatchMMS` take `order`.

### 3.4 `dispatchSession`

Holds `order`; `providerName()` starts with `order[0]` and replaces only on a strictly larger accepted count.

### 3.5 `runSMSChain` and `dispatchMMS`

`runSMSChain` walks `order`. `dispatchMMS`: the promoted pre-split (ineligible targets go to the SMS chain) is unchanged. The MMS stage is a loop over `order`:

```
remaining = mms targets
for name in order:
    stage = remaining targets eligible for name   (the others get skippedIneligible[i] = true and stay)
    if stage is empty: continue                    (no provider call with an empty stage)
    results = mmsProviders[name].SendMMS(stage)
    accepted -> accept, leave remaining
    rejected -> stay remaining, lastTried[i] = name   (set only for targets that were sent to name)
    failed   -> fail(i), metric (name, "error"), REMOVED from remaining (a promoted target does not go to the SMS chain)
after the loop, for each remaining target:
    promoted -> SMS chain (metric promoted_fallback reason "rejected")
    media    -> fail(i), metric (lastTried[i], or order[0] if no provider was tried, "ineligible" if skippedIneligible[i] else "rejected")
```

For a telnyx owned number the sequence of calls and the metric pairs are the ones of today: (telnyx, ineligible) for a refused non-+1 target, (telnyx, error), (messagebird, rejected), (messagebird, error). `promoted_mms_total` gets the provider name as the label as before.

### 3.6 Twilio request layer (`pkg/requestexternal/twilio.go`, `models/twilio`)

- `TwilioSendSMS(ctx, source, destination, text)` and `TwilioSendMMS(ctx, source, destination, text, mediaURLs)`; MMS without media uses `SendAsMms=true`. Both build one form request; `StatusCallback` is added when the setting is valid.
- Credentials come from a supplier `func(ctx) (sid, token string, err error)`. Missing sid or token returns an error wrapping `ErrTokenUnavailable` (not sent, outcome Failed like the telnyx and messagebird token errors). It can only happen if the environment changed after the order was computed, which a running process does not see, so it has no extra handling. The SID is path escaped.
- Non 2xx is a `ProviderError{Provider: "twilio"}` with the error code and message detail (digit runs masked, 200 characters), so `IsDefinitiveRejection` is the same F12 rule. Transport errors and unreadable 2xx answers are unconfirmed (Failed). A transport error is returned without the request URL (the `*url.Error` is unwrapped to its cause, so `errors.Is` on a context error still works), because the path of the URL holds the Account SID.
- `NewRequestExternal(messagebird, telnyx TokenFunc, twilio TwilioConfig)`: `TwilioConfig{Credentials func(ctx) (sid, token string, err error); CallbackURL func(ctx) string}`; the callers are `cmd/message-manager`, `cmd/message-control` and the tests, and `mock_main.go` of `requestexternal` is regenerated.
- Signature algorithm (from the Twilio webhook security page): the string to sign is the configured URL as is (no port or slash correction), then for every form key in byte order (`sort.Strings`) the key and its values (values of one key sorted), keys with an empty value included, with percent decoded values (`url.ParseQuery(string(body))`); HMAC-SHA1 with the Auth Token, base64, compared with `hmac.Equal`. Signing with a secondary token during a token rotation is out of scope.
- A zero `TwilioConfig{}` (nil functions, as in the existing tests) means "not configured" and never panics. The signature check is implemented here because the repository has no Twilio SDK dependency and one function of about 20 lines does not justify one.
- `TwilioAvailable(ctx)` is true only when sid and token are both set.
- One function validates the callback URL setting (an explicit default port such as `:443` is refused as not valid, because the signed string could differ from what Twilio signs); the send request and the validation both use it, so a value that is set but not valid counts as not set in both. `TwilioValidateCallback(ctx, signature, form url.Values)` returns a sentinel: `ErrTwilioNotConfigured` (token or URL not set), `ErrTwilioSignatureMissing`, `ErrTwilioSignatureInvalid`; compare with `hmac.Equal`.
- The API base URL is a package variable so tests can point it to `httptest`.
- Provider wrappers `smsTwilio` and `mmsTwilio` (`provider_twilio.go`): accepted result carries `sid` and `num_segments` (when greater than 0); metric `twilio_number_send_total{type}`.

### 3.7 Hook path

- `MessageHandler.Hook(ctx, uri, data, signature)` (the mock and about 25 test calls in `hook_*_test.go`, `inbound_media_test.go`, `error_paths_test.go` and `listenhandler/hooks_log_test.go` change with it); the `twilio` suffix calls `hookTwilio`.
- `hookTwilio`: first caps the body at 64 KiB, then parses the form only to compute the signature (a form that cannot be parsed is refused like a bad signature, reason `malformed`, errors carry no body; no value of the body is used before the check), then verifies the signature (a missing Auth Token or callback URL, a missing signature and a wrong one are all refused), then ignore a body without `MessageStatus`, validate the `MessageSid` shape (`SM` or `MM` and 32 hex), look it up with `MessageListByProviderReference`, map the status, apply with `updateTargets` and `statusRank` (never lowers). Unknown sid: `receipt_unmatched_total`. DB trouble: `ErrReceiptRetry` (as telnyx). A refused callback returns `ErrHookUnauthorized`; counter `twilio_callback_rejected_total{reason}`.
- Status map: `sent` to sent; `delivered` to delivered; `failed`, `undelivered`, `canceled` to failed (final states); `queued`, `accepted`, `scheduled`, `sending`, `receiving`, `received`, `read`, `partially_delivered` and unknown values change nothing. `num_segments` is used only when greater than 0 because Twilio can answer `"0"` right after queueing.
- `listenhandler`: `ErrReceiptRetry` stays 500; `ErrHookUnauthorized` is 401; everything else 200 as today.
- hook-manager `Message`: for a `/twilio` path set `ReceivedSignature` from the header; a message-manager 401 (`requesthandler.ErrUnauthorized`) becomes a `ValidationError`, and `messagesPOST` answers `ValidationError` with 400 (the 500 for every other error stays), so a refused callback is answered as a client error and not as a server fault. The `ValidationError` is used here for "the caller sent something that is refused"; a code comment says so. For the `/twilio` path only, the body is read through `io.LimitReader` at 64 KiB (over the limit is a `ValidationError`, 400), as `billing.go` does; the telnyx path keeps the unbounded read it has today. Telnyx requests are byte-identical to today. Tests: `servicehandler` (header forwarded for twilio only, telnyx unchanged, 401 mapping) and `messages_test.go` (400).
- Known limit, the same as telnyx: a callback that arrives before the accepted id is stored (the callback of the dispatch writes it right after the request returns) is counted as unmatched and not retried.

### 3.8 Configuration and deployment

- `configkeys`: three names. `messageconfig`: `TwilioCredentials()`, `TwilioStatusCallbackURL()`; `LogStates` logs the three names with their state only (unset is an info line).
- `komodo/docker-compose.yml`: the three variables with `[[BIN_MANAGER__...]]` placeholders (an unresolved placeholder means not configured). `k8s/deployment.yml`: three optional secret references.
- `cmd/message-manager` and `cmd/message-control` pass the new suppliers to `NewRequestExternal`.

### 3.9 Verification plan

All tests use mocks, `httptest` and fake values; nothing reaches a provider. `go test -race` is run for `pkg/messagehandler`.

| Test | Scenario | Expectation |
|---|---|---|
| T1 | messagebird owned number, SMS, 2 targets, messagebird rejects one | messagebird first with both; only the rejected one goes to telnyx |
| T2 | telnyx owned number: SMS, media MMS (+1 and non +1 targets), promoted MMS | the call order, failure conditions, final provider name and the four metric pairs (telnyx ineligible, telnyx error, messagebird rejected, messagebird error) are the ones of the existing telnyx owned scenarios of `dispatch_test.go` and `send_mms_test.go`, which pass unchanged except for the fixture and signature changes, with the four metric pairs asserted explicitly; twilio and any unexpected provider are called 0 times |
| T3 | owner empty | default order |
| T4 | messagebird owned, media MMS, one +1 and one non +1 target | +1 target: messagebird rejected then telnyx; non +1 target: telnyx only (messagebird 0 calls); a target that telnyx also rejects is failed and no SMS provider is called; empty stage means 0 calls |
| T5 | messagebird owned, promoted MMS rejected everywhere; a non +1 target | the SMS chain runs in order (messagebird, telnyx); the non +1 target skips MMS |
| T6 | Failed result in each position of each order, media and promoted | no next provider is called for that target |
| T7 | provider_name at create and at the end | create: first of the order; end: most accepted, tie goes to the earlier provider of the order, with 0 accepted for a messagebird owned and a twilio owned number |
| T8 | source omitted (default number) and source given | the owner of the row is used in both paths; `NumberV1NumberList` calls per message equal the old count |
| T9 | twilio owned, credentials set | SMS, media MMS and promoted MMS use (twilio, telnyx, messagebird); only Rejected moves on; a non +1 target skips twilio |
| T10 | twilio owned and (no sid, no token, neither, placeholder) | 0 twilio requests, default order, create-time provider_name telnyx |
| T11 | twilio request layer with `httptest` | 2xx: accepted, sid and segments recorded (`"0"` ignored); 4xx except 408: rejected; 408, 5xx, transport error, unreadable 2xx: unconfirmed; the form holds To, From, Body, MediaUrl, StatusCallback only when valid, `SendAsMms` only for text-only MMS; Basic auth; error strings and logs hold neither the Account SID nor the URL nor the token; missing credentials are not sent (token error, Failed) |
| T12 | twilio callback | the signature algorithm is fixed with the independent public vector of the Twilio helper library tests (token `12345`, a documented URL and parameter set; the source is named in the test comment) plus variants (multi value key, empty value, percent encoding, URL scheme, slash and port differences are refused); valid: status applied and never lowered; missing or wrong signature, token not set, URL not set, malformed form: refused (`ErrHookUnauthorized`, nothing written, counter); body without `MessageStatus`: ignored; unknown sid: unmatched counter; DB error: `ErrReceiptRetry`; the status map table; the sid shape |
| C1 | callback URL validation table | unset, placeholder, http, https, user information, query, wrong path, no host |
| C2 | `komodo_compose_test.go`, `Test_LogStates`, k8s env test | the three names are present with the same format as number-manager; LogStates shows names and states only |
| L1 | `listenhandler` | `ErrHookUnauthorized` 401, `ErrReceiptRetry` 500, others 200 |
| H1 | hook-manager `servicehandler` and `messages_test.go` | the header is forwarded for `/twilio` only; the telnyx hook content is equal to the old one; a 401 of message-manager becomes `ValidationError`; the API answers 400 for it and 500 for the other errors; the twilio body limit |

A13 review: `git diff` has no change under `bin-openapi-manager` and no customer API path.

### 3.10 Impact analysis

- No schema change. Customer API request and response contract: unchanged.
- Rolling deploy: the old and the new replica both send; an old replica never sends through Twilio, which is the same as today. A Twilio callback that reaches an old replica is answered 200 and dropped, so the callbacks of the messages that a new replica sent can be lost during the rollout window. An old hook-manager that does not forward the signature makes the new message-manager refuse (fail-closed) and answers 500 to Twilio for it; nothing changes in the data, and no callback exists before the callback URL setting is put into the deployment. The order of the two auto deployments is not enforced and either order is safe.
- With no Twilio credentials every number behaves as before, except that a messagebird owned number now tries messagebird first (the requested change).

- Operations notes: the signed URL is the setting at the time of receipt, so changing `TWILIO_STATUS_CALLBACK_URL` makes the callbacks of messages that were sent with the old value be refused (401); their targets keep the status of the acceptance. A Twilio opt-out style rejection and a 429 rate limit answer (4xx) count as a definitive rejection and falls back to the next provider, which is the existing fallback rule that the decision keeps. Whether another provider accepts a message from a number that it does not own was not verified (U1); the completion summary says so.

### 3.11 Out of scope

Inbound Twilio messages, messagebird receipts, telnyx signature check, billing and segment rules, install repository variables.
