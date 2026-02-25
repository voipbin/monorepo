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
| language         | (Optional, String) BCP47 language code (e.g., ``en-US``).    |
|                  | Defaults to provider default.                                |
+------------------+--------------------------------------------------------------+
| provider         | (Optional, String) TTS provider. Defaults to ``elevenlabs``. |
+------------------+--------------------------------------------------------------+
| voice_id         | (Optional, String) Provider-specific voice ID (e.g.,         |
|                  | ``21m00Tcm4TlvDq8ikWAM`` for ElevenLabs).                    |
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


Receiving Events
----------------

VoIPBIN delivers events to your backend through two methods: **webhooks** (push-based HTTP POST) and **WebSocket** (persistent bidirectional connection). You must configure at least one before starting the agent loop, since transcript and call events drive the conversation cycle.

**Webhook Delivery**

Webhooks push events to an HTTPS endpoint you register via ``PUT https://api.voipbin.net/v1.0/customer``. VoIPBIN sends an HTTP POST with the event payload each time a matching event occurs. Your endpoint must respond with HTTP ``200`` within 5 seconds or delivery may be retried.

Key event types for AI voice agent integration:

+---------------------------+--------------------------------------------------------------+
| Event Type                | Description                                                  |
+===========================+==============================================================+
| ``transcript_created``    | New transcript from the caller or TTS output                 |
+---------------------------+--------------------------------------------------------------+
| ``speaking_started``      | Speaking session is active and ready for ``/say``            |
+---------------------------+--------------------------------------------------------------+
| ``speaking_stopped``      | Speaking session has been terminated                         |
+---------------------------+--------------------------------------------------------------+
| ``call_progressing``      | Call answered — audio flowing, safe to start sessions        |
+---------------------------+--------------------------------------------------------------+
| ``call_hangup``           | Call ended — clean up transcribe and speaking sessions       |
+---------------------------+--------------------------------------------------------------+

.. note:: **AI Implementation Hint**

   Implement idempotent processing using the resource ``id`` and ``status`` fields, because VoIPBIN may retry delivery if your endpoint does not respond in time. See :ref:`Webhook Overview <webhook-overview>` for full configuration details.

**WebSocket Delivery**

WebSocket maintains a persistent connection for instant event delivery. Connect to ``wss://api.voipbin.net/v1.0/ws?token=<token>`` using the same JWT or access key token used for REST API calls. After connecting, send a subscribe message to start receiving events.

Subscribe to transcription and call events for your customer:

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:<your-customer-id>:transcription:*",
            "customer_id:<your-customer-id>:call:*"
        ]
    }

Events arrive as JSON messages on the open connection. No polling required.

.. note:: **AI Implementation Hint**

   Always implement automatic reconnection with exponential backoff (start at 1 second, cap at 30 seconds). When the connection drops, all subscriptions are lost and must be re-sent after reconnecting. See :ref:`WebSocket Overview <websocket_overview>` for the full topic format and subscription lifecycle.

**Choosing a Delivery Method**

+------------------+--------------------------------------+--------------------------------------+
| Aspect           | Webhook                              | WebSocket                            |
+==================+======================================+======================================+
| Latency          | Higher (HTTP round-trip per event)   | Lower (persistent connection)        |
+------------------+--------------------------------------+--------------------------------------+
| Connection model | Stateless — VoIPBIN POSTs to your    | Stateful — your client holds an open |
|                  | endpoint                             | connection to VoIPBIN                |
+------------------+--------------------------------------+--------------------------------------+
| Best for         | Serverless backends, simple setups,  | Real-time voice agents, low-latency  |
|                  | multi-region redundancy              | event loops, interactive dashboards  |
+------------------+--------------------------------------+--------------------------------------+

For real-time AI voice agent scenarios where latency directly impacts conversation quality, **WebSocket is recommended**. The persistent connection eliminates per-event HTTP overhead and delivers transcripts faster to your AI backend.


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
- :ref:`WebSocket Overview <websocket_overview>` — Real-time event delivery via WebSocket
