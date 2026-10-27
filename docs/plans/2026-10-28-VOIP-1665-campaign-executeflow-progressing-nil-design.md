# VOIP-1665 campaignhandler executeFlow nil dereference

## 1. Ticket review
- `executeFlow` (`bin-campaign-manager/pkg/campaignhandler/execute.go`) assigns `cc, err := h.campaigncallHandler.Progressing(ctx, tmpCC.ID)`.
- `campaigncallHandler.Progressing` (`pkg/campaigncallhandler/status.go`) returns `nil, err` when `updateStatus` fails.
- The error branch calls `h.campaigncallHandler.Done(ctx, cc.ID, ...)` with the nil `cc`, so it panics (nil pointer dereference).
- Reproduced by a table case in `Test_Execute_branches` (Progressing mock returns `nil, err`): fails with the panic before the fix.
- Other call sites (`executeCall`, activeflow create/execute branches) use a non-nil `cc` or `tmpCC`. No other error branch in the service dereferences a possibly nil return value.

## 2. Design
- In the Progressing error branch, call `Done` with `tmpCC.ID`, the campaigncall already created by `Create`.
- Return `nil, err` as before. No other state transition or return value changes.
- `Done` uses only the id: the campaigncall becomes done/fail and the outdialtarget returns to idle (`status.go` calcDialtargetStatus). This is the same cleanup the other error branches in executeFlow already perform, and it also works when only the follow-up `Get` inside `Progressing` failed.
- Later branches keep `cc.ID` (non-nil there).

## 3. Verification
- New case "flow type with progressing error marks the created campaigncall as fail": expects `Done(ctx, ccID, ResultFail)` and no panic.
- It fails before the fix (panic) and passes after.
- Service tests and golangci-lint pass; campaignhandler coverage stays at or above its current 88.4%.

## 4. Out of scope
- Changing `Progressing` to return a non-nil value on error.
- Same pattern in other services (not checked).
