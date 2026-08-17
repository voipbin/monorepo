# contact-manager

<!-- Updated dependencies: 2026-02-20 -->

# Deploy

`bin-contact-manager-build` pushes the image, and a single `build-approval` gate covers
the test -> build pipeline. The CircleCI `bin-contact-manager-deploy` job (direct SSH
deploy to bm-nyc-01) has been removed. Deploys to bm-nyc-01 are manual until
this service migrates to the Komodo-managed deploy path (see bin-call-manager
for the pattern).
