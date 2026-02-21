.. _quickstart_events:

.. _quickstart_event_subscription:

Receiving Events
----------------
VoIPBIN notifies you in real-time when things happen — calls connect, transcriptions arrive, recordings complete. There are two delivery methods:

- **WebSocket** — Maintain a persistent connection and receive events instantly. No public server needed.
- **Customer Webhook** — Configure a webhook URI on your customer account to receive all events via HTTP POST.

.. note:: **AI Implementation Hint**

   For AI agents and automated systems, **WebSocket is preferred** because it requires no public server endpoint and delivers events with lower latency. Use **Customer Webhook** when you have a persistent server that needs to process events asynchronously.

+-------------------+-------------------------------------------+-------------------------------------------+
|                   | WebSocket                                 | Customer Webhook                          |
+===================+===========================================+===========================================+
| Connection        | Persistent bidirectional connection       | Stateless HTTP pushes to your server      |
+-------------------+-------------------------------------------+-------------------------------------------+
| Setup             | Connect to ``wss://`` endpoint            | ``PUT /customer`` with webhook URI        |
+-------------------+-------------------------------------------+-------------------------------------------+
| Filtering         | Subscribe to specific topics              | Receives **all** events (no filtering)    |
+-------------------+-------------------------------------------+-------------------------------------------+
| Best for          | Real-time dashboards, AI agents           | Server-side catch-all processing          |
+-------------------+-------------------------------------------+-------------------------------------------+
| Requires          | WebSocket client library                  | Public HTTPS endpoint                     |
+-------------------+-------------------------------------------+-------------------------------------------+

WebSocket
~~~~~~~~~
Connect to VoIPBIN's WebSocket endpoint and subscribe to topics to receive events in real-time.

**Connect:**

.. code::

    wss://api.voipbin.net/v1.0/ws?token=<your-token>

**Subscribe** by sending a JSON message after connecting. Replace ``<your-customer-id>`` with your customer ID (UUID) obtained from ``GET https://api.voipbin.net/v1.0/customer``:

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:<your-customer-id>:call:*",
            "customer_id:<your-customer-id>:transcribe:*"
        ]
    }

The wildcard ``*`` subscribes to events from all resources of that type under your account.

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

.. note:: **AI Implementation Hint**

   The topic format is ``<scope>:<scope_id>:<resource>:<resource_id>``. Use ``*`` as the ``resource_id`` to subscribe to all resources of a type (e.g., ``customer_id:<id>:call:*`` for all call events, ``customer_id:<id>:transcribe:*`` for all transcription events). If the WebSocket connection drops, all subscriptions are lost — implement automatic reconnection and re-subscribe after reconnecting.

For the full WebSocket guide, see :ref:`WebSocket documentation <websocket-main>`.

Customer Webhook
~~~~~~~~~~~~~~~~
Configure a webhook URI on your customer account. VoIPBIN sends HTTP POST requests to this URI for **all** events associated with your account — no per-event-type filtering is needed.

**Update your customer's webhook configuration:**

.. code::

    $ curl --request PUT 'https://api.voipbin.net/v1.0/customer?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "webhook_method": "POST",
            "webhook_uri": "https://your-server.com/voipbin/events"
        }'

Response:

.. code::

    {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "webhook_method": "POST",
        "webhook_uri": "https://your-server.com/voipbin/events",
        ...
    }

Once configured, VoIPBIN sends a ``POST`` request to your ``webhook_uri`` each time any event occurs for your account. Your endpoint must respond with HTTP ``200`` within 5 seconds.

**Fields:**

- ``webhook_method`` (enum String): The HTTP method used for webhook delivery. Use ``"POST"``.
- ``webhook_uri`` (String): The URI where event notifications are sent. Must be publicly accessible.

.. note:: **AI Implementation Hint**

   The customer webhook receives **all** event types for your account — there is no filtering by event type. For local development, use a tunneling tool (e.g., ngrok) to expose your local server. To stop receiving events, set ``webhook_uri`` to an empty string ``""``.

Troubleshooting
+++++++++++++++

* **No events received via WebSocket:**
    * **Cause:** Subscription topic does not match your customer ID, or subscription was sent before the connection opened.
    * **Fix:** Verify the customer ID in the topic matches your account (from ``GET https://api.voipbin.net/v1.0/customer``). Send the subscribe message only after the ``on_open`` callback fires.

* **No events received via customer webhook:**
    * **Cause:** The ``webhook_uri`` is not publicly accessible, or the endpoint does not return HTTP ``200`` within 5 seconds.
    * **Fix:** Verify the URI is reachable from the internet. For local development, use a tunneling tool (e.g., ngrok). Check your server logs for incoming requests.

* **403 Forbidden on customer update:**
    * **Cause:** The authenticated user lacks admin permission to update the customer.
    * **Fix:** Verify you are using an admin-level token. Regular agents cannot modify customer settings.
