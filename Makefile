# Top-level monorepo Makefile.
# Per-service Go builds/tests live in each service's own toolchain — see
# docs/workflows/development-guide.md and docs/workflows/verification-workflows.md.
#
# This Makefile is reserved for repo-wide checks that don't belong in any one
# service.

.PHONY: lint-docs

lint-docs:
	@./scripts/check-docs.sh
