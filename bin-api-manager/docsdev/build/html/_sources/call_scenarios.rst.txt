.. _call-scenarios:

Advanced Call Scenarios
=======================

This section covers real-world call scenarios that combine multiple VoIPBIN features. Each scenario includes the complete flow, API examples, and best practices.

.. note:: **AI Implementation Hint**

   All scenarios in this section involve active outbound calls, which are chargeable. Each ``POST /calls`` request creates a billable call. Test your flow logic with a single destination before scaling to production. Use ``GET /calls/{call-id}`` to verify call status at each step.

IVR Menu with Queue Routing
---------------------------

A common contact center pattern: caller navigates an IVR menu, then enters a queue.

.. code::

    IVR to Queue Flow:

    Caller                          VoIPBIN                         Agent
       |                               |                              |
       | Calls support number          |                              |
       +------------------------------>|                              |
       |                               |                              |
       |<------------------------------+                              |
       | "Press 1 for Sales,           |                              |
       |  Press 2 for Support"         |                              |
       |                               |                              |
       | Press 2                       |                              |
       +------------------------------>|                              |
       |                               |                              |
       |<------------------------------+                              |
       | "Please hold, connecting      |                              |
       |  to support..."               |                              |
       |                               |                              |
       |<------------------------------+                              |
       | (Hold music plays)            |                              |
       |                               |                              |
       |                               | Find available agent         |
       |                               +----------------------------->|
       |                               |                              |
       |                               |<-----------------------------+
       |                               | Agent accepts                |
       |                               |                              |
       |<==============================+==============================>|
       | Connected to agent            |       Connected to caller    |
       |                               |                              |

**Flow Configuration:**

.. code::

    {
        "name": "Support IVR with Queue",
        "actions": [
            {
                "id": "welcome",
                "type": "talk",
                "option": {
                    "text": "Thank you for calling. Press 1 for Sales, Press 2 for Support, Press 3 for Billing.",
                    "language": "en-US"
                }
            },
            {
                "type": "digits_receive",
                "option": {
                    "duration": 10000,
                    "length": 1
                }
            },
            {
                "type": "branch",
                "option": {
                    "default_target_id": "invalid",
                    "target_ids": {
                        "1": "sales_queue",
                        "2": "support_queue",
                        "3": "billing_queue"
                    }
                }
            },
            {
                "id": "sales_queue",
                "type": "talk",
                "option": {
                    "text": "Connecting you to Sales. Please hold."
                }
            },
            {
                "type": "queue_join",
                "option": {
                    "queue_id": "sales-queue-uuid"
                }
            },
            {
                "type": "hangup"
            },
            {
                "id": "support_queue",
                "type": "talk",
                "option": {
                    "text": "Connecting you to Support. Please hold."
                }
            },
            {
                "type": "queue_join",
                "option": {
                    "queue_id": "support-queue-uuid"
                }
            },
            {
                "type": "hangup"
            },
            {
                "id": "billing_queue",
                "type": "talk",
                "option": {
                    "text": "Connecting you to Billing. Please hold."
                }
            },
            {
                "type": "queue_join",
                "option": {
                    "queue_id": "billing-queue-uuid"
                }
            },
            {
                "type": "hangup"
            },
            {
                "id": "invalid",
                "type": "talk",
                "option": {
                    "text": "Invalid selection."
                }
            },
            {
                "type": "goto",
                "option": {
                    "target_id": "welcome",
                    "loop_count": 2
                }
            },
            {
                "type": "talk",
                "option": {
                    "text": "Goodbye."
                }
            },
            {
                "type": "hangup"
            }
        ]
    }

.. note:: **AI Implementation Hint**

   The ``queue_id`` values (e.g., ``"sales-queue-uuid"``) must be valid UUIDs obtained from ``GET /queues``. The ``branch`` action's ``target_ids`` map DTMF digits to action ``id`` values within the same flow. If a ``target_id`` does not match any action ``id``, the flow will fail silently. Always include a ``default_target_id`` to handle unexpected input. The ``goto`` action's ``loop_count`` prevents infinite loops -- after the specified number of iterations, execution continues to the next action.

AI-Assisted Customer Service
----------------------------

Combine AI voice assistant with human escalation:

.. code::

    AI First, Human Backup:

    Caller                 AI Assistant              Agent
       |                        |                      |
       | "I need to check       |                      |
       |  my order status"      |                      |
       +----------------------->|                      |
       |                        |                      |
       |                        | Query CRM            |
       |                        | (tool call)          |
       |                        |                      |
       |<-----------------------+                      |
       | "Your order #12345     |                      |
       |  shipped yesterday.    |                      |
       |  Tracking: ABC123"     |                      |
       |                        |                      |
       | "I want to talk to     |                      |
       |  a real person"        |                      |
       +----------------------->|                      |
       |                        |                      |
       |<-----------------------+                      |
       | "Let me transfer you   |                      |
       |  to an agent"          |                      |
       |                        |                      |
       |                        | Transfer             |
       |                        | (tool call)          |
       |                        +--------------------->|
       |                        |                      |
       |<==============================================|
       | Connected to agent     |                      |
       |                        |                      |

**AI Assistant Configuration:**

.. code::

    POST /v1/calls
    {
        "source": {"type": "tel", "target": "+15551234567"},
        "destinations": [{"type": "tel", "target": "+15559876543"}],
        "actions": [
            {
                "type": "ai_talk",
                "option": {
                    "ai_id": "customer-service-ai-uuid",
                    "prompt": "You are a helpful customer service agent for Acme Corp. You can look up orders, check account balances, and answer questions about products. If the customer asks to speak to a human, transfer them to the support queue.",
                    "tools": [
                        {
                            "name": "lookup_order",
                            "description": "Look up order status by order ID or customer phone",
                            "webhook_url": "https://your-server.com/api/orders"
                        },
                        {
                            "name": "transfer_to_agent",
                            "description": "Transfer the call to a human agent",
                            "action": "transfer",
                            "destination": "queue:support-queue-uuid"
                        }
                    ],
                    "end_call_phrases": ["goodbye", "bye", "that's all"]
                }
            }
        ]
    }

.. note:: **AI Implementation Hint**

   The ``ai_id`` in the ``ai_talk`` action must be a valid UUID obtained from ``GET /ais``. Tool definitions with ``webhook_url`` will send HTTP POST requests to your server when the AI invokes the tool -- ensure your endpoint is publicly accessible and returns responses within 5 seconds. The ``end_call_phrases`` trigger automatic call hangup when the caller says any of the listed phrases.

Outbound Campaign with Voicemail Detection
------------------------------------------

Automated calling campaign that detects answering machines:

.. code::

    Campaign Call Flow:

    VoIPBIN             Destination              Voicemail
       |                    |                       |
       | Dial               |                       |
       +------------------->|                       |
       |                    |                       |
       | Answer?            |                       |
       |<-------------------+                       |
       |                    |                       |
       | AMD Analysis       |                       |
       | (first 3 seconds)  |                       |
       |                    |                       |
       +--- Human detected --+                      |
       |                    |                       |
       | Play message       |                       |
       +------------------->|                       |
       |                    |                       |
       | "Press 1 to speak  |                       |
       |  with an agent"    |                       |
       +------------------->|                       |
       |                    |                       |
       |                    |                       |
       +--- Machine detected ------------------------>|
       |                    |                       |
       | Leave voicemail    |                       |
       | message            |                       |
       +-------------------------------------->|    |
       |                    |                       |
       | Hangup             |                       |
       +-------------------------------------->|    |
       |                    |                       |

**Campaign Configuration:**

.. code::

    POST /v1/campaigns
    {
        "name": "Customer Reminder Campaign",
        "outplan_id": "outplan-uuid",
        "flow_id": "campaign-flow-uuid",
        "dial_timeout": 30000,
        "max_concurrent_calls": 10,
        "schedule": {
            "timezone": "America/New_York",
            "start_time": "09:00",
            "end_time": "17:00",
            "days": ["mon", "tue", "wed", "thu", "fri"]
        }
    }

    Campaign Flow:
    {
        "actions": [
            {
                "type": "amd",
                "option": {
                    "machine_action": "voicemail",
                    "human_action": "continue",
                    "timeout": 3000
                }
            },
            {
                "id": "human_path",
                "type": "talk",
                "option": {
                    "text": "Hello! This is a reminder from Acme Corp about your upcoming appointment. Press 1 to confirm, Press 2 to reschedule."
                }
            },
            {
                "type": "digits_receive",
                "option": {"duration": 10000, "length": 1}
            },
            {
                "type": "branch",
                "option": {
                    "target_ids": {
                        "1": "confirmed",
                        "2": "reschedule"
                    },
                    "default_target_id": "no_response"
                }
            },
            {
                "id": "confirmed",
                "type": "talk",
                "option": {"text": "Great! Your appointment is confirmed. Goodbye."}
            },
            {"type": "hangup"},
            {
                "id": "reschedule",
                "type": "talk",
                "option": {"text": "Please hold while we connect you to schedule a new time."}
            },
            {
                "type": "connect",
                "option": {
                    "destinations": [{"type": "tel", "target": "+15551234567"}]
                }
            },
            {
                "id": "voicemail",
                "type": "talk",
                "option": {
                    "text": "Hello, this is Acme Corp reminding you of your upcoming appointment. Please call us back at 555-123-4567 to confirm. Thank you."
                }
            },
            {"type": "hangup"},
            {
                "id": "no_response",
                "type": "goto",
                "option": {"target_id": "human_path", "loop_count": 2}
            },
            {"type": "hangup"}
        ]
    }

.. note:: **AI Implementation Hint**

   The ``outplan_id`` and ``flow_id`` must be valid UUIDs obtained from ``GET /outplans`` and ``GET /flows`` respectively. Each call placed by the campaign is individually chargeable. The ``amd`` (Answering Machine Detection) action analyzes the first few seconds of audio after answer -- if it detects a voicemail greeting, execution jumps to the action with ``id`` matching ``machine_action``. The ``max_concurrent_calls`` limits how many calls the campaign places at the same time.

Click-to-Call with Recording
----------------------------

Website visitor clicks to call, conversation is recorded:

.. code::

    Click-to-Call Flow:

    Website          Your Server        VoIPBIN           Visitor Phone      Agent
       |                 |                 |                    |              |
       | Click "Call Me" |                 |                    |              |
       +---------------->|                 |                    |              |
       |                 |                 |                    |              |
       |                 | POST /calls     |                    |              |
       |                 +---------------->|                    |              |
       |                 |                 |                    |              |
       |                 |                 | Call visitor       |              |
       |                 |                 +------------------->|              |
       |                 |                 |                    |              |
       |<----------------+<----------------+<-------------------+              |
       | "Calling you..."| Call created   | Ringing            |              |
       |                 |                 |                    |              |
       |                 |                 |<-------------------+              |
       |                 |                 | Answered           |              |
       |                 |                 |                    |              |
       |                 |                 | Start recording    |              |
       |                 |                 |                    |              |
       |                 |                 | Play greeting      |              |
       |                 |                 +------------------->|              |
       |                 |                 |                    |              |
       |                 |                 | Bridge to agent    |              |
       |                 |                 +---------------------------------->|
       |                 |                 |                    |              |
       |                 |                 |                    |<============>|
       |                 |                 |     Recording both directions     |
       |                 |                 |                    |              |

**API Request:**

.. code::

    POST /v1/calls
    {
        "source": {
            "type": "tel",
            "target": "+15551234567",
            "name": "Acme Support"
        },
        "destinations": [
            {
                "type": "tel",
                "target": "+15559876543"
            }
        ],
        "early_execution": false,
        "actions": [
            {
                "type": "talk",
                "option": {
                    "text": "Hello! Thank you for requesting a callback from Acme Corp. Please hold while we connect you to an agent.",
                    "language": "en-US"
                }
            },
            {
                "type": "record_start",
                "option": {
                    "direction": "both",
                    "format": "mp3"
                }
            },
            {
                "type": "connect",
                "option": {
                    "source": {"type": "tel", "target": "+15551234567"},
                    "destinations": [
                        {"type": "tel", "target": "+15552222222"}
                    ],
                    "ring_timeout": 30000
                }
            },
            {
                "type": "record_stop"
            }
        ]
    }

.. note:: **AI Implementation Hint**

   The ``source.target`` in ``POST /v1/calls`` must be a phone number you own, verified via ``GET /numbers``. The ``early_execution: false`` setting (default) ensures actions execute only after the destination answers. The ``record_start`` action must be placed before ``connect`` in the action list to capture the full conversation. After the call ends, retrieve the recording via ``GET /recordings/{recording-id}`` using the ID from the call's ``recording_ids`` array.

**Webhook Integration:**

.. code::

    Webhook: call_hungup
    {
        "type": "call_hungup",
        "data": {
            "id": "call-uuid",
            "duration": 245,
            "recording_ids": ["recording-uuid"],
            "hangup_by": "remote",
            "hangup_reason": "normal"
        }
    }

    Your Server Response:
    1. Fetch recording: GET /v1/recordings/{recording-uuid}
    2. Download audio: GET {recording.url}
    3. Store in your CRM
    4. Update call log with recording link

Multi-leg Conference Call
-------------------------

Create a conference with multiple participants joining at different times:

.. code::

    Multi-leg Conference:

    Organizer        VoIPBIN       Participant A    Participant B    Participant C
        |               |               |                |                |
        | Create        |               |                |                |
        | Conference    |               |                |                |
        +-------------->|               |                |                |
        |               |               |                |                |
        |<--------------+               |                |                |
        | Conf ID       |               |                |                |
        |               |               |                |                |
        | Add self      |               |                |                |
        +-------------->|               |                |                |
        |               |               |                |                |
        |<==============>               |                |                |
        | In conference |               |                |                |
        |               |               |                |                |
        | Dial out to A |               |                |                |
        +-------------->|               |                |                |
        |               | Call A        |                |                |
        |               +-------------->|                |                |
        |               |               |                |                |
        |               |<--------------+                |                |
        |               | Answered      |                |                |
        |               |               |                |                |
        |<==============><==============>                |                |
        | A joins conf  |               |                |                |
        |               |               |                |                |
        | Dial out to B |               |                |                |
        +-------------->|               |                |                |
        |               | Call B        |                |                |
        |               +------------------------------>|                |
        |               |               |                |                |
        |               |<------------------------------+                |
        |               | Answered      |                |                |
        |               |               |                |                |
        |<==============><==============><==============>|                |
        | B joins conf  |               |                |                |
        |               |               |                |                |
        |               |               |                | C dials in     |
        |               |               |                |                |
        |               |<---------------------------------------+-------+
        |               | Inbound call  |                |                |
        |               |               |                |                |
        |<==============><==============><==============>|<==============>|
        | All in conf   |               |                |                |

**Conference Creation:**

.. code::

    Step 1: Create Conference
    POST /v1/conferences
    {
        "name": "Weekly Team Sync",
        "customer_id": "customer-uuid"
    }

    Response:
    {
        "id": "conf-uuid",
        "name": "Weekly Team Sync",
        "status": "active",
        "participant_count": 0
    }

    Step 2: Add Organizer via Dial-in
    POST /v1/calls
    {
        "source": {"type": "tel", "target": "+15551111111"},
        "destinations": [{"type": "tel", "target": "+15550000000"}],
        "actions": [
            {
                "type": "conference_join",
                "option": {
                    "conference_id": "conf-uuid",
                    "role": "moderator",
                    "mute_on_join": false
                }
            }
        ]
    }

    Step 3: Dial Out to Participants
    POST /v1/calls
    {
        "source": {"type": "tel", "target": "+15551111111", "name": "Team Sync"},
        "destinations": [{"type": "tel", "target": "+15552222222"}],
        "actions": [
            {
                "type": "talk",
                "option": {
                    "text": "You are being connected to the Weekly Team Sync."
                }
            },
            {
                "type": "conference_join",
                "option": {
                    "conference_id": "conf-uuid",
                    "role": "participant",
                    "mute_on_join": false
                }
            }
        ]
    }

.. note:: **AI Implementation Hint**

   The ``conference_id`` in ``conference_join`` must be obtained from the ``id`` field in the ``POST /v1/conferences`` response or from ``GET /conferences``. The ``customer_id`` in the conference creation request must match your authenticated customer ID. Each dial-out to a participant creates a separate billable call. The ``role`` field controls permissions: ``moderator`` can mute/unmute others, while ``participant`` can only control their own audio.

Call Screening with Whisper
---------------------------

Screen calls before connecting to agent:

.. code::

    Call Screening Flow:

    Caller             VoIPBIN            Agent
       |                  |                  |
       | Incoming call    |                  |
       +----------------->|                  |
       |                  |                  |
       |<-----------------+                  |
       | "Please state    |                  |
       |  your name"      |                  |
       |                  |                  |
       | "John Smith"     |                  |
       +----------------->|                  |
       |                  |                  |
       | (Record name)    |                  |
       |                  |                  |
       |<-----------------+                  |
       | "Please hold"    |                  |
       |                  |                  |
       | (Hold music)     | Dial agent       |
       |                  +----------------->|
       |                  |                  |
       |                  |<-----------------+
       |                  | Agent answers    |
       |                  |                  |
       |                  | Whisper to agent |
       |                  | (caller can't hear)
       |                  +----------------->|
       |                  | "You have a call |
       |                  |  from John Smith"|
       |                  +----------------->|
       |                  | "Press 1 accept, |
       |                  |  2 to reject"    |
       |                  +----------------->|
       |                  |                  |
       |                  |<-----------------+
       |                  | Press 1          |
       |                  |                  |
       |<=================>==================>|
       | Connected        |                  |
       |                  |                  |

**Flow Configuration:**

.. code::

    {
        "actions": [
            {
                "type": "talk",
                "option": {
                    "text": "Please state your name after the beep."
                }
            },
            {
                "type": "record_voice",
                "option": {
                    "duration": 5000,
                    "silence_timeout": 2000,
                    "variable_name": "caller_name_recording"
                }
            },
            {
                "type": "talk",
                "option": {
                    "text": "Thank you. Please hold while we connect you."
                }
            },
            {
                "type": "moh_start",
                "option": {
                    "music_class": "default"
                }
            },
            {
                "type": "connect",
                "option": {
                    "destinations": [{"type": "tel", "target": "+15552222222"}],
                    "whisper": {
                        "enabled": true,
                        "message": "You have a call from:",
                        "play_recording": "{{caller_name_recording}}",
                        "accept_key": "1",
                        "reject_key": "2"
                    }
                }
            }
        ]
    }

.. note:: **AI Implementation Hint**

   The ``whisper`` feature in the ``connect`` action plays a message only to the agent (the caller cannot hear it). The ``record_voice`` action stores the recording in a flow variable (``caller_name_recording``) that can be referenced later using ``{{variable_name}}`` syntax. If the agent presses the ``reject_key``, the connect action fails and flow execution continues to the next action -- add a fallback (e.g., try another agent or go to voicemail).

Warm Transfer with Context
--------------------------

Transfer call with context passed to the receiving agent:

.. code::

    Warm Transfer Flow:

    Caller           Agent A          VoIPBIN         Agent B
       |                |                |                |
       |<===============>                |                |
       | Talking         |                |                |
       |                |                |                |
       |                | Initiate       |                |
       |                | transfer       |                |
       |                +--------------->|                |
       |                |                |                |
       |<---------------+                |                |
       | (On hold)      |                |                |
       |                |                | Call Agent B   |
       |                |                +--------------->|
       |                |                |                |
       |                |                |<---------------+
       |                |                | Answered       |
       |                |                |                |
       |                |<===============>================|
       |                | Talk to B      |                |
       |                | (Caller on hold)                |
       |                |                |                |
       |                | "Customer has  |                |
       |                |  billing issue"|                |
       |                |                |                |
       |                | Complete       |                |
       |                | transfer       |                |
       |                +--------------->|                |
       |                |                |                |
       |<===============================>=================|
       | Connected to B | (Disconnected) |                |
       |                |                |                |

**API for Attended Transfer:**

.. code::

    Step 1: Agent A initiates transfer
    POST /v1/calls/{call-id}/transfer
    {
        "type": "attended",
        "destination": {
            "type": "tel",
            "target": "+15553333333"
        },
        "context": {
            "customer_id": "cust-123",
            "issue": "billing dispute",
            "notes": "Customer called about incorrect charge on invoice #456"
        }
    }

    Response:
    {
        "transfer_id": "transfer-uuid",
        "consult_call_id": "consult-call-uuid",
        "status": "consulting"
    }

    Step 2: Agent A talks to Agent B, then completes
    POST /v1/transfers/{transfer-id}/complete

    Step 3: Agent B receives context via webhook or screen pop
    Webhook: transfer_completed
    {
        "type": "transfer_completed",
        "data": {
            "transfer_id": "transfer-uuid",
            "from_agent": "agent-a-uuid",
            "to_agent": "agent-b-uuid",
            "context": {
                "customer_id": "cust-123",
                "issue": "billing dispute",
                "notes": "Customer called about incorrect charge on invoice #456"
            }
        }
    }

.. note:: **AI Implementation Hint**

   The ``call-id`` in ``POST /v1/calls/{call-id}/transfer`` must be an active call in ``progressing`` status. The ``transfer_id`` returned in the response is used for subsequent operations: ``POST /v1/transfers/{transfer-id}/complete`` to finish the transfer, or ``POST /v1/transfers/{transfer-id}/cancel`` to abort it. The ``context`` object is passed through to the ``transfer_completed`` webhook, allowing Agent B's application to display caller information. The consult call to Agent B is a separate billable call.

Call with Real-time Transcription
---------------------------------

Transcribe call in real-time for live captioning or analysis:

.. code::

    Real-time Transcription:

    Caller          VoIPBIN         STT Service       Your Server
       |               |                |                  |
       | Speaking      |                |                  |
       +-------------->|                |                  |
       |               |                |                  |
       |               | Audio stream   |                  |
       |               +--------------->|                  |
       |               |                |                  |
       |               |<---------------+                  |
       |               | Transcript:    |                  |
       |               | "I need help"  |                  |
       |               |                |                  |
       |               | WebSocket push |                  |
       |               +---------------------------------->|
       |               |                |                  |
       |               |                |                  | Display
       |               |                |                  | caption
       |               |                |                  |
       | (continues)   |                |                  |
       +-------------->|                |                  |
       |               |                |                  |
       |               | More audio     |                  |
       |               +--------------->|                  |
       |               |                |                  |
       |               |<---------------+                  |
       |               | "with my order"|                  |
       |               |                |                  |
       |               +---------------------------------->|
       |               |                |                  |

**Enable Transcription:**

.. code::

    POST /v1/calls
    {
        "source": {"type": "tel", "target": "+15551234567"},
        "destinations": [{"type": "tel", "target": "+15559876543"}],
        "actions": [
            {
                "type": "transcribe_start",
                "option": {
                    "language": "en-US",
                    "direction": "both"
                }
            },
            {
                "type": "connect",
                "option": {
                    "destinations": [{"type": "tel", "target": "+15552222222"}]
                }
            },
            {
                "type": "transcribe_stop"
            }
        ]
    }

    WebSocket subscription for real-time transcripts:
    {
        "type": "subscribe",
        "topics": ["customer_id:<your-id>:transcript:*"]
    }

    Received events:
    {
        "type": "transcript_created",
        "data": {
            "transcribe_id": "transcribe-uuid",
            "direction": "in",
            "message": "I need help with my order",
            "tm_transcript": "0001-01-01 00:00:05.123"
        }
    }

.. note:: **AI Implementation Hint**

   Real-time transcription events are delivered via WebSocket, not webhooks. You must establish a WebSocket connection to ``wss://api.voipbin.net/v1.0/ws?token=<token>`` and subscribe to the ``customer_id:<your-id>:transcript:*`` topic before the call starts to receive all events. The ``direction`` field in transcript events indicates which party is speaking: ``in`` for the caller, ``out`` for the agent/system. Place ``transcribe_start`` before ``connect`` in the action list to capture transcription from the beginning of the conversation.
