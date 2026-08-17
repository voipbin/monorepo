# conversation-manager

Manages multi-channel conversations (SMS/MMS, LINE, WhatsApp) and their messages, handling bidirectional communication with external messaging platforms.

<!-- Updated dependencies: 2026-03-30 -->

# Deploy

`bin-conversation-manager-build` pushes the image, and a single `build-approval` gate covers
the test -> build pipeline. The CircleCI `bin-conversation-manager-deploy` job (direct SSH
deploy to bm-nyc-01) has been removed. Deploys to bm-nyc-01 are manual until
this service migrates to the Komodo-managed deploy path (see bin-call-manager
for the pattern).
