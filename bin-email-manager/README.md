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
