.. _quickstart_realtime:

Real-Time Voice Interaction
===========================
This scenario walks through creating a SIP extension, registering a softphone, making a call with live transcription, and speaking into the call using the real-time TTS API.

By the end, you will have:

- A SIP extension registered with a softphone (Linphone)
- A live call with real-time speech-to-text transcription
- Real-time text-to-speech injected into the call via the Speaking API

Prerequisites
+++++++++++++

* A valid authentication token (String) or accesskey (String). See :ref:`Authentication <quickstart_authentication>`.
* A source phone number in E.164 format (e.g., ``+15551234567``). Must be a number owned by your VoIPBIN account. Obtain available numbers via ``GET /numbers``.
* Your customer ID (UUID). Obtained from ``GET /customers`` or from your admin console profile.
* Linphone softphone installed on your computer or mobile device. Download from `linphone.org <https://www.linphone.org/>`_.

.. note:: **AI Implementation Hint**

   This scenario requires a real SIP phone (Linphone) to answer the call and speak. AI agents cannot complete this scenario fully autonomously — the SIP registration and call answering steps require a human with a softphone. AI agents can execute all API calls (Steps 1, 3, 4, 6, 7) and instruct the human for Steps 2 and 5.

Step 1: Create an extension
----------------------------
Create a SIP extension that your softphone will register to. The ``name`` (String, Required) identifies the extension. The ``username`` (String, Required) and ``password`` (String, Required) are used for SIP authentication.

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/extensions?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "quickstart-phone",
            "detail": "Quickstart softphone extension",
            "username": "quickstart1",
            "password": "your-secure-password-here"
        }'

Response:

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "quickstart-phone",
        "detail": "Quickstart softphone extension",
        "username": "quickstart1",
        "tm_create": "2026-02-21T10:00:00.000000Z",
        "tm_update": "",
        "tm_delete": ""
    }

The ``id`` (UUID) is the extension's unique identifier — use it for ``GET /extensions/{id}``, ``PUT /extensions/{id}``, or ``DELETE /extensions/{id}`` operations. For dialing, use the ``name`` field instead. Save the ``name`` (String) — you will use it as the call destination in Step 4.

.. note:: **AI Implementation Hint**

   The ``username`` and ``password`` are SIP credentials, not VoIPBIN login credentials. The ``name`` field is the extension identifier used when dialing (e.g., ``"target_name": "quickstart-phone"`` in the call request). Choose a memorable ``username`` and a strong ``password``.

Step 2: Register Linphone
--------------------------
Configure your Linphone softphone to register with VoIPBIN using the extension credentials from Step 1.

**Linphone configuration:**

+-------------------+------------------------------------------------------------+
| Field             | Value                                                      |
+===================+============================================================+
| Username          | ``quickstart1`` (from Step 1 ``username``)                 |
+-------------------+------------------------------------------------------------+
| Password          | The password you set in Step 1                             |
+-------------------+------------------------------------------------------------+
| Domain            | ``<your-customer-id>.registrar.voipbin.net``               |
+-------------------+------------------------------------------------------------+
| Transport         | UDP                                                        |
+-------------------+------------------------------------------------------------+

Replace ``<your-customer-id>`` with your customer ID (UUID) obtained from ``GET /customers``. For example, if your customer ID is ``550e8400-e29b-41d4-a716-446655440000``, the domain is ``550e8400-e29b-41d4-a716-446655440000.registrar.voipbin.net``.

**Setup steps (Linphone desktop):**

1. Open Linphone and go to **Preferences** > **Account** (or **SIP Account** on mobile).
2. Select **I already have a SIP account** (or **Use SIP account**).
3. Enter the username, password, and domain from the table above.
4. Save. Linphone should show **Registered** status within a few seconds.

If registration succeeds, the status indicator turns green. If it fails, see Troubleshooting below.

Step 3: Subscribe to events via WebSocket
------------------------------------------
Before making the call, connect to the VoIPBIN WebSocket to receive real-time transcription and call events.

**Connect:**

.. code::

    wss://api.voipbin.net/v1.0/ws?token=<your-token>

**Subscribe** by sending this JSON message after connecting. Replace ``<your-customer-id>`` with your customer ID (UUID) obtained from ``GET /customers``:

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:<your-customer-id>:call:*",
            "customer_id:<your-customer-id>:transcribe:*"
        ]
    }

**Python example:**

.. code::

    import websocket
    import json

    token = "<your-token>"
    customer_id = "<your-customer-id>"

    def on_message(ws, message):
        data = json.loads(message)
        event_type = data.get("event_type")
        if event_type:
            if "transcript" in event_type:
                transcript = data["data"]
                direction = transcript.get("direction", "?")
                text = transcript.get("message", "")
                print(f"[TRANSCRIBE {direction}] {text}")
            else:
                print(f"[EVENT] {event_type}")

    def on_open(ws):
        subscription = {
            "type": "subscribe",
            "topics": [
                f"customer_id:{customer_id}:call:*",
                f"customer_id:{customer_id}:transcribe:*"
            ]
        }
        ws.send(json.dumps(subscription))
        print("Subscribed to call and transcribe events. Waiting...")

    ws = websocket.WebSocketApp(
        f"wss://api.voipbin.net/v1.0/ws?token={token}",
        on_open=on_open,
        on_message=on_message
    )
    ws.run_forever()

Step 4: Make a call to the extension
--------------------------------------
With the WebSocket connected and Linphone registered, make an outbound call to the extension. This call starts real-time transcription, plays a TTS greeting, and then sleeps to keep the call alive while you interact.

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

    [
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
    ]

Save the call ``id`` (UUID) from the response — you will need it in Step 6.

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

   The ``source`` number must be a VoIPBIN-owned number (from ``GET /numbers``). The destination ``type`` is ``extension`` (not ``tel``), and ``target_name`` (String) is the extension's ``name`` field from Step 1. The ``sleep`` ``duration`` (Integer, milliseconds) keeps the call alive — ``600000`` = 10 minutes. The ``transcribe_start`` action uses BCP47 language codes (e.g., ``en-US``, ``ko-KR``, ``ja-JP``).

Step 5: Observe real-time transcription
----------------------------------------
After answering the call on Linphone, your WebSocket receives transcription events.

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

If you run the Python WebSocket example from Step 3, you will see output like:

.. code::

    Subscribed to call and transcribe events. Waiting...
    [EVENT] call_progressing
    [TRANSCRIBE out] Hello. This is the VoIPBIN real-time voice interaction test...
    [TRANSCRIBE in] Hi, this is a test of the transcription feature.

Step 6: Create a speaking stream
----------------------------------
While the call is active, you can inject real-time text-to-speech audio using the Speaking API. Create a speaking stream attached to the call.

``POST /speakings`` with:

- ``reference_type`` (String, Required): ``"call"``
- ``reference_id`` (UUID, Required): The call ``id`` from Step 4 response
- ``language`` (String, Optional): BCP47 language code (e.g., ``"en-US"``)
- ``provider`` (String, Optional): TTS provider. Use ``"elevenlabs"`` for high-quality streaming TTS
- ``direction`` (enum String, Optional): Audio direction. One of: ``"in"`` (caller hears, other side does not), ``"out"`` (other side hears, caller does not), ``"both"`` (both sides hear). Use ``"out"`` so the Linphone user hears the TTS.

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/speakings?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "reference_type": "call",
            "reference_id": "<call-id-from-step-4>",
            "language": "en-US",
            "provider": "elevenlabs",
            "direction": "out"
        }'

Response:

.. code::

    {
        "id": "f1e103d2-0429-4170-83b3-e95e29bb0ca8",
        "customer_id": "550e8400-e29b-41d4-a716-446655440000",
        "reference_type": "call",
        "reference_id": "<call-id-from-step-4>",
        "language": "en-US",
        "provider": "elevenlabs",
        "direction": "out",
        "status": "initiating",
        "tm_create": "2026-02-21T10:06:00.000000Z"
    }

Save the speaking ``id`` (UUID) — you will use it in Step 7.

**Speaking status lifecycle** (enum string):

- ``initiating``: The speaking stream is being set up and connecting to the TTS provider.
- ``active``: The TTS provider is connected. You can now call ``POST /speakings/{id}/say`` to speak.
- ``stopped``: The speaking stream has been stopped or the call has ended.

.. note:: **AI Implementation Hint**

   The speaking stream must be created while the call is in ``progressing`` status (answered and audio flowing). If the call has already hung up, the API returns ``400 Bad Request``. The ``direction`` field controls which side of the call hears the TTS: ``"out"`` means the called party (Linphone) hears it. The ``status`` transitions from ``initiating`` to ``active`` once the TTS provider connects.

Step 7: Speak via TTS API
---------------------------
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

   You can call ``POST /speakings/{id}/say`` multiple times. Each call queues text for sequential playback. If you need to interrupt, call ``POST /speakings/{id}/flush`` first, then ``POST /speakings/{id}/say`` with new text. The ``text`` field has a 5000-character limit per request. Since transcription is still active (from Step 4), the TTS output will also appear in transcription events as ``direction: "out"``.

Troubleshooting
+++++++++++++++

* **Extension creation returns 400 Bad Request:**
    * **Cause:** Missing required fields (``name``, ``username``, ``password``).
    * **Fix:** Ensure all three fields are present in the request body.

* **Linphone shows "Registration failed" or "408 Timeout":**
    * **Cause:** Incorrect domain, username, or password. The domain must include your customer ID.
    * **Fix:** Verify the domain is ``<your-customer-id>.registrar.voipbin.net``. Double-check the ``username`` and ``password`` match exactly what was set in Step 1. Ensure UDP port 5060 is not blocked by your firewall.

* **Call created but Linphone does not ring:**
    * **Cause:** Linphone is not registered, or the ``target_name`` does not match the extension ``name``.
    * **Fix:** Verify Linphone shows "Registered" status. Verify the ``target_name`` in the call request matches the extension ``name`` from Step 1 exactly (case-sensitive).

* **No transcription events in WebSocket:**
    * **Cause:** WebSocket subscription topic does not match your customer ID, or subscription was sent before the connection opened.
    * **Fix:** Verify the customer ID in the topic matches your account (from ``GET /customers``). Send the subscribe message only after the ``on_open`` callback fires.

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
