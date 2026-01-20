.. _architecture-communication:

Inter-Service Communication
============================

VoIPBIN's microservices communicate through multiple messaging patterns optimized for different use cases. The architecture uses RabbitMQ for RPC and pub/sub, ZeroMQ for high-performance events, and WebSocket for real-time client communication.

Communication Patterns Overview
--------------------------------

VoIPBIN uses three primary communication mechanisms:

.. code::

    Communication Architecture:

    ┌─────────────────────────────────────────────────────────┐
    │                  RabbitMQ (Primary Bus)                 │
    │                                                         │
    │  ┌───────────────────────┐  ┌───────────────────────┐   │
    │  │   RPC (Synchronous)   │  │  Pub/Sub (Async)      │   │
    │  │   Request-Response    │  │  Event Broadcasting   │   │
    │  └───────────────────────┘  └───────────────────────┘   │
    └─────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────┐
    │              ZeroMQ (High-Performance Events)           │
    │                                                         │
    │  • Real-time event streaming                            │
    │  • Agent presence updates                               │
    │  • Call state changes                                   │
    └─────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────┐
    │              WebSocket (Client Communication)           │
    │                                                         │
    │  • Real-time client notifications                       │
    │  • Bi-directional media streaming                       │
    │  • Live transcription feeds                             │
    └─────────────────────────────────────────────────────────┘

RabbitMQ RPC Pattern
--------------------

VoIPBIN uses RabbitMQ for synchronous request-response communication between services.

**RPC Flow**

.. code::

    RPC Request-Response Pattern:

    Client Service          RabbitMQ             Server Service
         │                     │                       │
         │  1. Send Request    │                       │
         │  ┌────────────┐     │                       │
         │  │ call_id    │     │                       │
         │  │ action     │     │                       │
         │  │ reply_to   │     │                       │
         │  └────────────┘     │                       │
         ├────────────────────▶│                       │
         │  Queue: bin-manager.│                       │
         │         call.request│                       │
         │                     │  2. Dequeue           │
         │                     ├──────────────────────▶│
         │                     │                       │
         │                     │  3. Process Request   │
         │                     │     (business logic)  │
         │                     │                       │
         │                     │  4. Send Response     │
         │                     │◀──────────────────────┤
         │                     │  Queue: reply_to      │
         │  5. Receive Response│                       │
         │◀────────────────────┤                       │
         │  ┌────────────┐     │                       │
         │  │ status     │     │                       │
         │  │ data       │     │                       │
         │  │ error      │     │                       │
         │  └────────────┘     │                       │
         │                     │                       │

**Queue Naming Convention**

All RPC queues follow a consistent naming pattern:

.. code::

    Queue Name Format:
    bin-manager.<service>.<operation>

    Examples:
    • bin-manager.call.request        → bin-call-manager
    • bin-manager.conference.request  → bin-conference-manager
    • bin-manager.sms.request         → bin-sms-manager
    • bin-manager.flow.request        → bin-flow-manager
    • bin-manager.billing.request     → bin-billing-manager

**Message Structure**

RPC messages use a standardized JSON format:

.. code::

    Request Message:
    {
      "message_id": "uuid-v4",
      "timestamp": "2026-01-20T12:00:00.000Z",
      "route": "/v1/calls",
      "method": "POST",
      "headers": {
        "customer_id": "customer-123",
        "agent_id": "agent-456"
      },
      "body": {
        "source": {"type": "tel", "target": "+15551234567"},
        "destinations": [{"type": "tel", "target": "+15559876543"}]
      }
    }

    Response Message:
    {
      "message_id": "uuid-v4",
      "timestamp": "2026-01-20T12:00:01.000Z",
      "status_code": 200,
      "body": {
        "id": "call-789",
        "status": "ringing",
        ...
      },
      "error": null
    }

**RPC Implementation Pattern**

Services implement RPC handlers following this pattern:

.. code::

    Service RPC Handler:

    ┌────────────────────────────────────────────────┐
    │        bin-call-manager                        │
    │                                                │
    │  1. Listen on Queue                            │
    │     ├─ bin-manager.call.request                │
    │     │                                          │
    │  2. Receive Message                            │
    │     ├─ Deserialize JSON                        │
    │     ├─ Validate request                        │
    │     │                                          │
    │  3. Route to Handler                           │
    │     ├─ Parse route: POST /v1/calls             │
    │     ├─ Call: CallCreate(ctx, req)              │
    │     │                                          │
    │  4. Execute Business Logic                     │
    │     ├─ Validate data                           │
    │     ├─ Create call record                      │
    │     ├─ Initiate SIP call                       │
    │     │                                          │
    │  5. Send Response                              │
    │     ├─ Serialize result                        │
    │     └─ Reply to reply_to queue                 │
    │                                                │
    └────────────────────────────────────────────────┘

**Load Balancing**

Multiple service instances share the same queue:

.. code::

    Load Balanced RPC:

    API Gateway                Queue              Service Instances
         │                      │                       │
         │  Request 1           │                       │
         ├─────────────────────▶│                       │
         │                      ├──────────────────────▶│ Instance 1
         │                      │  (round-robin)        │ (processes req 1)
         │                      │                       │
         │  Request 2           │                       │
         ├─────────────────────▶│                       │
         │                      ├──────────────────────▶│ Instance 2
         │                      │  (round-robin)        │ (processes req 2)
         │                      │                       │
         │  Request 3           │                       │
         ├─────────────────────▶│                       │
         │                      ├──────────────────────▶│ Instance 3
         │                      │  (round-robin)        │ (processes req 3)
         │                      │                       │

* **Fair Distribution**: RabbitMQ distributes messages evenly
* **No Coordination**: Instances don't need to know about each other
* **Dynamic Scaling**: Add/remove instances without configuration
* **Automatic Recovery**: If instance fails, messages redelivered

RabbitMQ Pub/Sub Pattern
-------------------------

For asynchronous event notifications, VoIPBIN uses RabbitMQ's pub/sub (fanout exchange) pattern.

**Pub/Sub Flow**

.. code::

    Event Publishing Pattern:

    Publisher               Exchange              Subscribers
         │                      │                       │
         │  1. Publish Event    │                       │
         │  ┌────────────┐      │                       │
         │  │event: call │      │                       │
         │  │      .created│    │                       │
         │  │data: {...} │      │                       │
         │  └────────────┘      │                       │
         ├─────────────────────▶│                       │
         │  Exchange:           │                       │
         │  call.events         │                       │
         │                      │  2. Fanout to all     │
         │                      │     subscribers       │
         │                      ├──────┬────────────────┤
         │                      │      │                │
         │                      │      ▼                ▼
         │                      │  ┌────────┐      ┌────────┐
         │                      │  │Billing │      │Webhook │
         │                      │  │Manager │      │Manager │
         │                      │  └────────┘      └────────┘
         │                      │      │                │
         │                      │  3. Process       3. Process
         │                      │     event             event
         │                      │     independently     independently

**Event Types**

VoIPBIN publishes events for major state changes:

.. code::

    Event Categories:

    Call Events:
    • call.created       - New call initiated
    • call.ringing       - Call ringing
    • call.answered      - Call answered
    • call.ended         - Call terminated

    Conference Events:
    • conference.created       - Conference created
    • conference.participant_joined
    • conference.participant_left
    • conference.ended

    SMS Events:
    • sms.sent           - SMS sent successfully
    • sms.delivered      - SMS delivered to recipient
    • sms.failed         - SMS delivery failed

    Agent Events:
    • agent.login        - Agent logged in
    • agent.logout       - Agent logged out
    • agent.status_change - Agent status changed

    Transcription Events:
    • transcribe.started - Transcription started
    • transcribe.completed
    • transcript.created - New transcript segment

**Event Message Structure**

.. code::

    Event Message Format:
    {
      "event_id": "uuid-v4",
      "event_type": "call.created",
      "timestamp": "2026-01-20T12:00:00.000Z",
      "customer_id": "customer-123",
      "resource_type": "call",
      "resource_id": "call-789",
      "data": {
        "id": "call-789",
        "source": "+15551234567",
        "destination": "+15559876543",
        "status": "ringing",
        ...
      }
    }

**Subscriber Pattern**

Services subscribe to events they're interested in:

.. code::

    Subscriber Implementation:

    ┌────────────────────────────────────────────────┐
    │         bin-billing-manager                    │
    │                                                │
    │  1. Declare Exchange                           │
    │     └─ call.events (fanout)                    │
    │                                                │
    │  2. Create Queue                               │
    │     └─ billing.call.events (unique)            │
    │                                                │
    │  3. Bind Queue to Exchange                     │
    │     └─ Receive all events from exchange        │
    │                                                │
    │  4. Consume Events                             │
    │     ├─ call.created → Track call start         │
    │     ├─ call.answered → Start billing           │
    │     ├─ call.ended → Calculate charges          │
    │     └─ Other events → Ignore                   │
    │                                                │
    └────────────────────────────────────────────────┘

**Event Processing Guarantees**

.. code::

    Event Processing:

    ┌──────────────┐
    │   Publish    │
    └──────┬───────┘
           │
           │  RabbitMQ persists event
           │  (survives broker restart)
           ▼
    ┌──────────────┐
    │   Deliver    │
    └──────┬───────┘
           │
           │  Subscriber processes
           │  (may retry on failure)
           ▼
    ┌──────────────┐
    │     ACK      │
    └──────────────┘
           │
           │  Remove from queue
           │  (event processed successfully)
           ▼
    ┌──────────────┐
    │   Complete   │
    └──────────────┘

* **At-Least-Once Delivery**: Events delivered at least once (may duplicate)
* **Persistent**: Events survive broker restart
* **Manual ACK**: Subscriber acknowledges after processing
* **Retry on Failure**: Redelivered if subscriber crashes

ZeroMQ Event Streaming
-----------------------

For high-performance, low-latency event streaming, VoIPBIN uses ZeroMQ pub/sub sockets.

**ZMQ Architecture**

.. code::

    ZeroMQ Pub/Sub Pattern:

    Publishers                               Subscribers
         │                                        │
         │  Call Manager                          │
         │  (publishes call events)               │
         ├──────────────────────┐                 │
         │  ZMQ PUB Socket      │                 │
         │  tcp://*:5555        │                 │
         └──────────┬───────────┘                 │
                    │                             │
                    │  Event Stream               │
                    │  (no broker)                │
                    │                             │
                    ├────────────────────────────▶│ Agent Manager
                    │                             │ (agent presence)
                    │                             │
                    ├────────────────────────────▶│ Webhook Manager
                    │                             │ (webhook delivery)
                    │                             │
                    └────────────────────────────▶│ Talk Manager
                                                  │ (agent UI updates)

**Key Differences from RabbitMQ**

.. code::

    RabbitMQ vs ZeroMQ:

    RabbitMQ:                          ZeroMQ:
    ┌────────────┐                     ┌────────────┐
    │ Publisher  │                     │ Publisher  │
    └──────┬─────┘                     └──────┬─────┘
           │                                  │
           │ Reliable                         │ Fast
           │ Persistent                       │ In-memory
           │ Broker-based                     │ Direct socket
           ▼                                  ▼
    ┌────────────┐                     ┌────────────┐
    │  RabbitMQ  │                     │ Subscriber │
    │   Broker   │                     │  (Direct)  │
    └──────┬─────┘                     └────────────┘
           │
           │ At-least-once
           ▼
    ┌────────────┐
    │ Subscriber │
    └────────────┘

**RabbitMQ:**
* Persistent, reliable
* Guaranteed delivery
* Message queuing
* Higher latency (~10ms)

**ZeroMQ:**
* In-memory, fast
* Best-effort delivery
* Direct sockets
* Lower latency (<1ms)

**Use Cases**

VoIPBIN uses ZeroMQ for:

.. code::

    ZeroMQ Use Cases:

    ✓ Agent Presence Updates
      • Agent login/logout
      • Status changes (available, busy, away)
      • Real-time UI updates
      • High frequency, acceptable loss

    ✓ Call State Changes
      • Call ringing, answered, ended
      • Conference participant updates
      • Duplicate with RabbitMQ (redundant)
      • Speed over reliability

    ✓ Real-Time Metrics
      • Queue statistics
      • Active call counts
      • System health metrics
      • Dashboard updates

    ✗ NOT Used For:
      • Billing events (use RabbitMQ)
      • Webhook delivery (use RabbitMQ)
      • Critical state changes (use RabbitMQ)

**ZMQ Message Format**

.. code::

    ZMQ Message Structure:

    Topic (routing key)
    │
    ├─ "agent.presence"
    │  {
    │    "agent_id": "agent-123",
    │    "status": "available",
    │    "timestamp": "2026-01-20T12:00:00.000Z"
    │  }
    │
    ├─ "call.state"
    │  {
    │    "call_id": "call-789",
    │    "status": "answered",
    │    "timestamp": "2026-01-20T12:00:01.000Z"
    │  }
    │
    └─ "queue.stats"
       {
         "queue_id": "queue-456",
         "waiting": 5,
         "active": 3
       }

**Topic Filtering**

Subscribers can filter events by topic:

.. code::

    Topic-Based Filtering:

    Subscriber A:
    • Subscribe to: "agent.*"
    • Receives:
      - agent.presence
      - agent.login
      - agent.logout

    Subscriber B:
    • Subscribe to: "call.*"
    • Receives:
      - call.state
      - call.metrics

    Subscriber C:
    • Subscribe to: ""  (empty = all)
    • Receives: everything

WebSocket Communication
-----------------------

For real-time client communication, VoIPBIN uses WebSocket connections.

**WebSocket Architecture**

.. code::

    WebSocket Connection Flow:

    Client (Browser/App)    API Gateway         Backend Services
         │                      │                       │
         │  1. HTTP Upgrade     │                       │
         │  (WebSocket)         │                       │
         ├─────────────────────▶│                       │
         │                      │  2. Authenticate      │
         │                      │     (JWT token)       │
         │                      │                       │
         │  3. Connection       │                       │
         │     Established      │                       │
         │◀─────────────────────┤                       │
         │                      │                       │
         │  4. Subscribe        │                       │
         │  {"type":"subscribe",│                       │
         │   "topics":["..."]}  │                       │
         ├─────────────────────▶│                       │
         │                      │  5. Register          │
         │                      │     subscription      │
         │                      │                       │
         │                      │  6. Backend Event     │
         │                      │◀──────────────────────┤
         │                      │  (via RabbitMQ/ZMQ)   │
         │                      │                       │
         │  7. Push to Client   │                       │
         │◀─────────────────────┤                       │
         │  {"event":"call.     │                       │
         │   created",...}      │                       │
         │                      │                       │

**Subscription Topics**

Clients subscribe to specific event topics:

.. code::

    Topic Pattern:
    customer_id:<id>:<resource>:<resource_id>

    Examples:
    • customer_id:123:call:*
      → All calls for customer 123

    • customer_id:123:call:call-789
      → Specific call updates

    • customer_id:123:agent:agent-456
      → Specific agent updates

    • customer_id:123:queue:*
      → All queues for customer

    • customer_id:123:conference:conf-999
      → Specific conference updates

**WebSocket Use Cases**

.. code::

    WebSocket Applications:

    Agent Dashboard:
    ┌──────────────────────────────────────┐
    │ • Real-time call notifications       │
    │ • Queue status updates               │
    │ • Agent presence                     │
    │ • Live chat messages                 │
    └──────────────────────────────────────┘

    Customer Portal:
    ┌──────────────────────────────────────┐
    │ • Call status updates                │
    │ • Campaign progress                  │
    │ • Billing updates                    │
    │ • System notifications               │
    └──────────────────────────────────────┘

    Media Streaming:
    ┌──────────────────────────────────────┐
    │ • Bi-directional audio (RTP)         │
    │ • Live transcription feed            │
    │ • Real-time metrics                  │
    └──────────────────────────────────────┘

**Connection Management**

.. code::

    WebSocket Lifecycle:

    ┌────────────┐
    │  Connect   │  Client establishes WebSocket
    └──────┬─────┘
           │
           ▼
    ┌────────────┐
    │ Authenticate│  Validate JWT token
    └──────┬─────┘
           │
           ▼
    ┌────────────┐
    │ Subscribe  │  Client subscribes to topics
    └──────┬─────┘
           │
           ▼
    ┌────────────┐
    │  Active    │  Bi-directional communication
    │            │  • Server pushes events
    │            │  • Client sends commands
    │            │  • Pinger sends ping frames
    └──────┬─────┘
           │
           │  (Keep-alive ping/pong)
           │
           ▼
    ┌────────────┐
    │ Disconnect │  Connection closed
    └────────────┘

**Keep-Alive Mechanism (Server-Side Ping/Pong)**

VoIPBIN implements server-side keep-alive to prevent load balancer timeouts:

.. code::

    Keep-Alive Configuration:

    ┌────────────────────────────────────────────────┐
    │  Ping Interval:  30 seconds                    │
    │  Pong Wait:      60 seconds                    │
    │  Write Timeout:  10 seconds                    │
    └────────────────────────────────────────────────┘

    Keep-Alive Flow:

    Server                                    Client
       │                                         │
       │  Every 30s: Send Ping Frame             │
       ├────────────────────────────────────────▶│
       │                                         │
       │  Automatic Pong Response                │
       │◀────────────────────────────────────────┤
       │                                         │
       │  Reset read deadline (60s)              │
       │                                         │

    Error Detection:
    ┌────────────────────────────────────────────────┐
    │  No pong within 60s → Connection dead          │
    │  Write failure → Connection broken             │
    │  Either error → Close and cleanup              │
    └────────────────────────────────────────────────┘

**Keep-Alive Benefits:**

* **Prevents Idle Drops**: Load balancers see regular traffic
* **Dead Connection Detection**: Server detects unresponsive clients
* **Automatic Cleanup**: Zombie connections closed promptly
* **RFC 6455 Compliant**: Uses standard WebSocket ping/pong frames

**Connection Features:**

* **Keepalive**: Server-side ping every 30 seconds
* **Dead Detection**: 60-second timeout for pong response
* **Auto-Reconnect**: Client should reconnect on disconnect
* **Subscription Restore**: Re-subscribe after reconnect
* **Write Protection**: Mutex prevents concurrent write race conditions

Message Reliability
-------------------

Different patterns provide different reliability guarantees:

.. code::

    Reliability Comparison:

    Pattern          Delivery          Persistence    Use Case
    ───────────────────────────────────────────────────────────
    RabbitMQ RPC     Exactly-once     Yes            Critical ops
                     (request-reply)

    RabbitMQ Pub/Sub At-least-once    Yes            Important events
                     (may duplicate)

    ZeroMQ Pub/Sub   Best-effort      No             Real-time updates
                     (may lose)

    WebSocket        Best-effort      No             Client notifications
                     (may lose)

**Reliability Patterns**

.. code::

    Ensuring Reliability:

    Critical Operations (RabbitMQ RPC):
    ┌────────────────────────────────────┐
    │ • Persistent messages              │
    │ • Manual acknowledgment            │
    │ • Automatic retry                  │
    │ • Timeout handling                 │
    │ • Idempotent operations            │
    └────────────────────────────────────┘

    Important Events (RabbitMQ Pub/Sub):
    ┌────────────────────────────────────┐
    │ • Persistent messages              │
    │ • Multiple subscribers             │
    │ • Redundant processing OK          │
    │ • Deduplication in subscriber      │
    └────────────────────────────────────┘

    Real-Time Updates (ZeroMQ):
    ┌────────────────────────────────────┐
    │ • No persistence                   │
    │ • Fast delivery                    │
    │ • Acceptable loss                  │
    │ • Often duplicated in RabbitMQ     │
    └────────────────────────────────────┘

Message Ordering
----------------

VoIPBIN guarantees ordering within specific boundaries:

.. code::

    Ordering Guarantees:

    Same Queue:              Different Queues:
    ┌──────────┐             ┌──────────┐  ┌──────────┐
    │ Message 1│             │ Message 1│  │ Message 2│
    └─────┬────┘             └─────┬────┘  └─────┬────┘
          │                        │             │
          │ Queue A                │ Queue A     │ Queue B
          │                        │             │
          ▼                        ▼             ▼
    ┌──────────┐             ┌──────────┐  ┌──────────┐
    │ Message 2│             │ Service A│  │ Service B│
    └─────┬────┘             └──────────┘  └──────────┘
          │                        │             │
          │                        │  May arrive in any order
          ▼                        ▼             ▼
    ┌──────────┐             ┌──────────┐  ┌──────────┐
    │ Message 3│             │ Ordered  │  │ No order │
    └──────────┘             │ delivery │  │ guarantee│
                             └──────────┘  └──────────┘

    Ordered ✓               Unordered ✗

**Ordering Strategy:**

* **Within Queue**: Messages delivered in order to same consumer
* **Across Queues**: No ordering guarantee
* **Single Publisher**: Maintains order if using single connection
* **Application Logic**: Handle out-of-order messages when necessary

Error Handling and Retries
---------------------------

VoIPBIN implements comprehensive error handling:

**Retry Strategy**

.. code::

    Exponential Backoff Retry:

    Attempt    Delay      Total Time
    ──────────────────────────────────
    1          0s         0s
    2          1s         1s
    3          2s         3s
    4          4s         7s
    5          8s         15s
    6          16s        31s
    7          32s        63s
    Max: 7 attempts, ~1 minute total

**Dead Letter Queue**

Failed messages move to dead letter queue for investigation:

.. code::

    Dead Letter Processing:

    Normal Flow:              Failed Flow:
    ┌──────────┐              ┌──────────┐
    │ Message  │              │ Message  │
    └─────┬────┘              └─────┬────┘
          │                         │
          │ Process                 │ Process (fails)
          ▼                         ▼
    ┌──────────┐              ┌──────────┐
    │  Success │              │  Retry   │
    └──────────┘              └─────┬────┘
                                    │ (max retries exceeded)
                                    ▼
                              ┌──────────┐
                              │   DLQ    │ Dead Letter Queue
                              └─────┬────┘
                                    │
                                    │ Manual investigation
                                    │ or automated recovery
                                    ▼
                              ┌──────────┐
                              │  Alert   │
                              └──────────┘

**Error Categories**

.. code::

    Error Handling by Type:

    Transient Errors (Retry):
    • Network timeout
    • Database connection lost
    • Service temporarily unavailable
    → Retry with exponential backoff

    Permanent Errors (Don't Retry):
    • Invalid data format
    • Resource not found
    • Permission denied
    → Send to DLQ, alert operator

    Business Errors (Log and Return):
    • Insufficient balance
    • Invalid phone number
    • Duplicate request
    → Return error to caller

Performance Optimization
------------------------

VoIPBIN optimizes messaging performance:

**Connection Pooling**

.. code::

    Connection Management:

    Service Instance
    ┌────────────────────────────────────┐
    │                                    │
    │  Connection Pool (5 connections)   │
    │  ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐│
    │  │ 1  │ │ 2  │ │ 3  │ │ 4  │ │ 5  ││
    │  └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘│
    │    │      │      │      │      │   │
    └────┼──────┼──────┼──────┼──────┼───┘
         │      │      │      │      │
         └──────┴──────┴──────┴──────┘
                    │
                    │ Single TCP connection
                    ▼
              ┌──────────┐
              │ RabbitMQ │
              └──────────┘

* **Reuse Connections**: Don't create per-request
* **Multiple Channels**: Use channels for concurrency
* **Connection Limits**: Pool size based on load
* **Health Checks**: Monitor connection health

**Batch Processing**

For high-volume operations:

.. code::

    Batch vs Individual:

    Individual Messages:     Batch Processing:
    ┌────┐ ┌────┐ ┌────┐    ┌──────────────┐
    │ M1 │ │ M2 │ │ M3 │    │ M1, M2, M3   │
    └─┬──┘ └─┬──┘ └─┬──┘    │ M4, M5, M6   │
      │      │      │       │ ... (100)    │
      ▼      ▼      ▼       └──────┬───────┘
    Send 100 times            Send once
    (high overhead)           (low overhead)

* **Bulk Publishing**: Send multiple messages at once
* **Bulk ACK**: Acknowledge multiple messages together
* **Reduced Overhead**: Fewer network round-trips
* **Higher Throughput**: 10x-100x improvement

Monitoring and Debugging
-------------------------

VoIPBIN monitors all communication channels:

**Metrics**

.. code::

    Message Queue Metrics:

    Queue Depth:
    ┌─────────────────────────────────┐
    │     Pending Messages            │
    │  ┌──┐┌──┐┌──┐┌──┐┌──┐           │
    │  │M1││M2││M3││M4││M5│...        │
    │  └──┘└──┘└──┘└──┘└──┘           │
    └─────────────────────────────────┘
    Alert if > 1000 messages

    Processing Rate:
    Messages/sec: ████████ 850/s
    Target:       ████████ 1000/s
    Alert if < 500/s

    Error Rate:
    Failures:     ██ 2%
    Target:       ██ < 5%
    Alert if > 10%

**Distributed Tracing**

Track requests across services:

.. code::

    Trace ID: trace-123

    1. API Gateway          [50ms]
       ├─ Authenticate      [5ms]
       ├─ Authorize         [10ms]
       └─ Send RPC          [35ms]
           │
           ▼
    2. Call Manager         [80ms]
       ├─ Validate          [10ms]
       ├─ Create Record     [20ms]
       └─ Initiate Call     [50ms]
           │
           ▼
    3. RTC Manager          [120ms]
       └─ Setup Media       [120ms]

    Total: 250ms

* **Correlation IDs**: Track requests across services
* **Timing**: Measure latency at each hop
* **Errors**: Identify where failures occur
* **Dependencies**: Visualize service interactions

Best Practices
--------------

**Message Design:**

* Keep messages small (<1MB)
* Use JSON for human-readable format
* Include timestamps for debugging
* Add correlation IDs for tracing

**Error Handling:**

* Always handle errors gracefully
* Implement retry with exponential backoff
* Use dead letter queues for failed messages
* Alert on high error rates

**Performance:**

* Use connection pooling
* Batch messages when possible
* Set appropriate timeouts
* Monitor queue depths

**Security:**

* Encrypt sensitive data in messages
* Validate all incoming messages
* Use authentication for connections
* Limit message size to prevent abuse
