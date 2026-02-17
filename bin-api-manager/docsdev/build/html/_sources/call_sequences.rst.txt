.. _call-sequences:

Internal Call Sequences
=======================

This section reveals how calls flow through VoIPBIN's internal components. Understanding these sequences helps developers debug issues and optimize their integrations.

.. note:: **AI Implementation Hint**

   These internal sequences are provided for debugging and understanding purposes. As an API consumer, you interact only with ``api-manager`` (via REST API) and receive events via webhooks or WebSocket. You do not need to interact with internal components (Kamailio, Asterisk, RabbitMQ) directly.

Inbound PSTN Call Flow
----------------------

When someone calls your VoIPBIN number from a regular phone:

.. code::

    Inbound PSTN Call - Complete Internal Flow:

    PSTN Carrier     Kamailio        Asterisk      asterisk-proxy    call-manager     flow-manager
         |              |               |               |                |                |
         | SIP INVITE   |               |               |                |                |
         +------------->|               |               |                |                |
         |              |               |               |                |                |
         |              | Route lookup  |               |                |                |
         |              | (destination  |               |                |                |
         |              |  number)      |               |                |                |
         |              |               |               |                |                |
         |              | SIP INVITE    |               |                |                |
         |              +-------------->|               |                |                |
         |              |               |               |                |                |
         |              |               | StasisStart   |                |                |
         |              |               | (ARI event)   |                |                |
         |              |               +-------------->|                |                |
         |              |               |               |                |                |
         |              |               |               | RabbitMQ:      |                |
         |              |               |               | asterisk.all.event              |
         |              |               |               +--------------->|                |
         |              |               |               |                |                |
         |              |               |               |                | Create Call    |
         |              |               |               |                | Record (MySQL) |
         |              |               |               |                |                |
         |              |               |               |                | Lookup Number  |
         |              |               |               |                | -> Flow ID     |
         |              |               |               |                |                |
         |              |               |               |                | RPC: Start     |
         |              |               |               |                | ActiveFlow     |
         |              |               |               |                +--------------->|
         |              |               |               |                |                |
         |              |               |               |                |                | Create
         |              |               |               |                |                | ActiveFlow
         |              |               |               |                |                |
         |              |               |               |                |                | Execute
         |              |               |               |                |                | Action 1
         |              |               |               |                |                |
         |              |               |               |  RPC: Answer   |                |
         |              |               |<--------------------------------------+---------+
         |              |               |               |                |                |
         |              | 200 OK        |               |                |                |
         |<-------------+<--------------+               |                |                |
         |              |               |               |                |                |
         | ACK          |               |               |                |                |
         +------------->+-------------->|               |                |                |
         |              |               |               |                |                |
         |   RTP Media  |   RTP Media   |               |                |                |
         |<============>|<=============>|               |                |                |
         |              |               |               |                |                |

**Key Internal Components:**

.. code::

    Component Responsibilities:

    Kamailio (SIP Proxy):
    +------------------------------------------+
    | o Receives SIP from carriers             |
    | o Authenticates SIP trunks               |
    | o Routes to appropriate Asterisk         |
    | o Load balances across Asterisk farm     |
    | o Handles NAT traversal                  |
    +------------------------------------------+

    Asterisk (Media Server):
    +------------------------------------------+
    | o Manages call channels                  |
    | o Handles media (RTP)                    |
    | o Executes Stasis application            |
    | o Provides ARI (Asterisk REST Interface) |
    | o Bridges channels for conferencing      |
    +------------------------------------------+

    asterisk-proxy:
    +------------------------------------------+
    | o Connects to Asterisk ARI WebSocket     |
    | o Translates ARI events to RabbitMQ      |
    | o Publishes to asterisk.all.event queue  |
    | o One proxy per Asterisk instance        |
    +------------------------------------------+

    call-manager:
    +------------------------------------------+
    | o Processes ARI events                   |
    | o Creates/updates call records           |
    | o Manages call lifecycle                 |
    | o Triggers flow execution                |
    | o Publishes call events                  |
    +------------------------------------------+

    flow-manager:
    +------------------------------------------+
    | o Executes flow actions sequentially     |
    | o Sends ARI commands via call-manager    |
    | o Manages activeflow state               |
    | o Handles branching and loops            |
    +------------------------------------------+

.. note:: **AI Implementation Hint**

   For inbound calls, the flow that executes is determined by the phone number configuration. The destination number is looked up to find the associated ``flow_id``. Configure your number's flow assignment via ``PUT /numbers/{number-id}`` with the ``flow_id`` field. Verify your number configuration with ``GET /numbers/{number-id}``.

Outbound API Call Flow
----------------------

When you create a call via the API:

.. code::

    Outbound API Call - Complete Flow:

    Your App      api-manager     call-manager    flow-manager     Asterisk       Destination
        |              |               |               |               |               |
        | POST /calls  |               |               |               |               |
        +------------->|               |               |               |               |
        |              |               |               |               |               |
        |              | Validate JWT  |               |               |               |
        |              |               |               |               |               |
        |              | RPC: Create   |               |               |               |
        |              | Call          |               |               |               |
        |              +-------------->|               |               |               |
        |              |               |               |               |               |
        |              |               | Create Call   |               |               |
        |              |               | Record        |               |               |
        |              |               |               |               |               |
        |              |               | Create temp   |               |               |
        |              |               | Flow (if      |               |               |
        |              |               | actions given)|               |               |
        |              |               |               |               |               |
        |              |               | RPC: Start    |               |               |
        |              |               | ActiveFlow    |               |               |
        |              |               +-------------->|               |               |
        |              |               |               |               |               |
        |              |               |               | Create        |               |
        |              |               |               | ActiveFlow    |               |
        |              |               |               |               |               |
        |              |               |               | RPC: Originate|               |
        |              |               |<--------------+               |               |
        |              |               |               |               |               |
        |              |               | ARI: Originate|               |               |
        |              |               +------------------------------>|               |
        |              |               |               |               |               |
        |              |               |               |               | SIP INVITE    |
        |              |               |               |               +-------------->|
        |              |               |               |               |               |
        |<-------------+<--------------+               |               |               |
        | Call object  |               |               |               |               |
        | status:dialing               |               |               |               |
        |              |               |               |               |               |
        |              |               |               |               | 180 Ringing   |
        |              |               |               |               |<--------------+
        |              |               |               |               |               |
        |              |               | ChannelState  |               |               |
        |              |               | (ringing)     |               |               |
        |              |               |<------------------------------+               |
        |              |               |               |               |               |
        | Webhook:     |               |               |               |               |
        | call_ringing |               |               |               |               |
        |<-------------+<--------------+               |               |               |
        |              |               |               |               |               |
        |              |               |               |               | 200 OK        |
        |              |               |               |               |<--------------+
        |              |               |               |               |               |
        |              |               | StasisStart   |               |               |
        |              |               | (answered)    |               |               |
        |              |               |<------------------------------+               |
        |              |               |               |               |               |
        |              |               | Update status |               |               |
        |              |               | = progressing |               |               |
        |              |               |               |               |               |
        |              |               | Event:        |               |               |
        |              |               | call_answered |               |               |
        |              |               +-------------->|               |               |
        |              |               |               |               |               |
        |              |               |               | Execute       |               |
        |              |               |               | Actions       |               |
        | Webhook:     |               |               |               |               |
        | call_answered|               |               |               |               |
        |<-------------+<--------------+               |               |               |
        |              |               |               |               |               |

.. note:: **AI Implementation Hint**

   The ``POST /calls`` response returns immediately with status ``dialing``. The call has not connected yet at this point. To know when the call is answered, either poll ``GET /calls/{call-id}`` for status ``progressing``, or subscribe to the ``call_answered`` webhook event. If you provide ``actions`` in the ``POST /calls`` request, a temporary flow is created automatically -- you do not need to create a flow separately via ``POST /flows``.

WebRTC Call Flow
----------------

WebRTC calls (browser-to-phone or browser-to-browser):

.. code::

    WebRTC Inbound Call Flow:

    Browser        api-manager    talk-manager     Asterisk        Destination
        |              |               |               |               |
        | WSS Connect  |               |               |               |
        +------------->|               |               |               |
        |              |               |               |               |
        | SDP Offer    |               |               |               |
        +------------->|               |               |               |
        |              |               |               |               |
        |              | RPC: Create   |               |               |
        |              | WebRTC Call   |               |               |
        |              +-------------->|               |               |
        |              |               |               |               |
        |              |               | ARI: Create   |               |
        |              |               | WebRTC Channel|               |
        |              |               +-------------->|               |
        |              |               |               |               |
        |              |               |<--------------+               |
        |              |               | SDP Answer    |               |
        |              |               |               |               |
        |<-------------+<--------------+               |               |
        | SDP Answer   |               |               |               |
        |              |               |               |               |
        | ICE Candidates               |               |               |
        |<------------>|<------------->|<------------>|               |
        |              |               |               |               |
        | DTLS-SRTP    |               |               |               |
        | Handshake    |               |               |               |
        |<============>|<=============>|<=============>|               |
        |              |               |               |               |
        |              |               |               | Bridge to     |
        |              |               |               | Destination   |
        |              |               |               +-------------->|
        |              |               |               |               |
        |     Secure Media (SRTP)      |               |               |
        |<============>|<=============>|<=============>|<=============>|
        |              |               |               |               |

**WebRTC vs PSTN Differences:**

.. code::

    Protocol Comparison:

    +------------------+------------------+------------------+
    | Aspect           | PSTN Call        | WebRTC Call      |
    +------------------+------------------+------------------+
    | Signaling        | SIP over UDP/TCP | WebSocket + SDP  |
    | Media            | RTP              | SRTP (encrypted) |
    | NAT Traversal    | Kamailio handles | ICE/STUN/TURN    |
    | Codec negotiation| SIP SDP          | WebRTC SDP       |
    | Entry point      | Kamailio         | api-manager      |
    | Audio quality    | G.711 (64kbps)   | Opus (variable)  |
    +------------------+------------------+------------------+

.. note:: **AI Implementation Hint**

   WebRTC calls enter through ``api-manager`` via WebSocket (WSS), not through Kamailio like PSTN calls. This means WebRTC calls require a valid authentication token for the WebSocket connection. If a WebRTC call fails to connect, check that the ICE candidate exchange completes successfully -- common issues include STUN/TURN server misconfiguration or corporate firewalls blocking UDP.

SIP Trunk Call Flow
-------------------

Calls between registered SIP endpoints (extensions):

.. code::

    SIP Trunk Extension-to-Extension:

    Extension A     Kamailio       registrar-mgr     Asterisk      Extension B
         |              |               |               |               |
         | REGISTER     |               |               |               |
         +------------->|               |               |               |
         |              | Store         |               |               |
         |              | Registration  |               |               |
         |              +-------------->|               |               |
         |              |               |               |               |
         |<-------------+               |               |               |
         | 200 OK       |               |               |               |
         |              |               |               |               |
         |              |               |               |               | REGISTER
         |              |               |               |               |<-----------
         |              |               |               |               |
         |              |               |               |               | 200 OK
         |              |               |               |               +----------->
         |              |               |               |               |
         | INVITE       |               |               |               |
         | (to ext B)   |               |               |               |
         +------------->|               |               |               |
         |              |               |               |               |
         |              | Lookup        |               |               |
         |              | Registration  |               |               |
         |              +-------------->|               |               |
         |              |               |               |               |
         |              |<--------------+               |               |
         |              | Contact URI   |               |               |
         |              |               |               |               |
         |              | Route via     |               |               |
         |              | Asterisk      |               |               |
         |              +------------------------------>|               |
         |              |               |               |               |
         |              |               |               | INVITE        |
         |              |               |               +-------------->|
         |              |               |               |               |
         |              |               |               |<--------------+
         |              |               |               | 200 OK        |
         |              |               |               |               |
         |<-------------+<------------------------------+               |
         | 200 OK       |               |               |               |
         |              |               |               |               |
         |   RTP Media (possibly via RTPEngine)         |               |
         |<============>|<=============>|<=============>|<=============>|
         |              |               |               |               |

Call Recording Sequence
-----------------------

How recording starts and stops:

.. code::

    Recording Start Sequence:

    flow-manager    call-manager     Asterisk      storage-manager    GCS
         |               |               |               |              |
         | Action:       |               |               |              |
         | record_start  |               |               |              |
         |               |               |               |              |
         | RPC: Start    |               |               |              |
         | Recording     |               |               |              |
         +-------------->|               |               |              |
         |               |               |               |              |
         |               | Create        |               |              |
         |               | Recording     |               |              |
         |               | Record (DB)   |               |              |
         |               |               |               |              |
         |               | ARI: Record   |               |              |
         |               | Channel       |               |              |
         |               +-------------->|               |              |
         |               |               |               |              |
         |               |               | Start mixing  |              |
         |               |               | audio to file |              |
         |               |               |               |              |
         |<--------------+               |               |              |
         | Recording ID  |               |               |              |
         |               |               |               |              |
         |               |               | (Call continues...)          |
         |               |               |               |              |
         | Action:       |               |               |              |
         | record_stop   |               |               |              |
         | (or hangup)   |               |               |              |
         |               |               |               |              |
         | RPC: Stop     |               |               |              |
         | Recording     |               |               |              |
         +-------------->|               |               |              |
         |               |               |               |              |
         |               | ARI: Stop     |               |              |
         |               | Recording     |               |              |
         |               +-------------->|               |              |
         |               |               |               |              |
         |               |<--------------+               |              |
         |               | Local file    |               |              |
         |               | path          |               |              |
         |               |               |               |              |
         |               | RPC: Upload   |               |              |
         |               | File          |               |              |
         |               +------------------------------>|              |
         |               |               |               |              |
         |               |               |               | Upload to    |
         |               |               |               | GCS bucket   |
         |               |               |               +------------->|
         |               |               |               |              |
         |               |               |               |<-------------+
         |               |               |               | URL          |
         |               |               |               |              |
         |               |<------------------------------+              |
         |               | Recording URL |               |              |
         |               |               |               |              |
         |               | Update        |               |              |
         |               | Recording     |               |              |
         |               | Record        |               |              |
         |               |               |               |              |

.. note:: **AI Implementation Hint**

   Recording upload to cloud storage is asynchronous -- the recording URL is not available immediately after the call ends. Poll ``GET /recordings/{recording-id}`` until the ``url`` field is populated. Signed URLs expire after 1 hour; fetch a fresh URL from the API each time you need to download. If the call hangs up before ``record_stop`` executes, the recording is stopped and uploaded automatically.

**Recording File Lifecycle:**

.. code::

    Recording Storage Flow:

    1. During Call:
       +------------------------------------------+
       | Location: Asterisk local disk            |
       | Format: WAV (uncompressed)               |
       | Path: /var/spool/asterisk/recording/     |
       +------------------------------------------+

    2. After Call Ends:
       +------------------------------------------+
       | Action: Convert to final format          |
       | Format: MP3 or WAV (configurable)        |
       | Compress for storage efficiency          |
       +------------------------------------------+

    3. Upload to Cloud:
       +------------------------------------------+
       | Destination: Google Cloud Storage        |
       | Bucket: recordings-<customer-id>         |
       | Path: /<date>/<recording-id>.mp3         |
       | Access: Signed URLs (time-limited)       |
       +------------------------------------------+

    4. Cleanup:
       +------------------------------------------+
       | Local file: Deleted after upload         |
       | Cloud retention: 90 days (default)       |
       | Customer can download before expiry      |
       +------------------------------------------+

AI Voice Call Sequence
----------------------

Calls with AI assistant (Pipecat integration):

.. code::

    AI Voice Call - Detailed Flow:

    Caller      Asterisk    pipecat-manager    pipecat-runner       LLM
       |           |              |                  |                |
       | Call      |              |                  |                |
       | Answered  |              |                  |                |
       +---------->|              |                  |                |
       |           |              |                  |                |
       |           | Audiosocket  |                  |                |
       |           | Connect      |                  |                |
       |           | (port 9000)  |                  |                |
       |           +------------->|                  |                |
       |           |              |                  |                |
       |           |              | Spawn Python     |                |
       |           |              | Process          |                |
       |           |              +----------------->|                |
       |           |              |                  |                |
       |           |              |<-----------------+                |
       |           |              | WebSocket Ready  |                |
       |           |              |                  |                |
       |           |<-------------+                  |                |
       |           | Audiosocket  |                  |                |
       |           | Connected    |                  |                |
       |           |              |                  |                |
       | Speak:    |              |                  |                |
       | "Hello"   |              |                  |                |
       +---------->|              |                  |                |
       |           |              |                  |                |
       |           | Audio Frame  |                  |                |
       |           | (8kHz ulaw)  |                  |                |
       |           +------------->|                  |                |
       |           |              |                  |                |
       |           |              | Resample         |                |
       |           |              | 8kHz->16kHz      |                |
       |           |              | ulaw->PCM        |                |
       |           |              |                  |                |
       |           |              | Protobuf Frame   |                |
       |           |              | INPUT_AUDIO_RAW  |                |
       |           |              +----------------->|                |
       |           |              |                  |                |
       |           |              |                  | STT: Deepgram  |
       |           |              |                  | "Hello"        |
       |           |              |                  +--------------->|
       |           |              |                  |                |
       |           |              |                  |<---------------+
       |           |              |                  | LLM Response   |
       |           |              |                  |                |
       |           |              |                  | TTS: Generate  |
       |           |              |                  | Audio          |
       |           |              |                  |                |
       |           |              |<-----------------+                |
       |           |              | Protobuf Frame   |                |
       |           |              | OUTPUT_AUDIO_RAW |                |
       |           |              |                  |                |
       |           |              | Resample         |                |
       |           |              | 16kHz->8kHz      |                |
       |           |              | PCM->ulaw        |                |
       |           |              |                  |                |
       |           |<-------------+                  |                |
       |           | Audio Frame  |                  |                |
       |           |              |                  |                |
       |<----------+              |                  |                |
       | AI speaks |              |                  |                |
       |           |              |                  |                |

.. note:: **AI Implementation Hint**

   The AI voice call uses Audiosocket (port 9000) to bridge Asterisk audio to the Pipecat pipeline. Audio is resampled from Asterisk's native 8kHz ulaw to 16kHz PCM for the STT/LLM processing, then resampled back for playback. This resampling happens automatically. If the AI voice sounds unnatural or there is high latency, the issue is typically in the LLM response time or TTS generation, not in the audio pipeline.

**AI Tool Calling Sequence:**

.. code::

    LLM Tool Call (e.g., Transfer):

    pipecat-runner    pipecat-manager    call-manager    transfer-manager
          |                 |                 |                |
          | Frame:          |                 |                |
          | LLM_FUNCTION_CALL                 |                |
          | tool: transfer_call               |                |
          | args: {dest: "sales"}             |                |
          +---------------->|                 |                |
          |                 |                 |                |
          |                 | RPC: Execute    |                |
          |                 | Tool            |                |
          |                 +---------------->|                |
          |                 |                 |                |
          |                 |                 | RPC: Transfer  |
          |                 |                 +--------------->|
          |                 |                 |                |
          |                 |                 |                | Initiate
          |                 |                 |                | Transfer
          |                 |                 |                |
          |                 |                 |<---------------+
          |                 |                 | Success        |
          |                 |                 |                |
          |                 |<----------------+                |
          |                 | Tool Result     |                |
          |                 |                 |                |
          |<----------------+                 |                |
          | Frame:          |                 |                |
          | FUNCTION_RESULT |                 |                |
          | result: success |                 |                |
          |                 |                 |                |
          | (LLM receives   |                 |                |
          |  result, speaks |                 |                |
          |  confirmation)  |                 |                |

Conference Bridge Sequence
--------------------------

Multi-party conference call setup:

.. code::

    Conference Join Sequence:

    Participant    flow-manager    conf-manager      Asterisk
         |              |               |               |
         | Call         |               |               |
         | Answered     |               |               |
         +------------->|               |               |
         |              |               |               |
         |              | Action:       |               |
         |              | conf_join     |               |
         |              |               |               |
         |              | RPC: Join     |               |
         |              | Conference    |               |
         |              +-------------->|               |
         |              |               |               |
         |              |               | Get/Create    |
         |              |               | Conference    |
         |              |               | Record        |
         |              |               |               |
         |              |               | ARI: Create   |
         |              |               | Bridge        |
         |              |               | (if not exist)|
         |              |               +-------------->|
         |              |               |               |
         |              |               |<--------------+
         |              |               | Bridge ID     |
         |              |               |               |
         |              |               | ARI: Add      |
         |              |               | Channel to    |
         |              |               | Bridge        |
         |              |               +-------------->|
         |              |               |               |
         |              |               |               | Mix audio
         |              |               |               | with other
         |              |               |               | participants
         |              |               |               |
         |              |               |<--------------+
         |              |               | Success       |
         |              |               |               |
         |              |<--------------+               |
         |              | Participant   |               |
         |              | Joined        |               |
         |              |               |               |
         |<-------------+               |               |
         | Hear other   |               |               |
         | participants |               |               |
         |              |               |               |

**Conference Audio Mixing:**

.. code::

    Conference Bridge Audio Flow:

    +----------------+       +----------------+       +----------------+
    | Participant A  |       | Participant B  |       | Participant C  |
    +-------+--------+       +-------+--------+       +-------+--------+
            |                        |                        |
            | Audio A                | Audio B                | Audio C
            v                        v                        v
    +-----------------------------------------------------------------------+
    |                        Asterisk Bridge                                |
    |                                                                       |
    |   Audio Mixing:                                                       |
    |   +-----------------------------------------------------------+       |
    |   |  To A: Mix(B + C)                                         |       |
    |   |  To B: Mix(A + C)                                         |       |
    |   |  To C: Mix(A + B)                                         |       |
    |   +-----------------------------------------------------------+       |
    |                                                                       |
    |   Each participant hears everyone except themselves                   |
    +-----------------------------------------------------------------------+
            |                        |                        |
            | Mix(B+C)               | Mix(A+C)               | Mix(A+B)
            v                        v                        v
    +-------+--------+       +-------+--------+       +-------+--------+
    | Participant A  |       | Participant B  |       | Participant C  |
    +----------------+       +----------------+       +----------------+

.. note:: **AI Implementation Hint**

   Conference audio mixing is handled by Asterisk bridges internally. As an API consumer, you join participants to a conference via the ``conference_join`` flow action with a ``conference_id``. Each participant is a separate call (and separately billed). The conference bridge automatically handles audio mixing so each participant hears all others but not themselves.

Event Publication Sequence
--------------------------

How call events propagate through the system:

.. code::

    Event Publication Flow:

    Asterisk       call-manager        RabbitMQ        webhook-mgr     billing-mgr
        |               |                  |               |               |
        | StasisEnd     |                  |               |               |
        | (call ended)  |                  |               |               |
        +-------------->|                  |               |               |
        |               |                  |               |               |
        |               | Update Call      |               |               |
        |               | Record:          |               |               |
        |               | status=hangup    |               |               |
        |               |                  |               |               |
        |               | Publish Event:   |               |               |
        |               | call_hungup      |               |               |
        |               +----------------->|               |               |
        |               |                  |               |               |
        |               |                  | Fanout to     |               |
        |               |                  | subscribers   |               |
        |               |                  +-------------->|               |
        |               |                  |               |               |
        |               |                  +------------------------------>|
        |               |                  |               |               |
        |               |                  |               | Lookup        |
        |               |                  |               | webhook       |
        |               |                  |               | config        |
        |               |                  |               |               |
        |               |                  |               | POST to       |
        |               |                  |               | customer      |
        |               |                  |               | endpoint      |
        |               |                  |               |               |
        |               |                  |               |               | Calculate
        |               |                  |               |               | call cost
        |               |                  |               |               |
        |               |                  |               |               | Update
        |               |                  |               |               | balance

.. note:: **AI Implementation Hint**

   Events are published to RabbitMQ and fanned out to all subscribers. As an API consumer, you receive events via webhooks (HTTP POST to your endpoint) or via WebSocket subscription. Webhook delivery is retried up to 3 times with exponential backoff if your endpoint returns a non-2xx status code. Always return HTTP 200 immediately and process the webhook asynchronously to avoid timeouts.

**Event Types and Subscribers:**

.. code::

    Call Event Subscriptions:

    Event Type          Subscribers
    ─────────────────────────────────────────────────────────
    call_created        webhook-manager, campaign-manager
    call_ringing        webhook-manager
    call_answered       webhook-manager, billing-manager
    call_hungup         webhook-manager, billing-manager,
                        campaign-manager, queue-manager,
                        transfer-manager, ai-manager
    call_recording      webhook-manager, storage-manager
    call_transcribing   webhook-manager, transcribe-manager
