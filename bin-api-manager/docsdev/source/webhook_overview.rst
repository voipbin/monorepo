.. _webhook-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free -- Webhook delivery does not incur charges. Events are pushed to your configured endpoint at no cost.
   * **Async:** No. ``POST /webhooks`` creates the webhook configuration synchronously. Event delivery to your endpoint happens asynchronously as events occur.

Webhooks, a robust feature offered by VoIPBIN, empower users to receive real-time event data for their calls and associated resources directly on their servers. By establishing custom endpoints, users can seamlessly configure their servers to receive timely notifications and updates related to VoIPBIN resources, thereby enhancing control, customization, and real-time visibility within their communication workflows.

Notification Mechanism
----------------------
Webhook events act as notifications sent by VoIPBIN, triggering when specific events or actions unfold within the system, such as call events, message events, or changes to resources like queues or agents. Configured webhook endpoints receive these events, ensuring users promptly receive pertinent data related to their VoIPBIN resources.

.. image:: _static/images/webhook_overview_notification.png

Types of Webhooks
-----------------
VoIPBIN tailors webhooks for each resource type, ensuring users receive granular progress and updates for various events. This resource-specific approach allows users to monitor their VoIPBIN resources with precision, obtaining notifications and data tailored to each resource type. For instance, call-specific webhook events furnish details on call status, duration, and caller ID, while message-specific events offer insights into SMS or MMS messages, including content, sender ID, and delivery status.

Custom Endpoints
----------------
To harness webhooks, users must configure custom webhook endpoints on their servers. These endpoints, serving as URLs, dictate where VoIPBIN transmits webhook events. Upon an event occurrence, VoIPBIN initiates an HTTP request to the configured endpoint, incorporating relevant data in the payload. This empowers users to process and respond to events according to their unique requirements.

.. note:: **AI Implementation Hint**

   Your webhook endpoint must respond with HTTP 200 within 5 seconds. VoIPBIN may retry delivery if your server is unreachable. Webhooks may be delivered more than once, so implement idempotent processing using the event type and resource ID to deduplicate.

Webhook Event Types
-------------------
VoIPBIN sends webhook events for various resource types. Each event includes the resource type, event type, and the full resource data.

========================= ======================================================
Resource Type             Events
========================= ======================================================
``call``                  Call status changes (dialing, ringing, progressing, hangup)
``groupcall``             Groupcall status changes
``conference``            Conference lifecycle events
``conferencecall``        Participant join/leave events
``message``               Message sent/received/delivered events
``recording``             Recording started/completed events
``transcribe``            Transcription completed events
``queue``                 Queue events
``queuecall``             Queue call events (joined, agent connected, timeout)
``activeflow``            Flow execution events
``campaign``              Campaign status changes
``number``                Number provisioning events
========================= ======================================================

Benefits of Webhooks
--------------------
Webhooks deliver a range of advantages for VoIPBIN users:

* **Real-Time Updates**: Offering immediate event notifications, webhooks keep users abreast of real-time changes to their VoIPBIN resources.
* **Customization**: Users can tailor webhook endpoints and process data as per their specific needs, facilitating the creation of customized workflows and integrations.
* **Automated Actions**: Webhooks enable users to automate actions based on event data, such as record updates, notifications, or the initiation of additional processes.
* **Enhanced Monitoring**: Providing a proactive monitoring solution, webhooks empower users to track and respond promptly to changes within the VoIPBIN system, ensuring informed decision-making.

Troubleshooting
---------------

* **Webhooks not being received:**
    * **Cause:** Your webhook endpoint URL is incorrect, unreachable, or not responding with HTTP 200.
    * **Fix:** Verify the endpoint URL via ``GET /webhooks``. Ensure your server is publicly accessible and responds with ``200 OK`` within 5 seconds.

* **Duplicate webhook events:**
    * **Cause:** VoIPBIN retries delivery if your endpoint did not respond in time.
    * **Fix:** Implement idempotent processing. Use the combination of resource ``id`` and ``status`` to deduplicate events.

* **400 Bad Request (creating webhook):**
    * **Cause:** Invalid URL format or missing required fields.
    * **Fix:** Ensure the ``url`` field is a valid HTTPS URL. Verify ``event_types`` is a non-empty array.
