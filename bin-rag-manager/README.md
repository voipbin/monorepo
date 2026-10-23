# rag-manager

<!-- Updated dependencies: 2026-02-20 -->

# Deploy

`bin-rag-manager-build` pushes the image, and the test -> build pipeline runs on every branch push
without an approval step. The CircleCI `bin-rag-manager-deploy` job (direct SSH
deploy to <prod-host>) has been removed. Deploys to <prod-host> are manual until
this service migrates to the Komodo-managed deploy path (see bin-call-manager
for the pattern).
