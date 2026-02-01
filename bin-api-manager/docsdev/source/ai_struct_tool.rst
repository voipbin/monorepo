.. _ai-struct-tool:

Tool Functions
==============

Tool functions enable the AI to perform actions during voice conversations. When the AI determines that an action is needed based on the conversation context, it can invoke the appropriate tool function.

Overview
--------

::

    Caller                      AI Engine                    VoIPBIN Platform
      |                            |                              |
      |  "Transfer me to sales"    |                              |
      +--------------------------->|                              |
      |                            |                              |
      |                            |  Detects intent              |
      |                            |  Invokes connect_call        |
      |                            +----------------------------->|
      |                            |                              |
      |                            |                   Tool result|
      |                            |<-----------------------------+
      |                            |                              |
      |   "Connecting you now..."  |                              |
      |<---------------------------+                              |
      |                            |                              |
      +-------------- Call transferred to sales ----------------->|


Available Tools
---------------

========================= =================================================
Tool Name                 Description
========================= =================================================
connect_call              Transfer or connect to another endpoint
send_email                Send an email message
send_message              Send an SMS text message
stop_media                Stop currently playing media
stop_service              End the AI conversation (soft stop)
stop_flow                 Terminate the entire flow (hard stop)
set_variables             Save data to flow context
get_variables             Retrieve data from flow context
get_aicall_messages       Get message history from an AI call
========================= =================================================

.. _ai-struct-tool-connect_call:

connect_call
------------

Connects or transfers the caller to another endpoint (person, department, or phone number).

**When to use:**

* Caller requests a transfer: "transfer me to...", "connect me to..."
* Caller wants to speak to a person: "let me talk to a human", "I need an agent"
* Caller requests a specific department: "sales", "support", "billing"
* Caller provides a phone number: "call +1234567890"

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to speak after connecting. Set false for silent transfer.",
                "default": false
            },
            "source": {
                "type": "object",
                "properties": {
                    "type": { "type": "string", "description": "agent, conference, extension, sip, or tel" },
                    "target": { "type": "string", "description": "Source address/identifier" },
                    "target_name": { "type": "string", "description": "Display name (optional)" }
                },
                "required": ["type", "target"]
            },
            "destinations": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "type": { "type": "string", "description": "agent, conference, extension, line, sip, tel" },
                        "target": { "type": "string", "description": "Destination address" },
                        "target_name": { "type": "string", "description": "Display name (optional)" }
                    },
                    "required": ["type", "target"]
                }
            }
        },
        "required": ["destinations"]
    }

**Examples:**

::

    "Transfer me to sales"      -> type="extension", target="sales"
    "Call my wife at 555-1234"  -> type="tel", target="+15551234"
    "I need a human agent"      -> type="agent", target=appropriate agent

.. _ai-struct-tool-send_email:

send_email
----------

Sends an email to one or more email addresses.

**When to use:**

* Caller explicitly requests email: "email me", "send me an email"
* Caller asks for documents to be emailed
* Caller provides an email address for receiving information

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to confirm verbally after sending.",
                "default": false
            },
            "destinations": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "type": { "type": "string", "enum": ["email"] },
                        "target": { "type": "string", "description": "Email address" },
                        "target_name": { "type": "string", "description": "Recipient name (optional)" }
                    },
                    "required": ["type", "target"]
                }
            },
            "subject": { "type": "string", "description": "Email subject line" },
            "content": { "type": "string", "description": "Email body content" },
            "attachments": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "reference_type": { "type": "string", "enum": ["recording"] },
                        "reference_id": { "type": "string", "description": "UUID of the attachment" }
                    }
                }
            }
        },
        "required": ["destinations", "subject", "content"]
    }

.. _ai-struct-tool-send_message:

send_message
------------

Sends an SMS text message to a phone number.

**When to use:**

* Caller explicitly requests a text: "text me", "send me a text", "SMS me"
* Caller asks for information sent to their phone
* Caller provides a phone number for messaging

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to confirm verbally after sending.",
                "default": false
            },
            "source": {
                "type": "object",
                "properties": {
                    "type": { "type": "string", "enum": ["tel"] },
                    "target": { "type": "string", "description": "Source phone number (+E.164)" },
                    "target_name": { "type": "string", "description": "Display name (optional)" }
                },
                "required": ["type", "target"]
            },
            "destinations": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "type": { "type": "string", "enum": ["tel"] },
                        "target": { "type": "string", "description": "Destination phone number (+E.164)" },
                        "target_name": { "type": "string", "description": "Recipient name (optional)" }
                    },
                    "required": ["type", "target"]
                }
            },
            "text": { "type": "string", "description": "SMS message content" }
        },
        "required": ["destinations", "text"]
    }

.. _ai-struct-tool-stop_media:

stop_media
----------

Stops media from a previous action that is currently playing on the call.

**When to use:**

* AI has finished loading and needs to stop hold music or greeting
* Previous flow action's media playback should stop before AI speaks
* Transitioning from pre-recorded media to live AI conversation

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to speak after stopping media.",
                "default": false
            }
        }
    }

**Comparison with other stop tools:**

::

    +-------------+------------------------------------------+
    | Tool        | Effect                                   |
    +=============+==========================================+
    | stop_media  | Stop previous action's media playback    |
    |             | AI conversation continues                |
    +-------------+------------------------------------------+
    | stop_service| End AI conversation                      |
    |             | Flow continues to next action            |
    +-------------+------------------------------------------+
    | stop_flow   | Terminate everything                     |
    |             | Call ends, no further actions            |
    +-------------+------------------------------------------+

.. _ai-struct-tool-stop_service:

stop_service
------------

Ends the AI conversation and proceeds to the next action in the flow.

**When to use:**

* Caller says goodbye: "bye", "goodbye", "thanks, that's all"
* Caller indicates they're done: "I'm all set", "that's everything"
* AI has successfully completed its purpose (appointment booked, issue resolved)
* Natural conversation conclusion

**When NOT to use:**

* Caller is frustrated but still needs help (de-escalate instead)
* Conversation has unresolved issues
* Caller wants to end the entire call (use stop_flow instead)

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {}
    }

**Examples:**

::

    "Thanks, bye!"                 -> stop_service (natural end)
    "I'm done here"                -> stop_service (completion signal)
    After booking appointment      -> stop_service (task complete)
    "Great, that's all I needed"   -> stop_service

.. _ai-struct-tool-stop_flow:

stop_flow
---------

Immediately terminates the entire flow and call. Nothing executes after this.

**When to use:**

* Caller explicitly wants to end everything: "hang up", "end the call", "disconnect"
* Critical error requiring full termination
* Emergency stop needed

**When NOT to use:**

* Caller just wants to end AI conversation (use stop_service instead)
* Caller says casual goodbye (use stop_service instead)
* There are more flow actions that should execute after AI

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {}
    }

**Examples:**

::

    "Hang up now"                  -> stop_flow
    "End this call immediately"    -> stop_flow
    "Terminate the call"           -> stop_flow

.. _ai-struct-tool-set_variables:

set_variables
-------------

Saves key-value data to the flow context for later use by downstream actions.

**When to use:**

* Save information collected during conversation (name, account number, preferences)
* Record conclusions (appointment time, issue category, resolution)
* Store data needed by subsequent flow actions

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to continue conversation after saving.",
                "default": false
            },
            "variables": {
                "type": "object",
                "description": "Key-value pairs to save",
                "additionalProperties": { "type": "string" }
            }
        },
        "required": ["variables"]
    }

**Examples:**

::

    "My name is John Smith"        -> set_variables({"customer_name": "John Smith"})
    "3pm works for me"             -> set_variables({"appointment_time": "15:00"})
    Issue categorized as billing   -> set_variables({"issue_category": "billing"})
    Account number provided        -> set_variables({"account_number": "12345"})

.. _ai-struct-tool-get_variables:

get_variables
-------------

Retrieves previously saved variables from the flow context.

**When to use:**

* Need context set earlier in the flow
* Need information from previous actions (confirmation number, customer info)
* Caller asks about something in saved context
* Before performing an action requiring previously collected data

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to respond using retrieved data.",
                "default": false
            }
        }
    }

**Examples:**

::

    Need customer name from earlier -> get_variables
    "What was my confirmation?"     -> get_variables
    Before sending SMS              -> get_variables (to get phone number)

.. _ai-struct-tool-get_aicall_messages:

get_aicall_messages
-------------------

Retrieves message history from a specific AI call session.

**When to use:**

* Need message history from a different AI call (not current conversation)
* Building summaries of past conversations
* Caller asks about previous interactions: "what did we discuss last time?"
* Referencing a specific past call by ID

**When NOT to use:**

* Current conversation history is sufficient (already in AI context)
* Need saved variables, not messages (use get_variables instead)
* No specific aicall_id to query

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to respond based on retrieved messages.",
                "default": false
            },
            "aicall_id": {
                "type": "string",
                "description": "UUID of the AI call to retrieve messages from"
            }
        },
        "required": ["aicall_id"]
    }


run_llm Parameter
-----------------

Most tools include a ``run_llm`` parameter that controls whether the AI should generate a response after the tool executes.

::

    +-------------------+--------------------------------------------------+
    | run_llm = true    | AI speaks after tool execution                   |
    |                   | Example: "I've sent that to your email"          |
    +-------------------+--------------------------------------------------+
    | run_llm = false   | Tool executes silently                           |
    |                   | Useful for chaining multiple tools               |
    +-------------------+--------------------------------------------------+

**Default:** Most tools default to ``run_llm = false`` for silent execution.


Tool Execution Flow
-------------------

::

    +-----------------------------------------------------------------+
    |                    Tool Execution Architecture                   |
    +-----------------------------------------------------------------+

    Caller speaks          Python Pipecat              Go AIcallHandler
         |                      |                            |
         |  "Transfer me to     |                            |
         |   sales please"      |                            |
         +--------------------->|                            |
         |                      |                            |
         |               STT converts                        |
         |               to text                             |
         |                      |                            |
         |               LLM detects intent                  |
         |               function_call: connect_call         |
         |                      |                            |
         |                      |  HTTP POST                 |
         |                      |  /tool/execute             |
         |                      +--------------------------->|
         |                      |                            |
         |                      |             Execute tool   |
         |                      |             (call-manager) |
         |                      |                            |
         |                      |  Tool result               |
         |                      |<---------------------------+
         |                      |                            |
         |               LLM generates                       |
         |               response                            |
         |                      |                            |
         |  TTS: "Connecting    |                            |
         |   you to sales now"  |                            |
         |<---------------------+                            |
         |                      |                            |


Best Practices
--------------

**1. Enable only needed tools**

.. code::

    // Good: Only enable tools the AI actually needs
    "tool_names": ["connect_call", "stop_service"]

    // Avoid: Enabling all tools when only some are needed
    "tool_names": ["all"]

**2. Use stop_service vs stop_flow correctly**

::

    stop_service = Soft stop (AI ends, flow continues)
        - User says "goodbye"
        - Task completed successfully

    stop_flow = Hard stop (everything ends)
        - User says "hang up"
        - Critical error

**3. Clarify ambiguous requests**

When a user says "send me that information," the AI should ask:

::

    "Would you like that by email or text message?"

This ensures the correct tool (send_email vs send_message) is used.

**4. Use run_llm appropriately**

::

    // Silent operations (chaining tools)
    "run_llm": false

    // User-facing confirmations
    "run_llm": true  -> "I've connected you to the sales department"
