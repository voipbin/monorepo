# pipecat-manager

# requirements.txt
poetry export -f requirements.txt --output requirements.txt --without-hashes
<!-- Updated dependencies: 2026-03-30 -->

# Deploy

`bin-pipecat-manager-build` pushes the image, and a single `build-approval` gate covers
the whole pipeline (test -> build -> deploy) through to production.
`bin-pipecat-manager-deploy` runs after `bin-pipecat-manager-build`, bumping this service's pin on
bm-nyc-01 and recreating the container. See
`.circleci/scripts/ssh-deploy.sh` (this pattern was piloted with
bin-call-manager). The previous GKE deploy path for this service has been
removed.
