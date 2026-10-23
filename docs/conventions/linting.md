# Linting

CI runs two golangci-lint passes on every Go service.

| Pass | Config | Scope |
|------|--------|-------|
| Full | `.golangci.yml` | All code: the `standard` linter set (errcheck skipped in tests), the convention linters below, and the gofmt formatter |
| New-line complexity | `.golangci-errorlint.yml` with `--new-from-merge-base=origin/main` | gocyclo and nestif on lines a branch adds or changes only |

## Rules enforced on all code

| Linter | What it prevents |
|--------|------------------|
| errorlint | `fmt.Errorf` without `%w` (error-handling.md). Only `fmt.Errorf` wrapping is checked, not error comparison or type assertions |
| noctx, rowserrcheck | Context-less I/O and DB calls, unchecked `rows.Err()` (database.md) |
| nilerr | Returning `nil` after checking a non-nil error without handling it |
| bodyclose | HTTP response bodies that are never closed |
| contextcheck | Dropping or replacing the caller's context |
| wastedassign, predeclared, copyloopvar | Dead assignments, shadowing of predeclared identifiers, redundant loop variable copies |
| gocritic | Common Go anti-patterns (default checks) |
| gosec | Security issues, except G101, G115, G117 and G304 (too noisy for this codebase) |
| forbidigo | `fmt.Print*`, `print` and `println` in service code. Use logrus (logging.md) |
| revive | error-naming, error-return, indent-error-flow, superfluous-else, context-as-argument, context-keys-type, empty-block, unreachable-code, var-naming |
| usestdlibvars | Magic strings and numbers for `net/http` methods and status codes |
| nolintlint | `//nolint` without a specific linter name or without an explanation |

## Rules enforced on new lines only

| Linter | What it prevents |
|--------|------------------|
| gocyclo, nestif | Cyclomatic complexity above 20 per function, nestif complexity of 6 or more per top-level `if` |

Existing functions above these limits are not reported until the declaration line is touched. Touching a legacy violating line makes it a new line, so fix it in the same change.

## Exemptions

- `_test.go` files are exempt from noctx, rowserrcheck, gosec, contextcheck, forbidigo, bodyclose, copyloopvar, gocritic, usestdlibvars and revive (and from gocyclo and nestif in the new-line pass). errorlint, nilerr, wastedassign, predeclared and nolintlint still apply to tests.
- `cmd/*-control/` operator CLIs and `bin-common-handler/pkg/migrationhandler/command.go` (the shared migrate CLI commands) are exempt from forbidigo, because they print to stdout on purpose. Any other file that prints to stdout needs `//nolint:forbidigo // <reason>`.
- `bin-api-manager/server/` methods that implement the generated `ServerInterface` are exempt from the revive `var-naming` method-name message, because the names must match `gens/openapi_server/gen.go` (for example `GetAccesskeysId`). Method parameter and local variable names in those files are still checked.

## Deliberate exceptions

Use `//nolint:<linter> // <reason>` on the line the linter reports. For a multi-line statement that is the line holding the reported position (often an argument line), not necessarily the first line. nolintlint rejects a bare `//nolint` and a missing reason. The `nolint:magicstring` marker read by `.claude/scripts/check-magic-strings.sh` is not a golangci-lint directive. Write it without a space after the slashes and with a reason (`//nolint:magicstring // <reason>`); nolintlint rejects the spaced form `// nolint:magicstring`.

golangci-lint reports one issue per line by default (`uniq-by-line`), so a line that breaks two rules shows the second one after the first is fixed.

## Known limits

- gocyclo and nestif report on the function or `if` declaration line. Adding code inside an existing function without touching that line can push it over the threshold without a report. The gate stops new complex functions, not the growth of legacy ones. About 90 existing reports remain (70 gocyclo and 21 nestif on 2026-10-23). A later phase decides whether to clean them up and promote the two linters.
- Import grouping (imports.md) is not enforced yet.
- `allow-unused: true` is set for nolintlint, so a `//nolint` that no longer suppresses anything is not reported.
- Style linters (revive unused-parameter and exported-comment rules, shadow, perfsprint, unparam, nilnil, prealloc, goimports) are not enabled.

## Running locally

```bash
cd bin-<service>
golangci-lint run                                                  # full pass
golangci-lint run -c "$(git rev-parse --show-toplevel)/.golangci-errorlint.yml" --new-from-merge-base=origin/main   # complexity on new lines
```
