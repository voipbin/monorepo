# bin-number-manager unit test coverage to 90% (VOIP-1663)

## 1. Analysis

Baseline at origin/main 0bfbe2a1d, measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`. Hand-written coverage excludes `gens/`, `mock_*.go` and `Code generated ... DO NOT EDIT` files; duplicate blocks are merged by max count. Hand-written total is 89.48% (2672 of 2986 statements). 90.0% needs 2688 covered (+16), 90.5% needs 2703 (+31). The reachable upper bound is well above 90.5%, so no block is needed.

## 2. Design

Test files only, no product code change, no new module (`miniredis` is already in `go.mod`).

- cmd/number-control: `initHandler` and the six `run*` functions with a closed loopback database port (the connection is refused at once, no sleep), `initCache` against a miniredis server and against a closed port, `main` with `--help`, `migrate status` against the closed port. Cobra runs pass `--database_dsn` explicitly (synthetic account, loopback closed port), so the `sync.Once` config load cannot pick up a default address.
- cmd/number-manager: `initCache` (miniredis and closed port), `initProm` (endpoint built from a package-level atomic counter so `-count` and `-shuffle` never register the same pattern twice, `127.0.0.1:0`, and an invalid listen address checked through a logrus hook), `runDaemon` up to a failed database connection, `main` with `--help`.
- pkg/listenhandler: `NewListenHandler`, a malformed request URI, the `message_flow_id` field of the flow ids update request, response marshal failures reached through a `number.Number` whose time is outside the JSON range, `jsonResponse` with an unencodable value, `errorResponse` with unencodable details.
- pkg/providerhandler: error propagation of the admin functions after a gomock db error, `ensureRow` for an unknown provider, `ObservePurchase`.
- pkg/secretcrypto: empty list entries, `Encrypt` and `Decrypt` with an invalid key length (same-package struct).
- models: `String` and `GoString` of `Provider` and `Credentials` must not contain the secret or ciphertext; MMS features of the Telnyx and Twilio converters; `CreateWebhookEvent` marshal failure.
- pkg/dbhandler, pkg/cachehandler, pkg/requestexternal, pkg/subscribehandler: duplicate-key detection, malformed stored rows, unencodable update values, a request that cannot be built or whose body cannot be read, the logged handler error.

Expected values are literals, never derived from the code under test; the redaction tests assert that the secret and the ciphertext are absent from the output. The cmd tests alone were measured (overlay run) to leave about 250 uncovered statements against the 283 allowed for 90.5%, so the other packages are a safety margin.

Isolation: no sleeps, no real clock, no real signals, no external network. Config, viper, logrus, `os.Args`, stdout and signal registration are restored in cleanup, and `chDone` is drained. A metrics server started by `initProm` cannot be stopped; it listens on `127.0.0.1:0` and is left to the process end, and the listen-error log is awaited through a logrus hook channel with a timeout, filtered by the listen address of the test, and the hook and output are restored in cleanup. The `initSignal` goroutine is ended by pushing one `SIGTERM` into `chSigs`. `main` is only run with `--help`, which returns without `os.Exit`. Tests also run alone with `-run`. Test values are synthetic.

## 3. Verify

- Hand-written total at or above 90.5% with the same exclusion rule as the baseline; the PR body lists the command and flags.
- Root verification workflow passes. New and existing tests pass with `-race -count=3 -shuffle=on` in every package except `pkg/dbhandler`, whose shared in-memory database already depends on test order and on a single run on origin/main; that package runs with `-race -count=1` in the default order; `scripts/check-test-conventions.sh` passes after `git fetch origin main`.
- Mutation spot check (each must fail a named new test): remove the secret redaction in `Provider.String`, drop the empty-entry skip in `secretcrypto.New`, and remove the `message_flow_id` assignment in the flow ids update request handler (the test sends only `message_flow_id` and asserts the exact update field map passed to the mocked number handler).
- The PR file list stays inside `bin-number-manager/`.
