# pipecat-manager

# requirements.txt
poetry export -f requirements.txt --output requirements.txt --without-hashes
<!-- Updated dependencies: 2026-03-30 -->

# Deploy

Komodo-managed (VOIP-1350), same as the other `bin-*-manager` services. CI runs
`bin-pipecat-manager-build` (pushes the image) then `bin-pipecat-manager-deploy`
(renders the image tag into `komodo/docker-compose.yml` and deploys via the
Komodo API), gated behind a single `build-approval` step. No manual deploy
step. See [docs/operations.md](docs/operations.md#deployment-komodo) for
details, including three intentional deviations from the standard Tier 1/2
template.

Redeploying (even with no source change here) is sometimes required to pick
up a `bin-ai-manager` change — see
[docs/dependencies.md](docs/dependencies.md) for why.
