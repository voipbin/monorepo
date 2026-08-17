# bin-email-manager
The bin-email-manager is responsible for managing email delivery within the VoIPBin platform. It integrates with various email providers and ensures reliable email communication for VoIPBin services.

## Features
* Send emails via supported providers.
* Handle webhook events for delivery status updates.
* Seamless integration with VoIPBin flows for automated email actions.

## Webhook Configuration
The bin-email-manager supports webhook event reception to track email delivery status and related events.

## Supported Providers

* SendGrid

For SendGrid, configure your webhook to point to the following endpoint:

```Sendgrid: https://hook.voipbin.net/v1.0/emails/sendgrid```

<!-- Updated dependencies: 2026-02-20 -->

# Deploy

`bin-email-manager-build` pushes the image, and a single `build-approval` gate covers
the test -> build pipeline. The CircleCI `bin-email-manager-deploy` job (direct SSH
deploy to bm-nyc-01) has been removed. Deploys to bm-nyc-01 are manual until
this service migrates to the Komodo-managed deploy path (see bin-call-manager
for the pattern).
