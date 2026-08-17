# number-manager
Number-manager for telephone numbers management.

# test numbers
* +821100000001: test talk
* +821100000002: test conference
* +821100000003: test transcribe

# telnyx
account: sungtae@voipbin.net
* +15734531118
* +14703298699

<!-- Updated dependencies: 2026-02-20 -->

# Deploy

`bin-number-manager-build` pushes the image, and a single `build-approval` gate covers
the test -> build pipeline. The CircleCI `bin-number-manager-deploy` job (direct SSH
deploy to bm-nyc-01) has been removed. Deploys to bm-nyc-01 are manual until
this service migrates to the Komodo-managed deploy path (see bin-call-manager
for the pattern).
