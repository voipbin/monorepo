.. _architecture-dataflow:

Data Flow Diagrams
==================

.. note:: **AI Context**

   This page illustrates end-to-end data flows through VoIPBIN for common operations: API request lifecycle, event publishing, WebSocket real-time updates, media streaming (audio pipeline), database write patterns, campaign execution, transcription, and webhook delivery. Relevant when an AI agent needs to trace how a request moves through the system or understand data transformations at each stage.

This section illustrates how data flows through VoIPBIN's components for common operations. Understanding these flows helps developers integrate with the platform and troubleshoot issues.

End-to-End Request Flow
-----------------------

Every API request follows a consistent path through the system:

.. code::

    Complete API Request Flow:

    Client          Load Balancer      API Gateway       Backend Service        Database
       |                 |                  |                  |                   |
       | HTTPS Request   |                  |                  |                   |
       +---------------->|                  |                  |                   |
       |                 | TLS Termination  |                  |                   |
       |                 +----------------->|                  |                   |
       |                 |                  |                  |                   |
       |                 |                  | 1. Parse Auth    |                   |
       |                 |                  |    Header        |                   |
       |                 |                  |                  |                   |
       |                 |                  | 2. Validate JWT  |                   |
       |                 |                  |    or AccessKey  |                   |
       |                 |                  |                  |                   |
       |                 |                  | 3. Extract       |                   |
       |                 |                  |    customer_id   |                   |
       |                 |                  |                  |                   |
       |                 |                  | 4. RabbitMQ RPC  |                   |
       |                 |                  +----------------->|                   |
       |                 |                  |                  |                   |
       |                 |                  |                  | 5. Check Redis   |
       |                 |                  |                  |    Cache          |
       |                 |                  |                  +------------------>|
       |                 |                  |                  |                   |
       |                 |                  |                  |<------------------+
       |                 |                  |                  | (cache hit/miss) |
       |                 |                  |                  |                   |
       |                 |                  |                  | 6. Query MySQL   |
       |                 |                  |                  |    (if cache miss)|
       |                 |                  |                  +------------------>|
       |                 |                  |                  |                   |
       |                 |                  |                  |<------------------+
       |                 |                  |                  | Data             |
       |                 |                  |                  |                   |
       |                 |                  |                  | 7. Update Cache  |
       |                 |                  |                  +------------------>|
       |                 |                  |                  |                   |
       |                 |                  |<-----------------+                   |
       |                 |                  | RPC Response     |                   |
       |                 |                  |                  |                   |
       |                 |                  | 8. Check         |                   |
       |                 |                  |    Authorization |                   |
       |                 |                  |    (customer_id) |                   |
       |                 |                  |                  |                   |
       |<----------------+-----------------+                   |                   |
       | JSON Response   |                  |                   |                   |
       |                 |                  |                  |                   |

**Key Data Transformations:**

.. code::

    Data Format at Each Stage:

    1. Client -> API Gateway:
       +------------------------------------------+
       | Format: HTTPS/JSON                       |
       | Auth: Bearer JWT or AccessKey header     |
       | Body: JSON request body                  |
       +------------------------------------------+

    2. API Gateway -> Backend Service:
       +------------------------------------------+
       | Format: RabbitMQ message (JSON)          |
       | Contains: customer_id, agent_id,         |
       |           original request data          |
       | Queue: bin-manager.<service>.request     |
       +------------------------------------------+

    3. Backend Service -> Database:
       +------------------------------------------+
       | Format: SQL queries (parameterized)      |
       | ORM: Squirrel query builder              |
       +------------------------------------------+

    4. Backend Service -> API Gateway:
       +------------------------------------------+
       | Format: RabbitMQ response (JSON)         |
       | Contains: status_code, data, error       |
       +------------------------------------------+

    5. API Gateway -> Client:
       +------------------------------------------+
       | Format: HTTPS/JSON                       |
       | Headers: Content-Type, Cache-Control     |
       +------------------------------------------+

Event Publishing Flow
---------------------

When resources change, events propagate through the system:

.. code::

    Event Publishing Flow:

    Source Service       RabbitMQ Exchange       Subscriber Services
         |                     |                        |
         | 1. Business Logic   |                        |
         |    (e.g., call ends)|                        |
         |                     |                        |
         | 2. Update Database  |                        |
         |                     |                        |
         | 3. Invalidate Cache |                        |
         |                     |                        |
         | 4. Publish Event    |                        |
         +-------------------->|                        |
         |  Exchange:          |                        |
         |  call.events        |                        |
         |                     |                        |
         |                     | 5. Fanout to Queues    |
         |                     +----------+-------------+
         |                     |          |             |
         |                     v          v             v
         |               +--------+ +--------+    +--------+
         |               |billing | |webhook |    |queue   |
         |               |.call   | |.call   |    |.call   |
         |               |.events | |.events |    |.events |
         |               +---+----+ +---+----+    +---+----+
         |                   |          |             |
         |                   | 6. Process             |
         |                   |    Event               |
         |                   v          v             v
         |              billing-   webhook-      queue-
         |              manager    manager       manager
         |                   |          |             |
         |                   | 7. Take  | 7. Send     | 7. Update
         |                   |    Action|    Webhook  |    Stats
         |                   |          |             |

**Event Data Structure:**

.. code::

    Published Event:

    Exchange: call.events
    Routing Key: call.hungup

    Message:
    {
      "event_id": "uuid",
      "event_type": "call_hungup",
      "timestamp": "2026-01-20T12:00:00.000Z",
      "customer_id": "uuid",
      "resource": {
        "id": "uuid",
        "type": "call",
        "source": "+15551234567",
        "destination": "+15559876543",
        "duration": 120,
        "status": "completed",
        "hangup_cause": "normal_clearing"
      }
    }

**Subscriber Processing:**

.. code::

    Event Processing by Service:

    billing-manager:
    +------------------------------------------+
    | On: call_hungup                          |
    | Action:                                  |
    |   1. Calculate call cost                 |
    |   2. Deduct from customer balance        |
    |   3. Create billing record               |
    +------------------------------------------+

    webhook-manager:
    +------------------------------------------+
    | On: call_hungup                          |
    | Action:                                  |
    |   1. Lookup customer webhook config      |
    |   2. Format webhook payload              |
    |   3. POST to customer endpoint           |
    |   4. Handle retries on failure           |
    +------------------------------------------+

    queue-manager:
    +------------------------------------------+
    | On: call_hungup                          |
    | Action:                                  |
    |   1. Check if call was from queue        |
    |   2. Update queue statistics             |
    |   3. Mark agent as available             |
    +------------------------------------------+

Real-Time Data Flow (WebSocket)
-------------------------------

WebSocket connections provide real-time updates to clients:

.. code::

    WebSocket Data Flow:

    Client           API Gateway          ZMQ Publisher        Backend Service
       |                  |                    |                      |
       | 1. WS Connect    |                    |                      |
       +----------------->|                    |                      |
       |                  | 2. Authenticate    |                      |
       |                  |    (JWT token)     |                      |
       |                  |                    |                      |
       | 3. Subscribe     |                    |                      |
       | {"type":"subscribe",                  |                      |
       |  "topics":["customer_id:123:call:*"]} |                      |
       +----------------->|                    |                      |
       |                  | 4. Register        |                      |
       |                  |    Subscription    |                      |
       |                  |                    |                      |
       |                  |                    |                      | 5. Call Starts
       |                  |                    |                      |    (business event)
       |                  |                    |                      |
       |                  |                    |<---------------------+
       |                  |                    | 6. ZMQ Publish       |
       |                  |                    |    topic: call.state |
       |                  |                    |                      |
       |                  |<-------------------+                      |
       |                  | 7. Match to        |                      |
       |                  |    Subscriptions   |                      |
       |                  |                    |                      |
       |<-----------------+                    |                      |
       | 8. Push Event    |                    |                      |
       | {"event":"call_created",...}          |                      |
       |                  |                    |                      |

**Topic Matching:**

.. code::

    Subscription Topic Matching:

    Subscribed Topic:
    customer_id:123:call:*

    Matches:
    +------------------------------------------+
    | customer_id:123:call:abc-456  [match]    |
    | customer_id:123:call:xyz-789  [match]    |
    | customer_id:123:call:*        [match]    |
    +------------------------------------------+

    Does Not Match:
    +------------------------------------------+
    | customer_id:456:call:abc-123  [no match] |
    | customer_id:123:conference:*  [no match] |
    +------------------------------------------+

Media Stream Data Flow
----------------------

Audio data flows through the media pipeline:

.. code::

    Audio Stream Flow (AI Voice):

    Caller        RTPEngine      Asterisk      pipecat-mgr       AI/LLM
       |              |             |               |               |
       | RTP Audio    |             |               |               |
       | (Various)    |             |               |               |
       +------------->|             |               |               |
       |              | Transcode   |               |               |
       |              | to ulaw     |               |               |
       |              +------------>|               |               |
       |              |             | Audiosocket   |               |
       |              |             | (8kHz ulaw)   |               |
       |              |             +-------------->|               |
       |              |             |               |               |
       |              |             |               | Resample to   |
       |              |             |               | 16kHz PCM     |
       |              |             |               |               |
       |              |             |               | WebSocket     |
       |              |             |               | (Protobuf)    |
       |              |             |               +-------------->|
       |              |             |               |               |
       |              |             |               |               | STT +
       |              |             |               |               | LLM +
       |              |             |               |               | TTS
       |              |             |               |               |
       |              |             |               |<--------------+
       |              |             |               | Audio Response|
       |              |             |               |               |
       |              |             |               | Resample to   |
       |              |             |               | 8kHz ulaw     |
       |              |             |               |               |
       |              |             |<--------------+               |
       |              |             | Audiosocket   |               |
       |              |             |               |               |
       |              |<------------+               |               |
       |              | RTP         |               |               |
       |<-------------+             |               |               |
       | Audio to     |             |               |               |
       | Caller       |             |               |               |

**Audio Format Transformations:**

.. code::

    Audio Format Pipeline:

    External (Varies)
    +------------------------------------------+
    | Codecs: G.711, G.722, Opus, etc.         |
    | Sample Rate: 8kHz - 48kHz                |
    | Bitrate: 64kbps - 510kbps                |
    +------------------------------------------+
              |
              | RTPEngine (Edge Transcoding)
              v
    Internal (Standard)
    +------------------------------------------+
    | Codec: G.711 ulaw                        |
    | Sample Rate: 8kHz                        |
    | Bitrate: 64kbps                          |
    +------------------------------------------+
              |
              | pipecat-manager (AI Processing)
              v
    AI Pipeline
    +------------------------------------------+
    | Format: PCM Linear                       |
    | Sample Rate: 16kHz                       |
    | Bit Depth: 16-bit                        |
    +------------------------------------------+

Database Write Flow
-------------------

Write operations follow a specific pattern for consistency:

.. code::

    Database Write Flow:

    Service Handler      Cache Handler      DB Handler       MySQL
          |                   |                 |              |
          | 1. Validate       |                 |              |
          |    Input          |                 |              |
          |                   |                 |              |
          | 2. Business       |                 |              |
          |    Logic          |                 |              |
          |                   |                 |              |
          | 3. Call DB Handler|                 |              |
          +---------------------------------->|              |
          |                   |                 |              |
          |                   |                 | 4. Begin    |
          |                   |                 |    Transaction
          |                   |                 +------------->|
          |                   |                 |              |
          |                   |                 | 5. INSERT/  |
          |                   |                 |    UPDATE   |
          |                   |                 +------------->|
          |                   |                 |              |
          |                   |                 |<-------------+
          |                   |                 | Success     |
          |                   |                 |              |
          |                   |                 | 6. COMMIT   |
          |                   |                 +------------->|
          |                   |                 |              |
          |                   |<----------------+              |
          |                   | Return ID       |              |
          |                   |                 |              |
          |                   | 7. Invalidate   |              |
          |                   |    Cache        |              |
          |<------------------+                 |              |
          |                   | DEL key         |              |
          |                   |                 |              |
          | 8. Publish Event  |                 |              |
          |    (RabbitMQ)     |                 |              |
          |                   |                 |              |

**Write Consistency Rules:**

.. code::

    Data Consistency:

    Order of Operations:
    +------------------------------------------+
    | 1. Write to database FIRST               |
    | 2. Invalidate cache SECOND               |
    | 3. Publish event THIRD                   |
    +------------------------------------------+

    Why This Order:
    +------------------------------------------+
    | o Database is source of truth            |
    | o Cache invalidation ensures freshness   |
    | o Events notify other services           |
    | o If publish fails, data still correct   |
    +------------------------------------------+

    Failure Handling:
    +------------------------------------------+
    | DB write fails  -> Rollback, return error|
    | Cache inv. fails-> Log, continue         |
    | Event pub. fails-> Log, retry async      |
    +------------------------------------------+

Campaign Execution Data Flow
----------------------------

Outbound campaigns involve complex data orchestration:

.. code::

    Campaign Data Flow:

    Scheduler      campaign-mgr    outdial-mgr       MySQL         call-mgr
        |              |               |               |               |
        | 1. Trigger   |               |               |               |
        |    Campaign  |               |               |               |
        +------------->|               |               |               |
        |              |               |               |               |
        |              | 2. Get        |               |               |
        |              |    Campaign   |               |               |
        |              +------------------------------>|               |
        |              |               |               |               |
        |              |<------------------------------+               |
        |              | Campaign Data |               |               |
        |              |               |               |               |
        |              | 3. Get Next   |               |               |
        |              |    Targets    |               |               |
        |              +-------------->|               |               |
        |              |               |               |               |
        |              |               | 4. Query      |               |
        |              |               |    Outplan    |               |
        |              |               +-------------->|               |
        |              |               |               |               |
        |              |               |<--------------+               |
        |              |               | Target List   |               |
        |              |               |               |               |
        |              |<--------------+               |               |
        |              | Dial Targets  |               |               |
        |              |               |               |               |
        |              | 5. For each target:           |               |
        |              | +-------------------------------------------+ |
        |              | |                             |             | |
        |              | | Create Call                 |             | |
        |              | +-------------------------------------------->|
        |              | |                             |             | |
        |              | |                             |<------------+ |
        |              | |                             | Call Created| |
        |              | |                             |             | |
        |              | +-------------------------------------------+ |
        |              |               |               |               |
        |              | 6. Subscribe  |               |               |
        |              |    call_hungup|               |               |
        |              |               |               |               |
        |              |                               |               |
        |              | (Later)       |               |               |
        |              | 7. Event:     |               |               |
        |              |    call_hungup|               |               |
        |              |<---------------------------------------------|
        |              |               |               |               |
        |              | 8. Update     |               |               |
        |              |    Campaign   |               |               |
        |              |    Status     |               |               |
        |              +------------------------------>|               |
        |              |               |               |               |

**Campaign State Machine:**

.. code::

    Campaign Data States:

    Campaign Record:
    +------------------------------------------+
    | status: pending -> running -> completed  |
    | total_targets: 1000                      |
    | dialed: 0 -> 500 -> 1000                 |
    | answered: 0 -> 250 -> 500                |
    | failed: 0 -> 50 -> 100                   |
    +------------------------------------------+

    Outplan (Dial Target):
    +------------------------------------------+
    | status: pending -> dialing -> completed  |
    | dial_count: 0 -> 1 -> 2                  |
    | last_dial_time: timestamp                |
    | result: null -> answered/busy/no_answer  |
    +------------------------------------------+

Transcription Data Flow
-----------------------

Real-time transcription processes audio streams:

.. code::

    Transcription Data Flow:

    Asterisk      call-mgr     transcribe-mgr     STT Provider      MySQL
        |             |              |                  |              |
        | Channel     |              |                  |              |
        | Up          |              |                  |              |
        +------------>|              |                  |              |
        |             |              |                  |              |
        |             | 1. Start     |                  |              |
        |             |    Transcribe|                  |              |
        |             +------------->|                  |              |
        |             |              |                  |              |
        |             |              | 2. Create        |              |
        |             |              |    Transcribe    |              |
        |             |              |    Record        |              |
        |             |              +-------------------------------->|
        |             |              |                  |              |
        |             |              | 3. Connect to    |              |
        |             |              |    STT Stream    |              |
        |             |              +----------------->|              |
        |             |              |                  |              |
        | Audio       |              |                  |              |
        | Stream      |              |                  |              |
        +-------------------------->|                  |              |
        |             |              | Audio Chunks     |              |
        |             |              +----------------->|              |
        |             |              |                  |              |
        |             |              |                  | 4. Process   |
        |             |              |                  |    Audio     |
        |             |              |                  |              |
        |             |              |<-----------------+              |
        |             |              | Transcript       |              |
        |             |              | Segment          |              |
        |             |              |                  |              |
        |             |              | 5. Save          |              |
        |             |              |    Transcript    |              |
        |             |              +-------------------------------->|
        |             |              |                  |              |
        |             |              | 6. Publish       |              |
        |             |              |    Event         |              |
        |             |              | (transcript_created)            |
        |             |              |                  |              |

**Transcript Data Structure:**

.. code::

    Transcript Record:

    transcribes table:
    +------------------------------------------+
    | id: uuid                                 |
    | customer_id: uuid                        |
    | reference_type: "call" | "conference"    |
    | reference_id: uuid (call_id)             |
    | language: "en-US"                        |
    | status: "running" | "completed"          |
    +------------------------------------------+

    transcripts table (segments):
    +------------------------------------------+
    | id: uuid                                 |
    | transcribe_id: uuid                      |
    | direction: "in" | "out"                  |
    | message: "Hello, how can I help?"        |
    | tm_transcript: relative timestamp        |
    | tm_create: absolute timestamp            |
    +------------------------------------------+

Webhook Delivery Data Flow
--------------------------

Webhooks deliver events to external systems:

.. code::

    Webhook Delivery Flow:

    Event Source    webhook-mgr        MySQL         HTTP Client      External
         |              |                |               |               |
         | Event:       |                |               |               |
         | call_hungup  |                |               |               |
         +------------->|                |               |               |
         |              |                |               |               |
         |              | 1. Lookup      |               |               |
         |              |    Webhook     |               |               |
         |              |    Config      |               |               |
         |              +--------------->|               |               |
         |              |                |               |               |
         |              |<---------------+               |               |
         |              | Webhook URL,   |               |               |
         |              | Secret         |               |               |
         |              |                |               |               |
         |              | 2. Format      |               |               |
         |              |    Payload     |               |               |
         |              |                |               |               |
         |              | 3. Sign        |               |               |
         |              |    Payload     |               |               |
         |              |    (HMAC-SHA256)|              |               |
         |              |                |               |               |
         |              | 4. Create      |               |               |
         |              |    Delivery    |               |               |
         |              |    Record      |               |               |
         |              +--------------->|               |               |
         |              |                |               |               |
         |              | 5. POST        |               |               |
         |              |    Webhook     |               |               |
         |              +------------------------------>|               |
         |              |                |               |               |
         |              |                |               +-------------->|
         |              |                |               | HTTPS POST    |
         |              |                |               |               |
         |              |                |               |<--------------+
         |              |                |               | 200 OK        |
         |              |                |               |               |
         |              |<------------------------------+               |
         |              | Success        |               |               |
         |              |                |               |               |
         |              | 6. Update      |               |               |
         |              |    Delivery    |               |               |
         |              |    Status      |               |               |
         |              +--------------->|               |               |
         |              |                |               |               |

**Webhook Payload:**

.. code::

    Webhook HTTP Request:

    POST https://customer.example.com/webhook
    Content-Type: application/json
    X-VoIPBIN-Signature: sha256=abc123...
    X-VoIPBIN-Timestamp: 2026-01-20T12:00:00.000Z
    X-VoIPBIN-Event: call_hungup

    {
      "id": "event-uuid",
      "type": "call_hungup",
      "created": "2026-01-20T12:00:00.000Z",
      "data": {
        "id": "call-uuid",
        "customer_id": "customer-uuid",
        "source": "+15551234567",
        "destination": "+15559876543",
        "duration": 120,
        "status": "completed",
        "hangup_cause": "normal_clearing"
      }
    }

**Signature Verification (Customer Side):**

.. code::

    Signature Verification:

    1. Extract signature from header:
       X-VoIPBIN-Signature: sha256=abc123...

    2. Compute expected signature:
       expected = HMAC-SHA256(
         secret = "webhook_secret",
         message = timestamp + "." + body
       )

    3. Compare:
       if (signature == expected) {
         // Valid webhook
       } else {
         // Reject - possible tampering
       }

Data Synchronization Patterns
-----------------------------

Services maintain data consistency through patterns:

.. code::

    Cache-Aside Pattern:

    Service              Redis              MySQL
       |                   |                  |
       | 1. Get Call       |                  |
       +------------------>|                  |
       |                   |                  |
       | Cache Miss        |                  |
       |<------------------+                  |
       |                   |                  |
       | 2. Query DB       |                  |
       +------------------------------------->|
       |                   |                  |
       |<-------------------------------------+
       | Call Data         |                  |
       |                   |                  |
       | 3. Store in Cache |                  |
       | (TTL: 24 hours)   |                  |
       +------------------>|                  |
       |                   |                  |
       | 4. Return Data    |                  |
       |                   |                  |

.. code::

    Write-Through Pattern:

    Service              MySQL              Redis
       |                   |                  |
       | 1. Update Call    |                  |
       +------------------>|                  |
       |                   |                  |
       |<------------------+                  |
       | Commit Success    |                  |
       |                   |                  |
       | 2. Invalidate     |                  |
       |    Cache          |                  |
       +------------------------------------->|
       |                   |                  |
       |<-------------------------------------+
       | DEL Success       |                  |
       |                   |                  |

.. code::

    Event Sourcing (for Audit):

    Service              MySQL              Audit Log
       |                   |                  |
       | 1. Action:        |                  |
       |    Delete Call    |                  |
       |                   |                  |
       | 2. Write to       |                  |
       |    calls table    |                  |
       +------------------>|                  |
       |                   |                  |
       | 3. Write to       |                  |
       |    audit_log      |                  |
       +------------------------------------->|
       |                   |                  |
       | Record:           |                  |
       | - action: delete  |                  |
       | - resource: call  |                  |
       | - actor: agent_id |                  |
       | - timestamp       |                  |
       | - before_state    |                  |
       |                   |                  |

