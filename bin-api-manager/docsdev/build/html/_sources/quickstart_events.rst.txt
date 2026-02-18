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
