# VOIP-1489: Return SIP 403 for inactive-account incoming calls

## Problem
`startCallTypeFlow` (bin-call-manager/pkg/callhandler/start.go) hangs up an incoming call of a
customer whose status is not active/initial with `ChannelCauseNetworkOutOfOrder` (38). Callers see
a platform failure instead of a policy rejection.

## Facts (origin/main 289c75162)
- `ValidateCustomerStatusIncoming` returns false only for frozen, deleted, expired. A lookup error
  is fail-open (nil, true), so this branch is a pure policy rejection.
- `ChannelCauseCallRejected` = 21. Upstream Asterisk (23.4.0, 23.5.0, master) `hangup_cause2sip`
  maps 21 -> 403; 38 has no case (0), which `ast_sip_session_terminate` turns into 603. The ticket's
  log shows tech_cause 500, which conflicts and is not resolved here. The target is 403 either way.
- The branch runs before Answer and before any call row, activeflow or balance work. After hangup,
  `ChannelDestroyed` fails on `CallGetByChannelID` as before; only the stored channel `hangup_cause`
  and the `channel_hangup_total` reason label move from 38 to 21. `isRetryable` is outgoing-only
  and 21 is already a non-retry cause.
- 15 call sites reach this branch (DID, registrar, SIP, direct, trunk). No RST change.

## Change
- start.go: this one branch uses `ChannelCauseCallRejected`, comment `// return 403. policy rejection`.
- Unchanged (cause 38): balance (mixes billing RPC errors with insufficient funds), addCallBridge,
  activeflow create, call create, setVariablesCall, ActionNext. No per-status codes, no body.

## Tests
- covB sip test: only the "customer not active after flow created" row changes; "flow create error" stays 38.
- covB direct flow test: the "customer not active" row changes.
- New `Test_startCallTypeFlow_inactiveCustomer` (frozen, deleted, expired): expects `UUIDCreate`,
  `CustomerV1CustomerGet`, `HangingUp(21)` only; any other mock call fails.

## Not verified
- Carrier behavior for 403 versus the current response; the deployed Asterisk build (source only).
- Unit tests assert `HangingUp(21)` only; the on-wire 403 needs a sandbox INVITE capture.
