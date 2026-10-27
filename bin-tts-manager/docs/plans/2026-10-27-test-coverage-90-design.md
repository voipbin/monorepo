# bin-tts-manager test coverage to 90% (VOIP-1663)

## Ticket review

Valid. On origin/main 0bfbe2a1d, `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` with generated
files, `gens/` and `mock_*.go` excluded and duplicate blocks merged by max count: hand-written 2486/2877 (86.41%),
raw 80.10%. 90.0% needs about 104 more statements, 90.5% about 118. Unit tests only, no product code changes.
cmd/tts-control and cmd/tts-manager are below 80% and overlap VOIP-1416; they are not absorbed here.

## Approach (all tests hermetic: loopback only, synthetic credentials, no fixed sleeps, no network)

Expected gain is in profile statements; the itemized reachable total is about 130, against 104 needed for 90.0% and 118 for 90.5%. Stable items
(GCP failure paths, ElevenLabs connect) are implemented first; the 4 second ticker items come last.

1. ElevenLabs `Init` and `connect` (about 67). Replace `websocket.DefaultDialer` with a dialer whose `NetDialContext`
   targets a local `httptest` TLS server and whose `RootCAs` holds that server's certificate and whose `ServerName` is a name in the certificate (no
   `InsecureSkipVerify`); restore the original pointer in `t.Cleanup`. Not parallel. The handler goroutine sends the
   request path, query and `xi-api-key` over a channel that the test goroutine reads with a bounded timed select, and asserts. A refused upgrade
   verifies the wrapped dial error; success verifies the config fields and the initiated event.
2. GCP failure paths (about 35, an estimate confirmed by measurement during implementation). Tests use `t.Setenv`, so they are not parallel. `Init` and `connect`: a missing
   credential file makes client creation fail ("could not create GCP TTS client"); synthetic authorized-user
   credentials with an already-cancelled context make stream creation fail ("could not start StreamingSynthesize").
   Both messages are confirmed in scratch before the assertions are fixed. `waitAndReconnectLocked`: a live session
   context plus a missing credential file gives the deterministic "failed to reconnect" error (`Stream` stays nil,
   the lock is released); a cancelled session context with an open `processDone` gives the cancellation error; an
   already-reconnected stream short-circuits. A cancelled context with a closed `processDone` is avoided because the
   select would be random. `closeStreamLocked` with a client built offline (no auth, loopback endpoint).
3. GCP `runKeepalive` (about 13). Reuse `fakeGCPStream`; no environment variables, so the two cases run in parallel
   against the 4 second interval (about 4 to 5 seconds per run, about 15 seconds under `-count=3`; the interval is a
   constant and is not changed): a nil stream ends the loop on the first tick, and a failing `Send` closes the stream
   and ends the loop. Fake stream fields are read under `muStream` or through a signal channel, and the loop end is
   awaited with a timed select. A "recent send is skipped" case is added only if the target is missed, because it
   needs a second tick (about 8 seconds).
4. Remaining small branches in ElevenLabs keepalive and read loop, `Start`, `StartWithID`, `Delete`,
   `websocketWrite` and `speakinghandler.Create` (about 12), and the `tts-control migrate` command wiring with an
   invalid DSN (1). `NewTTSHandler` is not covered: its constructor sends a warm-up RPC that cannot be isolated.

Cleanup: every session config created by a test is cancelled and its connections closed in `t.Cleanup`, the
`httptest` servers are closed, and the server handler goroutines are awaited.

## Verify

- Hand-written total >= 90.5% at the final HEAD with the baseline flags, then `-race -shuffle=on -count=3`,
  `go test ./...` and `-p 1` (no credential or dialer leakage between packages); lint clean.
- Mutation spot checks on a scratch copy, each must fail a new test: API key header, query parameter, dial error
  wrap, `cf.Stream = stream` assignment, `Stream != nil` short-circuit, `cf.Client = nil`, keepalive nil-stream
  return, `closeStreamLocked` call after a failed keepalive `Send`.
- `git diff --stat origin/main...HEAD` lists only `bin-tts-manager/` test files and this document.
