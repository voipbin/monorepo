.. _transcribe-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with transcription, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* An active call or conference in ``progressing`` status. Obtain the call ID via ``GET /calls`` or conference ID via ``GET /conferences``.
* A BCP47 language code matching the speaker's language (e.g., ``en-US``, ``ko-KR``). See :ref:`Supported Languages <transcribe-overview-supported_languages>`.
* (Optional for recording transcription) A recording ID from ``GET /recordings``.

.. note:: **AI Implementation Hint**

   Transcription can only be started on a call or conference that is in ``progressing`` status (i.e., answered and active). For recording transcription, the recording must exist and be in ``ended`` status. The ``language`` parameter is required and must be a valid BCP47 code; there is no auto-detect mode.

Start Transcription with Flow Action
-------------------------------------

The easiest way to enable transcription is by adding a ``transcribe_start`` action to your call flow. This automatically begins transcription when the call reaches that action.

**Create Call with Automatic Transcription:**

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "+15551234567"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "+15559876543"
                }
            ],
            "actions": [
                {
                    "type": "answer"
                },
                {
                    "type": "transcribe_start",
                    "option": {
                        "language": "en-US"
                    }
                },
                {
                    "type": "talk",
                    "option": {
                        "text": "This call is being transcribed for quality assurance",
                        "language": "en-US"
                    }
                }
            ]
        }'

Transcription starts when the call reaches the ``transcribe_start`` action and continues until the call ends.

Start Transcription via API (Manual)
-------------------------------------

For existing calls or conferences, start transcription manually by making an API request.

.. note:: **Optional parameter:** ``on_end_flow_id``

   You may include an optional ``on_end_flow_id`` (a flow UUID) in the request body. When the transcription ends, VoIPBIN executes that flow, which is useful for post-transcription processing (for example, running an AI summary flow). If omitted, no follow-up flow is triggered. The ``reference_type``, ``reference_id``, and ``language`` fields are required.

**Transcribe an Active Call:**

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/transcribes?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "reference_type": "call",
            "reference_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
            "language": "en-US",
            "direction": "both"
        }'

    {
        "id": "8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
        "customer_id": "12345678-1234-1234-1234-123456789012",
        "reference_type": "call",
        "reference_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "language": "en-US",
        "direction": "both",
        "status": "progressing",
        "tm_create": "2026-01-20 12:00:00.000000",
        "tm_update": "2026-01-20 12:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

**Transcribe a Conference:**

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/transcribes?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "reference_type": "confbridge",
            "reference_id": "c1d2e3f4-a5b6-7890-cdef-123456789abc",
            "language": "en-US",
            "direction": "both"
        }'

**Transcribe a Recording:**

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/transcribes?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "reference_type": "recording",
            "reference_id": "r1s2t3u4-v5w6-x789-yz01-234567890def",
            "language": "en-US",
            "direction": "both"
        }'

Get Transcription Results
--------------------------

Retrieve transcription data after the transcription completes or during real-time transcription.

**Get Transcription by ID:**

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/transcribes/8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
        "customer_id": "12345678-1234-1234-1234-123456789012",
        "reference_type": "call",
        "reference_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "language": "en-US",
        "direction": "both",
        "status": "done",
        "tm_create": "2026-01-20 12:00:00.000000",
        "tm_update": "2026-01-20 12:05:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

**Get Transcripts (Text Results):**

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/transcripts?token=<YOUR_AUTH_TOKEN>&transcribe_id=8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce'

    {
        "result": [
            {
                "id": "06af78f0-b063-48c0-b22d-d31a5af0aa88",
                "transcribe_id": "8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
                "direction": "in",
                "message": "Hi, good to see you. How are you today?",
                "tm_transcript": "0001-01-01 00:01:04.441160",
                "tm_create": "2024-04-01 07:22:07.229309"
            },
            {
                "id": "3c95ea10-a5b7-4a68-aebf-ed1903baf110",
                "transcribe_id": "8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
                "direction": "out",
                "message": "Welcome to the transcribe test. All your voice will be transcribed.",
                "tm_transcript": "0001-01-01 00:00:43.116830",
                "tm_create": "2024-04-01 07:17:27.208337"
            }
        ]
    }

**Find All Transcribes for a Call:**

If you only know the call ID (for example, when linking from a CRM interaction
record) and not the transcribe ID, filter ``GET /transcribes`` by
``reference_type`` and ``reference_id`` instead of calling ``GET
/transcribes/{id}``. Both parameters must be supplied together; supplying only
one returns a ``400`` error.

.. note:: **AI Implementation Hint**

   A single call can have more than one transcribe: one per language when
   multiple languages are captured, or multiple sessions if transcription was
   stopped and started again. This filter always returns a list — do not
   assume a single result.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/transcribes?token=<YOUR_AUTH_TOKEN>&reference_type=call&reference_id=a1b2c3d4-e5f6-7890-abcd-ef1234567890'

    {
        "result": [
            {
                "id": "8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
                "customer_id": "12345678-1234-1234-1234-123456789012",
                "reference_type": "call",
                "reference_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
                "language": "en-US",
                "direction": "both",
                "status": "done",
                "tm_create": "2026-01-20 12:00:00.000000",
                "tm_update": "2026-01-20 12:05:00.000000",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": ""
    }

An empty ``result`` array means no transcription has been started for that
call yet.

Understanding Transcription Direction
--------------------------------------

VoIPBIN distinguishes between incoming and outgoing audio:

**Direction: "in"** - Audio from the customer/caller to VoIPBIN

**Direction: "out"** - Audio from VoIPBIN to the customer/caller

.. code::

    Customer  -----"in"------>  VoIPBIN
             <----"out"-------

This helps identify who said what in the conversation:
- **"in"**: What the customer said
- **"out"**: What VoIPBIN played (TTS, recordings, or other party in the call)

Real-Time Transcription with WebSocket
---------------------------------------

Subscribe to real-time transcription events via WebSocket to get transcripts as they're generated during the call.

**1. Connect to WebSocket:**

.. code::

    wss://api.voipbin.net/v1.0/ws?token=<YOUR_AUTH_TOKEN>

**2. Subscribe to Transcription Events:**

Topics follow the pattern ``customer_id:<your-customer-id>:<resource>:<resource-id>`` and are matched by **prefix**. Transcribe session events (``transcribe_created``, ``transcribe_progressing``, ``transcribe_done``, ``transcribe_deleted``) are published under the ``transcribe`` resource, while individual transcript segments (``transcript_created``, ``transcript_deleted``) are published under the ``transcript`` resource. To receive transcript segments in real time, subscribe to the ``transcript`` resource. Subscribe by prefix (omit the trailing resource ID) to receive events for all resources of that type under your customer account.

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:12345678-1234-1234-1234-123456789012:transcript:",
            "customer_id:12345678-1234-1234-1234-123456789012:transcribe:"
        ]
    }

.. note::

   Topics are matched by prefix, not by glob. A trailing ``:`` (for example ``...:transcript:``) subscribes to every transcript event for your customer. To target a single resource, append its ID (for example ``...:transcribe:8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce``). Do not use ``*`` as a wildcard.

**3. Receive Real-Time Transcripts:**

Real-time events delivered over the WebSocket use the same envelope as webhooks: a ``type`` field (the event type) and a ``data`` field (the resource payload).

.. code::

    {
        "type": "transcript_created",
        "data": {
            "id": "9d59e7f0-7bdc-4c52-bb8c-bab718952050",
            "transcribe_id": "8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
            "direction": "out",
            "message": "Hello, this is a transcribe test call.",
            "tm_transcript": "0001-01-01 00:00:08.991840",
            "tm_create": "2024-04-04 07:15:59.233415"
        }
    }

**Python WebSocket Example:**

.. code::

    import websocket
    import json

    def on_message(ws, message):
        data = json.loads(message)

        if data.get('type') == 'transcript_created':
            transcript = data['data']
            direction = transcript['direction']
            text = transcript['message']

            print(f"[{direction}] {text}")

            # Process transcription in real-time
            # - Display in UI
            # - Run sentiment analysis
            # - Detect keywords

    def on_open(ws):
        # Subscribe to transcription events (prefix match, no wildcard)
        subscription = {
            "type": "subscribe",
            "topics": [
                "customer_id:12345678-1234-1234-1234-123456789012:transcript:",
                "customer_id:12345678-1234-1234-1234-123456789012:transcribe:"
            ]
        }
        ws.send(json.dumps(subscription))
        print("Subscribed to transcription events")

    token = "<YOUR_AUTH_TOKEN>"
    ws_url = f"wss://api.voipbin.net/v1.0/ws?token={token}"

    ws = websocket.WebSocketApp(
        ws_url,
        on_open=on_open,
        on_message=on_message
    )

    ws.run_forever()

Receive Transcripts via Webhook
--------------------------------

Webhooks are configured at the **customer account level**, not as separate resources. There is no ``/webhooks`` CRUD endpoint. You set a single delivery URL on your customer profile, and VoIPBIN sends **all** event types (including transcribe and transcript events) to that URL. For the full configuration guide, see :ref:`Webhook Tutorial <webhook-tutorial>`.

**1. Configure Your Webhook Endpoint:**

Set the ``webhook_uri`` and ``webhook_method`` on your customer profile.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/customer?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "webhook_uri": "https://your-server.com/voipbin/webhook",
            "webhook_method": "POST"
        }'

**2. Transcription-Related Event Types:**

The following event types are delivered to your webhook endpoint:

========================== ======================================================
Event type                 Description
========================== ======================================================
``transcribe_created``     A transcription session was created
``transcribe_progressing`` A transcription session started progressing
``transcribe_done``        A transcription session finished
``transcribe_deleted``     A transcription session was deleted
``transcript_created``     A new transcript (text segment) was generated
``transcript_deleted``     A transcript was deleted
========================== ======================================================

**3. Webhook Payload Example:**

Every webhook is a JSON object with a common two-field envelope: ``type`` (the event type) and ``data`` (the resource payload).

.. code::

    POST https://your-server.com/voipbin/webhook

    {
        "type": "transcript_created",
        "data": {
            "id": "9d59e7f0-7bdc-4c52-bb8c-bab718952050",
            "transcribe_id": "8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
            "direction": "in",
            "message": "I need help with my account",
            "tm_transcript": "0001-01-01 00:00:15.500000",
            "tm_create": "2024-04-04 07:16:05.100000"
        }
    }

**4. Process Webhook in Your Server:**

.. code::

    # Python Flask example
    from flask import Flask, request, jsonify

    app = Flask(__name__)

    @app.route('/voipbin/webhook', methods=['POST'])
    def transcription_webhook():
        payload = request.get_json()
        event_type = payload.get('type')

        if event_type == 'transcript_created':
            transcript = payload['data']
            transcribe_id = transcript['transcribe_id']
            message = transcript['message']
            direction = transcript['direction']

            # Store transcript in database
            store_transcript(transcribe_id, message, direction)

            # Analyze content
            sentiment = analyze_sentiment(message)
            keywords = extract_keywords(message)

            # Trigger actions based on content
            if 'urgent' in message.lower():
                alert_supervisor(transcribe_id)

        return jsonify({'status': 'received'}), 200

Supported Languages
-------------------

VoIPBIN supports transcription in multiple languages. See :ref:`supported languages <transcribe-overview-supported_languages>`.

**Common Languages:**
- ``en-US`` - English (United States)
- ``en-GB`` - English (United Kingdom)
- ``es-ES`` - Spanish (Spain)
- ``fr-FR`` - French (France)
- ``de-DE`` - German (Germany)
- ``ja-JP`` - Japanese (Japan)
- ``ko-KR`` - Korean (Korea)
- ``zh-CN`` - Chinese (Simplified)

**Example with Different Language:**

.. code::

    {
        "type": "transcribe_start",
        "option": {
            "language": "ja-JP"
        }
    }

Common Use Cases
----------------

**1. Customer Service Quality Assurance:**

.. code::

    # Monitor customer service calls
    def on_transcript(transcript):
        # Check for quality metrics
        if contains_greeting(transcript):
            mark_greeting_present()

        if contains_problem_resolution(transcript):
            mark_resolved()

        # Flag negative sentiment
        if analyze_sentiment(transcript) < 0.3:
            flag_for_review()

**2. Compliance and Record-Keeping:**

.. code::

    # Store all call transcripts for compliance
    def store_for_compliance(transcribe_id):
        transcripts = get_transcripts(transcribe_id)

        # Create formatted record
        record = {
            'call_id': call_id,
            'date': datetime.now(),
            'full_transcript': format_transcript(transcripts),
            'participants': get_participants(call_id)
        }

        # Store in compliance database
        compliance_db.store(record)

**3. Real-Time Agent Assistance:**

.. code::

    # Help agents during calls
    def on_real_time_transcript(transcript):
        # Detect customer questions
        if is_question(transcript['message']):
            # Suggest answers to agent
            answers = knowledge_base.search(transcript['message'])
            display_to_agent(answers)

        # Detect customer frustration
        if detect_frustration(transcript['message']):
            suggest_supervisor_escalation()

**4. Automated Call Summarization:**

.. code::

    # Generate call summaries
    def summarize_call(transcribe_id):
        transcripts = get_all_transcripts(transcribe_id)

        # Combine all transcripts
        full_text = ' '.join([t['message'] for t in transcripts])

        # Generate summary using AI
        summary = ai_summarize(full_text)

        # Extract key points
        action_items = extract_action_items(full_text)
        topics = extract_topics(full_text)

        return {
            'summary': summary,
            'action_items': action_items,
            'topics': topics
        }

**5. Keyword Detection and Alerting:**

.. code::

    # Monitor for important keywords
    ALERT_KEYWORDS = ['urgent', 'emergency', 'cancel', 'complaint', 'lawsuit']

    def on_transcript(transcript):
        message = transcript['message'].lower()

        for keyword in ALERT_KEYWORDS:
            if keyword in message:
                # Send immediate alert
                send_alert(
                    transcribe_id=transcript['transcribe_id'],
                    keyword=keyword,
                    context=message
                )

                # Escalate to supervisor
                escalate_call(transcript['transcribe_id'])

**6. Multi-Language Customer Support:**

VoIPBIN does not auto-detect the spoken language. The ``language`` parameter is required, so determine the language out of band (for example, from the customer's account profile, IVR menu selection, or the dialed number) and pass the corresponding BCP47 code when starting transcription.

.. code::

    # Transcribe in the customer's known language
    def start_multilingual_transcription(call_id, customer):
        # Resolve language from your own data (profile, IVR selection, etc.)
        language = customer.preferred_language  # e.g. "es-ES", "ko-KR"

        # Start transcription in the resolved language
        start_transcribe(
            resource_id=call_id,
            language=language
        )

Best Practices
--------------

**1. Choose the Right Trigger Method:**
- **Flow Action**: Use when transcription is always needed for specific flows
- **Manual API**: Use when transcription is conditional or triggered by user action

**2. Handle Real-Time Events Efficiently:**
- Process transcripts asynchronously to avoid blocking
- Buffer transcripts if processing takes time
- Use queues for high-volume scenarios

**3. Language Selection:**
- Set the correct ``language`` explicitly (there is no auto-detect mode)
- Resolve the language from your own data (customer profile, IVR selection, dialed number)
- Test with actual customer accents and dialects

**4. Data Management:**
- Store transcripts separately from call records
- Implement retention policies (GDPR, compliance)
- Encrypt sensitive transcriptions

**5. Error Handling:**
- Handle cases where transcription fails
- Retry logic for temporary failures
- Log failures for debugging

**6. Testing:**
- Test with various audio qualities
- Verify accuracy with different accents
- Test real-time latency

Transcription Lifecycle
-----------------------

**1. Start Transcription:**

.. code::

    POST /v1.0/transcribes
    → Returns transcribe_id

**2. Active Transcription:**

.. code::

    Status: "progressing"
    → Transcripts being generated in real-time

**3. Receive Transcripts:**

.. code::

    Via WebSocket: transcript_created events
    Via Webhook: POST to your endpoint
    Via API: GET /v1.0/transcripts?transcribe_id=...

**4. Completion:**

.. code::

    Status: "done"
    → All transcripts available via API

Troubleshooting
---------------

**Common Issues:**

**No transcripts generated:**
- Verify call has audio
- Check language setting is correct
- Ensure transcription started successfully

**Poor transcription accuracy:**
- Use correct language code
- Check audio quality
- Verify clear speech (no background noise)

**Missing real-time events:**
- Verify WebSocket subscription is active
- Check topic pattern matches transcribe_id
- Ensure network connection is stable

**Delayed transcripts:**
- Real-time transcription has ~2-5 second delay (normal)
- Check network latency
- Verify server can handle webhook volume

For more information about transcription features and configuration, see :ref:`Transcribe Overview <transcribe-overview>`.
