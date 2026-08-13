# webchat-manager

# Deploy

`bin-webchat-manager-build` pushes the image, and a single `build-approval` gate covers
the whole pipeline (test -> build -> deploy) through to production.
`bin-webchat-manager-deploy` runs after `bin-webchat-manager-build`, bumping this service's pin on
bm-nyc-01 and recreating the container. See
`.circleci/scripts/ssh-deploy.sh` (this pattern was piloted with
bin-call-manager). The previous GKE deploy path for this service has been
removed.
