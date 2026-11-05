# bin-email-manager
The bin-email-manager is responsible for managing email delivery within the VoIPBin platform. It integrates with various email providers and ensures reliable email communication for VoIPBin services.

## Features
* Send emails via supported providers.
* Handle webhook events for delivery status updates.
* Seamless integration with VoIPBin flows for automated email actions.

## Webhook Configuration
The bin-email-manager supports webhook event reception to track email delivery status and related events.

## Supported Providers

* Mailgun (primary)
* SendGrid (fallback)

Configure the provider webhooks to point to the following endpoints:

```Mailgun: https://hook.voipbin.net/v1.0/emails/mailgun```

```Sendgrid: https://hook.voipbin.net/v1.0/emails/sendgrid```

For Mailgun, register the URL for the accepted, delivered, temporary failure, permanent failure, opened, clicked, unsubscribed and complained events. Opened and clicked events need open and click tracking enabled on the Mailgun domain.

<!-- Updated dependencies: 2026-02-20 -->

# Deploy

`bin-email-manager-build` pushes the image, and the test -> build pipeline runs on every branch push
without an approval step. The CircleCI `bin-email-manager-deploy` job (direct SSH
deploy to <prod-host>) has been removed. Deploys to <prod-host> are manual until
this service migrates to the Komodo-managed deploy path (see bin-call-manager
for the pattern).
