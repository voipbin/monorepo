.. _architecture-flow:

System Request Flows
====================

.. note:: **AI Context**

   This page demonstrates detailed request flows through VoIPBIN: creating a call, reading with cache, event broadcasting, multi-service orchestration, real-time WebSocket notifications, and error handling with retries. Relevant when an AI agent needs to understand the step-by-step processing of API requests, timing breakdowns, caching behavior, or debugging strategies.

This section demonstrates how requests flow through VoIPBIN's architecture from client to backend services and back. Understanding these flows helps developers build integrations and debug issues.

Request Flow Overview
---------------------

All external requests follow this general pattern:

.. code::

    Complete Request Flow:

    Client App          API Gateway         Message Queue       Backend Service      Data Layer
        |                   |                    |                     |                  |
        |  HTTP Request     |                    |                     |                  |
        +------------------>|                    |                     |                  |
        |                   |  1. Authenticate   |                     |                  |
        |                   |  2. Authorize      |                     |                  |
        |                   |  3. Validate       |                     |                  |
        |                   |                    |                     |                  |
        |                   |  RPC Request       |                     |                  |
        |                   +------------------->|                     |                  |
        |                   |                    |  Dequeue            |                  |
        |                   |                    +-------------------->|                  |
        |                   |                    |                     |  Query           |
        |                   |                    |                     +----------------->|
        |                   |                    |                     |                  |
        |                   |                    |                     |  Result          |
        |                   |                    |                     |<-----------------+
        |                   |                    |  Response           |                  |
        |                   |                    |<--------------------+                  |
        |                   |  RPC Response      |                     |                  |
        |                   |<-------------------+                     |                  |
        |  JSON Response    |                    |                     |                  |
        |<------------------+                    |                     |                  |
        |                   |                    |                     |                  |

Flow 1: Create Call (Simple)
-----------------------------

This flow shows how a basic call creation request flows through the system.

**Step-by-Step Flow:**

.. code::

    1. Client Request:

    Client Application
        |
        |  POST /v1.0/calls
        |  Authorization: Bearer eyJhbGc...
        |  Content-Type: application/json
        |
        |  {
        |    "source": {"type": "tel", "target": "+15551234567"},
        |    "destinations": [{"type": "tel", "target": "+15559876543"}]
        |  }
        |
        v

    2. API Gateway (bin-api-manager):

    +-------------------------------------------------+
    |  a) Extract JWT token                           |
    |     → token = "eyJhbGc..."                      |
    |                                                 |
    |  b) Validate JWT signature                      |
    |     → customer_id = "customer-123"              |
    |     → agent_id = "agent-456"                    |
    |                                                 |
    |  c) Check permissions                           |
    |     → hasPermission(customer-123, "call.create")|
    |     → ✓ Allowed                                 |
    |                                                 |
    |  d) Validate request body                       |
    |     → Source phone valid                        |
    |     → Destination phone valid                   |
    |     → ✓ Valid                                   |
    |                                                 |
    |  e) Build RPC message                           |
    |     {                                           |
    |       "route": "POST /v1/calls",                |
    |       "headers": {                              |
    |         "customer_id": "customer-123",          |
    |         "agent_id": "agent-456"                 |
    |       },                                        |
    |       "body": {...}                             |
    |     }                                           |
    |                                                 |
    |  f) Send to RabbitMQ                            |
    |     → Queue: bin-manager.call.request           |
    +-------------------------------------------------+
        |
        v

    3. RabbitMQ:

    +------------------------------------------------+
    |  a) Receive message                            |
    |     → Queue: bin-manager.call.request          |
    |                                                |
    |  b) Route to available consumer                |
    |     → bin-call-manager instance 2 (of 3)       |
    +------------------------------------------------+
        |
        v

    4. Call Manager (bin-call-manager):

    +-------------------------------------------------+
    |  a) Receive RPC message                         |
    |     → Parse route: POST /v1/calls               |
    |     → Extract customer_id, agent_id             |
    |                                                 |
    |  b) Validate business logic                     |
    |     → Check billing balance                     |
    |     → ✓ Sufficient funds                        |
    |                                                 |
    |  c) Create call record                          |
    |     → Generate call_id = "call-789"             |
    |     → INSERT INTO calls (...)                   |
    |     → Status: "initiating"                      |
    |                                                 |
    |  d) Initiate SIP call                           |
    |     → Send to bin-rtc-manager                   |
    |     → Request Asterisk channel creation         |
    |                                                 |
    |  e) Update call status                          |
    |     → UPDATE calls SET status='ringing' WHERE...|
    |                                                 |
    |  f) Publish event                               |
    |     → Event: call.created                       |
    |     → RabbitMQ exchange: call.events            |
    |                                                 |
    |  g) Build response                              |
    |     {                                           |
    |       "id": "call-789",                         |
    |       "status": "ringing",                      |
    |       "source": "+15551234567",                 |
    |       "destination": "+15559876543",            |
    |       "tm_create": "2026-01-20T12:00:00.000Z"   |
    |     }                                           |
    |                                                 |
    |  h) Send RPC response                           |
    |     → Reply to: reply_to queue                  |
    +-------------------------------------------------+
        |
        v

    5. RabbitMQ (Response):

    +------------------------------------------------+
    |  a) Deliver response to API Gateway            |
    |     → Queue: amq.gen-xyz (reply_to)            |
    +------------------------------------------------+
        |
        v

    6. API Gateway (Response):

    +------------------------------------------------+
    |  a) Receive RPC response                       |
    |     → status_code: 200                         |
    |     → body: {...}                              |
    |                                                |
    |  b) Format HTTP response                       |
    |     → HTTP 201 Created                         |
    |     → Content-Type: application/json           |
    +------------------------------------------------+
        |
        v

    7. Client Response:

    HTTP/1.1 201 Created
    Content-Type: application/json

    {
      "id": "call-789",
      "status": "ringing",
      "source": "+15551234567",
      "destination": "+15559876543",
      "tm_create": "2026-01-20T12:00:00.000Z"
    }

**Timing Breakdown:**

.. code::

    Component               Time      Cumulative
    ---------------------------------------------
    API Gateway auth        5ms       5ms
    RabbitMQ routing        2ms       7ms
    Call Manager logic      30ms      37ms
    Database insert         8ms       45ms
    RTC Manager SIP setup   50ms      95ms
    Response routing        5ms       100ms
    ---------------------------------------------
    Total                   100ms

Flow 2: Get Call with Caching
------------------------------

This flow demonstrates cache-aside pattern for reading data.

.. code::

    1. Client Request:

    GET /v1.0/calls/call-789
    Authorization: Bearer eyJhbGc...

        |
        v

    2. API Gateway:

    +------------------------------------------------+
    |  • Authenticate (5ms)                          |
    |  • Build RPC message                           |
    |  • Send to bin-manager.call.request            |
    +------------------------------------------------+
        |
        v

    3. Call Manager:

    +------------------------------------------------+
    |  a) Check Redis cache first                    |
    |     key = "call:call-789"                      |
    |                                                |
    |     GET call:call-789                          |
    |     → Cache HIT! (90% of requests)             |
    |     → Return cached data (2ms)                 |
    |                                                |
    |     OR                                         |
    |                                                |
    |     → Cache MISS (10% of requests)             |
    |                                                |
    |  b) If cache miss, query MySQL                 |
    |     SELECT * FROM calls WHERE id='call-789'    |
    |     → Query time: 10ms                         |
    |                                                |
    |  c) Store in Redis for next time               |
    |     SET call:call-789 {...} EX 300  # 5 min    |
    |     → Store time: 2ms                          |
    |                                                |
    |  d) Check authorization                        |
    |     if call.customer_id != jwt.customer_id:    |
    |       return 404 (not 403, for security)       |
    |                                                |
    |  e) Return response                            |
    +------------------------------------------------+
        |
        v

    4. Response Times:

    Cache Hit Path:  ~12ms total
    • API Gateway: 5ms
    • Redis lookup: 2ms
    • Response: 5ms

    Cache Miss Path: ~27ms total
    • API Gateway: 5ms
    • Redis lookup: 2ms (miss)
    • MySQL query: 10ms
    • Redis store: 2ms
    • Response: 5ms
    • Authorization: 3ms

Flow 3: Call with Event Broadcasting
-------------------------------------

This flow shows asynchronous event publishing to multiple subscribers.

.. code::

    Call State Change Flow:

    1. Call Answered (in bin-call-manager):

    +------------------------------------------------+
    |  a) Receive SIP 200 OK from Asterisk           |
    |     → Call answered                            |
    |                                                |
    |  b) Update database                            |
    |     UPDATE calls                               |
    |     SET status='active', tm_answer=NOW()       |
    |     WHERE id='call-789'                        |
    |                                                |
    |  c) Invalidate cache                           |
    |     DEL call:call-789                          |
    |                                                |
    |  d) Publish event to RabbitMQ                  |
    |     Exchange: call.events                      |
    |     Event: call.answered                       |
    |     {                                          |
    |       "event_type": "call.answered",           |
    |       "call_id": "call-789",                   |
    |       "timestamp": "2026-01-20T12:00:05.000Z"  |
    |     }                                          |
    |                                                |
    |  e) Publish to ZeroMQ (fast path)              |
    |     Topic: "call.state"                        |
    |     {                                          |
    |       "call_id": "call-789",                   |
    |       "status": "active"                       |
    |     }                                          |
    +------------------------------------------------+
        |
        |
        +----------------------+----------------------+----------------------+
        |                      |                      |                      |
        v                      v                      v                      v

    2a. Billing Manager    2b. Webhook Manager   2c. Talk Manager      2d. Agent Manager

    +----------------+    +----------------+    +----------------+   +----------------+
    | Start billing  |    | Send webhook   |    | Update agent   |   | Update agent   |
    | for call       |    | to customer    |    | dashboard      |   | stats          |
    |                |    | endpoint       |    | via WebSocket  |   |                |
    | • Calculate    |    |                |    |                |   | • Active calls |
    |   charges      |    | POST https://  |    | {              |   | • Talk time    |
    | • Create       |    | customer.com/  |    |   "event":     |   | • Status       |
    |   billing      |    | webhook        |    |   "call.       |   |                |
    |   record       |    |                |    |   answered",   |   |                |
    |                |    | {              |    |   "call_id":   |   |                |
    | INSERT INTO    |    |   "event_type":|    |   "call-789"   |   |                |
    | billings       |    |   "call.       |    | }              |   |                |
    | (...)          |    |   answered",   |    |                |   |                |
    |                |    |   ...          |    |                |   |                |
    |                |    | }              |    |                |   |                |
    +----------------+    +----------------+    +----------------+   +----------------+

    All subscribers process event independently and concurrently

Flow 4: Complex Multi-Service Flow
-----------------------------------

This flow demonstrates a complex operation involving multiple services.

.. code::

    Conference Join with Flow Execution:

    Client                API Gateway         Flow Manager        Conference Mgr      Call Manager
      |                       |                    |                    |                  |
      |  POST /conferences/   |                    |                    |                  |
      |  conf-123/join        |                    |                    |                  |
      +---------------------->|                    |                    |                  |
      |                       |  Auth + RPC        |                    |                  |
      |                       +------------------->|                    |                  |
      |                       |                    |                    |                  |
      |                       |                    |  1. Get Conference |                  |
      |                       |                    +------------------->|                  |
      |                       |                    |                    |  [conf data]     |
      |                       |                    |<-------------------+                  |
      |                       |                    |                    |                  |
      |                       |                    |  2. Get Flow       |                  |
      |                       |                    |  (from conf)       |                  |
      |                       |                    |                    |                  |
      |                       |                    |  3. Execute Flow   |                  |
      |                       |                    |  Actions:          |                  |
      |                       |                    |                    |                  |
      |                       |                    |  Action 1: Answer  |                  |
      |                       |                    +-------------------------------------->|
      |                       |                    |                    |                  |
      |                       |                    |  Action 2: Talk    |                  |
      |                       |                    |  "Welcome to conf" |                  |
      |                       |                    +-------------------------------------->|
      |                       |                    |                    |                  |
      |                       |                    |  Action 3: Join    |                  |
      |                       |                    |  Conference        |                  |
      |                       |                    +------------------->|                  |
      |                       |                    |                    |  Add participant |
      |                       |                    |                    |  to bridge       |
      |                       |                    |<-------------------+                  |
      |                       |                    |                    |                  |
      |                       |  Response          |                    |                  |
      |                       |<-------------------+                    |                  |
      |  Success              |                    |                    |                  |
      |<----------------------+                    |                    |                  |
      |                       |                    |                    |                  |

    Services Involved:
    • API Gateway (authentication, routing)
    • Flow Manager (orchestration)
    • Conference Manager (conference state)
    • Call Manager (call handling)
    • RTC Manager (not shown, handles SIP/media)

    Total Time: ~200ms
    • Gateway: 5ms
    • Conference lookup: 10ms
    • Flow execution: 150ms (multiple actions)
    • Conference join: 30ms
    • Response: 5ms

Flow 5: Real-Time Event Notification
-------------------------------------

This flow shows how real-time events reach clients via WebSocket.

.. code::

    Real-Time Call Status Updates:

    1. Client Subscribes:

    Client (Browser)        API Gateway (WebSocket)    Backend Services
        |                           |                       |
        |  WebSocket Connect        |                       |
        +-------------------------->|                       |
        |  wss://api.voipbin.net/ws |                       |
        |  ?token=eyJhbGc...        |                       |
        |                           |  Validate JWT         |
        |                           |  → customer_id: 123   |
        |                           |                       |
        |  Subscribe                |                       |
        |  {                        |                       |
        |    "type": "subscribe",   |                       |
        |    "topics": [            |                       |
        |      "customer_id:123:    |                       |
        |       call:*"             |                       |
        |    ]                      |                       |
        |  }                        |                       |
        +-------------------------->|                       |
        |                           |  Register             |
        |                           |  subscription         |
        |                           |                       |
        |  ACK                      |                       |
        |<--------------------------+                       |
        |                           |                       |

    2. Event Occurs:

    Call Manager                RabbitMQ/ZMQ          API Gateway (WS)      Client
        |                           |                       |                  |
        |  Call status changed      |                       |                  |
        |  (answered)               |                       |                  |
        |                           |                       |                  |
        |  Publish event            |                       |                  |
        +-------------------------->|                       |                  |
        |  {                        |                       |                  |
        |    "event": "call.        |                       |                  |
        |     answered",            |                       |                  |
        |    "customer_id": "123",  |                       |                  |
        |    "call_id": "call-789"  |                       |                  |
        |  }                        |                       |                  |
        |                           |                       |                  |
        |                           |  Fanout to            |                  |
        |                           |  subscribers          |                  |
        |                           +---------------------->|                  |
        |                           |                       |  Match topic     |
        |                           |                       |  filter          |
        |                           |                       |                  |
        |                           |                       |  Push to client  |
        |                           |                       +----------------->|
        |                           |                       |                  |
        |                           |                       |  {               |
        |                           |                       |    "event_type": |
        |                           |                       |    "call.        |
        |                           |                       |    answered",    |
        |                           |                       |    "call_id":    |
        |                           |                       |    "call-789",   |
        |                           |                       |    "timestamp":  |
        |                           |                       |    "..."         |
        |                           |                       |  }               |

    Latency: < 100ms from event to client notification

Flow 6: Error Handling Flow
----------------------------

This flow demonstrates error handling and retry logic.

.. code::

    Failed Request with Retry:

    1. Initial Request (Fails):

    API Gateway         Call Manager        Database
        |                   |                   |
        |  RPC: Create Call |                   |
        +------------------>|                   |
        |                   |  INSERT INTO      |
        |                   |  calls (...)      |
        |                   +------------------>|
        |                   |                   X  Connection lost
        |                   |                   |
        |                   |  ← Error          |
        |                   |<------------------+
        |                   |                   |
        |                   |  Retry (1s delay) |
        |                   |                   |

    2. Automatic Retry (Attempt 2):

        |                   |  Reconnect        |
        |                   +------------------>|
        |                   |                   |
        |                   |  INSERT INTO      |
        |                   |  calls (...)      |
        |                   +------------------>|
        |                   |                   ✓  Success
        |                   |                   |
        |                   |  Success          |
        |                   |<------------------+
        |                   |                   |
        |  Success          |                   |
        |<------------------+                   |
        |                   |                   |

    3. Permanent Error (No Retry):

    API Gateway         Call Manager        Billing Manager
        |                   |                   |
        |  RPC: Create Call |                   |
        +------------------>|                   |
        |                   |  Check balance    |
        |                   +------------------>|
        |                   |                   |
        |                   |  Insufficient     |
        |                   |  balance          |
        |                   |<------------------+
        |                   |                   |
        |  Error 402        |  Don't retry      |
        |  Payment Required |  (permanent error)|
        |<------------------+                   |
        |                   |                   |

    Error Categories:
    • Transient → Retry (network, timeout, connection)
    • Permanent → Don't retry (invalid data, permissions)
    • Business → Return error (insufficient balance)

Performance Optimization
------------------------

VoIPBIN optimizes flow performance through several techniques:

**Parallel Processing**

.. code::

    Sequential vs Parallel:

    Sequential (Slow):           Parallel (Fast):
    +----------+                 +----------+
    | Task A   | 50ms            | Task A   | 50ms
    +----+-----+                 +----+-----+
         |                            |
         v                            |
    +----------+                      |
    | Task B   | 50ms                 +-------------+
    +----+-----+                      |             |
         |                            v             v
         v                        +----------+ +----------+
    +----------+                  | Task B   | | Task C   |
    | Task C   | 50ms             | 50ms     | | 50ms     |
    +----------+                  +----------+ +----------+

    Total: 150ms                 Total: 50ms (3x faster)

**Caching Strategy**

.. code::

    Without Cache:               With Cache:
    Every request → DB           First request → DB
    Query time: 10ms             Query time: 10ms
                                 Cache for 5 minutes

                                 Subsequent requests → Cache
                                 Query time: 2ms

    1000 requests = 10s          1000 requests = 2s (5x faster)

**Connection Pooling**

.. code::

    No Pooling:                  With Pooling:
    Each request:                Each request:
    • Connect: 20ms              • Get from pool: 1ms
    • Query: 10ms                • Query: 10ms
    • Disconnect: 5ms            • Return to pool: 1ms
    Total: 35ms                  Total: 12ms (3x faster)

Best Practices for Developers
------------------------------

**When Integrating with VoIPBIN:**

1. **Always Include Authentication**
   - Include JWT token in Authorization header
   - Handle 401 responses (refresh token)

2. **Handle Asynchronous Operations**
   - Many operations are asynchronous
   - Use webhooks or WebSocket for notifications
   - Poll with reasonable intervals if needed

3. **Implement Retry Logic**
   - Retry on 5xx errors
   - Use exponential backoff
   - Don't retry on 4xx errors

4. **Subscribe to Events**
   - Use WebSocket for real-time updates
   - Configure webhooks for important events
   - Handle duplicate events gracefully

5. **Optimize Requests**
   - Use pagination for lists
   - Request only needed fields
   - Cache responses when appropriate

6. **Monitor Performance**
   - Track response times
   - Alert on high error rates
   - Monitor webhook delivery

Debugging Request Flows
------------------------

**Using Correlation IDs:**

.. code::

    Request Tracing:

    1. Client sends request with X-Request-ID header:
       POST /v1.0/calls
       X-Request-ID: req-abc-123

    2. API Gateway logs:
       [req-abc-123] Authenticated customer-123
       [req-abc-123] Sending RPC to call-manager

    3. Call Manager logs:
       [req-abc-123] Creating call record
       [req-abc-123] Call created: call-789

    4. Search logs by correlation ID to trace full flow

**Common Issues:**

.. code::

    Issue: 401 Unauthorized
    → Check JWT token validity
    → Ensure token not expired
    → Verify customer_id matches resource

    Issue: 404 Not Found
    → May be authorization failure (returns 404 for security)
    → Check customer_id ownership
    → Verify resource exists

    Issue: 500 Internal Server Error
    → Backend service error
    → Check logs with correlation ID
    → May require retry

    Issue: Slow Response
    → Check cache hit rate
    → Review database query performance
    → Monitor service health

Summary
-------

VoIPBIN's request flows are designed for:

* **Performance**: Caching, connection pooling, parallel processing
* **Reliability**: Retry logic, circuit breakers, health checks
* **Scalability**: Stateless services, horizontal scaling, queue-based communication
* **Observability**: Correlation IDs, distributed tracing, comprehensive logging
* **Security**: Gateway authentication, authorization checks, encrypted communication

Understanding these flows helps developers build efficient integrations and troubleshoot issues effectively.
