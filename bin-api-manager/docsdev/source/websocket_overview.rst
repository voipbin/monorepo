.. _websocket-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Medium -- Requires persistent connection management, topic subscription, and reconnection logic.
   * **Cost:** Free -- WebSocket connections and event delivery do not incur charges.
   * **Async:** Yes. Connect via ``wss://api.voipbin.net/v1.0/ws?token=<token>``, then send subscribe messages to receive events. Events are pushed asynchronously as they occur on subscribed topics.

VoIPBIN's WebSocket API enables real-time, bi-directional communication for receiving instant event notifications. WebSockets maintain persistent connections, allowing immediate delivery of call status changes, message arrivals, flow updates, and other platform events without polling.

The WebSocket API provides:

- Real-time event streaming for calls, messages, and flows
- Topic-based subscription filtering
- Bi-directional communication channel
- Low-latency event delivery
- Prefix-based topic matching for broad or narrow subscriptions


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
       |     (no ack is sent back --   |                               |
       |      subscribe is fire-and-   |                               |
       |      forget from the client's |                               |
       |      point of view)           |                               |
       |                               |                               |
       |                               |        4. Event occurs        |
       |                               |<------------------------------+
       |                               |                               |
       |   5. Event delivered          |                               |
       |<------------------------------+                               |
       |                               |                               |

    Continuous Connection:
    +-----------------------------------------------------------------------+
    | Unlike HTTP (request-response), WebSocket maintains an open channel   |
    | Events are pushed instantly as they occur - no polling needed         |
    +-----------------------------------------------------------------------+

.. note:: **AI Implementation Hint**

   The topic format is ``<scope>:<scope_id>:<resource>:<resource_id>``. To subscribe to all resources of a type, omit the ``resource_id`` segment entirely (e.g., ``customer_id:abc123:call`` for all call events) -- matching is prefix-based, so a topic without a trailing resource ID matches every event whose topic starts with that prefix. Always implement automatic reconnection with exponential backoff since connections can drop due to network issues or token expiration.

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
                    | - call      |   | - message   |
                    | - queue     |   | - activeflow|
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
    | customer_id:abc123:call             | All calls for customer          |
    | customer_id:abc123:call:xyz789      | Specific call                   |
    | customer_id:abc123:message          | All messages for customer       |
    | agent_id:agent123:queue             | All queues for agent            |
    +-----------------------------------------------------------------------+

    Matching all resources of a type:
    +-----------------------------------------------------------------------+
    | Topic matching is prefix-based: a subscribed topic matches any        |
    | delivered event topic that starts with it. Omit the resource_id       |
    | segment (and its leading colon) to receive every event for that       |
    | resource type.                                                        |
    | customer_id:abc123:call    -> All call events for customer abc123    |
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
Server-pushed events are **not** wrapped in a ``topic``/``timestamp`` envelope. However, the platform is mid-migration to a new event-routing path (backend "Task 4.6" cutover), and two different payload shapes currently coexist on the same topic:

* **Wrapped (legacy fanout path)** -- the classic webhook envelope documented in :ref:`Webhook Struct <webhook-struct-webhook>`: a top-level ``type`` field naming the event (e.g. ``"call_created"``) plus a ``data`` field holding the resource.
* **Unwrapped (new routing-keyed path)** -- the raw resource object only, with **no** top-level ``type``/``data`` wrapper at all. There is no event-action indicator (created/updated/deleted) in this shape; only the resource's own fields (``id``, ``customer_id``, etc.) are present.

Until the migration completes, treat both as possible: check for a top-level ``type`` field first -- if present, unwrap ``data`` per the Webhook Struct docs; if absent, the entire message is the raw resource object for whatever topic you subscribed to.

**Wrapped example**

::

    {
        "type": "call_created",
        "data": {
            "id": "xyz789",
            "customer_id": "abc123",
            ...
        }
    }

**Unwrapped example**

::

    {
        "id": "xyz789",
        "customer_id": "abc123",
        ...
    }

.. note:: **AI Implementation Hint**

   Because neither shape includes a ``topic`` field, correlate an incoming event to the subscription that produced it using the resource fields in the payload itself (``id``, ``customer_id``, ``owner_id``), not by comparing against the topic string you subscribed with. This dual-shape behavior is a temporary migration state, not a stable contract -- expect it to collapse to a single shape once the backend cutover completes.

**Common Event Types**

The webhook event type strings below use the same underscored naming as :ref:`Webhook Struct <webhook-struct-webhook>` (not the dotted ``resource.action`` style used elsewhere in this page for topic *resource* names).

.. list-table::
   :header-rows: 1

   * - Event Type
     - Description
   * - call_created
     - New call initiated
   * - call_ringing
     - Call is ringing
   * - call_progressing
     - Call is in progress (answered)
   * - call_updated
     - Call info updated
   * - call_hangup
     - Call ended
   * - activeflow_created
     - Flow execution started
   * - activeflow_updated
     - Flow action executed
   * - activeflow_deleted
     - Flow execution ended
   * - queue_created
     - Queue created
   * - queue_updated
     - Queue info updated
   * - queuecall_created
     - Call entered a queue
   * - queuecall_connecting
     - Call connecting to an agent
   * - queuecall_serviced
     - Call connected to an agent
   * - agent_status_updated
     - Agent availability changed

See :ref:`Webhook Struct <webhook-struct-webhook>` for the full list of event types and their exact payload shapes.


Common Scenarios
----------------

**Scenario 1: Real-Time Call Dashboard**

Build a live dashboard showing all active calls.

::

    Setup:
    +--------------------------------------------+
    | Subscribe to: customer_id:<id>:call        |
    |                                            |
    | Events received:                           |
    | - call_created  -> Add to active list      |
    | - call_updated  -> Update call state       |
    | - call_hangup   -> Remove from list        |
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
    | agent_id:<agent-id>:queue                  |
    | agent_id:<agent-id>:call                   |
    | customer_id:<cust-id>:message              |
    +--------------------------------------------+

    Event Handling:
    +--------------------------------------------+
    | queuecall_created -> Show notification      |
    |                     "New caller waiting"    |
    |                                            |
    | call_updated      -> Pop customer info     |
    |                     Show call controls      |
    |                                            |
    | message_created   -> Display in chat panel |
    |                     Enable quick reply      |
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
    | customer_id:<id>:message                   |
    +--------------------------------------------+

    Event Processing:
    +--------------------------------------------+
    | 1. Receive message_created event           |
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
- Use specific resource IDs when possible (not the broad resource-type-only prefix)
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
| Not receiving events      | Verify topic format is correct; there is no    |
|                           | subscription acknowledgment message, so        |
|                           | confirm events are actually occurring for      |
|                           | the resource                                   |
+---------------------------+------------------------------------------------+
| Receiving wrong events    | Review topic patterns; remember matching is    |
|                           | prefix-based, so an overly short topic can     |
|                           | match more than intended; check customer_id    |
|                           | is correct                                     |
+---------------------------+------------------------------------------------+
| Connection closes right   | The server closes the WebSocket connection     |
| after sending subscribe   | (no error frame) if a subscribed topic fails   |
|                           | validation -- e.g., a customer_id that does    |
|                           | not match the token, or an agent_id that is    |
|                           | not the token owner. Verify agent_id matches   |
|                           | token owner and customer_id matches the        |
|                           | authenticated customer.                        |
+---------------------------+------------------------------------------------+

**Message Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Malformed messages        | Validate JSON parsing; handle unexpected       |
|                           | fields gracefully; log raw messages            |
+---------------------------+------------------------------------------------+
| Missing event data        | The delivered message has no wrapper --        |
|                           | inspect its own ``type`` field (e.g.           |
|                           | ``call_created``) and resource fields          |
|                           | directly, per :ref:`Webhook Struct             |
|                           | <webhook-struct-webhook>`                      |
+---------------------------+------------------------------------------------+
| Delayed events            | Check client processing time; verify           |
|                           | network latency; monitor server health         |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`WebSocket Tutorial <websocket-tutorial>` - Implementation examples and code
- :ref:`WebSocket Structures <websocket-struct>` - Message format specifications
- :ref:`Authentication Quickstart <quickstart-authentication>` - Token generation
- :ref:`Call Overview <call-overview>` - Call event details
- :ref:`Message Overview <message-overview>` - Message event details
- :ref:`Flow Overview <flow-overview>` - Activeflow event details

