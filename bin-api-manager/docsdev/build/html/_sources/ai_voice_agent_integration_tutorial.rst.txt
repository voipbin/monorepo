.. _ai-voice-agent-integration-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before building a custom AI voice agent, you need:

* An authentication token (String). Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* A source phone number in E.164 format (e.g., ``+15551234567``). Obtain one via ``GET /numbers``.
* A destination phone number or extension to call.
* A webhook URL or WebSocket connection to receive ``transcript_created`` events. Configure your webhook URL via ``PUT https://api.voipbin.net/v1.0/customer``.
* An AI backend (LLM, RAG, or NLP system) that can receive text and return a response.

.. note:: **AI Implementation Hint**

   The call must reach ``progressing`` status (answered, audio flowing) before you can start transcription or speaking sessions. If the call is still in ``dialing`` or ``ringing`` status, the API will reject the request.


Step 1: Create an Outbound Call
-------------------------------

Create a call to the destination. The call must be answered before starting STT/TTS.

**Request:**

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
            "flow_id": "00000000-0000-0000-0000-000000000000"
        }'

**Response (201 Created):**

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "status": "dialing",
        ...
    }

Save the ``id`` value — this is your ``call_id`` (UUID). Wait for the call to reach ``progressing`` status by listening for a ``call_progressing`` webhook event, or poll ``GET /calls/{call_id}`` until ``status`` is ``progressing``.


Step 2: Start Transcription
----------------------------

Once the call is in ``progressing`` status, start a transcription session.

**Request:**

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/transcribes?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "reference_type": "call",
            "reference_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
            "language": "en-US",
            "direction": "both"
        }'

**Response (200 OK):**

.. code::

    {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "customer_id": "7c4d2f3a-1b8e-4f5c-9a6d-3e2f1a0b4c5d",
        "reference_type": "call",
        "reference_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "status": "progressing",
        "language": "en-US",
        "direction": "both",
        "tm_create": "2026-01-15T09:30:00.000000Z",
        "tm_update": "2026-01-15T09:30:00.000000Z",
        "tm_delete": null
    }

Save the ``id`` value — this is your ``transcribe_id`` (UUID).


Step 3: Create a Speaking Session
----------------------------------

Create a speaking session on the same call so you can inject TTS audio.

**Request:**

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/speakings?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "reference_type": "call",
            "reference_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
            "language": "en-US",
            "provider": "elevenlabs",
            "voice_id": "21m00Tcm4TlvDq8ikWAM",
            "direction": "both"
        }'

**Response (201 Created):**

.. code::

    {
        "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "customer_id": "7c4d2f3a-1b8e-4f5c-9a6d-3e2f1a0b4c5d",
        "reference_type": "call",
        "reference_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "language": "en-US",
        "provider": "elevenlabs",
        "voice_id": "21m00Tcm4TlvDq8ikWAM",
        "direction": "both",
        "status": "initiating",
        "tm_create": "2026-01-15T09:30:01.000000Z",
        "tm_update": "2026-01-15T09:30:01.000000Z",
        "tm_delete": null
    }

Save the ``id`` value — this is your ``speaking_id`` (UUID). Wait for the ``speaking_started`` webhook event or poll ``GET /speakings/{speaking_id}`` until ``status`` is ``active`` before sending text.


Step 4: Send an Initial Greeting
----------------------------------

Once the speaking session is ``active``, send an initial greeting to the caller.

**Request:**

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/speakings/b2c3d4e5-f6a7-8901-bcde-f12345678901/say?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "text": "Hello, thank you for calling. How can I help you today?"
        }'

**Response (200 OK):**

.. code::

    {
        "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "status": "active",
        ...
    }

The caller now hears the greeting. VoIPBIN synthesizes the text using the configured TTS provider and voice.


Step 5: Receive and Process Transcripts
-----------------------------------------

When the caller responds, VoIPBIN delivers a ``transcript_created`` event to your webhook URL.

**Webhook Payload:**

.. code::

    {
        "type": "transcript_created",
        "data": {
            "id": "9d59e7f0-7bdc-4c52-bb8c-bab718952050",
            "transcribe_id": "550e8400-e29b-41d4-a716-446655440000",
            "direction": "in",
            "message": "Hi, I need help with my account balance.",
            "tm_transcript": "0001-01-01 00:00:08.991840",
            "tm_create": "2026-01-15T09:30:15.000000Z"
        }
    }

**Process in your AI backend:**

.. code::

    # Python example — webhook handler
    from flask import Flask, request, jsonify
    import requests

    VOIPBIN_TOKEN = "<YOUR_AUTH_TOKEN>"
    SPEAKING_ID = "b2c3d4e5-f6a7-8901-bcde-f12345678901"

    app = Flask(__name__)

    @app.route('/webhook', methods=['POST'])
    def handle_webhook():
        payload = request.get_json()

        if payload.get('type') == 'transcript_created':
            transcript = payload['data']

            # Only process caller's speech (direction: "in")
            if transcript['direction'] == 'in':
                caller_text = transcript['message']

                # Send to your LLM
                ai_response = call_your_llm(caller_text)

                # Send AI response back to the call via Speaking API
                requests.post(
                    f'https://api.voipbin.net/v1.0/speakings/{SPEAKING_ID}/say?token={VOIPBIN_TOKEN}',
                    json={'text': ai_response}
                )

        return jsonify({'status': 'ok'}), 200

    def call_your_llm(text):
        # Replace with your actual LLM call
        # e.g., OpenAI, Anthropic, local model, etc.
        return f"I understand you need help with: {text}"


Step 6: Handle Interruptions
------------------------------

If the caller speaks while TTS is playing, you receive a new ``transcript_created`` event. Flush the current playback before sending the new response.

.. code::

    @app.route('/webhook', methods=['POST'])
    def handle_webhook():
        payload = request.get_json()

        if payload.get('type') == 'transcript_created':
            transcript = payload['data']

            if transcript['direction'] == 'in':
                # Flush current TTS playback
                requests.post(
                    f'https://api.voipbin.net/v1.0/speakings/{SPEAKING_ID}/flush?token={VOIPBIN_TOKEN}'
                )

                # Process and respond
                ai_response = call_your_llm(transcript['message'])
                requests.post(
                    f'https://api.voipbin.net/v1.0/speakings/{SPEAKING_ID}/say?token={VOIPBIN_TOKEN}',
                    json={'text': ai_response}
                )

        return jsonify({'status': 'ok'}), 200


Step 7: Clean Up on Call End
-----------------------------

When the call ends, stop the transcription and speaking sessions to release resources.

Listen for the ``call_hangup`` webhook event and clean up:

.. code::

    if payload.get('type') == 'call_hangup':
        call_id = payload['data']['id']

        # Stop transcription
        requests.post(
            f'https://api.voipbin.net/v1.0/transcribes/{TRANSCRIBE_ID}/stop?token={VOIPBIN_TOKEN}'
        )

        # Stop speaking
        requests.post(
            f'https://api.voipbin.net/v1.0/speakings/{SPEAKING_ID}/stop?token={VOIPBIN_TOKEN}'
        )

.. note:: **AI Implementation Hint**

   Transcription and speaking sessions may stop automatically when the call ends. However, explicitly stopping them ensures immediate resource cleanup and prevents charges for idle sessions.


Complete Python Example
------------------------

This example combines all steps into a complete webhook server that implements a custom AI voice agent.

.. code::

    from flask import Flask, request, jsonify
    import requests
    import os

    app = Flask(__name__)

    VOIPBIN_BASE = "https://api.voipbin.net/v1.0"
    TOKEN = os.environ.get("VOIPBIN_TOKEN")

    # Store session IDs per call (in production, use a database)
    call_sessions = {}

    @app.route('/webhook', methods=['POST'])
    def webhook():
        payload = request.get_json()
        event_type = payload.get('type')

        if event_type == 'call_progressing':
            handle_call_progressing(payload['data'])

        elif event_type == 'transcript_created':
            handle_transcript(payload['data'])

        elif event_type == 'call_hangup':
            handle_call_hangup(payload['data'])

        return jsonify({'status': 'ok'}), 200

    def handle_call_progressing(call):
        """Call answered — start transcription and speaking."""
        call_id = call['id']

        # Start transcription
        resp = requests.post(
            f'{VOIPBIN_BASE}/transcribes?token={TOKEN}',
            json={
                'reference_type': 'call',
                'reference_id': call_id,
                'language': 'en-US',
                'direction': 'both'
            }
        )
        transcribe_id = resp.json()['id']

        # Create speaking session
        resp = requests.post(
            f'{VOIPBIN_BASE}/speakings?token={TOKEN}',
            json={
                'reference_type': 'call',
                'reference_id': call_id,
                'language': 'en-US',
                'provider': 'elevenlabs',
                'direction': 'both'
            }
        )
        speaking_id = resp.json()['id']

        call_sessions[call_id] = {
            'transcribe_id': transcribe_id,
            'speaking_id': speaking_id
        }

        # Send initial greeting (after short delay for session to become active)
        requests.post(
            f'{VOIPBIN_BASE}/speakings/{speaking_id}/say?token={TOKEN}',
            json={'text': 'Hello, how can I help you today?'}
        )

    def handle_transcript(transcript):
        """Process caller speech and respond."""
        if transcript['direction'] != 'in':
            return

        # Find the speaking session for this transcribe
        transcribe_id = transcript['transcribe_id']
        session = next(
            (s for s in call_sessions.values() if s['transcribe_id'] == transcribe_id),
            None
        )
        if not session:
            return

        speaking_id = session['speaking_id']

        # Flush any current playback
        requests.post(
            f'{VOIPBIN_BASE}/speakings/{speaking_id}/flush?token={TOKEN}'
        )

        # Generate AI response
        ai_response = call_your_llm(transcript['message'])

        # Send response to caller
        requests.post(
            f'{VOIPBIN_BASE}/speakings/{speaking_id}/say?token={TOKEN}',
            json={'text': ai_response}
        )

    def handle_call_hangup(call):
        """Clean up sessions when call ends."""
        call_id = call['id']
        session = call_sessions.pop(call_id, None)
        if session:
            requests.post(
                f'{VOIPBIN_BASE}/transcribes/{session["transcribe_id"]}/stop?token={TOKEN}'
            )
            requests.post(
                f'{VOIPBIN_BASE}/speakings/{session["speaking_id"]}/stop?token={TOKEN}'
            )

    def call_your_llm(text):
        """Replace with your actual LLM integration."""
        # Example: OpenAI, Anthropic, local model, etc.
        return f"I understand you said: {text}. Let me help you with that."

    if __name__ == '__main__':
        app.run(host='0.0.0.0', port=8080)
