.. _webhook-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before configuring webhooks, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* A publicly accessible HTTPS endpoint URL where VoIPBIN will send event notifications.
* Knowledge of which event types you want to receive (e.g., ``call_created``, ``message_created``). See :ref:`Webhook Structure <webhook-struct-webhook>` for the full list.

.. note:: **AI Implementation Hint**

   Webhooks are configured at the **customer account level**, not as separate resources. Use ``PUT https://api.voipbin.net/v1.0/customer`` to set the ``webhook_uri`` and ``webhook_method`` fields on your customer profile. There is no ``/webhooks`` CRUD endpoint. Your webhook endpoint must be publicly reachable from the internet and respond with HTTP 200 within 5 seconds. For local development, use tools like ngrok to expose a local server.

Configure Webhook Endpoint
--------------------------

Set your webhook delivery URL and HTTP method by updating your customer profile. VoIPBIN will send all event notifications to this URL.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/customer?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "webhook_uri": "https://your-server.com/voipbin/webhook",
            "webhook_method": "POST"
        }'

.. note:: **AI Implementation Hint**

   The ``webhook_method`` field accepts one of: ``POST``, ``PUT``, ``GET``, ``DELETE``. Most implementations use ``POST``. The ``webhook_uri`` should be a valid HTTPS URL for production use. To verify your current webhook configuration, call ``GET https://api.voipbin.net/v1.0/customer`` and check the ``webhook_uri`` and ``webhook_method`` fields in the response.

Verify Webhook Configuration
-----------------------------

Retrieve your current customer profile to confirm the webhook settings.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/customer?token=<YOUR_AUTH_TOKEN>'

Check the ``webhook_uri`` and ``webhook_method`` fields in the response to verify your configuration.

Disable Webhooks
----------------

To stop receiving webhook notifications, set the ``webhook_uri`` to an empty string.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/customer?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "webhook_uri": "",
            "webhook_method": ""
        }'

Webhook Event Types
-------------------

VoIPBIN sends different event types to your webhook endpoint. For the complete list, see :ref:`Webhook Structure <webhook-struct-webhook>`.

**Call Events:**

- ``call_created`` - Call initiated
- ``call_ringing`` - Call is ringing
- ``call_answered`` - Call was answered
- ``call_updated`` - Call status changed
- ``call_hungup`` - Call ended

**Message Events:**

- ``message_created`` - Message created
- ``message_updated`` - Message status changed
- ``message_deleted`` - Message deleted

**Recording Events:**

- ``recording_created`` - Recording started
- ``recording_updated`` - Recording status changed
- ``recording_deleted`` - Recording deleted

**Transcription Events:**

- ``transcribe_created`` - Transcription session created
- ``transcribe_progressing`` - Transcription session started progressing
- ``transcribe_done`` - Transcription session finished
- ``transcribe_deleted`` - Transcription session deleted
- ``transcript_created`` - Transcript segment created
- ``transcript_deleted`` - Transcript segment deleted

**Queue Events:**

- ``queue_created`` - Queue created
- ``queue_updated`` - Queue updated
- ``queuecall_created`` - Call joined queue
- ``queuecall_kicking`` - Agent assigned to queue call
- ``queuecall_serviced`` - Queue call being handled

**Conference Events:**

- ``conference_created`` - Conference created
- ``conference_updated`` - Conference updated
- ``confbridge_created`` - Conference bridge created
- ``confbridge_updated`` - Participant joined/left

**Activeflow Events:**

- ``activeflow_created`` - Flow execution started
- ``activeflow_updated`` - Flow execution progressed
- ``activeflow_deleted`` - Flow execution ended

Receiving Webhook Events
-------------------------

Your webhook endpoint should accept POST requests and process the JSON payload. Here's an example webhook server implementation:

**Python (Flask) Example:**

.. code::

    from flask import Flask, request, jsonify

    app = Flask(__name__)

    @app.route('/voipbin/webhook', methods=['POST'])
    def voipbin_webhook():
        # Get the webhook payload
        payload = request.get_json()

        # Process different event types
        event_type = payload.get('type')

        if event_type == 'call_hungup':
            call_id = payload['data']['id']
            status = payload['data']['hangup_reason']

            print(f"Call {call_id} ended: {status}")

            # Your business logic here
            # - Update CRM
            # - Send notifications
            # - Trigger workflows

        elif event_type == 'message_created':
            message_id = payload['data']['id']
            from_number = payload['data']['source']['target']
            text = payload['data']['text']

            print(f"Message from {from_number}: {text}")

            # Process the message
            # - Auto-reply
            # - Route to agent
            # - Store in database

        elif event_type == 'recording_updated':
            recording_id = payload['data']['id']
            status = payload['data']['status']

            print(f"Recording {recording_id} status: {status}")

            # Handle recording
            # - Download and store
            # - Transcribe
            # - Send to customer

        # Return 200 OK to acknowledge receipt
        return jsonify({'status': 'received'}), 200

    if __name__ == '__main__':
        app.run(host='0.0.0.0', port=5000)

**Node.js (Express) Example:**

.. code::

    const express = require('express');
    const app = express();

    app.use(express.json());

    app.post('/voipbin/webhook', (req, res) => {
        const payload = req.body;
        const eventType = payload.type;

        console.log(`Received event: ${eventType}`);

        switch(eventType) {
            case 'call_hungup':
                handleCallHungup(payload.data);
                break;
            case 'message_created':
                handleMessageCreated(payload.data);
                break;
            case 'recording_updated':
                handleRecordingUpdated(payload.data);
                break;
            default:
                console.log(`Unknown event type: ${eventType}`);
        }

        // Acknowledge receipt
        res.status(200).json({ status: 'received' });
    });

    function handleCallHungup(data) {
        console.log(`Call ${data.id} ended`);
        // Your logic here
    }

    function handleMessageCreated(data) {
        console.log(`Message from ${data.source.target}: ${data.text}`);
        // Your logic here
    }

    function handleRecordingUpdated(data) {
        console.log(`Recording ${data.id} status: ${data.status}`);
        // Your logic here
    }

    app.listen(5000, () => {
        console.log('Webhook server listening on port 5000');
    });

Testing Webhooks
----------------

**Local Development:**

Use tools like ngrok to expose your local server for testing:

.. code::

    # Install ngrok
    $ brew install ngrok  # macOS
    $ snap install ngrok  # Linux

    # Start your webhook server locally
    $ python webhook_server.py

    # Expose it via ngrok
    $ ngrok http 5000

    # Use the ngrok URL in your webhook configuration
    # Example: https://abc123.ngrok.io/voipbin/webhook

**Testing with curl:**

Simulate a webhook event to test your endpoint:

.. code::

    $ curl --location --request POST 'http://localhost:5000/voipbin/webhook' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "type": "call_hungup",
            "data": {
                "id": "test-call-id",
                "hangup_reason": "normal",
                "source": {
                    "type": "tel",
                    "target": "+15551234567"
                },
                "destination": {
                    "type": "tel",
                    "target": "+15559876543"
                }
            }
        }'

Webhook Payload Structure
--------------------------

All webhook events follow this structure:

.. code::

    {
        "type": "call_hungup",
        "data": {
            // Event-specific data
            // Structure varies by event type
        }
    }

**Fields:**

- ``type``: Type of event that occurred (e.g., ``call_hungup``, ``message_created``)
- ``data``: Event-specific data (varies by event type). See :ref:`Webhook Structure <webhook-struct-webhook>` for details.

Best Practices
--------------

**1. Acknowledge Quickly:**
- Return 200 OK immediately upon receiving the webhook
- Process time-consuming tasks asynchronously (queue jobs, background workers)
- VoIPBIN expects a response within 5 seconds

**2. Handle Duplicates:**
- Webhooks may be delivered more than once
- Use the event ID or timestamp to detect and ignore duplicates
- Implement idempotent processing

**3. Secure Your Endpoint:**
- Use HTTPS for production webhooks
- Validate webhook authenticity (check source IP, use signatures)
- Implement rate limiting to prevent abuse

**4. Error Handling:**
- Log all webhook events for debugging
- Return appropriate HTTP status codes
- Implement retry logic for failed processing

**5. Event Filtering:**
- Filter events in your webhook handler based on the ``type`` field
- Process only events relevant to your application

**6. Monitoring:**
- Track webhook delivery success/failure rates
- Set up alerts for high failure rates
- Monitor webhook processing times

Common Use Cases
----------------

**CRM Integration:**
Automatically update your CRM when calls end:

.. code::

    if event_type == 'call_hungup':
        call_data = payload['data']

        # Update CRM contact record
        crm.update_contact(
            phone=call_data['destination']['target'],
            last_call_date=call_data['tm_hangup'],
            hangup_reason=call_data['hangup_reason']
        )

**Auto-Reply to Messages:**
Respond automatically to incoming messages:

.. code::

    if event_type == 'message_created':
        message = payload['data']
        from_number = message['source']['target']

        # Send auto-reply
        voipbin_api.send_message(
            to=from_number,
            text="Thanks for your message! We'll respond soon."
        )

**Recording Distribution:**
Email recordings to stakeholders when ready:

.. code::

    if event_type == 'recording_updated':
        recording = payload['data']

        if recording['status'] == 'ended':
            # Send email with recording info
            email.send(
                to='team@company.com',
                subject='Call Recording Available',
                body=f'Recording ID: {recording["id"]}'
            )

For more information about webhook configuration and event types, see :ref:`Webhook Overview <webhook-overview>`.
