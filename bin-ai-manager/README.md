# ai-manager

<!-- Updated dependencies: 2026-02-20 -->

# Deploy

`bin-ai-manager-build` pushes the image, and a single `build-approval` gate covers
the test -> build pipeline. The CircleCI `bin-ai-manager-deploy` job (direct SSH
deploy to <prod-host>) has been removed. Deploys to <prod-host> are manual until
this service migrates to the Komodo-managed deploy path (see bin-call-manager
for the pattern).
