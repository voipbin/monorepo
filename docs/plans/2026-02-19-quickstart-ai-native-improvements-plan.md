# Quickstart AI-Native Improvements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Bring all quickstart RST documentation pages to AI-native standard by adding missing troubleshooting sections, a headless signup flow, and event delivery introduction (webhook + websocket).

**Architecture:** 5 RST file edits + 1 new RST file in `bin-api-manager/docsdev/source/`, plus a `GET /openapi.json` endpoint to serve the bundled OpenAPI spec for AI agent consumption. After RST edits, rebuild Sphinx HTML and commit both together.

**Tech Stack:** RST (reStructuredText), Sphinx documentation builder

---

### Task 1: Rewrite quickstart_signup.rst with headless API flow

**Files:**
- Modify: `bin-api-manager/docsdev/source/quickstart_signup.rst`

**Step 1: Replace the entire file content**

Write the following content to `bin-api-manager/docsdev/source/quickstart_signup.rst`:

```rst
.. _quickstart_signup:

Signup
======
To use the VoIPBIN production API, you need your own account. There are two ways to sign up: via the Admin Console (browser) or via the API (headless, for automated systems).

Sign up via Admin Console
-------------------------
1. Go to the `admin console <https://admin.voipbin.net>`_.
2. Click **Sign Up**.
3. Enter your email address and submit.
4. Check your inbox for a verification email.
5. Click the verification link in the email to verify your address.
6. You will receive a welcome email with instructions to set your password.

Once your password is set, you can log in to the `admin console <https://admin.voipbin.net>`_ and start making API requests.

Sign up via API (Headless)
--------------------------
For automated systems and AI agents, use the headless signup flow. This requires two API calls: one to initiate signup and one to verify with a 6-digit OTP code sent to your email.

.. note:: **AI Implementation Hint**

   The headless signup path is preferred for AI agents and automated systems. The ``POST /auth/complete-signup`` response includes an ``accesskey`` with an API token that can be used immediately — no need to call ``POST /auth/login`` separately. Note that ``POST /auth/signup`` always returns HTTP 200 regardless of success to prevent email enumeration. An empty ``temp_token`` in the response may indicate the email is already registered or invalid.

**Step 1: Initiate signup**

Send a ``POST`` request to ``/auth/signup`` with your email address (String, Required). All other fields are optional.

.. code::

    $ curl --request POST 'https://api.voipbin.net/auth/signup' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "email": "your-email@example.com",
            "name": "Your Company Name"
        }'

Response (always HTTP 200):

.. code::

    {
        "temp_token": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"
    }

Save the ``temp_token`` (String) — you will need it in the next step. A 6-digit OTP verification code is sent to the email address you provided.

**Step 2: Complete signup with OTP**

Send a ``POST`` request to ``/auth/complete-signup`` with the ``temp_token`` (String, Required) from Step 1 and the 6-digit ``code`` (String, Required) from the verification email.

.. code::

    $ curl --request POST 'https://api.voipbin.net/auth/complete-signup' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "temp_token": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
            "code": "123456"
        }'

Response:

.. code::

    {
        "customer_id": "550e8400-e29b-41d4-a716-446655440000",
        "accesskey": {
            "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
            "token": "your-api-access-key-token"
        }
    }

The ``accesskey.token`` (String) is your API key. Use it immediately for authentication — see :ref:`Authentication <quickstart_authentication>`.

Troubleshooting
---------------

* **200 OK but empty ``temp_token``:**
    * **Cause:** The email may already be registered, or the email format is invalid.
    * **Fix:** Try logging in via ``POST /auth/login`` with the email. If the account exists, use the existing credentials. ``POST /auth/signup`` always returns 200 to prevent email enumeration.

* **400 Bad Request on ``/auth/complete-signup``:**
    * **Cause:** The ``temp_token`` has expired or the ``code`` is incorrect.
    * **Fix:** Verify the 6-digit code from the verification email. Codes expire after a limited time — if expired, re-submit ``POST /auth/signup`` to receive a new code.

* **429 Too Many Requests on ``/auth/complete-signup``:**
    * **Cause:** Too many verification attempts (maximum 5 per ``temp_token``).
    * **Fix:** Re-submit ``POST /auth/signup`` with the same email to receive a new ``temp_token`` and verification code, then retry.
```

**Step 2: Verify the file was written correctly**

Read back the file and confirm it contains both the Admin Console and API signup sections, the AI hint, and the troubleshooting section.

---

### Task 2: Create quickstart_events.rst (new file)

**Files:**
- Create: `bin-api-manager/docsdev/source/quickstart_events.rst`

**Step 1: Write the new file**

Write the following content to `bin-api-manager/docsdev/source/quickstart_events.rst`:

```rst
.. _quickstart_events:

Receiving Events
================
VoIPBIN notifies you in real-time when things happen — calls connect, messages arrive, recordings complete, transcriptions finish. There are two delivery methods:

- **Webhook** — VoIPBIN sends HTTP POST requests to your server endpoint.
- **WebSocket** — You maintain a persistent connection and receive events instantly.

.. note:: **AI Implementation Hint**

   For AI agents and automated systems, **WebSocket is preferred** because it requires no public server endpoint and delivers events with lower latency. Use **Webhook** when you have a persistent server that needs to process events asynchronously (e.g., updating a database, sending notifications).

+-------------------+-------------------------------------------+-------------------------------------------+
|                   | Webhook                                   | WebSocket                                 |
+===================+===========================================+===========================================+
| Connection        | Stateless HTTP pushes to your server      | Persistent bidirectional connection       |
+-------------------+-------------------------------------------+-------------------------------------------+
| Setup             | Register URL via ``POST /webhooks``       | Connect to ``wss://`` endpoint            |
+-------------------+-------------------------------------------+-------------------------------------------+
| Best for          | Server-side integrations, CI/CD pipelines | Real-time dashboards, AI agents           |
+-------------------+-------------------------------------------+-------------------------------------------+
| Requires          | Public HTTPS endpoint                     | WebSocket client library                  |
+-------------------+-------------------------------------------+-------------------------------------------+

Webhook
-------
Register an endpoint URL, and VoIPBIN sends HTTP POST requests to it when events occur.

**Create a webhook:**

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/webhooks?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "My webhook",
            "uri": "https://your-server.com/voipbin/webhook",
            "method": "POST",
            "event_types": [
                "call_created",
                "call_updated",
                "call_hungup"
            ]
        }'

Response:

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "name": "My webhook",
        "uri": "https://your-server.com/voipbin/webhook",
        "method": "POST",
        "event_types": [
            "call_created",
            "call_updated",
            "call_hungup"
        ],
        "status": "active",
        "tm_create": "2026-02-19 10:00:00.000000",
        "tm_update": "2026-02-19 10:00:00.000000",
        "tm_delete": ""
    }

VoIPBIN sends a ``POST`` request to your ``uri`` each time a matching event occurs. Your endpoint must respond with HTTP ``200`` within 5 seconds.

.. note:: **AI Implementation Hint**

   Event types in the ``event_types`` registration and the delivered payload both use underscore notation (e.g., ``call_created``, ``message_received``). Your endpoint must be publicly accessible — for local development, use a tunneling tool (e.g., ngrok).

For the full webhook guide, see :ref:`Webhook documentation <webhook-main>`.

WebSocket
---------
Connect to VoIPBIN's WebSocket endpoint and subscribe to topics to receive events in real-time.

**Connect:**

.. code::

    wss://api.voipbin.net/v1.0/ws?token=<your-token>

**Subscribe to events** by sending a JSON message after connecting. Replace ``<your-customer-id>`` with your customer ID (UUID) obtained from ``GET /customers``:

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:<your-customer-id>:call:*"
        ]
    }

The wildcard ``*`` subscribes to events from all calls under your account.

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
            print(f"Event: {event_type}")
            print(f"Data: {json.dumps(data.get('data', {}), indent=2)}")

    def on_open(ws):
        subscription = {
            "type": "subscribe",
            "topics": [
                f"customer_id:{customer_id}:call:*"
            ]
        }
        ws.send(json.dumps(subscription))
        print("Subscribed to call events. Waiting...")

    ws = websocket.WebSocketApp(
        f"wss://api.voipbin.net/v1.0/ws?token={token}",
        on_open=on_open,
        on_message=on_message
    )
    ws.run_forever()

.. note:: **AI Implementation Hint**

   The topic format is ``<scope>:<scope_id>:<resource>:<resource_id>``. Use ``*`` as the ``resource_id`` to subscribe to all resources of a type (e.g., ``customer_id:<id>:call:*`` for all call events, ``customer_id:<id>:message:*`` for all message events). If the WebSocket connection drops, all subscriptions are lost — implement automatic reconnection and re-subscribe after reconnecting.

For the full WebSocket guide, see :ref:`WebSocket documentation <websocket-main>`.
```

**Step 2: Verify the file was created**

Read back the file and confirm it contains the comparison table, webhook section, websocket section with Python example, and AI hints.

---

### Task 3: Update quickstart.rst to include events section

**Files:**
- Modify: `bin-api-manager/docsdev/source/quickstart.rst`

**Step 1: Add the events include**

In `quickstart.rst`, insert `.. include:: quickstart_events.rst` between the `quickstart_authentication.rst` and `quickstart_call.rst` includes. The includes section should become:

```rst
.. include:: quickstart_sandbox.rst
.. include:: quickstart_signup.rst
.. include:: quickstart_authentication.rst
.. include:: quickstart_events.rst
.. include:: quickstart_call.rst
.. include:: quickstart_queue.rst
.. include:: quickstart_transcribe.rst
```

Use Edit tool to add the single line `.. include:: quickstart_events.rst` after line 16 (`.. include:: quickstart_authentication.rst`).

---

### Task 4: Add troubleshooting to quickstart_sandbox.rst

**Files:**
- Modify: `bin-api-manager/docsdev/source/quickstart_sandbox.rst`

**Step 1: Append troubleshooting section**

Add the following after the last line (line 55, the closing `'`) of the sandbox curl example:

```rst

Troubleshooting
---------------

* **"init" or "start" command fails:**
    * **Cause:** Docker daemon is not running or Docker Compose v2 is not installed.
    * **Fix:** Verify Docker is running with ``docker compose version``. Must show v2.x or later.

* **Port 8443 already in use:**
    * **Cause:** Another service is occupying port 8443.
    * **Fix:** Stop the conflicting service, or check with ``ss -tlnp | grep 8443`` (Linux) / ``lsof -i :8443`` (macOS).

* **SSL certificate error (without ``-k`` flag):**
    * **Cause:** The Sandbox uses a self-signed SSL certificate that is not trusted by default.
    * **Fix:** Add the ``-k`` flag to all curl commands when using the Sandbox.

* **401 Unauthorized with default credentials:**
    * **Cause:** The Sandbox has not fully initialized.
    * **Fix:** Run ``voipbin> init`` again and wait for it to complete before running ``voipbin> start``.
```

---

### Task 5: Add call status enum and troubleshooting to quickstart_call.rst

**Files:**
- Modify: `bin-api-manager/docsdev/source/quickstart_call.rst`

**Step 1: Add call status lifecycle after the response example**

After line 69 (the closing `]` of the inline-actions response example), add:

```rst

**Call status lifecycle:**

- ``dialing``: The system is currently dialing the destination number.
- ``ringing``: The destination device is ringing, awaiting answer.
- ``progressing``: The call is answered. Audio is flowing between parties.
- ``terminating``: The system is ending the call.
- ``canceling``: The originator canceled the call before it was answered (outgoing calls only).
- ``hangup``: The call has ended. This is the final state.

For the full call lifecycle, see :ref:`Call overview <call-overview>`.
```

**Step 2: Add troubleshooting section at the end of the file**

After line 97 (`For more details on flows, see the :ref:`Flow tutorial <flow-main>`.`), add:

```rst

Troubleshooting
+++++++++++++++

* **400 Bad Request:**
    * **Cause:** The ``source`` number is not owned by your VoIPBIN account, or the phone number is not in E.164 format.
    * **Fix:** Verify your numbers via ``GET /numbers``. Ensure all phone numbers start with ``+`` followed by digits only (e.g., ``+15551234567``).

* **404 Not Found (when using ``flow_id``):**
    * **Cause:** The ``flow_id`` does not reference an existing flow.
    * **Fix:** Verify the flow exists via ``GET /flows``. Create one via ``POST /flows`` if needed.

* **Call status immediately shows "hangup":**
    * **Cause:** The destination number is unreachable, or the source number has no telephony provider attached.
    * **Fix:** For testing, use virtual numbers (``+899`` prefix) as the destination — these are free and route internally within VoIPBIN.
```

---

### Task 6: Add troubleshooting to quickstart_queue.rst

**Files:**
- Modify: `bin-api-manager/docsdev/source/quickstart_queue.rst`

**Step 1: Add troubleshooting section at the end of the file**

After line 71 (`For more details, see the :ref:`Queue tutorial <queue-tutorial>`.`), add:

```rst

Troubleshooting
+++++++++++++++

* **400 Bad Request:**
    * **Cause:** ``tag_ids`` contains an invalid UUID, or required fields are missing.
    * **Fix:** Verify tag IDs via ``GET /tags``. Ensure ``tag_ids`` is a non-empty array of valid UUIDs.

* **Queue created but calls are not routed to agents:**
    * **Cause:** No agents with matching tags are online or in ``available`` status.
    * **Fix:** Verify agents have matching tags via ``GET /agents``. Check that at least one agent is in ``available`` status.

* **Callers timing out immediately:**
    * **Cause:** ``timeout_wait`` is set too low. The value is in **milliseconds**, not seconds.
    * **Fix:** For a 100-second wait, set ``timeout_wait: 100000``. Setting ``100`` means only 0.1 seconds.
```

---

### Task 7: Rebuild Sphinx HTML and commit

**Files:**
- All RST files modified in Tasks 1-6
- Generated HTML in `bin-api-manager/docsdev/build/`

**Step 1: Rebuild Sphinx HTML**

Run from the worktree root:

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Quickstart-ai-native-improvements/bin-api-manager/docsdev && python3 -m sphinx -M html source build
```

Expected: Build completes with no errors (warnings about missing references are acceptable).

**Step 2: Verify the build output**

Check that the quickstart HTML was rebuilt:

```bash
ls -la ~/gitvoipbin/monorepo-worktrees/NOJIRA-Quickstart-ai-native-improvements/bin-api-manager/docsdev/build/html/quickstart.html
```

**Step 3: Stage all changes**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Quickstart-ai-native-improvements
git add bin-api-manager/docsdev/source/quickstart.rst
git add bin-api-manager/docsdev/source/quickstart_signup.rst
git add bin-api-manager/docsdev/source/quickstart_events.rst
git add bin-api-manager/docsdev/source/quickstart_sandbox.rst
git add bin-api-manager/docsdev/source/quickstart_call.rst
git add bin-api-manager/docsdev/source/quickstart_queue.rst
git add docs/plans/2026-02-19-quickstart-ai-native-improvements-design.md
git add docs/plans/2026-02-19-quickstart-ai-native-improvements-plan.md
git add -f bin-api-manager/docsdev/build/
```

**Step 4: Commit**

```bash
git commit -m "NOJIRA-Quickstart-ai-native-improvements

Bring all quickstart pages to AI-native documentation standard.

- bin-api-manager: Rewrite quickstart_signup.rst with headless API signup flow
- bin-api-manager: Add quickstart_events.rst introducing webhook and websocket
- bin-api-manager: Add events include to quickstart.rst index
- bin-api-manager: Add troubleshooting to quickstart_sandbox.rst
- bin-api-manager: Add call status enum and troubleshooting to quickstart_call.rst
- bin-api-manager: Add troubleshooting to quickstart_queue.rst
- docs: Add design and implementation plan documents"
```

**Step 5: Verify commit**

```bash
git log --oneline -1
git diff --stat HEAD~1
```

Expected: Single commit with ~8 files changed (6 RST + 2 plan docs + build directory).
