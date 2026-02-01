.. _websocket_overview:

Overview
========
VoIPBIN's WebSocket API enables real-time, bi-directional communication for receiving instant event notifications. WebSockets maintain persistent connections, allowing immediate delivery of call status changes, message arrivals, flow updates, and other platform events without polling.

The WebSocket API provides:

- Real-time event streaming for calls, messages, and flows
- Topic-based subscription filtering
- Bi-directional communication channel
- Low-latency event delivery
- Wildcard pattern matching for subscriptions


How WebSocket Works
-------------------
WebSocket maintains a persistent connection for instant event delivery.

**WebSocket Architecture**

::

    +-----------------------------------------------------------------------+
    |                       WebSocket System                                |
    +-----------------------------------------------------------------------+

    Client                          VoIPBIN                        Services
       |                               |                               |
       | 1. WSS Connection             |                               |
       +------------------------------>|                               |
       |                               |                               |
       |   2. Connection Accepted      |                               |
       |<------------------------------+                               |
       |                               |                               |
       | 3. Subscribe to topics        |                               |
       +------------------------------>|                               |
       |                               |                               |
       |   4. Subscription confirmed   |                               |
       |<------------------------------+                               |
       |                               |                               |
       |                               |        5. Event occurs        |
       |                               |<------------------------------+
       |                               |                               |
       |   6. Event delivered          |                               |
       |<------------------------------+                               |
       |                               |                               |

    Continuous Connection:
    +-----------------------------------------------------------------------+
    | Unlike HTTP (request-response), WebSocket maintains an open channel   |
    | Events are pushed instantly as they occur - no polling needed         |
    +-----------------------------------------------------------------------+

**Key Components**

- **WebSocket Connection**: Persistent bi-directional channel
- **Topics**: Event filters for specific resources
- **Subscriptions**: Active topic registrations
- **Events**: Real-time notifications pushed to clients


Connection Architecture
-----------------------
WebSocket connections integrate with VoIPBIN's event system.

**Event Flow Architecture**

::

    +-----------------------------------------------------------------------+
    |                    WebSocket Event Pipeline                           |
    +-----------------------------------------------------------------------+

    +-------------------+     +-------------------+     +-------------------+
    |   VoIPBIN         |     |   Event           |     |   WebSocket       |
    |   Services        |---->|   Router          |---->|   Handler         |
    +-------------------+     +-------------------+     +-------------------+
                                      |
                                      | Route by topic
                                      v
                              +-------+-------+
                              |               |
                              v               v
                    +-------------+   +-------------+
                    | Client A    |   | Client B    |
                    | Topics:     |   | Topics:     |
                    | - call:*    |   | - msg:*     |
                    | - queue:*   |   | - flow:*    |
                    +-------------+   +-------------+

    Event Publishing:
    +-----------------------------------------------------------------------+
    | 1. Service generates event (call answered, message received, etc.)    |
    | 2. Event router matches topic patterns                                 |
    | 3. Event pushed to all subscribed clients                             |
    +-----------------------------------------------------------------------+


Topic System
------------
Topics define which events a client receives.

**Topic Format**

::

    +-----------------------------------------------------------------------+
    |                         Topic Structure                               |
    +-----------------------------------------------------------------------+

    Format: <scope>:<scope_id>:<resource>:<resource_id>

    Examples:
    +-----------------------------------------------------------------------+
    | customer_id:abc123:call:*           | All calls for customer          |
    | customer_id:abc123:call:xyz789      | Specific call                   |
    | customer_id:abc123:message:*        | All messages for customer       |
    | agent_id:agent123:queue:*           | All queues for agent            |
    +-----------------------------------------------------------------------+

    Wildcard (*):
    +-----------------------------------------------------------------------+
    | Use * to match all resources of a type                                |
    | customer_id:abc123:call:*  -> All call events                        |
    | customer_id:abc123:*:*     -> All events (not recommended)           |
    +-----------------------------------------------------------------------+

**Available Resource Types**

+-------------------+------------------------------------------------------------------+
| Resource          | Description                                                      |
+===================+==================================================================+
| call              | Call status changes, connection, hangup                          |
+-------------------+------------------------------------------------------------------+
| message           | SMS/MMS received, sent, delivery status                          |
+-------------------+------------------------------------------------------------------+
| activeflow        | Flow execution updates, action changes                           |
+-------------------+------------------------------------------------------------------+
| conference        | Conference events, participant join/leave                        |
+-------------------+------------------------------------------------------------------+
| queue             | Queue entry, exit, agent assignment                              |
+-------------------+------------------------------------------------------------------+
| agent             | Agent status changes                                             |
+-------------------+------------------------------------------------------------------+
| recording         | Recording start, stop, completion                                |
+-------------------+------------------------------------------------------------------+
| transcription     | Transcription results                                            |
+-------------------+------------------------------------------------------------------+


Connection Lifecycle
--------------------
WebSocket connections follow a defined lifecycle.

**Connection States**

::

    +-------------------+
    |   Disconnected    |
    +--------+----------+
             |
             | WSS Connect with token
             v
    +-------------------+
    |   Connecting      |
    +--------+----------+
             |
             | Token validated
             v
    +-------------------+     Subscribe
    |   Connected       |<-------------------+
    +--------+----------+                    |
             |                               |
             | Send subscription             |
             v                               |
    +-------------------+     Unsubscribe    |
    |   Subscribed      +--------------------+
    +--------+----------+
             |
             | Connection lost / Close
             v
    +-------------------+
    |   Disconnected    |
    +-------------------+
             |
             | Reconnect with backoff
             v
    +-------------------+
    |   Reconnecting    |
    +-------------------+

**Connection Endpoint**

::

    WebSocket URL:
    +-----------------------------------------------------------------------+
    | wss://api.voipbin.net/v1.0/ws?token=<YOUR_AUTH_TOKEN>                |
    +-----------------------------------------------------------------------+

    Authentication:
    +-----------------------------------------------------------------------+
    | Token passed as query parameter                                       |
    | Same JWT or AccessKey token used for REST API                        |
    +-----------------------------------------------------------------------+


Event Message Format
--------------------
All events follow a consistent structure.

**Event Structure**

::

    {
        "event_type": "call.status",
        "timestamp": "2024-01-15T10:30:00.000000Z",
        "topic": "customer_id:abc123:call:xyz789",
        "data": {
            // Resource-specific payload
        }
    }

**Common Event Types**

+-------------------------+--------------------------------------------------------+
| Event Type              | Description                                            |
+=========================+========================================================+
| call.status             | Call state changed (ringing, answered, hangup)         |
+-------------------------+--------------------------------------------------------+
| call.created            | New call initiated                                     |
+-------------------------+--------------------------------------------------------+
| message.received        | Incoming SMS/MMS received                              |
+-------------------------+--------------------------------------------------------+
| message.sent            | Outgoing message sent                                  |
+-------------------------+--------------------------------------------------------+
| message.delivery        | Message delivery status update                         |
+-------------------------+--------------------------------------------------------+
| activeflow.updated      | Flow action executed                                   |
+-------------------------+--------------------------------------------------------+
| activeflow.completed    | Flow execution completed                               |
+-------------------------+--------------------------------------------------------+
| conference.joined       | Participant joined conference                          |
+-------------------------+--------------------------------------------------------+
| conference.left         | Participant left conference                            |
+-------------------------+--------------------------------------------------------+
| queue.joined            | Call entered queue                                     |
+-------------------------+--------------------------------------------------------+
| queue.connected         | Call connected to agent                                |
+-------------------------+--------------------------------------------------------+
| agent.status            | Agent availability changed                             |
+-------------------------+--------------------------------------------------------+
| recording.completed     | Recording finished and available                       |
+-------------------------+--------------------------------------------------------+


Common Scenarios
----------------

**Scenario 1: Real-Time Call Dashboard**

Build a live dashboard showing all active calls.

::

    Setup:
    +--------------------------------------------+
    | Subscribe to: customer_id:<id>:call:*      |
    |                                            |
    | Events received:                           |
    | - call.created  -> Add to active list      |
    | - call.status   -> Update call state       |
    | - call.ended    -> Remove from list        |
    +--------------------------------------------+

    Dashboard Updates:
    +--------------------------------------------+
    | Incoming Call  |  +1-555-1234  |  Ringing  |
    | Active Call    |  +1-555-5678  |  Answered |
    | In Queue       |  +1-555-9012  |  Waiting  |
    +--------------------------------------------+

    Benefits:
    +--------------------------------------------+
    | - Instant visibility into call status      |
    | - No polling required                      |
    | - Real-time metrics and KPIs               |
    +--------------------------------------------+

**Scenario 2: Agent Desktop Application**

Power a contact center agent interface.

::

    Subscriptions:
    +--------------------------------------------+
    | agent_id:<agent-id>:queue:*                |
    | agent_id:<agent-id>:call:*                 |
    | customer_id:<cust-id>:message:*            |
    +--------------------------------------------+

    Event Handling:
    +--------------------------------------------+
    | queue.joined     -> Show notification       |
    |                    "New caller waiting"     |
    |                                            |
    | call.assigned    -> Pop customer info      |
    |                    Show call controls       |
    |                                            |
    | message.received -> Display in chat panel  |
    |                    Enable quick reply       |
    +--------------------------------------------+

    Agent Interface:
    +--------------------------------------------+
    | [New Call Alert]      [Customer: John D.]  |
    | Queue: Sales          Previous: 3 calls    |
    | Wait time: 45s        Last purchase: $299  |
    |                                            |
    | [Accept Call] [Transfer] [Send to VM]     |
    +--------------------------------------------+

**Scenario 3: Message Auto-Response System**

Automatically respond to incoming messages.

::

    Subscription:
    +--------------------------------------------+
    | customer_id:<id>:message:*                 |
    +--------------------------------------------+

    Event Processing:
    +--------------------------------------------+
    | 1. Receive message.received event          |
    | 2. Analyze message content                 |
    | 3. Match against response rules            |
    | 4. Send appropriate auto-reply             |
    +--------------------------------------------+

    Example Flow:
    +--------------------------------------------+
    | Incoming: "What are your hours?"           |
    |                                            |
    | -> Match keyword: "hours"                  |
    | -> Auto-reply: "We're open Mon-Fri 9-5!"  |
    |                                            |
    | Incoming: "STOP"                           |
    |                                            |
    | -> Match keyword: "STOP"                   |
    | -> Unsubscribe user from messages          |
    | -> Confirm: "You've been unsubscribed"    |
    +--------------------------------------------+


Best Practices
--------------

**1. Connection Management**

- Implement automatic reconnection with exponential backoff
- Start with 1 second delay, double on each failure (max 30 seconds)
- Monitor connection health with ping/pong messages
- Handle network transitions gracefully (WiFi to cellular)

**2. Subscription Strategy**

- Subscribe only to events your application needs
- Use specific resource IDs when possible (not just wildcards)
- Unsubscribe when events are no longer needed
- Resubscribe after reconnection

**3. Error Handling**

- Handle connection errors gracefully
- Parse messages safely with try/catch
- Log all errors for debugging
- Implement timeout handling for stale connections

**4. Performance Optimization**

- Process events asynchronously for heavy operations
- Batch UI updates to avoid excessive re-rendering
- Use message queues for high-volume scenarios
- Avoid blocking operations in event handlers


Troubleshooting
---------------

**Connection Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Connection refused        | Verify token is valid; check endpoint URL;     |
|                           | ensure WSS (not WS) for production             |
+---------------------------+------------------------------------------------+
| Connection drops          | Implement reconnection logic; check network;   |
|                           | verify token hasn't expired                    |
+---------------------------+------------------------------------------------+
| Authentication failure    | Token may be expired; regenerate token;        |
|                           | verify token has WebSocket permissions         |
+---------------------------+------------------------------------------------+

**Subscription Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Not receiving events      | Verify topic format is correct; check          |
|                           | subscription was acknowledged; confirm events  |
|                           | are actually occurring                         |
+---------------------------+------------------------------------------------+
| Receiving wrong events    | Review topic patterns; avoid overly broad      |
|                           | wildcards; check customer_id is correct        |
+---------------------------+------------------------------------------------+
| Permission denied         | Verify user has access to the resource;        |
|                           | check agent_id matches token owner             |
+---------------------------+------------------------------------------------+

**Message Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Malformed messages        | Validate JSON parsing; handle unexpected       |
|                           | fields gracefully; log raw messages            |
+---------------------------+------------------------------------------------+
| Missing event data        | Check event_type for proper handling;          |
|                           | verify data field exists before access         |
+---------------------------+------------------------------------------------+
| Delayed events            | Check client processing time; verify           |
|                           | network latency; monitor server health         |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`WebSocket Tutorial <websocket-tutorial>` - Implementation examples and code
- :ref:`WebSocket Structures <websocket_struct>` - Message format specifications
- :ref:`Authentication Quickstart <quickstart_authentication>` - Token generation
- :ref:`Call Overview <call-overview>` - Call event details
- :ref:`Message Overview <message-overview>` - Message event details
- :ref:`Flow Overview <flow-overview>` - Activeflow event details

