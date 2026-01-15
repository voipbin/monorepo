# update-all-services

Update all services in monorepo after bin-common-handler changes.

## Usage

```bash
/update-all-services [--skip-tests] [--service-pattern <pattern>]
```

## Flags

- `--skip-tests`: Skip `go test ./...` step (faster, use for quick dependency sync checks)
- `--service-pattern <pattern>`: Filter services by pattern (e.g., "bin-*-manager", "bin-call-manager")

## What This Does

Updates all 30+ services in the monorepo with the standard workflow:

**For each service:**
1. Navigate to service directory
2. Run `go mod tidy` - Sync go.mod with dependencies
3. Run `go mod vendor` - Update vendored dependencies
4. Run `go generate ./...` - Regenerate mocks and generated code
5. Run `go mod vendor` again - Re-vendor after generate (catches dependencies added during code generation)
6. Run `go test ./...` - Verify nothing broke (unless --skip-tests)
7. Report progress and timing

**Progress display:**
- Shows current service being updated (e.g., [3/32])
- Shows progress for each step with timing
- Collects failures and continues with remaining services
- Displays summary at end with pass/fail counts
- Saves detailed log to file

## Examples

```bash
# Update all services (standard workflow)
/update-all-services

# Skip tests for faster dependency sync
/update-all-services --skip-tests

# Update only specific services
/update-all-services --service-pattern "bin-call-manager"
/update-all-services --service-pattern "bin-*-manager"

# Update multiple specific services
/update-all-services --service-pattern "bin-call-manager|bin-flow-manager"
```

## Output Example

```
Discovering services in monorepo...
Found 32 services with go.mod files

Starting updates (estimated 15-20 minutes)...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[1/32] bin-ai-manager
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✓ go mod tidy         1.2s
  ✓ go mod vendor       3.4s
  ✓ go generate ./...   2.1s
  ✓ go mod vendor       0.2s (re-vendor after generate)
  ✓ go test ./...       8.7s (23/23 tests passed)
  ✓ Complete           15.6s total

[2/32] bin-api-manager
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✓ go mod tidy         1.5s
  ✓ go mod vendor       4.2s
  ✓ go generate ./...   3.8s
  ✓ go mod vendor       0.3s (re-vendor after generate)
  ✗ go test ./...      FAILED after 12.3s

    Error: 1 test failed
    --- FAIL: TestConversationGet (0.01s)
        Expected: 200
        Got: 400

  ✗ Skipping remaining steps

[3/32] bin-billing-manager
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✓ go mod tidy         1.3s
  ✓ go mod vendor       3.9s
  ✓ go generate ./...   2.5s
  ✓ go mod vendor       0.2s (re-vendor after generate)
  ✓ go test ./...       7.2s (18/18 tests passed)
  ✓ Complete           15.2s total

... (continues for all 32 services) ...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total services: 32
✓ Passed:       30 (93.8%)
✗ Failed:       2 (6.2%)

Failed services:
  - bin-api-manager (test failure)
  - bin-conference-manager (generate error)

Total time: 18m 42s
Average:    35s per service

Detailed log saved to: /tmp/monorepo-update-2026-01-15-143022.log

Next steps:
  1. Review failures in log file
  2. Fix issues in failed services
  3. Re-run for failed services:
     /update-all-services --service-pattern "bin-api-manager|bin-conference-manager"
```

## When to Use

**CRITICAL: Use this command after ANY changes to:**
- `bin-common-handler` (shared library used by all services)
- `bin-openapi-manager` (OpenAPI specs and generated types)
- Any shared model or interface changes

**Why:** Changes to shared libraries require all dependent services to update their vendored dependencies and regenerate code.

## What It Prevents

- ❌ Missing services when updating dependencies
- ❌ No visibility into which service is being updated
- ❌ Unable to resume if one service fails
- ❌ No summary of overall success/failure
- ❌ Difficult to identify which services failed
- ❌ Complex find command syntax errors

## Why This Command Exists

- CRITICAL operation after bin-common-handler changes (mandatory)
- Affects all 30+ services in monorepo
- Takes 10-20+ minutes to complete
- Currently requires complex find command from CLAUDE.md
- Easy to lose track of progress
- Failures are hard to diagnose without progress output

## Implementation Details

This command:
1. Finds all `go.mod` files in monorepo (maxdepth 2)
2. Filters by service pattern if provided
3. For each service:
   - Changes to service directory
   - Runs workflow steps sequentially
   - Captures output and timing
   - Continues on failure (doesn't stop)
4. Collects all results
5. Displays summary with pass/fail counts
6. Saves detailed log to `/tmp/monorepo-update-<timestamp>.log`

## Exit Codes

- `0`: All services passed
- `1`: At least one service failed (check summary for details)

## Related Documentation

See `CLAUDE.md` section "Special Case: Changes to bin-common-handler" for complete workflow and when to use this command.
