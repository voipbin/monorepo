# voip-kamailio-proxy: cover the UDP write error path of the SIP OPTIONS check

## Goal

Add the remaining unit test that is reachable without changing production code. The hand-written statement coverage of the service moves from 294/330 (89.09%) to 296/330 (89.70%).

## Scope

- Test only, in `pkg/siphandler/health_test.go`. No production code, signature, or `cmd/` change.
- No new module. No network traffic leaves the host.

## Design

`SendOptionsCheck` converts every failure into an unhealthy result with response code `timeout`. The write step is the one failure path not yet exercised.

The new test calls `SendOptionsCheck` with `127.0.0.1` and a negative timeout. The socket deadline is then already in the past, so the UDP write fails locally with an i/o timeout and nothing is sent.

## Verify

1. The result is `Healthy == false`, `ResponseCode == "timeout"`, and the returned error is nil.
2. A debug log hook (same style as the existing dial failure test: global logger level and hooks restored in `t.Cleanup`, entries filtered by the `hostname` field, no `t.Parallel()`) shows the write-failure message, and none of the resolve, dial, set-deadline, or read failure messages. This proves the intended branch was reached and not an earlier one.
3. The test does not occupy a fixed port and does not depend on the clock beyond the already-expired deadline.
4. Mutation check: making the write failure return a healthy result makes the test fail.
5. Coverage of the service measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile` is 296/330 or higher. Existing tests and lint pass.
