# Top-level monorepo Makefile.
# Per-service Go builds/tests live in each service's own toolchain — see
# docs/workflows/development-guide.md and docs/workflows/verification-workflows.md.
#
# This Makefile is reserved for repo-wide checks that don't belong in any one
# service.

.DEFAULT_GOAL := lint-docs

.PHONY: lint-docs lint-error-envelope

lint-docs:
	@./scripts/check-docs.sh

lint-error-envelope:
	@./scripts/check-error-envelope.sh
