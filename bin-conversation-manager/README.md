# conversation-manager

Manages multi-channel conversations (SMS/MMS, LINE, WhatsApp) and their messages, handling bidirectional communication with external messaging platforms.

<!-- Updated dependencies: 2026-03-30 -->

# Deploy

`bin-conversation-manager-build` pushes the image, and a single `build-approval` gate covers
the whole pipeline (test -> build -> deploy) through to production.
`bin-conversation-manager-deploy` runs after `bin-conversation-manager-build`, bumping this service's pin on
bm-nyc-01 and recreating the container. See
`.circleci/scripts/ssh-deploy.sh` (this pattern was piloted with
bin-call-manager). The previous GKE deploy path for this service has been
removed.
