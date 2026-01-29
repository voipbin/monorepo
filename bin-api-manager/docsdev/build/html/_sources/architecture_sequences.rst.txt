.. _architecture-sequences:

Call Flow Sequences
===================

This section provides detailed sequence diagrams for VoIPBIN's core call flows, showing how components interact during real-world scenarios.

Inbound Call Flow
-----------------

When an external caller dials a VoIPBIN number, the following sequence occurs:

.. code::

    Inbound Call Flow:

    PSTN Carrier    Kamailio     Asterisk    asterisk-proxy   call-manager    flow-manager
         |             |            |              |                |               |
         |  SIP INVITE |            |              |                |               |
         +------------>|            |              |                |               |
         |             | Route      |              |                |               |
         |             +----------->|              |                |               |
         |             |            | Channel      |                |               |
         |             |            | Created      |                |               |
         |             |            +------------->|                |               |
         |             |            |              | Publish:       |               |
         |             |            |              | asterisk.all.event             |
         |             |            |              +--------------->|               |
         |             |            |              |                |               |
         |             |            |              |                | Create Call   |
         |             |            |              |                | Record        |
         |             |            |              |                |               |
         |             |            |              |                | Lookup Number |
         |             |            |              |                | -> Flow ID    |
         |             |            |              |                |               |
         |             |            |              |                | RPC: Start    |
         |             |            |              |                | ActiveFlow    |
         |             |            |              |                +-------------->|
         |             |            |              |                |               |
         |             |            |              |                |               | Execute
         |             |            |              |                |               | Actions
         |             |            |              |                |               |
         |             |            |              |  RPC: Answer   |               |
         |             |            |<-----------------------------------+----------+
         |             |            |              |                |               |
         |  200 OK     |            |              |                |               |
         |<------------+------------+              |                |               |
         |             |            |              |                |               |
         |   RTP Media Established  |              |                |               |
         |<------------------------>|              |                |               |
         |             |            |              |                |               |

**Key Components:**

1. **Kamailio** - Receives SIP INVITE, routes to appropriate Asterisk instance
2. **Asterisk** - Creates SIP channel, generates ARI events via ARI WebSocket
3. **asterisk-proxy** - Bridges ARI events to RabbitMQ (``asterisk.all.event`` queue)
4. **call-manager** - Processes events, creates call record, initiates flow
5. **flow-manager** - Executes the configured call flow (IVR actions)

**Event Routing in call-manager:**

.. code::

    asterisk-proxy Event Routing:

    asterisk.all.event
          |
          v
    +------------------+
    | subscribehandler |
    +--------+---------+
             |
             | Routes by event type
             v
    +------------------+
    | arieventhandler  |
    +--------+---------+
             |
     +-------+-------+
     |               |
     v               v
    +----------+ +----------+
    |channelhdl| |bridgehdl |
    +----------+ +----------+
         |            |
         v            v
    Channel      Bridge
    Events       Events
    (create,     (join,
    hangup,      leave)
    dtmf)

**Channel Events:**

* ``StasisStart`` - Channel enters Stasis application (call starts)
* ``StasisEnd`` - Channel exits Stasis (call ends)
* ``ChannelDtmfReceived`` - DTMF digit pressed
* ``ChannelHangupRequest`` - Hangup initiated
* ``ChannelStateChange`` - Channel state changed (ringing, up, etc.)

**Bridge Events:**

* ``ChannelEnteredBridge`` - Participant joined bridge (conference)
* ``ChannelLeftBridge`` - Participant left bridge

Outbound Campaign Flow
----------------------

Outbound campaigns automate calling lists of targets:

.. code::

    Campaign Execution Flow:

    API Request    campaign-mgr    outdial-mgr    call-manager    flow-manager
         |             |               |               |               |
         | Start       |               |               |               |
         | Campaign    |               |               |               |
         +------------>|               |               |               |
         |             |               |               |               |
         |             | Get Targets   |               |               |
         |             | (Outplan)     |               |               |
         |             +-------------->|               |               |
         |             |               |               |               |
         |             |<--------------+               |               |
         |             | Dial Targets  |               |               |
         |             |               |               |               |
         |             | For each target:              |               |
         |             +------------------------------------------+    |
         |             |               |               |          |    |
         |             | RPC: Create   |               |          |    |
         |             | Outbound Call |               |          |    |
         |             +------------------------------>|          |    |
         |             |               |               |          |    |
         |             |               |               | Asterisk |    |
         |             |               |               | Originate|    |
         |             |               |               +--------->|    |
         |             |               |               |          |    |
         |             |               |               | Answer?  |    |
         |             |               |               |<---------+    |
         |             |               |               |          |    |
         |             |               |               | If answered:  |
         |             |               |               | Start Flow    |
         |             |               |               +-------------->|
         |             |               |               |               |
         |             |               |               |               | Execute
         |             |               |               |               | Actions
         |             |               |               |               | (play,
         |             |               |               |               |  gather,
         |             |               |               |               |  ai_talk)
         |             |               |               |               |
         |             | Event:        |               |               |
         |             | call_hungup   |               |               |
         |             |<------------------------------+               |
         |             |               |               |               |
         |             | Update        |               |               |
         |             | Campaign      |               |               |
         |             | Status        |               |               |
         |             +------------------------------------------+    |
         |             |                                               |
         |             | Continue with next target...                  |
         |             |                                               |

**Campaign Components:**

* **campaign-manager** - Orchestrates campaign execution, tracks progress
* **outdial-manager** - Manages dial targets (outplans), provides next numbers to dial
* **call-manager** - Creates and manages individual calls
* **flow-manager** - Executes call flow when target answers

**Campaign Events:**

.. code::

    Event Subscriptions:

    campaign-manager subscribes to:
    +--------------------------------+
    | o call_hungup                  | - Track call completion
    | o activeflow_deleted           | - Track flow completion
    | o call_answered                | - Track answer rates
    +--------------------------------+

    Event triggers campaign state updates:
    o Calculate dial success rate
    o Move to next target
    o Update campaign statistics

AI Voice Assistant Flow (Pipecat)
---------------------------------

VoIPBIN's AI voice assistant uses a hybrid Go+Python architecture:

.. code::

    Pipecat AI Voice Architecture:

    Asterisk        pipecat-manager (Go)      pipecat-runner (Python)      LLM
       |                   |                          |                     |
       |                   |                          |                     |
       |  Audiosocket      |                          |                     |
       |  (8kHz ulaw)      |                          |                     |
       +------------------>|                          |                     |
       |                   |                          |                     |
       |                   | WebSocket                |                     |
       |                   | (16kHz PCM)              |                     |
       |                   +------------------------->|                     |
       |                   |                          |                     |
       |                   |                          | STT: Deepgram       |
       |                   |                          | "What's the weather?"|
       |                   |                          +-------------------->|
       |                   |                          |                     |
       |                   |                          |<--------------------+
       |                   |                          | LLM Response        |
       |                   |                          |                     |
       |                   |                          | TTS: Generate Audio |
       |                   |                          |                     |
       |                   |<-------------------------+                     |
       |                   | Audio Response           |                     |
       |                   |                          |                     |
       |<------------------+                          |                     |
       | Play to Caller    |                          |                     |
       |                   |                          |                     |

**Audio Processing Pipeline:**

.. code::

    Audio Resampling:

    Asterisk                 pipecat-manager               pipecat-runner
    (8kHz ulaw)             (Go Resampler)                (16kHz PCM)
        |                        |                             |
        | Audiosocket           |                             |
        | (8kHz ulaw)           |                             |
        +---------------------->|                             |
        |                       |                             |
        |                       | Resample                    |
        |                       | 8kHz -> 16kHz               |
        |                       | ulaw -> PCM                 |
        |                       |                             |
        |                       | WebSocket                   |
        |                       | (Protobuf frame)            |
        |                       +---------------------------->|
        |                       |                             |
        |                       | WebSocket                   |
        |                       | (Protobuf response)         |
        |                       |<----------------------------+
        |                       |                             |
        |                       | Resample                    |
        |                       | 16kHz -> 8kHz               |
        |                       | PCM -> ulaw                 |
        |                       |                             |
        |<----------------------+                             |
        | Audiosocket           |                             |
        | (8kHz ulaw)           |                             |
        |                       |                             |

**Why Hybrid Architecture:**

* **Go (pipecat-manager)**: Efficient audio handling, low-latency resampling, integration with VoIPBIN RPC
* **Python (pipecat-runner)**: Rich AI/ML ecosystem, Pipecat framework, easy LLM integration

**Protobuf Frame Format:**

.. code::

    Frame Message:
    +----------------------------------+
    | type: FrameType                  |
    |   o INPUT_AUDIO_RAW (16kHz PCM)  |
    |   o OUTPUT_AUDIO_RAW             |
    |   o CONTROL (start/stop)         |
    |   o LLM_FUNCTION_CALL            |
    |   o LLM_FUNCTION_CALL_RESULT     |
    +----------------------------------+
    | data: bytes (audio samples)      |
    +----------------------------------+
    | timestamp: int64                 |
    +----------------------------------+

**LLM Tool Calling:**

.. code::

    Tool Call Flow:

    LLM                    pipecat-runner          pipecat-manager        External API
     |                          |                        |                    |
     | "Transfer to sales"      |                        |                    |
     +------------------------->|                        |                    |
     |                          |                        |                    |
     |                          | Frame: LLM_FUNCTION_CALL                    |
     |                          | tool: "transfer_call"  |                    |
     |                          | args: {dept: "sales"}  |                    |
     |                          +----------------------->|                    |
     |                          |                        |                    |
     |                          |                        | RPC: Transfer      |
     |                          |                        | Call               |
     |                          |                        +------------------->|
     |                          |                        |                    |
     |                          |                        |<-------------------+
     |                          |                        | Success            |
     |                          |                        |                    |
     |                          | Frame: FUNCTION_CALL_RESULT                 |
     |                          | result: "transferred"  |                    |
     |                          |<-----------------------+                    |
     |                          |                        |                    |
     |<-------------------------+                        |                    |
     | "Transferred to sales"   |                        |                    |
     |                          |                        |                    |

**Available AI Tools:**

* ``transfer_call`` - Transfer to another extension/queue
* ``end_call`` - End the conversation
* ``send_sms`` - Send SMS to caller
* ``create_ticket`` - Create support ticket
* ``lookup_customer`` - Query CRM for customer info
* ``schedule_callback`` - Schedule callback appointment

Call Transfer Sequence
----------------------

Call transfers involve coordination between multiple services:

.. code::

    Blind Transfer Flow:

    Agent A      call-manager    flow-manager    Asterisk       Agent B
       |              |              |              |              |
       |  Transfer    |              |              |              |
       |  Request     |              |              |              |
       +------------->|              |              |              |
       |              |              |              |              |
       |              | Create       |              |              |
       |              | Transfer     |              |              |
       |              | Record       |              |              |
       |              |              |              |              |
       |              | RPC: Start   |              |              |
       |              | Transfer Flow|              |              |
       |              +------------->|              |              |
       |              |              |              |              |
       |              |              | Action:      |              |
       |              |              | Redirect     |              |
       |              |              +------------->|              |
       |              |              |              |              |
       |              |              |              | REFER        |
       |              |              |              +------------->|
       |              |              |              |              |
       |              |              |              |<-------------+
       |              |              |              | 200 OK       |
       |              |              |              |              |
       |              | Event:       |              |              |
       | Disconnected | transfer_    |              |              |
       |<-------------+ completed    |              |              |
       |              |              |              |              |
       |              |              |              | RTP Media    |
       |              |              |   Caller <------------------>|
       |              |              |              |              |

**Attended Transfer Flow:**

.. code::

    Attended Transfer:

    Agent A      call-manager    Asterisk      Agent B      Caller
       |              |              |            |            |
       |              |              |            |            |
       | Consult B    |              |            |            |
       +------------->|              |            |            |
       |              | Create       |            |            |
       |              | Consult Call |            |            |
       |              +------------->|            |            |
       |              |              +----------->|            |
       |              |              |            |            |
       |<------- Consult Active ---->|            |            |
       |              |              |            |            |
       | (Discusses with B)          |            |            |
       |              |              |            |            |
       | Complete     |              |            |            |
       | Transfer     |              |            |            |
       +------------->|              |            |            |
       |              | Bridge       |            |            |
       |              | B <-> Caller |            |            |
       |              +------------->|            |            |
       |              |              |<----------------------->|
       |              |              |    RTP Media            |
       |              |              |            |            |
       | Disconnected |              |            |            |
       |<-------------+              |            |            |
       |              |              |            |            |

Queue Call Distribution
-----------------------

Queue management distributes calls to available agents:

.. code::

    Queue Call Flow:

    Caller       flow-manager    queue-manager    agent-manager    Agent
       |              |              |                 |             |
       | Incoming Call|              |                 |             |
       +------------->|              |                 |             |
       |              |              |                 |             |
       |              | Action:      |                 |             |
       |              | queue_join   |                 |             |
       |              +------------->|                 |             |
       |              |              |                 |             |
       |              |              | Get Available   |             |
       |              |              | Agents          |             |
       |              |              +---------------->|             |
       |              |              |                 |             |
       |              |              |<----------------+             |
       |              |              | [agent1, agent2]|             |
       |              |              |                 |             |
       |              |              | Ring Strategy   |             |
       |              |              | (round-robin,   |             |
       |              |              |  longest-idle)  |             |
       |              |              |                 |             |
       |              |              | Offer Call      |             |
       |              |              +------------------------------>|
       |              |              |                 |             |
       |              |              |<------------------------------+
       |              |              | Agent Accepts   |             |
       |              |              |                 |             |
       |              |<-------------+                 |             |
       |              | Exit Queue   |                 |             |
       |              |              |                 |             |
       |              | Action:      |                 |             |
       |              | Connect      |                 |             |
       |              | Agent<->Caller                 |             |
       |              |              |                 |             |
       |<------- Media Connected ------------------------>|         |
       |              |              |                 |             |

**Queue Features:**

* **Ring Strategies**: round-robin, longest-idle, least-calls, ring-all
* **Queue Timeout**: Max wait time before alternative action
* **Queue Music**: Hold music or announcements while waiting
* **Position Announcements**: "You are caller number 3 in queue"
* **Agent Wrap-up**: Post-call processing time before next call

Conference Join Sequence
------------------------

Multi-party conferences use dedicated infrastructure:

.. code::

    Conference Join Flow:

    Participant    flow-manager    conf-manager    Asterisk-Conf
         |              |               |                |
         | Call Arrives |               |                |
         +------------->|               |                |
         |              |               |                |
         |              | Action:       |                |
         |              | conference_join               |
         |              +-------------->|                |
         |              |               |                |
         |              |               | Get/Create     |
         |              |               | Conference     |
         |              |               +--------------->|
         |              |               |                |
         |              |               |  ARI: Create   |
         |              |               |  Bridge        |
         |              |               |<---------------+
         |              |               |  bridge_id     |
         |              |               |                |
         |              |               | ARI: Add       |
         |              |               | Channel to     |
         |              |               | Bridge         |
         |              |               +--------------->|
         |              |               |                |
         |              |<--------------+                |
         |              | Participant   |                |
         |              | Joined        |                |
         |              |               |                |
         | Audio Mixed  |               |                |
         |<-------------------------------------------->|
         |              |               |                |
         |              | Event:        |                |
         |              | confbridge_   |                |
         |              | joined        |                |
         |              +-------------->|                |
         |              |               |                |

**Conference Events Published:**

.. code::

    Conference Events:

    confbridge_joined
    +----------------------------------+
    | conference_id: uuid              |
    | participant_id: uuid             |
    | call_id: uuid                    |
    | participant_count: int           |
    +----------------------------------+

    confbridge_left
    +----------------------------------+
    | conference_id: uuid              |
    | participant_id: uuid             |
    | reason: "hangup" | "kick"        |
    | participant_count: int           |
    +----------------------------------+

    confbridge_record_started
    +----------------------------------+
    | conference_id: uuid              |
    | recording_id: uuid               |
    +----------------------------------+

Webhook Delivery Flow
---------------------

Events trigger webhook notifications to customer endpoints:

.. code::

    Webhook Delivery:

    call-manager    RabbitMQ     webhook-manager    Customer Endpoint
         |              |               |                   |
         | Event:       |               |                   |
         | call_hungup  |               |                   |
         +------------->|               |                   |
         |              |               |                   |
         |              | Fanout to     |                   |
         |              | Subscribers   |                   |
         |              +-------------->|                   |
         |              |               |                   |
         |              |               | Lookup Webhook    |
         |              |               | Config for        |
         |              |               | Customer          |
         |              |               |                   |
         |              |               | POST Event        |
         |              |               +------------------>|
         |              |               |                   |
         |              |               | Retry on Failure  |
         |              |               | (exponential      |
         |              |               |  backoff)         |
         |              |               |                   |
         |              |               |<------------------+
         |              |               | 200 OK            |
         |              |               |                   |
         |              |               | Mark Delivered    |
         |              |               |                   |

**Webhook Retry Policy:**

.. code::

    Retry Strategy:
    +----------------------------------+
    | Attempt 1:  Immediate            |
    | Attempt 2:  1 minute delay       |
    | Attempt 3:  5 minutes delay      |
    | Attempt 4:  30 minutes delay     |
    | Attempt 5:  2 hours delay        |
    +----------------------------------+
    | Max Attempts: 5                  |
    | Total Window: ~2.5 hours         |
    +----------------------------------+

**Webhook Payload:**

.. code::

    POST https://customer.example.com/webhook
    Content-Type: application/json
    X-VoIPBIN-Signature: sha256=...

    {
      "type": "call_hungup",
      "timestamp": "2026-01-20T12:00:00.000Z",
      "data": {
        "id": "call-123",
        "customer_id": "customer-456",
        "source": "+15551234567",
        "destination": "+15559876543",
        "duration": 120,
        "status": "completed",
        "hangup_cause": "normal_clearing"
      }
    }

