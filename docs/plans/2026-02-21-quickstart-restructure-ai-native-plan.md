# Quickstart Restructure: AI-Native Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Restructure the API manager quickstart documentation into 3 progressive scenarios (first call, events, real-time voice interaction) with a new section covering extensions, SIP phone registration, real-time transcription, and the speaking TTS API.

**Architecture:** Modify 2 existing RST files and create 1 new RST file. Remove queue and transcribe includes from the parent quickstart. Rebuild Sphinx HTML after all RST changes. No Go code changes.

**Tech Stack:** RST (reStructuredText), Sphinx, Python venv

**Design doc:** `docs/plans/2026-02-21-quickstart-restructure-ai-native-design.md`

---

### Task 1: Update `quickstart.rst` parent file

**Files:**
- Modify: `bin-api-manager/docsdev/source/quickstart.rst`

**Step 1: Edit the parent quickstart file**

Replace the entire content of `quickstart.rst` with the following. Changes:
- Add a "Three scenarios" intro paragraph explaining the progressive structure
- Remove `.. include:: quickstart_queue.rst`
- Replace `.. include:: quickstart_transcribe.rst` with `.. include:: quickstart_realtime.rst`
- Update "What's Next" section to reference the 3 scenarios completed

```rst
.. _quickstart-main:

*******************
Quickstart
*******************

There are two ways to get started with VoIPBIN:

- **Try the Demo** — Click the **Try Demo Account** button at `admin.voipbin.net <https://admin.voipbin.net>`_ to explore VoIPBIN instantly. No setup or sign-up needed.
- **Run the Sandbox** — Run the full VoIPBIN platform on your local machine using Docker. See the :ref:`Sandbox <quickstart_sandbox>` section below.

For production use, you can :ref:`sign up <quickstart_signup>` for your own account.

This quickstart walks you through three progressive scenarios:

1. **Your First Call** — Sign up, authenticate, and make an outbound voice call with text-to-speech.
2. **Receiving Events** — Set up webhooks and WebSocket subscriptions to receive real-time notifications.
3. **Real-Time Voice Interaction** — Create a SIP extension, register a softphone, make a call with live transcription, and speak into the call using the TTS API.

.. include:: quickstart_sandbox.rst
.. include:: quickstart_signup.rst
.. include:: quickstart_authentication.rst
.. include:: quickstart_call.rst
.. include:: quickstart_events.rst
.. include:: quickstart_realtime.rst

.. _quickstart_next:

What's Next
===========
Now that you have completed all three scenarios, explore the full capabilities of VoIPBIN:

- :ref:`Flow <flow-main>` — Build programmable voice workflows with the visual flow builder.
- :ref:`AI <ai-main>` — Integrate AI-powered voice agents with real-time speech processing.
- :ref:`Conference <conference-main>` — Set up multi-party conferencing.
- :ref:`Conversation <conversation-main>` — Manage messaging conversations.
- :ref:`Queue <queue-main>` — Route incoming calls to available agents with queue management.

For the complete API reference, visit the `API documentation <https://api.voipbin.net/redoc/index.html>`_.

.. note:: **AI Implementation Hint**

   The full OpenAPI 3.0 specification is available as machine-readable JSON at ``https://api.voipbin.net/openapi.json``. Use this for automated client generation, API discovery, and building integrations programmatically.
```

**Step 2: Verify no syntax errors**

Visually review the file for RST syntax correctness — matching indentation, correct `.. include::` directives, proper ref targets.

**Step 3: Commit**

Do not commit yet — wait until all tasks are complete.

---

### Task 2: Create `quickstart_realtime.rst` (Scenario 3)

**Files:**
- Create: `bin-api-manager/docsdev/source/quickstart_realtime.rst`

**Step 1: Write the new quickstart_realtime.rst file**

Create the file with the following content. This is the main new content covering extensions, SIP phone registration, calling an extension with transcription, and real-time TTS speaking.

```rst
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

Save the ``name`` (String) — you will use it as the call destination in Step 4.

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

**What happens next:**

1. Linphone rings. **Answer the call.**
2. The TTS greeting plays (you hear it through Linphone).
3. The call enters the ``sleep`` action (600 seconds = 10 minutes), keeping the call alive.
4. Transcription is active — anything you say into Linphone is transcribed.

.. note:: **AI Implementation Hint**

   The ``source`` number must be a VoIPBIN-owned number (from ``GET /numbers``). The destination ``type`` is ``extension`` (not ``tel``), and ``target_name`` (String) is the extension's ``name`` field from Step 1. The ``sleep`` duration is in milliseconds — ``600000`` = 10 minutes. The ``transcribe_start`` action uses BCP47 language codes (e.g., ``en-US``, ``ko-KR``, ``ja-JP``).

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

* **One-way audio (can hear TTS but Linphone speech is not transcribed):**
    * **Cause:** NAT or firewall blocking RTP traffic.
    * **Fix:** Enable STUN in Linphone settings (use ``stun.linphone.org``). Ensure your network allows UDP traffic on ports 5060 and 10000-20000.
```

**Step 2: Verify RST syntax**

Check that all ref targets, code blocks, note directives, and table formatting are syntactically correct.

---

### Task 3: Build HTML documentation

**Files:**
- Rebuild: `bin-api-manager/docsdev/build/` (generated HTML)

**Step 1: Set up Python venv (if not already present)**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-api-manager/docsdev
python3 -m venv .venv_docs
source .venv_docs/bin/activate
pip install sphinx sphinx-rtd-theme sphinx-wagtail-theme sphinxcontrib-youtube
```

**Step 2: Build the HTML**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-api-manager/docsdev
source .venv_docs/bin/activate
python3 -m sphinx -M html source build
```

Expected: Build completes with 0 errors. Warnings about missing refs to other docs pages are acceptable.

**Step 3: Verify the build output**

Open `build/html/quickstart.html` in a browser and verify:
- The 3-scenario intro paragraph appears
- Scenarios 1 (Call), 2 (Events), 3 (Real-Time Voice) render correctly
- Queue section is gone
- The old transcribe section is replaced by the new real-time section
- All code blocks render properly
- Tables render properly
- AI Implementation Hint notes render as note boxes

---

### Task 4: Commit and push

**Step 1: Create worktree and branch**

```bash
cd /home/pchero/gitvoipbin/monorepo
git worktree add /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-quickstart-restructure-ai-native -b NOJIRA-quickstart-restructure-ai-native
```

**Step 2: Copy changed files to worktree**

Copy the modified/new RST files and the rebuilt HTML to the worktree.

**Step 3: Stage and commit**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-quickstart-restructure-ai-native
git add bin-api-manager/docsdev/source/quickstart.rst
git add bin-api-manager/docsdev/source/quickstart_realtime.rst
git add -f bin-api-manager/docsdev/build/
git add docs/plans/2026-02-21-quickstart-restructure-ai-native-design.md
git add docs/plans/2026-02-21-quickstart-restructure-ai-native-plan.md
git commit -m "NOJIRA-quickstart-restructure-ai-native

Restructure quickstart into 3 progressive scenarios with new real-time
voice interaction section covering extensions, SIP registration,
transcription, and the Speaking TTS API.

- bin-api-manager: Restructure quickstart.rst into 3-scenario progressive guide
- bin-api-manager: Add quickstart_realtime.rst covering extensions, Linphone SIP setup, real-time transcribe, and Speaking API
- bin-api-manager: Remove queue section from quickstart includes
- bin-api-manager: Replace transcribe quickstart with comprehensive real-time voice interaction scenario
- bin-api-manager: Rebuild HTML documentation
- docs: Add design and implementation plan documents"
```

**Step 4: Push and create PR**

```bash
git push -u origin NOJIRA-quickstart-restructure-ai-native
```

Then create PR with title `NOJIRA-quickstart-restructure-ai-native`.
