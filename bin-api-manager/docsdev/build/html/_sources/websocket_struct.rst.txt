.. _websocket_struct:

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
    | event         | Real-time event notification with resource data       |
    | ack           | Acknowledgment of subscription changes                |
    | error         | Error message for failed operations                   |
    +-----------------------------------------------------------------------+


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

+-------------------+----------+--------------------------------------------------+
| Field             | Type     | Description                                      |
+===================+==========+==================================================+
| type              | string   | Must be "subscribe"                              |
+-------------------+----------+--------------------------------------------------+
| topics            | array    | List of topic patterns to subscribe to           |
+-------------------+----------+--------------------------------------------------+

**Example: Subscribe to all calls**

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:12345678-1234-1234-1234-123456789012:call:*"
        ]
    }

**Example: Subscribe to multiple resource types**

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:12345678-1234-1234-1234-123456789012:call:*",
            "customer_id:12345678-1234-1234-1234-123456789012:message:*",
            "customer_id:12345678-1234-1234-1234-123456789012:activeflow:*"
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
            "agent_id:98765432-4321-4321-4321-210987654321:queue:*",
            "agent_id:98765432-4321-4321-4321-210987654321:call:*"
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

+-------------------+----------+--------------------------------------------------+
| Field             | Type     | Description                                      |
+===================+==========+==================================================+
| type              | string   | Must be "unsubscribe"                            |
+-------------------+----------+--------------------------------------------------+
| topics            | array    | List of topic patterns to unsubscribe from       |
+-------------------+----------+--------------------------------------------------+

**Example: Unsubscribe from calls**

.. code::

    {
        "type": "unsubscribe",
        "topics": [
            "customer_id:12345678-1234-1234-1234-123456789012:call:*"
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

+-------------------+------------------------------------------------------------------+
| Component         | Description                                                      |
+===================+==================================================================+
| scope             | Access level: "customer_id" or "agent_id"                        |
+-------------------+------------------------------------------------------------------+
| scope_id          | UUID of the customer or agent                                    |
+-------------------+------------------------------------------------------------------+
| resource_type     | Type of resource: call, message, activeflow, conference, etc.    |
+-------------------+------------------------------------------------------------------+
| resource_id       | UUID of specific resource or "*" for all                         |
+-------------------+------------------------------------------------------------------+

**Valid Scopes**

+-------------------+------------------------------------------------------------------+
| Scope             | Permission Required                                              |
+===================+==================================================================+
| customer_id       | Admin or Manager permission for the customer                     |
+-------------------+------------------------------------------------------------------+
| agent_id          | Must be the owner of the agent                                   |
+-------------------+------------------------------------------------------------------+

**Valid Resource Types**

+-------------------+------------------------------------------------------------------+
| Resource Type     | Events                                                           |
+===================+==================================================================+
| call              | call.created, call.status, call.ended                            |
+-------------------+------------------------------------------------------------------+
| message           | message.received, message.sent, message.delivery                 |
+-------------------+------------------------------------------------------------------+
| activeflow        | activeflow.updated, activeflow.completed                         |
+-------------------+------------------------------------------------------------------+
| conference        | conference.joined, conference.left, conference.ended             |
+-------------------+------------------------------------------------------------------+
| queue             | queue.joined, queue.connected, queue.left                        |
+-------------------+------------------------------------------------------------------+
| agent             | agent.status                                                     |
+-------------------+------------------------------------------------------------------+
| recording         | recording.started, recording.completed                           |
+-------------------+------------------------------------------------------------------+
| transcription     | transcription.completed                                          |
+-------------------+------------------------------------------------------------------+

**Wildcard Usage**

::

    +-----------------------------------------------------------------------+
    |                         Wildcard Patterns                             |
    +-----------------------------------------------------------------------+

    Specific resource:
    customer_id:abc123:call:xyz789
    -> Only events for call xyz789

    All resources of type:
    customer_id:abc123:call:*
    -> All call events for customer abc123

    Multiple types (separate subscriptions):
    customer_id:abc123:call:*
    customer_id:abc123:message:*
    -> All calls AND all messages


Event Message Structure
-----------------------
Events are pushed from the server when subscribed resources change.

**Structure**

.. code::

    {
        "event_type": "<event-type>",
        "timestamp": "<ISO-8601-timestamp>",
        "topic": "<topic-that-matched>",
        "data": {
            // Resource-specific payload
        }
    }

**Fields**

+-------------------+----------+--------------------------------------------------+
| Field             | Type     | Description                                      |
+===================+==========+==================================================+
| event_type        | string   | Type of event (e.g., "call.status")              |
+-------------------+----------+--------------------------------------------------+
| timestamp         | string   | ISO 8601 timestamp in UTC                        |
+-------------------+----------+--------------------------------------------------+
| topic             | string   | Topic pattern that triggered this event          |
+-------------------+----------+--------------------------------------------------+
| data              | object   | Resource-specific data payload                   |
+-------------------+----------+--------------------------------------------------+


Event Type Reference
--------------------
Complete list of event types and their data structures.

**Call Events**

.. code::

    // call.created
    {
        "event_type": "call.created",
        "timestamp": "2024-01-15T10:30:00.000000Z",
        "topic": "customer_id:abc123:call:xyz789",
        "data": {
            "id": "xyz789",
            "customer_id": "abc123",
            "direction": "inbound",
            "source": {
                "type": "tel",
                "target": "+15551234567"
            },
            "destination": {
                "type": "tel",
                "target": "+15559876543"
            },
            "status": "ringing",
            "tm_create": "2024-01-15T10:30:00.000000Z"
        }
    }

    // call.status
    {
        "event_type": "call.status",
        "timestamp": "2024-01-15T10:30:05.000000Z",
        "topic": "customer_id:abc123:call:xyz789",
        "data": {
            "id": "xyz789",
            "status": "answered",
            "previous_status": "ringing",
            "tm_update": "2024-01-15T10:30:05.000000Z"
        }
    }

**Message Events**

.. code::

    // message.received
    {
        "event_type": "message.received",
        "timestamp": "2024-01-15T10:30:00.000000Z",
        "topic": "customer_id:abc123:message:msg789",
        "data": {
            "id": "msg789",
            "customer_id": "abc123",
            "direction": "inbound",
            "source": {
                "type": "tel",
                "target": "+15551234567"
            },
            "destination": {
                "type": "tel",
                "target": "+15559876543"
            },
            "text": "Hello, I need help with my order",
            "tm_create": "2024-01-15T10:30:00.000000Z"
        }
    }

    // message.delivery
    {
        "event_type": "message.delivery",
        "timestamp": "2024-01-15T10:30:02.000000Z",
        "topic": "customer_id:abc123:message:msg789",
        "data": {
            "id": "msg789",
            "status": "delivered",
            "tm_update": "2024-01-15T10:30:02.000000Z"
        }
    }

**Activeflow Events**

.. code::

    // activeflow.updated
    {
        "event_type": "activeflow.updated",
        "timestamp": "2024-01-15T10:30:00.000000Z",
        "topic": "customer_id:abc123:activeflow:flow789",
        "data": {
            "id": "flow789",
            "flow_id": "template123",
            "status": "executing",
            "current_action": {
                "id": "action456",
                "type": "play",
                "name": "welcome_message"
            },
            "tm_update": "2024-01-15T10:30:00.000000Z"
        }
    }

    // activeflow.completed
    {
        "event_type": "activeflow.completed",
        "timestamp": "2024-01-15T10:35:00.000000Z",
        "topic": "customer_id:abc123:activeflow:flow789",
        "data": {
            "id": "flow789",
            "flow_id": "template123",
            "status": "completed",
            "result": "success",
            "tm_end": "2024-01-15T10:35:00.000000Z"
        }
    }

**Queue Events**

.. code::

    // queue.joined
    {
        "event_type": "queue.joined",
        "timestamp": "2024-01-15T10:30:00.000000Z",
        "topic": "customer_id:abc123:queue:queue789",
        "data": {
            "queue_id": "queue789",
            "call_id": "call456",
            "position": 3,
            "estimated_wait": 120,
            "tm_join": "2024-01-15T10:30:00.000000Z"
        }
    }

    // queue.connected
    {
        "event_type": "queue.connected",
        "timestamp": "2024-01-15T10:32:00.000000Z",
        "topic": "customer_id:abc123:queue:queue789",
        "data": {
            "queue_id": "queue789",
            "call_id": "call456",
            "agent_id": "agent123",
            "wait_time": 120,
            "tm_connect": "2024-01-15T10:32:00.000000Z"
        }
    }

**Conference Events**

.. code::

    // conference.joined
    {
        "event_type": "conference.joined",
        "timestamp": "2024-01-15T10:30:00.000000Z",
        "topic": "customer_id:abc123:conference:conf789",
        "data": {
            "conference_id": "conf789",
            "call_id": "call456",
            "participant_count": 3,
            "tm_join": "2024-01-15T10:30:00.000000Z"
        }
    }

    // conference.left
    {
        "event_type": "conference.left",
        "timestamp": "2024-01-15T10:45:00.000000Z",
        "topic": "customer_id:abc123:conference:conf789",
        "data": {
            "conference_id": "conf789",
            "call_id": "call456",
            "participant_count": 2,
            "reason": "hangup",
            "tm_leave": "2024-01-15T10:45:00.000000Z"
        }
    }

**Agent Events**

.. code::

    // agent.status
    {
        "event_type": "agent.status",
        "timestamp": "2024-01-15T10:30:00.000000Z",
        "topic": "agent_id:agent123:agent:agent123",
        "data": {
            "agent_id": "agent123",
            "status": "available",
            "previous_status": "busy",
            "tm_update": "2024-01-15T10:30:00.000000Z"
        }
    }

**Recording Events**

.. code::

    // recording.completed
    {
        "event_type": "recording.completed",
        "timestamp": "2024-01-15T10:45:00.000000Z",
        "topic": "customer_id:abc123:recording:rec789",
        "data": {
            "id": "rec789",
            "call_id": "call456",
            "duration": 300,
            "format": "wav",
            "size": 2400000,
            "reference_url": "https://storage.voipbin.net/recordings/rec789.wav",
            "tm_complete": "2024-01-15T10:45:00.000000Z"
        }
    }


Acknowledgment Messages
-----------------------
Server may send acknowledgments for subscription operations.

**Success Acknowledgment**

.. code::

    {
        "type": "ack",
        "action": "subscribe",
        "topics": [
            "customer_id:abc123:call:*"
        ],
        "status": "success"
    }

**Error Response**

.. code::

    {
        "type": "error",
        "action": "subscribe",
        "topics": [
            "customer_id:abc123:call:*"
        ],
        "code": "PERMISSION_DENIED",
        "message": "You do not have permission to subscribe to this topic"
    }

**Error Codes**

+-------------------------+--------------------------------------------------------+
| Code                    | Description                                            |
+=========================+========================================================+
| PERMISSION_DENIED       | User lacks permission for the topic                    |
+-------------------------+--------------------------------------------------------+
| INVALID_TOPIC           | Topic format is invalid                                |
+-------------------------+--------------------------------------------------------+
| INVALID_MESSAGE         | Message structure is invalid                           |
+-------------------------+--------------------------------------------------------+
| RATE_LIMITED            | Too many subscription requests                         |
+-------------------------+--------------------------------------------------------+


Message Handling Examples
-------------------------
Code examples for processing WebSocket messages.

**JavaScript**

.. code::

    ws.onmessage = function(event) {
        const message = JSON.parse(event.data);

        switch(message.type || message.event_type) {
            case 'ack':
                console.log('Subscription confirmed:', message.topics);
                break;

            case 'error':
                console.error('Error:', message.code, message.message);
                break;

            case 'call.status':
                handleCallStatus(message.data);
                break;

            case 'message.received':
                handleMessage(message.data);
                break;

            default:
                console.log('Received:', message.event_type, message.data);
        }
    };

**Python**

.. code::

    def on_message(ws, raw_message):
        message = json.loads(raw_message)

        msg_type = message.get('type') or message.get('event_type')

        if msg_type == 'ack':
            print(f"Subscription confirmed: {message['topics']}")

        elif msg_type == 'error':
            print(f"Error: {message['code']} - {message['message']}")

        elif msg_type == 'call.status':
            handle_call_status(message['data'])

        elif msg_type == 'message.received':
            handle_message(message['data'])

        else:
            print(f"Received: {msg_type}")


Related Documentation
---------------------

- :ref:`WebSocket Overview <websocket_overview>` - Connection and topic concepts
- :ref:`WebSocket Tutorial <websocket-tutorial>` - Implementation examples
- :ref:`Call Struct <call-struct-call>` - Complete call data structure
- :ref:`Message Struct <message-struct-message>` - Complete message data structure
- :ref:`Activeflow Struct <activeflow-struct-activeflow>` - Complete activeflow data structure

