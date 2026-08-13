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
the whole pipeline (test -> build -> deploy) through to production.
`bin-number-manager-deploy` runs after `bin-number-manager-build`, bumping this service's pin on
bm-nyc-01 and recreating the container. See
`.circleci/scripts/ssh-deploy.sh` (this pattern was piloted with
bin-call-manager). The previous GKE deploy path for this service has been
removed.
