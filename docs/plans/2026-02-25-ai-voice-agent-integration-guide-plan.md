# AI Voice Agent Integration Guide Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add RST documentation for customers building custom AI voice agents using VoIPBin's individual `/speakings` (TTS) and `/transcribes` (STT) APIs with their own AI backend.

**Architecture:** Three RST files (index + overview + tutorial) added to `bin-api-manager/docsdev/source/`, registered in `index.rst` under "AI & Automation". Follows existing documentation patterns (see `transcribe.rst`, `transcribe_overview.rst`, `transcribe_tutorial.rst` for reference).

**Tech Stack:** RST (reStructuredText), Sphinx documentation builder

**Working directory:** `~/gitvoipbin/monorepo-worktrees/NOJIRA-add-ai-voice-agent-integration-guide/`

---

### Task 1: Create the index file

**Files:**
- Create: `bin-api-manager/docsdev/source/ai_voice_agent_integration.rst`

**Step 1: Write the index RST file**

This file follows the pattern of `transcribe.rst` and `ai.rst`. It includes the overview and tutorial via `.. include::`.

```rst
.. _ai-voice-agent-integration-main:

******************************
AI Voice Agent Integration
******************************
Build custom AI-powered voice agents using VoIPBIN's Speech-to-Text (Transcribe API) and Text-to-Speech (Speaking API) with your own AI backend. This guide covers the architecture and step-by-step integration for customers who manage their own LLM or conversational AI logic.

**API Reference:** `Speaking endpoints <https://api.voipbin.net/redoc/#tag/Speaking>`_ | `Transcribe endpoints <https://api.voipbin.net/redoc/#tag/Transcribe>`_

.. include:: ai_voice_agent_integration_overview.rst
.. include:: ai_voice_agent_integration_tutorial.rst
```

**Step 2: Verify file exists**

Run: `ls -la bin-api-manager/docsdev/source/ai_voice_agent_integration.rst`

---

### Task 2: Create the overview file

**Files:**
- Create: `bin-api-manager/docsdev/source/ai_voice_agent_integration_overview.rst`

**Step 1: Write the overview RST file**

This is the main content file. It must follow the AI-Native RST Writing Guidelines from `bin-api-manager/CLAUDE.md`:
- AI Context block at top
- AI Implementation Hints (at least one)
- Explicit state transitions for enums
- Data provenance for all IDs
- Cause/fix pairs in troubleshooting

Full content below. Key sections:

1. **AI Context block** (mandatory)
2. **What is Custom AI Voice Agent Integration** — explains the "bring your own AI backend" model vs managed `ai_talk`
3. **Architecture** — ASCII diagram showing Caller ↔ VoIPBIN ↔ Customer AI Backend with the 3-step loop
4. **API Components** — Speaking API (POST /speakings, /say, /stop, /flush) and Transcribe API (POST /transcribes, /stop), with cross-refs to existing docs
5. **Integration Workflow** — 6-step numbered cycle
6. **Handling Interruptions (Barge-in)** — flush + new transcript pattern
7. **Custom Integration vs ai_talk** — comparison table
8. **Best Practices** — latency, session lifecycle, error handling
9. **Troubleshooting** — cause/fix pairs per the RST guidelines

```rst
.. _ai-voice-agent-integration-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Medium
   * **Cost:** Chargeable (STT per minute of audio transcribed + TTS per character synthesized)
   * **Async:** Yes. Both ``POST /speakings`` and ``POST /transcribes`` return immediately. Transcripts are delivered asynchronously via webhook (``transcript_created`` events) or WebSocket subscription.

VoIPBIN enables you to build fully custom AI voice agents by combining two independent APIs:

- **Transcribe API** (``/transcribes``): Converts caller speech to text in real-time (STT)
- **Speaking API** (``/speakings``): Converts your AI-generated text to speech and injects it into the call (TTS)

By using these APIs individually, you retain full control over the AI logic — choose any LLM, RAG pipeline, or custom NLP system as your backend. VoIPBIN handles the telecom layer (SIP, RTP, codecs) and the speech processing; your backend handles the intelligence.

.. note:: **AI Implementation Hint**

   This guide describes the **custom integration** approach where you manage your own AI backend. If you want VoIPBIN to manage the entire STT → LLM → TTS pipeline automatically, use the ``ai_talk`` flow action instead. See :ref:`AI Overview <ai-overview>` for the managed approach.


Architecture
------------
The custom AI voice agent architecture follows a three-step loop:

::

    +-----------------+                    +----------------------------+
    |                 | <=== SIP/RTP ====> |                            |
    |  Caller (Phone) |                    |  VoIPBIN (CPaaS)           |
    |                 |                    |  [SIP, RTP, STT, TTS]      |
    +-----------------+                    +----------------------------+
                                               |                  ^
                                               |                  |
                                    (1) transcript_created     (3) POST
                                        webhook/WS            /speakings/{id}/say
                                    [caller's speech as text]  [AI response text]
                                               |                  |
                                               v                  |
                                           +----------------------------+
                                           |                            |
                                           |   Your AI Backend          |
                                           |   [LLM / RAG / NLP]       |
                                           |                            |
                                           |   (2) Process text,        |
                                           |       generate response    |
                                           +----------------------------+

**The Loop:**

1. VoIPBIN transcribes the caller's speech and delivers text to your backend via ``transcript_created`` webhook or WebSocket event
2. Your AI backend processes the text (e.g., sends to an LLM) and generates a response
3. Your backend sends the response text to VoIPBIN via ``POST /speakings/{id}/say``, which synthesizes and plays it to the caller


API Components
--------------

**Speaking API (Text-to-Speech)**

The Speaking API creates a streaming TTS session on an active call or conference. You send text, and VoIPBIN synthesizes it into speech and injects the audio into the call.

+-------------------------------------------+-----------------------------------------------------------+
| Endpoint                                  | Description                                               |
+===========================================+===========================================================+
| ``POST /speakings``                       | Create a new speaking session on a call or conference     |
+-------------------------------------------+-----------------------------------------------------------+
| ``POST /speakings/{id}/say``              | Send text to be spoken. Can be called multiple times.     |
+-------------------------------------------+-----------------------------------------------------------+
| ``POST /speakings/{id}/flush``            | Cancel current speech and clear queued text.              |
|                                           | Session stays open.                                       |
+-------------------------------------------+-----------------------------------------------------------+
| ``POST /speakings/{id}/stop``             | Terminate the speaking session.                           |
+-------------------------------------------+-----------------------------------------------------------+
| ``GET /speakings``                        | List speaking sessions.                                   |
+-------------------------------------------+-----------------------------------------------------------+
| ``GET /speakings/{id}``                   | Get speaking session details.                             |
+-------------------------------------------+-----------------------------------------------------------+

**Speaking Session Lifecycle:**

::

    POST /speakings
         |
         v
    +-------------+     POST /say      +--------+     POST /stop     +---------+
    | initiating  |-------------------->| active |-------------------->| stopped |
    +-------------+                     +--------+                    +---------+
                                            |
                                     POST /flush
                                     (clears queue,
                                      stays active)

**Speaking Status Values:**

+---------------+------------------------------------------------------------------+
| Status        | Description                                                      |
+===============+==================================================================+
| ``initiating``| Session is being created. TTS provider connection in progress.   |
+---------------+------------------------------------------------------------------+
| ``active``    | Session is ready. Text sent via ``/say`` is synthesized and      |
|               | played into the call.                                            |
+---------------+------------------------------------------------------------------+
| ``stopped``   | Session has been terminated. No more text can be sent.           |
+---------------+------------------------------------------------------------------+

**Speaking Parameters:**

+------------------+--------------------------------------------------------------+
| Parameter        | Description                                                  |
+==================+==============================================================+
| reference_type   | (Required, String) Type of resource: ``call`` or             |
|                  | ``confbridge``                                               |
+------------------+--------------------------------------------------------------+
| reference_id     | (Required, UUID) ID of the call or conference. Obtained      |
|                  | from ``POST /calls`` or ``GET /calls``.                      |
+------------------+--------------------------------------------------------------+
| language         | (Optional, String) BCP47 language code (e.g., ``en-US``).   |
|                  | Defaults to provider default.                                |
+------------------+--------------------------------------------------------------+
| provider         | (Optional, String) TTS provider. Defaults to ``elevenlabs``. |
+------------------+--------------------------------------------------------------+
| voice_id         | (Optional, String) Provider-specific voice ID (e.g.,         |
|                  | ``21m00Tcm4TlvDq8ikWAM`` for ElevenLabs).                   |
+------------------+--------------------------------------------------------------+
| direction        | (Optional, String) Audio injection direction: ``in``,        |
|                  | ``out``, ``both``, or empty. See Direction Values below.     |
+------------------+--------------------------------------------------------------+

**Direction Values:**

+---------------+------------------------------------------------------------------+
| Direction     | Meaning                                                          |
+===============+==================================================================+
| ``in``        | Inject audio to the incoming side only (caller hears it)         |
+---------------+------------------------------------------------------------------+
| ``out``       | Inject audio to the outgoing side only (connected party hears)   |
+---------------+------------------------------------------------------------------+
| ``both``      | Inject audio to both sides                                       |
+---------------+------------------------------------------------------------------+
| (empty)       | Default behavior                                                 |
+---------------+------------------------------------------------------------------+

**Speaking Webhook Events:**

+-----------------------+--------------------------------------------------------------+
| Event                 | Description                                                  |
+=======================+==============================================================+
| ``speaking_started``  | Speaking session is active and ready to accept text          |
+-----------------------+--------------------------------------------------------------+
| ``speaking_stopped``  | Speaking session has been terminated                         |
+-----------------------+--------------------------------------------------------------+

**Transcribe API (Speech-to-Text)**

The Transcribe API captures audio from an active call or conference and converts it to text in real-time. For full details, see :ref:`Transcribe Overview <transcribe-overview>`.

Key endpoints for this integration:

+-------------------------------------------+-----------------------------------------------------------+
| Endpoint                                  | Description                                               |
+===========================================+===========================================================+
| ``POST /transcribes``                     | Start a transcription session on a call or conference     |
+-------------------------------------------+-----------------------------------------------------------+
| ``POST /transcribes/{id}/stop``           | Stop a transcription session                              |
+-------------------------------------------+-----------------------------------------------------------+

Transcripts are delivered via ``transcript_created`` webhook events or WebSocket subscription. Each transcript includes:

- ``direction``: ``in`` (caller's speech) or ``out`` (VoIPBIN's speech toward caller)
- ``message``: The transcribed text
- ``transcribe_id``: UUID linking back to the transcription session


Integration Workflow
--------------------
The complete integration involves six steps in a repeating loop:

::

    Step 1          Step 2              Step 3               Step 4
    Create Call --> Start Transcribe --> Receive Transcripts --> AI Processes
         |                                                        |
         |              Step 6                Step 5              |
         +--- Repeat <-- Listen for next <-- Send via /say <------+
                         transcript

**Step 1: Establish a call**

Create an outbound call or receive an inbound call. The call must reach ``progressing`` status (answered, audio flowing) before starting transcription or speaking.

**Step 2: Start transcription**

Create a transcription session on the active call using ``POST /transcribes`` with ``reference_type: "call"`` and ``reference_id`` set to the call UUID.

**Step 3: Receive transcript events**

VoIPBIN delivers ``transcript_created`` events to your webhook URL or WebSocket connection. Each event contains the transcribed text and direction.

**Step 4: Process with your AI backend**

Your backend receives the text, sends it to your LLM or NLP system, and generates a response.

**Step 5: Send AI response as speech**

Send the response text to VoIPBIN via ``POST /speakings/{id}/say``. VoIPBIN synthesizes the text and plays it into the call.

**Step 6: Repeat**

Continue listening for new ``transcript_created`` events and responding. The loop runs until the call ends or you stop the sessions.

.. note:: **AI Implementation Hint**

   Create the speaking session (``POST /speakings``) early — ideally right after the call reaches ``progressing`` status, alongside the transcribe session. The speaking session must be in ``active`` status before you can call ``/say``. Creating it early avoids latency when your AI generates its first response.


Handling Interruptions (Barge-in)
---------------------------------
When the caller speaks while TTS audio is playing, VoIPBIN detects the incoming speech and generates a new ``transcript_created`` event. Your backend should handle this by:

1. Calling ``POST /speakings/{id}/flush`` to stop the current TTS playback and clear queued text
2. Processing the new transcript through your AI
3. Sending the new response via ``POST /speakings/{id}/say``

::

    Your AI Backend                VoIPBIN                    Caller
         |                            |                         |
         | POST /say "Let me check"   |                         |
         +--------------------------->| Playing TTS audio...    |
         |                            |<---"Wait, cancel that"--|
         |                            |                         |
         | transcript_created         |                         |
         |<---------------------------| (caller interrupted)    |
         |                            |                         |
         | POST /flush                |                         |
         +--------------------------->| Stops TTS playback      |
         |                            |                         |
         | POST /say "Sure, what..."  |                         |
         +--------------------------->| Playing new TTS audio   |
         |                            |                         |

.. note:: **AI Implementation Hint**

   Always call ``/flush`` before sending a new response after an interruption. If you skip the flush, the new response is queued after the current playback, and the caller hears both the old and new responses.


Custom Integration vs ai_talk
------------------------------
VoIPBIN offers two approaches for AI voice agents:

+---------------------+----------------------------------+----------------------------------+
| Aspect              | Custom Integration               | ai_talk (Managed)                |
|                     | (this guide)                     |                                  |
+=====================+==================================+==================================+
| AI Backend          | You manage your own LLM          | VoIPBIN manages the LLM          |
|                     | (any provider)                   | (configured via AI resource)     |
+---------------------+----------------------------------+----------------------------------+
| STT/TTS Control     | Individual API calls             | Automatic pipeline               |
|                     | (``/transcribes``, ``/speakings``)| (configured in flow action)     |
+---------------------+----------------------------------+----------------------------------+
| LLM Provider        | Any (OpenAI, Anthropic, local,   | Must be one of the supported     |
|                     | custom, etc.)                    | providers (see AI docs)          |
+---------------------+----------------------------------+----------------------------------+
| Tools/Functions     | Implement your own               | Built-in tools                   |
|                     |                                  | (connect_call, send_email, etc.) |
+---------------------+----------------------------------+----------------------------------+
| Setup Complexity    | Higher (webhook server, loop     | Lower (configure AI resource     |
|                     | management)                      | and flow action)                 |
+---------------------+----------------------------------+----------------------------------+
| Flexibility         | Full control over AI logic,      | Limited to supported providers   |
|                     | prompts, and conversation flow   | and tool set                     |
+---------------------+----------------------------------+----------------------------------+
| Best For            | Custom AI logic, proprietary     | Quick setup, standard            |
|                     | models, complex pipelines        | conversational agents            |
+---------------------+----------------------------------+----------------------------------+

See :ref:`AI Overview <ai-overview>` for full details on the managed ``ai_talk`` approach.


Best Practices
--------------

**1. Minimize Latency**

- Create both transcribe and speaking sessions immediately when the call reaches ``progressing`` status
- Keep your AI backend response time under 2 seconds for natural conversation flow
- Use WebSocket subscription instead of webhooks for lower-latency transcript delivery
- Pre-connect to your LLM with a persistent session if supported

**2. Session Lifecycle Management**

- Create one speaking session per call and reuse it for all responses (call ``/say`` multiple times)
- Stop both transcribe and speaking sessions when the call ends to release resources
- Listen for ``call_hangup`` webhook events to trigger cleanup

**3. Error Handling**

- If ``POST /speakings/{id}/say`` returns an error, the speaking session may have been stopped — create a new one
- If transcript events stop arriving, check that the transcribe session is still in ``progressing`` status via ``GET /transcribes/{id}``
- Implement a timeout: if no transcript arrives within 30 seconds, consider prompting the caller

**4. Audio Direction**

- For most AI agent scenarios, use ``direction: "both"`` for the speaking session so both parties hear the AI
- Use ``direction: "both"`` for the transcribe session to capture the full conversation


Troubleshooting
---------------

* **Speaking session stuck in ``initiating``:**
    * **Cause:** The call is not in ``progressing`` status, or the TTS provider connection failed.
    * **Fix:** Verify the call status via ``GET /calls/{call-id}``. The call must be answered and in ``progressing`` status. Retry creating the speaking session.

* **No transcript events received:**
    * **Cause:** Transcription not started, or webhook URL not configured.
    * **Fix:** Verify the transcribe session exists and is in ``progressing`` status via ``GET /transcribes/{id}``. Check that your webhook URL is configured in customer settings via ``PUT https://api.voipbin.net/v1.0/customer``.

* **TTS audio not playing after ``/say``:**
    * **Cause:** Speaking session is ``stopped`` or ``initiating`` (not yet ``active``).
    * **Fix:** Check speaking session status via ``GET /speakings/{id}``. If ``stopped``, create a new session. If ``initiating``, wait for ``speaking_started`` webhook event before calling ``/say``.

* **Caller hears old and new responses after interruption:**
    * **Cause:** ``/flush`` was not called before sending the new response.
    * **Fix:** Always call ``POST /speakings/{id}/flush`` before ``POST /speakings/{id}/say`` when handling an interruption.

* **High latency in conversation:**
    * **Cause:** Webhook delivery overhead or slow AI backend response.
    * **Fix:** Switch from webhooks to WebSocket subscription for transcript delivery. Optimize your AI backend response time.


Related Documentation
---------------------

- :ref:`Transcribe Overview <transcribe-overview>` — Full STT documentation
- :ref:`AI Overview <ai-overview>` — Managed AI pipeline with ``ai_talk``
- :ref:`Call Overview <call-overview>` — Call lifecycle and status
- :ref:`Webhook Overview <webhook-overview>` — Webhook configuration
```

**Step 2: Verify file exists**

Run: `ls -la bin-api-manager/docsdev/source/ai_voice_agent_integration_overview.rst`

---

### Task 3: Create the tutorial file

**Files:**
- Create: `bin-api-manager/docsdev/source/ai_voice_agent_integration_tutorial.rst`

**Step 1: Write the tutorial RST file**

Must follow tutorial guidelines: prerequisites block, complete request AND response examples, AI Implementation Hints.

```rst
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
```

**Step 2: Verify file exists**

Run: `ls -la bin-api-manager/docsdev/source/ai_voice_agent_integration_tutorial.rst`

---

### Task 4: Update index.rst

**Files:**
- Modify: `bin-api-manager/docsdev/source/index.rst` (line 56, after `ai`)

**Step 1: Add ai_voice_agent_integration to index.rst**

In the "AI & Automation" toctree section, add `ai_voice_agent_integration` after `ai`:

```rst
.. toctree::
   :maxdepth: 5
   :caption: AI & Automation

   ai
   ai_voice_agent_integration
   campaign
   outdial
   outplan
```

**Step 2: Verify change**

Run: `grep -A 6 "AI & Automation" bin-api-manager/docsdev/source/index.rst`

---

### Task 5: Build and verify Sphinx docs

**Step 1: Build the documentation**

Run:
```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```

Expected: Build completes without errors.

**Step 2: Check for warnings**

Review the build output for:
- Broken cross-references (`:ref:` targets that don't exist)
- RST formatting errors
- Missing files

If there are warnings about undefined labels like `ai-overview` or `call-overview`, check exact label names in the referenced files and fix accordingly.

**Step 3: Fix any issues**

If build errors or warnings appear, fix the RST files and rebuild.

---

### Task 6: Commit all changes

**Step 1: Stage files**

Run:
```bash
git add \
    bin-api-manager/docsdev/source/ai_voice_agent_integration.rst \
    bin-api-manager/docsdev/source/ai_voice_agent_integration_overview.rst \
    bin-api-manager/docsdev/source/ai_voice_agent_integration_tutorial.rst \
    bin-api-manager/docsdev/source/index.rst \
    bin-api-manager/docsdev/build/ \
    docs/plans/2026-02-25-ai-voice-agent-integration-guide-design.md \
    docs/plans/2026-02-25-ai-voice-agent-integration-guide-plan.md
```

Note: `git add -f bin-api-manager/docsdev/build/` may be needed because root `.gitignore` excludes `build/`.

**Step 2: Commit**

```bash
git commit -m "NOJIRA-add-ai-voice-agent-integration-guide

Add RST documentation for building custom AI voice agents using VoIPBIN's
individual Speaking (TTS) and Transcribe (STT) APIs with a customer-managed
AI backend.

- bin-api-manager: Add ai_voice_agent_integration overview, tutorial, and index RST files
- bin-api-manager: Register new docs in index.rst under AI & Automation section
- bin-api-manager: Rebuild Sphinx HTML documentation
- docs: Add design doc and implementation plan"
```
