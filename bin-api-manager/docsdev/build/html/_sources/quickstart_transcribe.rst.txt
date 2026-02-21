.. _quickstart_transcribe:

Transcribe
==========
Set up real-time speech-to-text transcription on a voice call using the VoIPBIN API. This walkthrough creates a flow that answers a call, starts transcription, speaks a greeting, and pauses — then shows how to receive transcription events via WebSocket and webhook.

Prerequisites
+++++++++++++

* A valid authentication token (String) or accesskey (String). See :ref:`Authentication <quickstart_authentication>`.
* A source phone number in E.164 format (e.g., ``+15551234567``). Must be a number owned by your VoIPBIN account. Obtain available numbers via ``GET /numbers``.
* Your customer ID (UUID). Obtained from ``GET https://api.voipbin.net/v1.0/customer`` or from your admin console profile.

.. note:: **AI Implementation Hint**

   This guide uses virtual numbers (``+899`` prefix) which are free and do not require a provider purchase. The TTS ``talk`` action produces audio that is transcribed as ``direction: "out"`` (VoIPBIN to caller). If you speak into the call, your speech appears as ``direction: "in"`` (caller to VoIPBIN). The ``sleep`` action keeps the call alive for 30 seconds so you can observe transcription events in real time.

Step 1: Create a transcription flow
------------------------------------
Create a flow that answers the call, starts transcription, speaks a greeting, and then pauses for 30 seconds. The ``transcribe_start`` action begins real-time speech-to-text; all subsequent audio (both TTS output and caller speech) is transcribed.

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/flows?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "quickstart transcribe flow",
            "detail": "Answer, start transcription, speak greeting, pause 30s",
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
                        "text": "Hello. This is a VoIPBIN transcription test. Everything you say will be transcribed in real time. Please speak now.",
                        "language": "en-US"
                    }
                },
                {
                    "type": "sleep",
                    "option": {
                        "duration": 30000
                    }
                }
            ]
        }'

The response includes the created flow with a server-generated ``id`` (UUID). Save this — you will need it in the next step:

.. code::

    {
        "id": "b3f7a1d2-9c4e-4f8a-b6d1-2e5f8a3c7d90",
        "name": "quickstart transcribe flow",
        "detail": "Answer, start transcription, speak greeting, pause 30s",
        "actions": [
            {
                "id": "a1b2c3d4-0001-0000-0000-000000000001",
                "type": "answer"
            },
            {
                "id": "a1b2c3d4-0001-0000-0000-000000000002",
                "type": "transcribe_start",
                "option": {
                    "language": "en-US"
                }
            },
            {
                "id": "a1b2c3d4-0001-0000-0000-000000000003",
                "type": "talk",
                "option": {
                    "text": "Hello. This is a VoIPBIN transcription test. Everything you say will be transcribed in real time. Please speak now.",
                    "language": "en-US"
                }
            },
            {
                "id": "a1b2c3d4-0001-0000-0000-000000000004",
                "type": "sleep",
                "option": {
                    "duration": 30000
                }
            }
        ],
        "tm_create": "2026-02-18 10:00:00.000000",
        "tm_update": "",
        "tm_delete": ""
    }

Key actions:

- ``answer``: Answers the incoming call. Required before any media actions on inbound calls.
- ``transcribe_start``: Starts real-time speech-to-text. The ``language`` field (String, BCP47 format) must match the speaker's language (e.g., ``en-US``, ``ko-KR``). See :ref:`Supported Languages <transcribe-overview-supported_languages>`.
- ``talk``: Generates TTS audio from the ``text`` field and plays it. This audio is transcribed as ``direction: "out"``.
- ``sleep``: Pauses flow execution for ``duration`` milliseconds (Integer). ``30000`` = 30 seconds. Keeps the call alive so you can speak and observe transcription.

Step 2: Create a virtual number and assign the flow
----------------------------------------------------
Create a virtual number (free, ``+899`` prefix) and assign the transcription flow to handle inbound calls.

**Create the virtual number:**

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/numbers?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "number": "+899100000001"
        }'

.. code::

    {
        "id": "d4e5f6a7-b8c9-0d1e-2f3a-4b5c6d7e8f90",
        "number": "+899100000001",
        "type": "virtual",
        "call_flow_id": "00000000-0000-0000-0000-000000000000",
        "message_flow_id": "00000000-0000-0000-0000-000000000000",
        "name": "",
        "detail": "",
        "status": "active",
        "t38_enabled": false,
        "emergency_enabled": false,
        "tm_create": "2026-02-18 10:01:00.000000",
        "tm_update": "",
        "tm_delete": ""
    }

Save the number's ``id`` (UUID) from the response.

**Assign the flow to the number:**

Update the number's ``call_flow_id`` to point to the flow you created in Step 1. Replace ``<number-id>`` with the ``id`` from the response above, and ``<flow-id>`` with the flow ``id`` from Step 1:

.. code::

    $ curl --request PUT 'https://api.voipbin.net/v1.0/numbers/<number-id>?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "call_flow_id": "<flow-id>"
        }'

.. code::

    {
        "id": "d4e5f6a7-b8c9-0d1e-2f3a-4b5c6d7e8f90",
        "number": "+899100000001",
        "type": "virtual",
        "call_flow_id": "b3f7a1d2-9c4e-4f8a-b6d1-2e5f8a3c7d90",
        "message_flow_id": "00000000-0000-0000-0000-000000000000",
        "name": "",
        "detail": "",
        "status": "active",
        ...
    }

Now, any inbound call to ``+899100000001`` will execute your transcription flow.

.. note:: **AI Implementation Hint**

   Virtual numbers (``+899`` prefix) are free and routed internally within VoIPBIN. They do not require a provider purchase. If ``+899100000001`` is already taken, try ``+899100000002`` or search for available virtual numbers via ``GET /available_numbers?type=virtual``.

Step 3: Subscribe to transcribe events via WebSocket
-----------------------------------------------------
Before making the call, connect to the VoIPBIN WebSocket and subscribe to transcription events. This way you receive transcripts as they are generated during the call.

**Connect to WebSocket:**

.. code::

    wss://api.voipbin.net/v1.0/ws?token=<your-token>

**Send a subscription message** after connecting. Replace ``<your-customer-id>`` with your customer ID (UUID) obtained from ``GET https://api.voipbin.net/v1.0/customer``:

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:<your-customer-id>:transcribe:*"
        ]
    }

The wildcard ``*`` subscribes to events from all transcriptions under your account.

**Python example:**

.. code::

    import websocket
    import json

    token = "<your-token>"
    customer_id = "<your-customer-id>"

    def on_message(ws, message):
        data = json.loads(message)
        if data.get("event_type") == "transcript_created":
            transcript = data["data"]
            direction = transcript["direction"]  # "in" = caller, "out" = VoIPBIN TTS
            text = transcript["message"]
            print(f"[{direction}] {text}")

    def on_open(ws):
        subscription = {
            "type": "subscribe",
            "topics": [
                f"customer_id:{customer_id}:transcribe:*"
            ]
        }
        ws.send(json.dumps(subscription))
        print("Subscribed to transcribe events. Waiting for transcripts...")

    ws = websocket.WebSocketApp(
        f"wss://api.voipbin.net/v1.0/ws?token={token}",
        on_open=on_open,
        on_message=on_message
    )
    ws.run_forever()

Step 4: Make a call to the virtual number
------------------------------------------
With the WebSocket connected, make an outbound call to the virtual number. The call's destination is the virtual number you created; the source is any number owned by your account. In this quickstart scenario, the TTS greeting will generate ``direction: "out"`` transcripts. To also see ``direction: "in"`` transcripts (caller speech), you would need a SIP or WebRTC client connected to the call.

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
                    "type": "tel",
                    "target": "+899100000001"
                }
            ]
        }'

.. code::

    [
        {
            "id": "c7d8e9f0-a1b2-3c4d-5e6f-7a8b9c0d1e2f",
            "flow_id": "b3f7a1d2-9c4e-4f8a-b6d1-2e5f8a3c7d90",
            "source": {
                "type": "tel",
                "target": "<your-source-number>"
            },
            "destination": {
                "type": "tel",
                "target": "+899100000001"
            },
            "status": "dialing",
            "direction": "outgoing",
            ...
        }
    ]

The call dials the virtual number, which triggers the transcription flow: answer, start transcription, speak the greeting, and pause for 30 seconds.

.. note:: **AI Implementation Hint**

   The ``source`` number must be a VoIPBIN-owned number (obtained from ``GET /numbers``). The ``destinations`` target is the virtual number you created. Since the virtual number has a ``call_flow_id`` assigned, VoIPBIN executes the flow when the call is answered. No ``flow_id`` or ``actions`` field is needed in the call request — the number's assigned flow handles everything.

Step 5: Receive real-time transcription events
-----------------------------------------------
Within seconds of the call being answered, the WebSocket begins delivering ``transcript_created`` events. The TTS greeting appears first (``direction: "out"``), followed by any speech from the caller (``direction: "in"``).

**Example WebSocket event:**

.. code::

    {
        "event_type": "transcript_created",
        "timestamp": "2026-02-18T10:02:05.000000Z",
        "topic": "customer_id:<your-customer-id>:transcribe:8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
        "data": {
            "id": "9d59e7f0-7bdc-4c52-bb8c-bab718952050",
            "transcribe_id": "8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
            "direction": "out",
            "message": "Hello. This is a VoIPBIN transcription test. Everything you say will be transcribed in real time. Please speak now.",
            "tm_transcript": "0001-01-01 00:00:08.991840",
            "tm_create": "2026-02-18 10:02:05.233415"
        }
    }

**Fields:**

- ``event_type`` (String): Always ``transcript_created`` for new transcript segments.
- ``data.transcribe_id`` (UUID): The transcription session ID. Use this to query all transcripts for this session via ``GET /transcripts?transcribe_id=<transcribe_id>``.
- ``data.direction`` (enum string): ``"out"`` = audio from VoIPBIN to the caller (TTS output). ``"in"`` = audio from the caller to VoIPBIN (caller speech).
- ``data.message`` (String): The transcribed text.

If you run the Python WebSocket example from Step 3, you will see output like:

.. code::

    Subscribed to transcribe events. Waiting for transcripts...
    [out] Hello. This is a VoIPBIN transcription test. Everything you say will be transcribed in real time. Please speak now.
    [in] Hi, this is a test of the transcription feature.

Receive events via webhook (alternative)
-----------------------------------------
Instead of WebSocket, you can receive transcription events via HTTP webhook. Create a webhook that listens for ``transcript.created`` events:

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/webhooks?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "Transcription events",
            "uri": "https://your-server.com/webhook",
            "method": "POST",
            "event_types": [
                "transcript.created"
            ]
        }'

Note: the ``event_types`` registration uses dot notation (``transcript.created``), while the delivered payload's ``event_type`` field uses underscore notation (``transcript_created``).

VoIPBIN sends a ``POST`` request to your endpoint each time a transcript segment is generated:

.. code::

    POST https://your-server.com/webhook

    {
        "event_type": "transcript_created",
        "timestamp": "2026-02-18T10:02:05.000000Z",
        "topic": "customer_id:<your-customer-id>:transcribe:8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
        "data": {
            "id": "9d59e7f0-7bdc-4c52-bb8c-bab718952050",
            "transcribe_id": "8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
            "direction": "in",
            "message": "Hi, this is a test of the transcription feature.",
            "tm_transcript": "0001-01-01 00:00:15.500000",
            "tm_create": "2026-02-18 10:02:10.100000"
        }
    }

.. note:: **AI Implementation Hint**

   Webhook delivery requires your ``uri`` endpoint to be publicly accessible and to return HTTP ``200`` within 5 seconds. For local development, use a tunneling tool (e.g., ngrok) to expose your local server. WebSocket is simpler for development and testing since it requires no public endpoint.

Step 6: Verify transcription results
--------------------------------------
After the call ends, you can retrieve the full transcript via the API. Use the ``transcribe_id`` (UUID) from the WebSocket or webhook events. In this flow-based scenario, no ``POST /transcribes`` call is made directly — the ``transcribe_id`` is generated internally when the ``transcribe_start`` action executes and is surfaced only via events. If you did not capture it from an event, you can also find it via ``GET /transcribes?resource_id=<call-id>``.

.. code::

    $ curl --request GET 'https://api.voipbin.net/v1.0/transcripts?token=<your-token>&transcribe_id=<transcribe-id>'

.. code::

    {
        "result": [
            {
                "id": "3c95ea10-a5b7-4a68-aebf-ed1903baf110",
                "transcribe_id": "8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
                "direction": "out",
                "message": "Hello. This is a VoIPBIN transcription test. Everything you say will be transcribed in real time. Please speak now.",
                "tm_transcript": "0001-01-01 00:00:08.991840",
                "tm_create": "2026-02-18 10:02:05.233415"
            },
            {
                "id": "06af78f0-b063-48c0-b22d-d31a5af0aa88",
                "transcribe_id": "8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
                "direction": "in",
                "message": "Hi, this is a test of the transcription feature.",
                "tm_transcript": "0001-01-01 00:00:15.500000",
                "tm_create": "2026-02-18 10:02:10.100000"
            }
        ]
    }

The ``direction`` field distinguishes speakers: ``"out"`` is the TTS greeting played by VoIPBIN, ``"in"`` is the caller's speech.

Troubleshooting
+++++++++++++++

* **No transcription events received:**
    * **Cause:** WebSocket subscription topic does not match your customer ID, or subscription was sent before the connection opened.
    * **Fix:** Verify the customer ID in the topic matches your account (from ``GET https://api.voipbin.net/v1.0/customer``). Send the subscribe message only after the ``on_open`` callback fires.

* **400 Bad Request on call creation:**
    * **Cause:** The ``source`` number is not owned by your account, or the number format is not E.164.
    * **Fix:** Check your numbers via ``GET /numbers``. Ensure the source starts with ``+`` and contains only digits after it.

* **Transcripts show only "out" direction, no "in":**
    * **Cause:** The call is outbound to a virtual number with no real caller speaking.
    * **Fix:** This is expected for the TTS-only portion. If you are connected to the call (e.g., via WebRTC or SIP phone), speak during the 30-second ``sleep`` period to generate ``"in"`` transcripts.

* **404 Not Found on number update:**
    * **Cause:** The number ID does not exist or belongs to a different customer.
    * **Fix:** Verify the number ID from the ``POST /numbers`` response. Use ``GET /numbers`` to list your numbers.

For more details, see the full :ref:`Transcribe tutorial <transcribe-tutorial>` and :ref:`WebSocket tutorial <websocket-tutorial>`.
