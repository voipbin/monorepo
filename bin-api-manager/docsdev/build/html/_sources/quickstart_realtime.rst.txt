.. _quickstart_realtime:

Real-Time Voice Interaction
---------------------------
This scenario walks through making a call with live transcription and speaking into the call using the real-time TTS API.

By the end, you will have:

- A live call with real-time speech-to-text transcription
- Real-time text-to-speech injected into the call via the Speaking API

Prerequisites
+++++++++++++

* A valid authentication token (String) or accesskey (String). See :ref:`Authentication <quickstart_authentication>`.
* A source phone number in E.164 format (e.g., ``+15551234567``). Must be a number owned by your VoIPBIN account. Obtain available numbers via ``GET /numbers``.
* Your customer ID (UUID). Obtained from ``GET https://api.voipbin.net/v1.0/customer`` or from your admin console profile.
* A registered SIP extension and softphone. See :ref:`Extension & Softphone Setup <quickstart_extension>`.
* Event subscription set up (WebSocket or customer webhook). See :ref:`Receiving Events <quickstart_events>`.

.. note:: **AI Implementation Hint**

   This scenario requires a registered softphone (Linphone) to answer the call and speak. AI agents can execute all API calls and instruct the human for answering the call on Linphone. Set up event delivery before starting — you will need it to observe transcription events during the call.

Make a call to the extension
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
With event subscription configured and Linphone registered, make an outbound call to the extension. This call starts real-time transcription, plays a TTS greeting, and then sleeps to keep the call alive while you interact.

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/calls?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "<your-source-number>"
            },
            "destinations": [
                {
                    "type": "extension",
                    "target_name": "quickstart-phone"
                }
            ],
            "actions": [
                {
                    "type": "transcribe_start",
                    "option": {
                        "language": "en-US"
                    }
                },
                {
                    "type": "talk",
                    "option": {
                        "text": "Hello. This is the VoIPBIN real-time voice interaction test. You can speak now and your speech will be transcribed. The call will stay open for you to test the Speaking API.",
                        "gender": "female",
                        "language": "en-US"
                    }
                },
                {
                    "type": "sleep",
                    "option": {
                        "duration": 600000
                    }
                }
            ]
        }'

Response:

.. code::

    {
        "calls": [
            {
                "id": "e2a65df2-4e50-4e37-8628-df07b3cec579",
                "source": {
                    "type": "tel",
                    "target": "<your-source-number>",
                    "target_name": ""
                },
                "destination": {
                    "type": "extension",
                    "target_name": "quickstart-phone"
                },
                "status": "dialing",
                "direction": "outgoing",
                ...
            }
        ],
        "groupcalls": []
    }

Save the call ``id`` (UUID) from ``calls[0].id`` in the response — you will need it when creating a speaking stream.

**Call status lifecycle** (enum string):

- ``dialing``: The system is currently dialing the destination extension.
- ``ringing``: The destination device (Linphone) is ringing, awaiting answer.
- ``progressing``: The call is answered. Audio is flowing between parties.
- ``terminating``: The system is ending the call.
- ``canceling``: The originator canceled the call before it was answered (outgoing calls only).
- ``hangup``: The call has ended. Final state — no further changes possible.

**What happens next:**

1. Linphone rings. **Answer the call.**
2. The TTS greeting plays (you hear it through Linphone).
3. The call enters the ``sleep`` action (600 seconds = 10 minutes), keeping the call alive.
4. Transcription is active — anything you say into Linphone is transcribed.

.. note:: **AI Implementation Hint**

   The ``source`` number must be a VoIPBIN-owned number (from ``GET /numbers``). The destination ``type`` is ``extension`` (not ``tel``), and ``target_name`` (String) is the extension's ``name`` field from the :ref:`Extension & Softphone Setup <quickstart_extension>`. The ``sleep`` ``duration`` (Integer, milliseconds) keeps the call alive — ``600000`` = 10 minutes. The ``transcribe_start`` action uses BCP47 language codes (e.g., ``en-US``, ``ko-KR``, ``ja-JP``).

Observe real-time transcription
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
After answering the call on Linphone, you receive transcription events via your configured event subscription (WebSocket or customer webhook). The examples below show WebSocket event payloads.

**The TTS greeting appears first** (``direction: "out"`` — VoIPBIN to caller):

.. code::

    {
        "event_type": "transcript_created",
        "timestamp": "2026-02-21T10:05:02.000000Z",
        "topic": "customer_id:<your-customer-id>:transcribe:<transcribe-id>",
        "data": {
            "id": "9d59e7f0-7bdc-4c52-bb8c-bab718952050",
            "transcribe_id": "8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
            "direction": "out",
            "message": "Hello. This is the VoIPBIN real-time voice interaction test. You can speak now and your speech will be transcribed.",
            "tm_create": "2026-02-21T10:05:02.233415Z"
        }
    }

**Transcript event fields:**

- ``data.transcribe_id`` (UUID): The transcription session ID, generated internally when the ``transcribe_start`` action executes. Query all transcripts for this session via ``GET /transcripts?transcribe_id=<transcribe_id>``.
- ``data.direction`` (enum String): ``"in"`` — speech from the caller to VoIPBIN. ``"out"`` — speech from VoIPBIN to the caller (TTS output).
- ``data.message`` (String): The transcribed text.

**When you speak into Linphone**, your speech appears as ``direction: "in"`` (caller to VoIPBIN):

.. code::

    {
        "event_type": "transcript_created",
        "data": {
            "transcribe_id": "8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
            "direction": "in",
            "message": "Hi, this is a test of the transcription feature.",
            "tm_create": "2026-02-21T10:05:15.100000Z"
        }
    }

If you run the Python WebSocket example from :ref:`Receiving Events <quickstart_events>`, you will see output like:

.. code::

    Subscribed to call and transcribe events. Waiting...
    [EVENT] call_progressing
    [TRANSCRIBE out] Hello. This is the VoIPBIN real-time voice interaction test...
    [TRANSCRIBE in] Hi, this is a test of the transcription feature.

Create a speaking stream
~~~~~~~~~~~~~~~~~~~~~~~~~
While the call is active, you can inject real-time text-to-speech audio using the Speaking API. Create a speaking stream attached to the call.

``POST /speakings`` with:

- ``reference_type`` (String, Required): ``"call"``
- ``reference_id`` (UUID, Required): The call ``id`` from the call response above
- ``language`` (String, Optional): BCP47 language code (e.g., ``"en-US"``)
- ``provider`` (String, Optional): TTS provider. Use ``"elevenlabs"`` for high-quality streaming TTS
- ``direction`` (enum String, Optional): Controls which side of the call hears the TTS audio. One of: ``"out"`` (the call's destination hears the TTS), ``"in"`` (the call's source hears the TTS), ``"both"`` (both sides hear). In this scenario, use ``"out"`` so the Linphone user (destination) hears the TTS.

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/speakings?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "reference_type": "call",
            "reference_id": "<call-id>",
            "language": "en-US",
            "provider": "elevenlabs",
            "direction": "out"
        }'

Response (HTTP 201 Created):

.. code::

    {
        "id": "f1e103d2-0429-4170-83b3-e95e29bb0ca8",
        "customer_id": "550e8400-e29b-41d4-a716-446655440000",
        "reference_type": "call",
        "reference_id": "<call-id>",
        "language": "en-US",
        "provider": "elevenlabs",
        "voice_id": "",
        "direction": "out",
        "status": "initiating",
        "pod_id": "",
        "tm_create": "2026-02-21T10:06:00.000000Z",
        "tm_update": "",
        "tm_delete": ""
    }

Save the speaking ``id`` (UUID) — you will use it to send text-to-speech.

**Speaking status lifecycle** (enum string):

- ``initiating``: The speaking stream is being set up and connecting to the TTS provider.
- ``active``: The TTS provider is connected. You can now call ``POST /speakings/{id}/say`` to speak.
- ``stopped``: The speaking stream has been stopped or the call has ended.

.. note:: **AI Implementation Hint**

   The speaking stream must be created while the call is in ``progressing`` status (answered and audio flowing). If the call has already hung up, the API returns ``400 Bad Request``. The ``direction`` field controls which side of the call hears the TTS: ``"out"`` means the called party (Linphone) hears it. The ``status`` transitions from ``initiating`` to ``active`` once the TTS provider connects.

Speak via TTS API
~~~~~~~~~~~~~~~~~~~
Send text to the speaking stream to have it spoken into the call in real time.

``POST /speakings/{id}/say`` with:

- ``text`` (String, Required): The text to speak. Maximum 5000 characters.

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/speakings/<speaking-id>/say?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "text": "Hello, how are you today? This is VoIPBIN speaking to you in real time using the ElevenLabs text-to-speech engine."
        }'

Response:

.. code::

    {
        "id": "f1e103d2-0429-4170-83b3-e95e29bb0ca8",
        "reference_type": "call",
        "reference_id": "<call-id>",
        "language": "en-US",
        "provider": "elevenlabs",
        "direction": "out",
        "status": "active",
        ...
    }

You should hear the text spoken through Linphone within a second or two. You can call ``/say`` multiple times to queue additional speech.

**Additional speaking operations:**

- ``POST /speakings/{id}/flush`` — Clear the speech queue (stop any pending text from being spoken).
- ``POST /speakings/{id}/stop`` — Stop the current speech immediately.
- ``DELETE /speakings/{id}`` — Delete the speaking stream entirely.

.. note:: **AI Implementation Hint**

   You can call ``POST /speakings/{id}/say`` multiple times. Each call queues text for sequential playback. If you need to interrupt, call ``POST /speakings/{id}/flush`` first, then ``POST /speakings/{id}/say`` with new text. The ``text`` field has a 5000-character limit per request. Since transcription is still active, the TTS output will also appear in transcription events as ``direction: "out"``.

Troubleshooting
+++++++++++++++

* **Call created but Linphone does not ring:**
    * **Cause:** Linphone is not registered, or the ``target_name`` does not match the extension ``name``.
    * **Fix:** Verify Linphone shows "Registered" status. Verify the ``target_name`` in the call request matches the extension ``name`` from the :ref:`Extension & Softphone Setup <quickstart_extension>` exactly (case-sensitive). See :ref:`Extension troubleshooting <quickstart_extension>` for registration issues.

* **No transcription events received:**
    * **Cause:** Event subscription is not set up, or configuration is incorrect.
    * **Fix:** See :ref:`Event Subscription troubleshooting <quickstart_event_subscription>`.

* **Speaking creation returns 400 Bad Request:**
    * **Cause:** The call is not in ``progressing`` status (not yet answered or already hung up), or the ``reference_id`` is invalid.
    * **Fix:** Verify the call status via ``GET /calls/<call-id>``. The call must be answered (``status: "progressing"``) before creating a speaking stream.

* **Speaking say returns 400 Bad Request:**
    * **Cause:** The ``text`` field is empty or exceeds 5000 characters, or the speaking stream is no longer active.
    * **Fix:** Verify the text is non-empty and under 5000 characters. Check the speaking status via ``GET /speakings/<speaking-id>``.

* **401 Unauthorized on any API call:**
    * **Cause:** Token has expired (older than 7 days) or is malformed.
    * **Fix:** Generate a new token via ``POST /auth/login``. Ensure the ``Authorization`` header uses the format ``Bearer <token>`` (with a space after "Bearer"), or pass the token as a query parameter ``?token=<token>``.

* **404 Not Found on speaking or call operations:**
    * **Cause:** The call ID or speaking ID is incorrect, or the resource has been deleted.
    * **Fix:** Verify the ID by listing resources via ``GET /calls`` or ``GET /speakings``.

* **One-way audio (can hear TTS but Linphone speech is not transcribed):**
    * **Cause:** NAT or firewall blocking RTP traffic.
    * **Fix:** Enable STUN in Linphone settings (use ``stun.linphone.org``). Ensure your network allows UDP traffic on ports 5060 and 10000-20000.
