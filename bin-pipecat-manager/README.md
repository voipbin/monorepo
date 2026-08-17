# pipecat-manager

# requirements.txt
poetry export -f requirements.txt --output requirements.txt --without-hashes
<!-- Updated dependencies: 2026-03-30 -->

# Deploy

`bin-pipecat-manager-build` pushes the image, and a single `build-approval` gate covers
the test -> build pipeline. The CircleCI `bin-pipecat-manager-deploy` job (direct SSH
deploy to bm-nyc-01) has been removed. Deploys to bm-nyc-01 are manual until
this service migrates to the Komodo-managed deploy path (see bin-call-manager
for the pattern).
