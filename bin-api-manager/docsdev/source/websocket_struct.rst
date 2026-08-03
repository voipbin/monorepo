.. _websocket-struct:

Structures
==========
This section documents the message structures used in VoIPBIN's WebSocket API for subscription management and event delivery.


Message Overview
----------------
WebSocket communication uses JSON messages for all operations.

**Message Categories**

::

    +-----------------------------------------------------------------------+
    |                    WebSocket Message Types                            |
    +-----------------------------------------------------------------------+

    Client -> Server:
    +-----------------------------------------------------------------------+
    | subscribe     | Register for event notifications                      |
    | unsubscribe   | Stop receiving event notifications                    |
    +-----------------------------------------------------------------------+

    Server -> Client:
    +-----------------------------------------------------------------------+
    | (unwrapped webhook payload) | Real-time event notification. Pushed as |
    |                             | the raw resource payload with no        |
    |                             | envelope -- see "Event Message          |
    |                             | Structure" below.                       |
    +-----------------------------------------------------------------------+

.. note:: **AI Implementation Hint**

   There is no server-to-client acknowledgment or error message for ``subscribe``/``unsubscribe`` requests. If a subscribed topic fails permission validation, the server closes the WebSocket connection instead of sending an error frame.


Subscribe Message
-----------------
Send a subscribe message to receive events for specific topics.

**Structure**

.. code::

    {
        "type": "subscribe",
        "topics": [
            "<topic-pattern-1>",
            "<topic-pattern-2>",
            ...
        ]
    }

**Fields**

* ``type`` (enum string): Must be ``"subscribe"``.
* ``topics`` (Array of String): List of topic patterns to subscribe to. Each topic follows the format ``<scope>:<scope_id>:<resource_type>:<resource_id>``. Omit the ``resource_id`` segment (and its leading colon) to match all resources of a type -- see :ref:`Topic Pattern Structure <websocket-struct>` below.

**Example: Subscribe to all calls**

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:12345678-1234-1234-1234-123456789012:call"
        ]
    }

**Example: Subscribe to multiple resource types**

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:12345678-1234-1234-1234-123456789012:call",
            "customer_id:12345678-1234-1234-1234-123456789012:message",
            "customer_id:12345678-1234-1234-1234-123456789012:activeflow"
        ]
    }

**Example: Subscribe to specific resource**

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:12345678-1234-1234-1234-123456789012:call:a1b2c3d4-e5f6-7890-abcd-ef1234567890"
        ]
    }

**Example: Agent-level subscription**

.. code::

    {
        "type": "subscribe",
        "topics": [
            "agent_id:98765432-4321-4321-4321-210987654321:queue",
            "agent_id:98765432-4321-4321-4321-210987654321:call"
        ]
    }


Unsubscribe Message
-------------------
Send an unsubscribe message to stop receiving events for specific topics.

**Structure**

.. code::

    {
        "type": "unsubscribe",
        "topics": [
            "<topic-pattern-1>",
            "<topic-pattern-2>",
            ...
        ]
    }

**Fields**

* ``type`` (enum string): Must be ``"unsubscribe"``.
* ``topics`` (Array of String): List of topic patterns to unsubscribe from. Must match the exact topic patterns used during subscription.

**Example: Unsubscribe from calls**

.. code::

    {
        "type": "unsubscribe",
        "topics": [
            "customer_id:12345678-1234-1234-1234-123456789012:call"
        ]
    }

**Example: Unsubscribe from specific resource**

.. code::

    {
        "type": "unsubscribe",
        "topics": [
            "customer_id:12345678-1234-1234-1234-123456789012:call:a1b2c3d4-e5f6-7890-abcd-ef1234567890"
        ]
    }


Topic Pattern Structure
-----------------------
Topics follow a consistent format for event filtering.

**Topic Format**

::

    <scope>:<scope_id>:<resource_type>:<resource_id>

**Topic Components**

* ``scope`` (enum string): Access level. Either ``"customer_id"`` or ``"agent_id"``.
* ``scope_id`` (UUID): The UUID of the customer or agent. Obtained from ``GET /customers`` or ``GET /agents``. For ``agent_id`` scope, this must match the authenticated agent's own ID.
* ``resource_type`` (String): Type of resource, derived from the first underscore-delimited segment of the resource's webhook event type (e.g. ``call`` from ``call_created``, ``activeflow`` from ``activeflow_updated``, ``queuecall`` from ``queuecall_created``). See :ref:`Webhook Struct <webhook-struct-webhook>` for the full list of event types.
* ``resource_id`` (UUID, optional): UUID of a specific resource. Omit this segment (and its leading colon) to match every event of the given resource type -- see "Matching All Resources of a Type" below.

.. note:: **AI Implementation Hint**

   There is no ``*`` wildcard character. Matching is prefix-based: the server delivers an event to a subscription whenever the event's topic string starts with the subscribed topic string. To subscribe to a single resource, use the full 4-part topic with a real UUID. To subscribe to every resource of a type, subscribe with just the 3-part prefix (no trailing resource ID).

**Valid Scopes**

.. list-table::
   :header-rows: 1

   * - Scope
     - Permission Required
   * - customer_id
     - Admin or Manager permission for the customer (non-Direct tokens)
   * - agent_id
     - ``scope_id`` must equal the authenticated agent's own ID

**Common Resource Types**

.. list-table::
   :header-rows: 1

   * - Resource Type
     - Example Events (see Webhook Struct for the full list)
   * - call
     - call_created, call_ringing, call_progressing, call_updated, call_hangup
   * - message
     - message_created, message_updated
   * - activeflow
     - activeflow_created, activeflow_updated, activeflow_deleted
   * - queue
     - queue_created, queue_updated, queue_deleted
   * - queuecall
     - queuecall_created, queuecall_connecting, queuecall_serviced, queuecall_done, queuecall_abandoned
   * - agent
     - agent_created, agent_updated, agent_status_updated
   * - chat
     - chat_created, chatmessage_created, chatparticipant_added

**Matching All Resources of a Type**

::

    +-----------------------------------------------------------------------+
    |                    Prefix-Based Topic Matching                        |
    +-----------------------------------------------------------------------+

    Specific resource:
    customer_id:abc123:call:xyz789
    -> Only events for call xyz789

    All resources of type (omit the resource_id segment):
    customer_id:abc123:call
    -> All call events for customer abc123

    Multiple types (separate subscriptions):
    customer_id:abc123:call
    customer_id:abc123:message
    -> All calls AND all messages


Event Message Structure
-----------------------
Events are pushed from the server when subscribed resources change. The pushed text frame is **not** wrapped in a topic/timestamp envelope. The platform is currently mid-migration to a new event-routing path, so two payload shapes coexist on the same topic until the migration completes -- see :ref:`Event Message Format <websocket-overview>` for the full explanation. In short:

* **Wrapped (legacy)** -- the classic webhook envelope from :ref:`Webhook Struct <webhook-struct-webhook>`: an object with a ``type`` field naming the event and a ``data`` field holding the resource.
* **Unwrapped (new)** -- just the raw resource object, with no ``type``/``data`` wrapper and no event-action indicator.

**Wrapped structure**

.. code::

    {
        "type": "<event-type>",
        "data": {
            // Resource-specific payload, e.g. the Call struct for call_* events
        }
    }

**Unwrapped structure**

.. code::

    {
        // Resource-specific payload directly at the top level, e.g. the Call struct itself for call_* events
    }

**Fields (wrapped shape)**

* ``type`` (enum string): The webhook event type (e.g., ``"call_created"``, ``"call_updated"``). See :ref:`Webhook Struct <webhook-struct-webhook>` for the full list.
* ``data`` (Object): Resource-specific data payload. Structure varies by ``type`` and corresponds to the relevant resource struct (e.g., :ref:`Call <call-struct-call>` for ``call_*`` events).

.. note:: **AI Implementation Hint**

   Check for a top-level ``type`` field first to distinguish the two shapes: if present, unwrap ``data`` as above; if absent, treat the whole message as the raw resource. Neither shape has a separate ``event_type``, ``topic``, or ``timestamp`` field -- use the resource's own ``tm_create``/``tm_update`` fields for timing, and its own ``id``/``customer_id``/``owner_id`` fields to correlate the event. This dual-shape behavior is a temporary migration state, not a stable contract.


Event Type Reference
--------------------
The wrapped WebSocket event payloads are identical to the corresponding webhook payloads. Below is one representative example per resource type (wrapped shape); see :ref:`Webhook Struct <webhook-struct-webhook>` for the complete, authoritative list of event types and full field sets.

**Call Events**

.. code::

    // call_created
    {
        "type": "call_created",
        "data": {
            "id": "5371e9db-d035-4db6-a8d6-0994d33e744e",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "status": "ringing",
            "direction": "incoming",
            "tm_create": "2022-04-11 00:23:53.636000",
            "tm_update": "9999-01-01 00:00:00.000000"
        }
    }

    // call_hangup
    {
        "type": "call_hangup",
        "data": {
            "id": "5371e9db-d035-4db6-a8d6-0994d33e744e",
            "status": "terminated",
            "hangup_by": "remote",
            "hangup_reason": "normal",
            "tm_hangup": "2022-04-11 00:25:10.000000"
        }
    }

**Activeflow Events**

.. code::

    // activeflow_updated
    {
        "type": "activeflow_updated",
        "data": {
            "id": "flow789",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "tm_update": "2024-01-15 10:30:00.000000"
        }
    }

**Queue Events**

.. code::

    // queuecall_created
    {
        "type": "queuecall_created",
        "data": {
            "id": "queuecall789",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "queue_id": "queue789",
            "reference_id": "call456",
            "tm_create": "2024-01-15 10:30:00.000000"
        }
    }

**Agent Events**

.. code::

    // agent_status_updated
    {
        "type": "agent_status_updated",
        "data": {
            "id": "agent123",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "status": "available",
            "tm_update": "2024-01-15 10:30:00.000000"
        }
    }

**Talk (Chat) Events**

.. code::

    // chatmessage_created
    {
        "type": "chatmessage_created",
        "data": {
            "id": "a3c5e8f2-4d3a-4b5c-9e7f-1a2b3c4d5e6f",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
            "text": "Hello team!",
            "tm_create": "2024-01-17 10:32:00.000000"
        }
    }


Acknowledgment Messages
-----------------------
The server does **not** send acknowledgment or error messages for ``subscribe``/``unsubscribe`` requests. There is no ``ack`` or ``error`` message type in the current implementation.

If a subscribed topic fails permission validation (e.g., a ``customer_id`` that does not match the caller's own customer, or an ``agent_id`` that is not the authenticated agent), the server closes the WebSocket connection outright rather than returning an error frame. Design your client to detect an unexpected close shortly after sending a ``subscribe`` message as a signal of a permission or format problem, and to reconnect with corrected topics.


Message Handling Examples
-------------------------
Code examples for processing WebSocket messages. Since server-pushed events reuse the ``type``/``data`` shape but with resource event-type values (not ``"ack"``/``"error"``, which do not exist), a simple switch on ``type`` is enough to distinguish them from client-sent message echoes (which a well-behaved client should not receive back, since the server never echoes ``subscribe``/``unsubscribe`` messages).

**JavaScript**

.. code::

    ws.onmessage = function(event) {
        const message = JSON.parse(event.data);

        switch(message.type) {
            case 'call_created':
            case 'call_ringing':
            case 'call_progressing':
            case 'call_updated':
            case 'call_hangup':
                handleCallEvent(message.type, message.data);
                break;

            case 'message_created':
                handleMessage(message.data);
                break;

            default:
                console.log('Received:', message.type, message.data);
        }
    };

**Python**

.. code::

    def on_message(ws, raw_message):
        message = json.loads(raw_message)

        msg_type = message.get('type')

        if msg_type and msg_type.startswith('call_'):
            handle_call_event(msg_type, message['data'])

        elif msg_type == 'message_created':
            handle_message(message['data'])

        else:
            print(f"Received: {msg_type}")


Related Documentation
---------------------

- :ref:`WebSocket Overview <websocket-overview>` - Connection and topic concepts
- :ref:`WebSocket Tutorial <websocket-tutorial>` - Implementation examples
- :ref:`Webhook Struct <webhook-struct-webhook>` - Authoritative list of event types and payload shapes (identical to what is pushed over the WebSocket)
- :ref:`Call Struct <call-struct-call>` - Complete call data structure
- :ref:`Message Struct <message-struct-message>` - Complete message data structure
- :ref:`Activeflow Struct <activeflow-struct-activeflow>` - Complete activeflow data structure

