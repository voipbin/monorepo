# Fix Number Renew Process

## Problem Statement

The number-renew process has several bugs and reliability issues that can cause incorrect billing, data corruption, and incomplete renewals.

## Current Behavior

A Kubernetes CronJob triggers `POST /v1/numbers/renew` with `{"days":28}`. The number-manager queries numbers where `tm_renew < (now - 28 days)`, checks each customer's billing balance, and either renews the number (updates `tm_renew` to now, publishes `number_renewed` event) or deletes it (insufficient balance).

## Issues Found

### Bug 1: Deleted numbers still get renewed (Critical)

In `renewNumbersByTMRenew` (`renew.go:71-92`), when a number is deleted for insufficient balance, execution falls through to the renewal code because there is no `continue` after the delete block. The deleted number then has its `tm_renew` updated and a `number_renewed` event is published, creating a billing record for a deleted number.

### Bug 2: CronJob runs 60 times per hour instead of once per day

`k8s/cronjob.yml` uses schedule `"* 1 * * *"` which means "every minute during the 1 AM hour" (60 runs). It should be `"0 1 * * *"` for once at 1:00 AM daily.

### Reliability: Hard-coded limit of 100 with no pagination

`dbListByTMRenew` always fetches at most 100 numbers. If more than 100 numbers need renewal, the rest are silently skipped. With the buggy 60x cron, subsequent runs accidentally pick up remaining numbers, but with a corrected schedule this would leave numbers un-renewed.

### Code Quality: nil slice initialization

`RenewNumbers` (`renew.go:25`) uses `var res []*number.Number` which serializes to JSON `null` instead of `[]`.

### Code Quality: Typos and comment mismatch

- `renew.go:117`: Comment says `renewNumbersByDays` but function is `renewNumbersByHours`
- `renew.go:106, 125`: "Renwing" should be "Renewing"

### Test Coverage: Only happy path tested

No tests for insufficient balance (delete path), balance check errors, DB update errors, empty results, or mixed valid/invalid numbers.

## Approach

### Fix 1: Add `continue` after delete block

After deleting a number for insufficient balance, add `continue` to skip the renewal logic for that number.

### Fix 2: Correct cron schedule

Change `"* 1 * * *"` to `"0 1 * * *"`.

### Fix 3: Add pagination loop in `renewNumbersByTMRenew`

Wrap the DB query and processing in a loop. Since each renewed number's `tm_renew` gets updated to now (moving it past the threshold) and deleted numbers are removed from future queries, re-querying with the same threshold naturally returns the next batch. Loop until the query returns an empty set.

### Fix 4: nil slice initialization

Change `var res []*number.Number` to `res := []*number.Number{}`.

### Fix 5: Typos and comment

Correct the comment on `renewNumbersByHours` and fix "Renwing" typos.

### Fix 6: Add test cases

Add tests for: insufficient balance (delete path), balance check error (skip), DB update error (skip), empty DB result, and mixed valid/invalid numbers.

## Files to Change

| File | Changes |
|------|---------|
| `bin-number-manager/pkg/numberhandler/renew.go` | Fix fallthrough bug, nil slice, pagination loop, typos, comment |
| `bin-number-manager/pkg/numberhandler/renew_test.go` | Add error/edge case test cases |
| `bin-number-manager/k8s/cronjob.yml` | Fix cron schedule |

## Out of Scope

- Adding a country field to the Number model (the `country` parameter in balance validation is accepted but not used in billing logic today)
- Changing the page size from 100 (pagination loop handles arbitrary counts)
- Adding retry/circuit-breaker for balance check RPC failures (current skip-and-retry-next-cycle behavior is acceptable)
