# Workflows

Process documentation for git operations, verification, common multi-service workflows, and known production gotchas.

| File | Description |
|---|---|
| [git-workflow-guide.md](git-workflow-guide.md) | Branch naming, commit message format, PR creation, squash-merge rules, and conflict-precheck workflow |
| [verification-workflows.md](verification-workflows.md) | The 5-step verification workflow (`go mod tidy` → `vendor` → `generate` → `test` → `lint`) and why each step is mandatory |
| [common-workflows.md](common-workflows.md) | Step-by-step guides for adding a new API endpoint, creating a flow action, adding a manager service, modifying shared models, parsing filters from request body, and Alembic migrations |
| [special-cases.md](special-cases.md) | bin-common-handler updates (cross-monorepo verification), OpenAPI sync, and other workflows that span multiple services |
| [development-guide.md](development-guide.md) | Local build commands, testing patterns, code generation, and linting reference |
| [common-gotchas.md](common-gotchas.md) | Hard-won production lessons: shared-library signature updates, Prometheus metric collisions, UUID `db:` tags, model/OpenAPI sync, RST docs sync |
