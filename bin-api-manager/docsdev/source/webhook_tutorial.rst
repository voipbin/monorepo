.. _webhook-tutorial:

Tutorial
========

Create a Webhook
----------------

Register a webhook endpoint to receive real-time event notifications from VoIPBIN. Webhooks notify your server when events occur (call status changes, messages received, etc.).

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/webhooks?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "Production Webhook",
            "detail": "Main webhook for production events",
            "uri": "https://your-server.com/voipbin/webhook",
            "method": "POST",
            "event_types": [
                "call.status",
                "message.received",
                "recording.completed"
            ]
        }'

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "12345678-1234-1234-1234-123456789012",
        "name": "Production Webhook",
        "detail": "Main webhook for production events",
        "uri": "https://your-server.com/voipbin/webhook",
        "method": "POST",
        "event_types": [
            "call.status",
            "message.received",
            "recording.completed"
        ],
        "status": "active",
        "tm_create": "2026-01-20 10:30:00.000000",
        "tm_update": "2026-01-20 10:30:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

The webhook is now active and will receive POST requests when specified events occur.

Get List of Webhooks
---------------------

Retrieve all registered webhooks for your account.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/webhooks?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
                "customer_id": "12345678-1234-1234-1234-123456789012",
                "name": "Production Webhook",
                "detail": "Main webhook for production events",
                "uri": "https://your-server.com/voipbin/webhook",
                "method": "POST",
                "event_types": [
                    "call.status",
                    "message.received"
                ],
                "status": "active",
                "tm_create": "2026-01-20 10:30:00.000000",
                "tm_update": "2026-01-20 10:30:00.000000",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ]
    }

Update a Webhook
----------------

Modify an existing webhook's configuration, such as changing the URI or event types.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/webhooks/a1b2c3d4-e5f6-7890-abcd-ef1234567890?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "Production Webhook - Updated",
            "event_types": [
                "call.status",
                "call.completed",
                "message.received",
                "recording.completed",
                "transcribe.completed"
            ]
        }'

Delete a Webhook
----------------

Remove a webhook when it's no longer needed.

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/webhooks/a1b2c3d4-e5f6-7890-abcd-ef1234567890?token=<YOUR_AUTH_TOKEN>'

Webhook Event Types
-------------------

VoIPBIN sends different event types to your webhook endpoint. Common event types include:

**Call Events:**
- ``call.status`` - Call status changed (dialing, ringing, answered, hangup)
- ``call.completed`` - Call ended
- ``call.recording`` - Recording status changed

**Message Events:**
- ``message.received`` - Incoming message received
- ``message.sent`` - Outgoing message sent
- ``message.delivered`` - Message delivery confirmed

**Recording Events:**
- ``recording.started`` - Recording started
- ``recording.completed`` - Recording finished and available

**Transcription Events:**
- ``transcribe.started`` - Transcription started
- ``transcribe.completed`` - Transcription finished
- ``transcribe.updated`` - Real-time transcription update

**Queue Events:**
- ``queue.joined`` - Caller joined queue
- ``queue.answered`` - Agent answered queued call
- ``queue.abandoned`` - Caller left queue

**Conference Events:**
- ``conference.participant.joined`` - Participant joined conference
- ``conference.participant.left`` - Participant left conference
- ``conference.ended`` - Conference ended

Receiving Webhook Events
-------------------------

Your webhook endpoint should accept POST requests and process the JSON payload. Here's an example webhook server implementation:

**Python (Flask) Example:**

.. code::

    from flask import Flask, request, jsonify
    import hmac
    import hashlib

    app = Flask(__name__)

    @app.route('/voipbin/webhook', methods=['POST'])
    def voipbin_webhook():
        # Get the webhook payload
        payload = request.get_json()

        # Process different event types
        event_type = payload.get('event_type')

        if event_type == 'call.completed':
            call_id = payload['data']['id']
            duration = payload['data']['duration']
            status = payload['data']['status']

            print(f"Call {call_id} completed: {duration}s, status: {status}")

            # Your business logic here
            # - Update CRM
            # - Send notifications
            # - Trigger workflows

        elif event_type == 'message.received':
            message_id = payload['data']['id']
            from_number = payload['data']['source']['target']
            text = payload['data']['text']

            print(f"Message from {from_number}: {text}")

            # Process the message
            # - Auto-reply
            # - Route to agent
            # - Store in database

        elif event_type == 'recording.completed':
            recording_id = payload['data']['id']
            url = payload['data']['url']

            print(f"Recording {recording_id} available at: {url}")

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
        const eventType = payload.event_type;

        console.log(`Received event: ${eventType}`);

        switch(eventType) {
            case 'call.completed':
                handleCallCompleted(payload.data);
                break;
            case 'message.received':
                handleMessageReceived(payload.data);
                break;
            case 'recording.completed':
                handleRecordingCompleted(payload.data);
                break;
            default:
                console.log(`Unknown event type: ${eventType}`);
        }

        // Acknowledge receipt
        res.status(200).json({ status: 'received' });
    });

    function handleCallCompleted(data) {
        console.log(`Call ${data.id} completed`);
        // Your logic here
    }

    function handleMessageReceived(data) {
        console.log(`Message from ${data.source.target}: ${data.text}`);
        // Your logic here
    }

    function handleRecordingCompleted(data) {
        console.log(`Recording available: ${data.url}`);
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
            "event_type": "call.completed",
            "timestamp": "2026-01-20T10:30:00.000000Z",
            "data": {
                "id": "test-call-id",
                "status": "completed",
                "duration": 120,
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
        "event_type": "call.completed",
        "timestamp": "2026-01-20T10:30:00.000000Z",
        "webhook_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "12345678-1234-1234-1234-123456789012",
        "data": {
            // Event-specific data
            // Structure varies by event type
        }
    }

**Fields:**
- ``event_type``: Type of event that occurred
- ``timestamp``: When the event occurred (ISO 8601 format in UTC)
- ``webhook_id``: ID of the webhook that triggered this event
- ``customer_id``: Your VoIPBIN customer ID
- ``data``: Event-specific data (varies by event type)

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
- Subscribe only to events you need
- Filter events in your webhook handler
- Update webhook configuration as requirements change

**6. Monitoring:**
- Track webhook delivery success/failure rates
- Set up alerts for high failure rates
- Monitor webhook processing times

Common Use Cases
----------------

**CRM Integration:**
Automatically update your CRM when calls complete:

.. code::

    if event_type == 'call.completed':
        call_data = payload['data']

        # Update CRM contact record
        crm.update_contact(
            phone=call_data['destination']['target'],
            last_call_date=call_data['tm_hangup'],
            call_duration=call_data['duration'],
            call_status=call_data['status']
        )

**Auto-Reply to Messages:**
Respond automatically to incoming messages:

.. code::

    if event_type == 'message.received':
        message = payload['data']
        from_number = message['source']['target']

        # Send auto-reply
        voipbin_api.send_message(
            to=from_number,
            text="Thanks for your message! We'll respond soon."
        )

**Recording Distribution:**
Email recordings to stakeholders when completed:

.. code::

    if event_type == 'recording.completed':
        recording = payload['data']

        # Send email with recording link
        email.send(
            to='team@company.com',
            subject=f'Call Recording Available',
            body=f'Recording URL: {recording["url"]}'
        )

For more information about webhook configuration and event types, see :ref:`Webhook Overview <webhook-overview>`.
