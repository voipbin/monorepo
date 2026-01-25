.. _flow-advanced-patterns:

Advanced Flow Patterns
======================

This section covers advanced flow design patterns for building sophisticated communication applications.

Multi-Level IVR with Sub-Menus
------------------------------

Building complex IVR systems with nested menus:

.. code::

    Multi-Level IVR Structure:

                              Main Menu
                                  |
            +---------------------+---------------------+
            |                     |                     |
         Press 1              Press 2              Press 3
         Sales                Support              Billing
            |                     |                     |
        +---+---+             +---+---+             +---+---+
        |       |             |       |             |       |
      Press 1 Press 2      Press 1 Press 2      Press 1 Press 2
      New     Existing     Tech    Account     Payment  Disputes
      Customer Customer    Support Support     Info

    Implementation Strategy:
    +------------------------------------------------------------------+
    | Use fetch_flow to load sub-menus dynamically                     |
    | This keeps each flow focused and maintainable                    |
    +------------------------------------------------------------------+


.. code::

    Main Menu Flow:

    {
      "name": "Main IVR Menu",
      "actions": [
        {
          "id": "main-answer",
          "type": "answer"
        },
        {
          "id": "main-greeting",
          "type": "talk",
          "option": {
            "text": "Welcome to Acme Corp. Press 1 for Sales, 2 for Support, 3 for Billing.",
            "language": "en-US"
          }
        },
        {
          "id": "main-input",
          "type": "digits_receive",
          "option": {
            "duration": 5000,
            "length": 1
          }
        },
        {
          "id": "main-branch",
          "type": "branch",
          "option": {
            "variable": "voipbin.call.digits",
            "default_target_id": "invalid-input",
            "target_ids": {
              "1": "goto-sales",
              "2": "goto-support",
              "3": "goto-billing"
            }
          }
        },
        {
          "id": "goto-sales",
          "type": "fetch_flow",
          "option": {
            "flow_id": "sales-submenu-flow-id"
          }
        },
        {
          "id": "goto-support",
          "type": "fetch_flow",
          "option": {
            "flow_id": "support-submenu-flow-id"
          }
        },
        {
          "id": "goto-billing",
          "type": "fetch_flow",
          "option": {
            "flow_id": "billing-submenu-flow-id"
          }
        },
        {
          "id": "invalid-input",
          "type": "talk",
          "option": {
            "text": "Invalid selection.",
            "language": "en-US"
          }
        },
        {
          "id": "retry-goto",
          "type": "goto",
          "option": {
            "target_id": "main-greeting",
            "loop_count": 3
          }
        },
        {
          "type": "talk",
          "option": {
            "text": "Goodbye.",
            "language": "en-US"
          }
        },
        {
          "type": "hangup"
        }
      ]
    }


Business Hours Routing
----------------------

Route calls differently based on time of day:

.. code::

    Business Hours Flow:

                           Incoming Call
                                |
                                v
                    +---------------------+
                    | condition_datetime  |
                    | (9 AM - 5 PM,       |
                    |  Mon-Fri)           |
                    +----------+----------+
                               |
              +----------------+----------------+
              |                                 |
         During Hours                    After Hours
              |                                 |
              v                                 v
    +------------------+              +------------------+
    | queue_join       |              | Voicemail Flow   |
    | (live agents)    |              | (leave message)  |
    +------------------+              +------------------+


.. code::

    {
      "name": "Business Hours Router",
      "actions": [
        {
          "type": "answer"
        },
        {
          "id": "check-hours",
          "type": "condition_datetime",
          "option": {
            "condition": ">=",
            "hour": 9,
            "minute": 0,
            "day": -1,
            "month": 0,
            "weekdays": [1, 2, 3, 4, 5],
            "false_target_id": "after-hours"
          }
        },
        {
          "id": "check-closing",
          "type": "condition_datetime",
          "option": {
            "condition": "<",
            "hour": 17,
            "minute": 0,
            "day": -1,
            "month": 0,
            "weekdays": [1, 2, 3, 4, 5],
            "false_target_id": "after-hours"
          }
        },
        {
          "id": "during-hours",
          "type": "talk",
          "option": {
            "text": "Thank you for calling. Connecting you to an agent.",
            "language": "en-US"
          }
        },
        {
          "type": "queue_join",
          "option": {
            "queue_id": "support-queue-id"
          }
        },
        {
          "type": "hangup"
        },
        {
          "id": "after-hours",
          "type": "talk",
          "option": {
            "text": "Our office is currently closed. Our hours are Monday through Friday, 9 AM to 5 PM. Please leave a message after the tone.",
            "language": "en-US"
          }
        },
        {
          "type": "beep"
        },
        {
          "type": "recording_start",
          "option": {
            "format": "mp3",
            "end_of_silence": 5,
            "duration": 120
          }
        },
        {
          "type": "talk",
          "option": {
            "text": "Thank you for your message. Goodbye.",
            "language": "en-US"
          }
        },
        {
          "type": "hangup"
        }
      ]
    }


Dynamic Flow with External Data
-------------------------------

Fetch customer data from your API to personalize the experience:

.. code::

    External Data Integration:

    +------------------------------------------------------------------+
    |                     Dynamic Flow Pattern                         |
    +------------------------------------------------------------------+

    1. Call arrives
    +----------------+
    | answer         |
    +----------------+
           |
           v
    2. Fetch customer data from your API
    +----------------+     +---------------------------+
    | fetch          |---->| Your API:                 |
    | (sync: true)   |     | GET /customers?phone=...  |
    +----------------+     +---------------------------+
           |                          |
           |     Response: {"name": "John", "tier": "premium"}
           |                          |
           v                          v
    3. Variables are set from response
    +------------------------------------------+
    | customer.name = "John"                   |
    | customer.tier = "premium"                |
    +------------------------------------------+
           |
           v
    4. Personalize the flow
    +------------------------------------------+
    | talk: "Hello ${customer.name}..."        |
    | branch on ${customer.tier}               |
    +------------------------------------------+


.. code::

    Implementation with webhook_send:

    {
      "name": "Personalized Greeting",
      "actions": [
        {
          "type": "answer"
        },
        {
          "id": "fetch-customer",
          "type": "webhook_send",
          "option": {
            "sync": true,
            "uri": "https://your-api.com/customer-lookup",
            "method": "POST",
            "data_type": "application/json",
            "data": "{\"phone\": \"${voipbin.call.source.target}\"}"
          }
        },
        {
          "type": "talk",
          "option": {
            "text": "Hello ${customer.name}. Welcome back to our service.",
            "language": "en-US"
          }
        },
        {
          "id": "tier-check",
          "type": "branch",
          "option": {
            "variable": "customer.tier",
            "default_target_id": "standard-service",
            "target_ids": {
              "premium": "premium-service",
              "vip": "vip-service"
            }
          }
        },
        {
          "id": "premium-service",
          "type": "talk",
          "option": {
            "text": "As a premium member, you're being connected to our priority support line.",
            "language": "en-US"
          }
        },
        {
          "type": "queue_join",
          "option": {
            "queue_id": "premium-queue-id"
          }
        },
        {
          "type": "hangup"
        },
        {
          "id": "vip-service",
          "type": "connect",
          "option": {
            "source": {
              "type": "tel",
              "target": "${voipbin.call.destination.target}"
            },
            "destinations": [
              {
                "type": "tel",
                "target": "+15551234567"
              }
            ]
          }
        },
        {
          "type": "hangup"
        },
        {
          "id": "standard-service",
          "type": "talk",
          "option": {
            "text": "Please hold while we connect you.",
            "language": "en-US"
          }
        },
        {
          "type": "queue_join",
          "option": {
            "queue_id": "standard-queue-id"
          }
        },
        {
          "type": "hangup"
        }
      ]
    }


AI-Powered Conversational Flow
------------------------------

Integrate AI for natural language interactions:

.. code::

    AI Voice Assistant Pattern:

    +------------------------------------------------------------------+
    |                     AI Talk Integration                          |
    +------------------------------------------------------------------+

    Traditional IVR:
    +-------+     +----------+     +--------+     +---------+
    | talk  |---->| digits   |---->| branch |---->| action  |
    | menu  |     | receive  |     | 1,2,3  |     |         |
    +-------+     +----------+     +--------+     +---------+

    AI-Powered:
    +-------+     +----------+     +----------+     +---------+
    | ai    |---->| AI       |---->| Intent   |---->| action  |
    | talk  |     | processes|     | detected |     |         |
    +-------+     | speech   |     | routing  |     +---------+
                  +----------+     +----------+

    AI Talk handles:
    - Speech recognition (STT)
    - Natural language understanding
    - Response generation
    - Text-to-speech (TTS)
    - Context maintenance


.. code::

    AI Talk Flow Example:

    {
      "name": "AI Customer Service",
      "actions": [
        {
          "type": "answer"
        },
        {
          "id": "ai-greeting",
          "type": "ai_talk",
          "option": {
            "ai_id": "your-ai-agent-id",
            "language": "en-US",
            "gender": "female",
            "duration": 300
          }
        },
        {
          "id": "post-ai-branch",
          "type": "branch",
          "option": {
            "variable": "voipbin.ai.intent",
            "default_target_id": "transfer-support",
            "target_ids": {
              "billing": "transfer-billing",
              "technical": "transfer-technical",
              "cancel": "transfer-retention",
              "resolved": "goodbye"
            }
          }
        },
        {
          "id": "transfer-billing",
          "type": "talk",
          "option": {
            "text": "I'll transfer you to our billing department.",
            "language": "en-US"
          }
        },
        {
          "type": "queue_join",
          "option": {
            "queue_id": "billing-queue-id"
          }
        },
        {
          "type": "hangup"
        },
        {
          "id": "transfer-technical",
          "type": "queue_join",
          "option": {
            "queue_id": "tech-support-queue-id"
          }
        },
        {
          "type": "hangup"
        },
        {
          "id": "transfer-retention",
          "type": "queue_join",
          "option": {
            "queue_id": "retention-queue-id"
          }
        },
        {
          "type": "hangup"
        },
        {
          "id": "transfer-support",
          "type": "queue_join",
          "option": {
            "queue_id": "general-support-queue-id"
          }
        },
        {
          "type": "hangup"
        },
        {
          "id": "goodbye",
          "type": "talk",
          "option": {
            "text": "Thank you for calling. Have a great day!",
            "language": "en-US"
          }
        },
        {
          "type": "hangup"
        }
      ]
    }


Call Recording with Consent
---------------------------

Legal compliance pattern for recording calls:

.. code::

    Recording Consent Flow:

    +------------------------------------------------------------------+
    |                     Recording with Consent                       |
    +------------------------------------------------------------------+

                    Incoming Call
                          |
                          v
                    +-----------+
                    |  Answer   |
                    +-----------+
                          |
                          v
    +---------------------------------------------------+
    | "This call may be recorded for quality assurance. |
    |  Press 1 to continue, or 2 to opt out."           |
    +---------------------------------------------------+
                          |
                          v
                    +------------+
                    |   Branch   |
                    +-----+------+
                          |
            +-------------+-------------+
            |                           |
         Press 1                     Press 2
         (consent)                   (opt-out)
            |                           |
            v                           v
    +---------------+           +---------------+
    | recording_    |           | Continue      |
    | start         |           | without       |
    +---------------+           | recording     |
            |                   +---------------+
            v                           |
    +---------------+                   |
    | Continue      |                   |
    | call flow     |<------------------+
    +---------------+


.. code::

    {
      "name": "Recording Consent Flow",
      "actions": [
        {
          "type": "answer"
        },
        {
          "id": "consent-prompt",
          "type": "talk",
          "option": {
            "text": "This call may be recorded for quality assurance and training purposes. Press 1 to continue with recording, or press 2 to continue without recording.",
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
            "variable": "voipbin.call.digits",
            "default_target_id": "consent-prompt",
            "target_ids": {
              "1": "start-recording",
              "2": "no-recording"
            }
          }
        },
        {
          "id": "start-recording",
          "type": "recording_start",
          "option": {
            "format": "mp3"
          }
        },
        {
          "type": "variable_set",
          "option": {
            "key": "recording.consent",
            "value": "yes"
          }
        },
        {
          "id": "continue-call",
          "type": "talk",
          "option": {
            "text": "Thank you. How may I help you today?",
            "language": "en-US"
          }
        },
        {
          "type": "queue_join",
          "option": {
            "queue_id": "support-queue-id"
          }
        },
        {
          "type": "hangup"
        },
        {
          "id": "no-recording",
          "type": "variable_set",
          "option": {
            "key": "recording.consent",
            "value": "no"
          }
        },
        {
          "type": "goto",
          "option": {
            "target_id": "continue-call"
          }
        }
      ]
    }


Callback Request Pattern
------------------------

Allow customers to request a callback instead of waiting:

.. code::

    Callback Flow Pattern:

    +------------------------------------------------------------------+
    |                     Callback Request                             |
    +------------------------------------------------------------------+

                    Incoming Call
                          |
                          v
    +---------------------------------------------------+
    | "All agents are busy. Press 1 to wait on hold,    |
    |  or press 2 to receive a callback."               |
    +---------------------------------------------------+
                          |
            +-------------+-------------+
            |                           |
         Press 1                     Press 2
         (wait)                      (callback)
            |                           |
            v                           v
    +--------------+            +---------------+
    | queue_join   |            | Confirm phone |
    +--------------+            +---------------+
                                        |
                                        v
                                +---------------+
                                | webhook_send  |
                                | (create task) |
                                +---------------+
                                        |
                                        v
                                +---------------+
                                | "We will call |
                                |  you back"    |
                                +---------------+
                                        |
                                        v
                                    hangup


.. code::

    {
      "name": "Callback Option Flow",
      "actions": [
        {
          "type": "answer"
        },
        {
          "id": "callback-offer",
          "type": "talk",
          "option": {
            "text": "All of our agents are currently busy. Your estimated wait time is 10 minutes. Press 1 to wait on hold, or press 2 to receive a callback when an agent is available.",
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
            "variable": "voipbin.call.digits",
            "default_target_id": "callback-offer",
            "target_ids": {
              "1": "wait-queue",
              "2": "request-callback"
            }
          }
        },
        {
          "id": "wait-queue",
          "type": "talk",
          "option": {
            "text": "Please hold. An agent will be with you shortly.",
            "language": "en-US"
          }
        },
        {
          "type": "queue_join",
          "option": {
            "queue_id": "support-queue-id"
          }
        },
        {
          "type": "hangup"
        },
        {
          "id": "request-callback",
          "type": "talk",
          "option": {
            "text": "We will call you back at the number you called from. If you'd like to be called at a different number, please enter it now followed by the pound key. Otherwise, press pound to confirm.",
            "language": "en-US"
          }
        },
        {
          "type": "digits_receive",
          "option": {
            "duration": 15000,
            "length": 15,
            "key": "#"
          }
        },
        {
          "id": "check-digits",
          "type": "condition_variable",
          "option": {
            "condition": "==",
            "variable": "voipbin.call.digits",
            "value_type": "string",
            "value_string": "#",
            "false_target_id": "custom-number"
          }
        },
        {
          "type": "variable_set",
          "option": {
            "key": "callback.number",
            "value": "${voipbin.call.source.target}"
          }
        },
        {
          "id": "schedule-callback",
          "type": "webhook_send",
          "option": {
            "sync": true,
            "uri": "https://your-api.com/schedule-callback",
            "method": "POST",
            "data_type": "application/json",
            "data": "{\"phone\": \"${callback.number}\", \"call_id\": \"${voipbin.call.id}\"}"
          }
        },
        {
          "type": "talk",
          "option": {
            "text": "Thank you. You will receive a callback at ${callback.number} within the next 30 minutes. Goodbye.",
            "language": "en-US"
          }
        },
        {
          "type": "hangup"
        },
        {
          "id": "custom-number",
          "type": "variable_set",
          "option": {
            "key": "callback.number",
            "value": "${voipbin.call.digits}"
          }
        },
        {
          "type": "goto",
          "option": {
            "target_id": "schedule-callback"
          }
        }
      ]
    }


Survey After Call
-----------------

Using on_complete_flow_id for post-call surveys:

.. code::

    Post-Call Survey Pattern:

    +------------------------------------------------------------------+
    |                     Main Call Flow                               |
    +------------------------------------------------------------------+
    | answer -> talk -> queue_join -> hangup                           |
    |                                                                  |
    | on_complete_flow_id: "survey-flow-id"                           |
    +------------------------------------------------------------------+
                          |
                          | (call ends)
                          v
    +------------------------------------------------------------------+
    |                     Survey Flow                                  |
    +------------------------------------------------------------------+
    | (New session, inherits variables)                                |
    |                                                                  |
    | Create outbound call to customer                                 |
    | -> Play survey questions                                         |
    | -> Collect responses                                             |
    | -> Send results to webhook                                       |
    +------------------------------------------------------------------+


.. code::

    Main Call Flow (with on_complete_flow_id):

    {
      "name": "Support Call",
      "on_complete_flow_id": "post-call-survey-flow-id",
      "actions": [
        {
          "type": "answer"
        },
        {
          "type": "recording_start",
          "option": {
            "format": "mp3"
          }
        },
        {
          "type": "queue_join",
          "option": {
            "queue_id": "support-queue-id"
          }
        },
        {
          "type": "hangup"
        }
      ]
    }

    Survey Flow (executed after call ends):

    {
      "name": "Post-Call Survey",
      "actions": [
        {
          "type": "call",
          "option": {
            "source": {
              "type": "tel",
              "target": "+15551234567"
            },
            "destinations": [
              {
                "type": "tel",
                "target": "${voipbin.call.source.target}"
              }
            ],
            "actions": [
              {
                "type": "talk",
                "option": {
                  "text": "Thank you for contacting us. Please help us improve by answering a brief survey. On a scale of 1 to 5, how satisfied were you with your support experience? Press 1 for very unsatisfied, 5 for very satisfied.",
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
                "type": "variable_set",
                "option": {
                  "key": "survey.satisfaction",
                  "value": "${voipbin.call.digits}"
                }
              },
              {
                "type": "talk",
                "option": {
                  "text": "Was your issue resolved? Press 1 for yes, 2 for no.",
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
                "type": "variable_set",
                "option": {
                  "key": "survey.resolved",
                  "value": "${voipbin.call.digits}"
                }
              },
              {
                "type": "webhook_send",
                "option": {
                  "uri": "https://your-api.com/survey-results",
                  "method": "POST",
                  "data_type": "application/json",
                  "data": "{\"call_id\": \"${voipbin.call.id}\", \"satisfaction\": \"${survey.satisfaction}\", \"resolved\": \"${survey.resolved}\"}"
                }
              },
              {
                "type": "talk",
                "option": {
                  "text": "Thank you for your feedback. Goodbye.",
                  "language": "en-US"
                }
              },
              {
                "type": "hangup"
              }
            ]
          }
        }
      ]
    }


Failover and Redundancy
-----------------------

Handle failures gracefully with fallback options:

.. code::

    Failover Pattern:

    +------------------------------------------------------------------+
    |                     Multi-Destination Failover                   |
    +------------------------------------------------------------------+

    Primary:     connect to Agent 1
                       |
                 +-----------+
                 | answered? |
                 +-----+-----+
                       |
            +----------+----------+
            |                     |
           Yes                   No
            |                     |
            v                     v
    +-------------+       Secondary: connect to Agent 2
    | Continue    |               |
    +-------------+         +-----------+
                            | answered? |
                            +-----+-----+
                                  |
                       +----------+----------+
                       |                     |
                      Yes                   No
                       |                     |
                       v                     v
               +-------------+       Tertiary: connect to Queue
               | Continue    |               |
               +-------------+               v
                                     +-------------+
                                     | queue_join  |
                                     +-------------+


.. code::

    {
      "name": "Failover Flow",
      "actions": [
        {
          "type": "answer"
        },
        {
          "type": "talk",
          "option": {
            "text": "Please hold while we connect you.",
            "language": "en-US"
          }
        },
        {
          "id": "try-primary",
          "type": "connect",
          "option": {
            "source": {
              "type": "tel",
              "target": "+15551234567"
            },
            "destinations": [
              {
                "type": "tel",
                "target": "+15559876543"
              }
            ]
          }
        },
        {
          "type": "condition_variable",
          "option": {
            "condition": "==",
            "variable": "voipbin.call.status",
            "value_type": "string",
            "value_string": "progressing",
            "false_target_id": "try-secondary"
          }
        },
        {
          "type": "hangup"
        },
        {
          "id": "try-secondary",
          "type": "talk",
          "option": {
            "text": "Agent unavailable. Trying alternate contact.",
            "language": "en-US"
          }
        },
        {
          "type": "connect",
          "option": {
            "source": {
              "type": "tel",
              "target": "+15551234567"
            },
            "destinations": [
              {
                "type": "tel",
                "target": "+15551112222"
              }
            ]
          }
        },
        {
          "type": "condition_variable",
          "option": {
            "condition": "==",
            "variable": "voipbin.call.status",
            "value_type": "string",
            "value_string": "progressing",
            "false_target_id": "try-queue"
          }
        },
        {
          "type": "hangup"
        },
        {
          "id": "try-queue",
          "type": "talk",
          "option": {
            "text": "All direct contacts unavailable. Placing you in the support queue.",
            "language": "en-US"
          }
        },
        {
          "type": "queue_join",
          "option": {
            "queue_id": "support-queue-id"
          }
        },
        {
          "type": "hangup"
        }
      ]
    }


Language Selection
------------------

Multi-language IVR with persistent language preference:

.. code::

    Language Selection Pattern:

    +------------------------------------------------------------------+
    |                     Language Router                              |
    +------------------------------------------------------------------+

                    Incoming Call
                          |
                          v
    +---------------------------------------------------+
    | "For English, press 1. Para Espanol, marque 2.    |
    |  Pour Francais, appuyez 3."                       |
    +---------------------------------------------------+
                          |
            +-------------+-------------+
            |             |             |
         Press 1       Press 2       Press 3
         English       Spanish       French
            |             |             |
            v             v             v
    +------------+ +------------+ +------------+
    | Set lang:  | | Set lang:  | | Set lang:  |
    | en-US      | | es-ES      | | fr-FR      |
    +------------+ +------------+ +------------+
            |             |             |
            +-------------+-------------+
                          |
                          v
                  Main Flow (uses
                  ${language} variable)


.. code::

    {
      "name": "Multi-Language IVR",
      "actions": [
        {
          "type": "answer"
        },
        {
          "id": "lang-prompt",
          "type": "talk",
          "option": {
            "text": "<speak><lang xml:lang='en-US'>For English, press 1.</lang> <lang xml:lang='es-ES'>Para Espanol, marque 2.</lang> <lang xml:lang='fr-FR'>Pour Francais, appuyez 3.</lang></speak>",
            "language": "en-US"
          }
        },
        {
          "type": "digits_receive",
          "option": {
            "duration": 5000,
            "length": 1
          }
        },
        {
          "type": "branch",
          "option": {
            "variable": "voipbin.call.digits",
            "default_target_id": "set-english",
            "target_ids": {
              "1": "set-english",
              "2": "set-spanish",
              "3": "set-french"
            }
          }
        },
        {
          "id": "set-english",
          "type": "variable_set",
          "option": {
            "key": "language",
            "value": "en-US"
          }
        },
        {
          "type": "goto",
          "option": {
            "target_id": "main-menu"
          }
        },
        {
          "id": "set-spanish",
          "type": "variable_set",
          "option": {
            "key": "language",
            "value": "es-ES"
          }
        },
        {
          "type": "goto",
          "option": {
            "target_id": "main-menu"
          }
        },
        {
          "id": "set-french",
          "type": "variable_set",
          "option": {
            "key": "language",
            "value": "fr-FR"
          }
        },
        {
          "type": "goto",
          "option": {
            "target_id": "main-menu"
          }
        },
        {
          "id": "main-menu",
          "type": "fetch_flow",
          "option": {
            "flow_id": "main-menu-flow-id"
          }
        }
      ]
    }


    Main Menu Flow (uses language variable):

    {
      "name": "Main Menu",
      "actions": [
        {
          "type": "talk",
          "option": {
            "text": "Welcome to our service. Press 1 for sales, 2 for support.",
            "language": "${language}"
          }
        }
      ]
    }


Parallel Call Attempts (Ring All)
---------------------------------

Ring multiple destinations simultaneously:

.. code::

    Ring All Pattern:

    +------------------------------------------------------------------+
    |                     Parallel Ring                                |
    +------------------------------------------------------------------+

                    Incoming Call
                          |
                          v
                    +-----------+
                    |  Connect  |
                    | (multiple |
                    |  dests)   |
                    +-----------+
                          |
            +-------------+-------------+
            |             |             |
         Agent 1       Agent 2       Agent 3
         ringing       ringing       ringing
            |             |             |
            +------+------+             |
                   |                    |
                   | First to answer    |
                   | wins               |
                   v                    v
            +-----------+         +-----------+
            | Connected |         | Cancelled |
            +-----------+         +-----------+


.. code::

    {
      "name": "Ring All Flow",
      "actions": [
        {
          "type": "answer"
        },
        {
          "type": "talk",
          "option": {
            "text": "Please hold while we connect you to the next available agent.",
            "language": "en-US"
          }
        },
        {
          "type": "connect",
          "option": {
            "source": {
              "type": "tel",
              "target": "+15551234567"
            },
            "destinations": [
              {
                "type": "tel",
                "target": "+15559876543"
              },
              {
                "type": "tel",
                "target": "+15551112222"
              },
              {
                "type": "tel",
                "target": "+15553334444"
              }
            ],
            "early_media": true
          }
        },
        {
          "type": "hangup"
        }
      ]
    }


Real-Time Transcription with Keyword Detection
-----------------------------------------------

Monitor calls for specific keywords:

.. code::

    Transcription Pattern:

    +------------------------------------------------------------------+
    |                     Keyword Monitoring                           |
    +------------------------------------------------------------------+

                    Incoming Call
                          |
                          v
                    +-----------+
                    | Start     |
                    | transcribe|
                    +-----------+
                          |
                          v
                    +-----------+
                    | Connect   |
                    | to agent  |
                    +-----------+
                          |
                          |  Transcripts sent to webhook
                          |  in real-time
                          v
    +---------------------------------------------------+
    | Your API monitors for keywords:                   |
    | - "cancel subscription" -> Alert retention team   |
    | - "legal action" -> Alert legal team              |
    | - "supervisor" -> Trigger escalation              |
    +---------------------------------------------------+


.. code::

    {
      "name": "Transcribed Call Flow",
      "actions": [
        {
          "type": "answer"
        },
        {
          "type": "transcribe_start",
          "option": {
            "language": "en-US"
          }
        },
        {
          "type": "talk",
          "option": {
            "text": "This call is being transcribed for quality assurance. Connecting you now.",
            "language": "en-US"
          }
        },
        {
          "type": "queue_join",
          "option": {
            "queue_id": "support-queue-id"
          }
        },
        {
          "type": "transcribe_stop"
        },
        {
          "type": "hangup"
        }
      ]
    }

