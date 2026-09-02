.. _ai-struct-tool:

Tool Functions
==============

Tool functions enable the AI to perform actions during voice conversations. When the AI determines that an action is needed based on the conversation context, it can invoke the appropriate tool function.

.. note:: **AI Implementation Hint**

   Tool functions are invoked by the LLM, not by your application code. You configure which tools are available via the ``tool_names`` field in the AI configuration (``POST /ais`` or inline flow action). The LLM decides when to call a tool based on conversation context and the tool's parameter schema. Your prompt should describe when each tool should be used.

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

.. note:: **AI Implementation Hint**

   The ``tool_names`` field accepts an enum of valid tool names defined in the OpenAPI spec (``AIManagerToolName``).
   Use ``["all"]`` to enable every tool, or specify individual names: ``["connect_call", "search_knowledge", "stop_service"]``.

========================= ================================================= ===============
Tool Name                 Description                                       run_llm Default
========================= ================================================= ===============
connect_call              Transfer or connect to another endpoint            ``false``
create_call               Place a new, independent outbound call             ``true``
send_email                Send an email message                              ``false``
send_message              Send an SMS text message                           ``false``
stop_media                Stop currently playing media                       ``false``
stop_service              End the AI conversation (soft stop)                ``false``
stop_flow                 Terminate the entire flow (hard stop)              ``false``
set_variables             Save data to flow context                          ``false``
get_variables             Retrieve data from flow context                    ``false``
get_aicall_messages       Get message history from an AI call                ``false``
search_knowledge          Search the configured knowledge base (RAG)         ``true``
get_correlation           List resources linked to an activeflow             ``true``
get_resource              Fetch the content of a related resource            ``true``
describe_action           Look up a flow action's option schema              ``true``
case_create               Create a CRM case for the current contact          ``true``
========================= ================================================= ===============

.. note:: **AI Implementation Hint**

   ``get_correlation``, ``get_resource``, and ``describe_action`` are diagnostic/orchestration tools intended for troubleshooting and assembling ``create_call`` action lists. ``create_call`` and ``case_create`` are only available for ``type=normal`` AIs (via ``tool_names``); they are not part of the Insight tool set. See :ref:`Insight Tools <ai-struct-tool-insight>` for the separate tool set used by ``type=insight`` AIs.

.. _ai-struct-tool-connect_call:

connect_call
------------

Connects or transfers the user to another endpoint (person, department, or phone number).

.. note:: **AI Implementation Hint**

   The ``connect_call`` tool creates a new outgoing call and bridges it with the current call. The ``source.type`` determines the caller ID shown to the destination. For PSTN transfers, use ``type: "tel"`` with an E.164 phone number (e.g., ``+15551234567``). For internal transfers, use ``type: "extension"`` with the extension name.

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

.. note:: **AI Implementation Hint**

   Phone numbers in ``source.target`` and ``destinations[].target`` must be in E.164 format (e.g., ``+15551234567``). If the user provides a local number like ``555-1234``, the LLM must normalize it to E.164 before invoking this tool. The ``source`` phone number must be a number owned by your VoIPBIN account (obtainable via ``GET /numbers``).

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

.. note:: **AI Implementation Hint**

   Variables set via ``set_variables`` are accessible in subsequent flow actions using the ``${variable_name}`` syntax. Variable names are case-sensitive strings. Values are always stored as strings. These variables persist for the duration of the flow execution and are included in webhook events.

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

.. note:: **AI Implementation Hint**

   The ``aicall_id`` (UUID) must reference a completed or active AI call session. You can obtain this ID from the ``${voipbin.aicall.id}`` flow variable during the current call, or from a previous call's webhook event data. This tool is most useful for multi-call workflows where AI needs context from a prior conversation.

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

.. _ai-struct-tool-search_knowledge:

search_knowledge
----------------

Searches the configured knowledge base (RAG) for information relevant to the user's question or topic. Returns matching content from uploaded documents.

.. note:: **AI Implementation Hint**

   The ``search_knowledge`` tool requires a knowledge base (RAG) to be configured on the AI via the ``rag_id`` field. If no ``rag_id`` is set (or it is ``00000000-0000-0000-0000-000000000000``), this tool will not be available to the AI during the call. Create a knowledge base via ``POST /rags`` and assign its ``id`` to the AI's ``rag_id`` field. The tool returns up to 5 matching sources with relevance scores.

**When to use:**

* Caller asks a question that may be answered by company documentation or uploaded files
* Caller needs product specifications, policy details, or reference information
* AI needs to look up facts before answering: "What is our return policy?", "How do I configure X?"
* Caller asks about topics covered by the knowledge base

**When NOT to use:**

* Caller asks general conversational questions unrelated to the knowledge base
* Information is already in the conversation context
* No knowledge base is configured (tool will not be available)

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "query": {
                "type": "string",
                "description": "The search query to find relevant information in the knowledge base"
            },
            "run_llm": {
                "type": "boolean",
                "description": "Set true to respond using retrieved knowledge.",
                "default": true
            }
        },
        "required": ["query"]
    }

**Response format:**

The tool returns matching document sections with metadata:

::

    [Source 1: "Product Guide" > "Return Policy" (relevance: 0.92)]
    Items may be returned within 30 days of purchase...

    [Source 2: "FAQ" > "Refunds" (relevance: 0.85)]
    Refunds are processed within 5-7 business days...

**Examples:**

::

    "What's your return policy?"        -> search_knowledge(query="return policy")
    "How do I set up the widget?"       -> search_knowledge(query="widget setup instructions")
    "What are the pricing tiers?"       -> search_knowledge(query="pricing tiers plans")

.. _ai-struct-tool-create_call:

create_call
-----------

Places a **new, independent** outbound call that is **not** bridged into the current conversation. The new call runs its own flow while the current AI session continues normally (it is not ended).

.. note:: **AI Implementation Hint**

   Provide **either** ``flow_id`` (reuse a flow already built on the account) **or** ``actions`` (assemble the call scenario inline), never both. Use ``actions`` for ad-hoc scenarios not covered by an existing flow (e.g. "call John, say the meeting moved to 3pm, then hang up"). Use :ref:`describe_action <ai-struct-tool-describe_action>` to look up an action type's option fields before assembling an inline action.

**Differs from connect_call:**

::

    +-------------+------------------------------------------------------+
    | create_call | NEW independent call, NOT bridged, current            |
    |             | AI session continues                                 |
    +-------------+------------------------------------------------------+
    | connect_call| Bridges another party INTO the current call,          |
    |             | ends the AI session                                   |
    +-------------+------------------------------------------------------+

**When to use:**

* Caller wants a separate call placed to someone: "call John and remind him about the meeting"
* A callback or notification call should be triggered to a third party
* An outbound call should run a predefined scenario (``flow_id``) or an ad-hoc one assembled inline (``actions``)

**When NOT to use:**

* Caller wants to be transferred/connected to someone in the current call (use ``connect_call``)
* Caller wants to end the current call (use ``stop_flow`` / ``stop_service``)

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true (default) to confirm verbally after placing the call.",
                "default": true
            },
            "flow_id": {
                "type": "string",
                "description": "UUID of a pre-existing flow the new call will execute. Provide EITHER flow_id OR actions, not both. Must belong to the account."
            },
            "actions": {
                "type": "array",
                "description": "Ordered list of inline flow actions, assembled INSTEAD OF flow_id. Each item has a 'type', optional 'id'/'next_id' for branching, and a type-specific 'option' object.",
                "items": {
                    "type": "object",
                    "properties": {
                        "id": { "type": "string", "description": "Optional UUID assigned to this action so other actions can target it via target_id/false_target_id/default_target_id/target_ids." },
                        "next_id": { "type": "string", "description": "Optional UUID of the action to run next instead of the following array item." },
                        "type": { "type": "string", "description": "Flow action type (e.g. talk, play, hangup, connect, variable_set, branch, goto, sleep, digits_receive). Use describe_action to look up option fields first." },
                        "option": { "type": "object", "description": "Action-type-specific options; shape depends on type." }
                    },
                    "required": ["type"]
                }
            },
            "source": {
                "type": "object",
                "description": "Optional source endpoint. If omitted, a default account number is used.",
                "properties": {
                    "type": { "type": "string", "description": "Source endpoint type: tel or sip" },
                    "target": { "type": "string", "description": "Source address (e.g., +E.164 phone number)" },
                    "target_name": { "type": "string", "description": "Display name (optional)" }
                }
            },
            "destinations": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "type": { "type": "string", "description": "Destination type: tel, sip, extension, agent" },
                        "target": { "type": "string", "description": "Destination address" },
                        "target_name": { "type": "string", "description": "Display name (optional)" }
                    },
                    "required": ["type", "target"]
                }
            },
            "anonymous": {
                "type": "string",
                "description": "Optional caller-ID privacy: yes | no | auto (default auto)."
            },
            "variables": {
                "type": "object",
                "description": "Optional flat key-value context seeded into the new call's flow as runtime variables (readable via ${key}). String values only. Keys starting with 'voipbin.' are reserved and ignored. Max 100 keys, 64KB total.",
                "additionalProperties": { "type": "string" }
            }
        },
        "required": ["destinations"]
    }

**Examples:**

::

    "Call John and tell him the meeting moved to 3pm" -> actions=[talk, hangup]
    "Run the appointment-reminder flow on +15551234567" -> flow_id=<uuid>, destinations=[{type: tel, target: "+15551234567"}]

.. _ai-struct-tool-get_correlation:

get_correlation
----------------

Retrieves the correlation graph for a resource: the related resources (calls, messages, recordings, transcribes, aicalls, etc.) linked to the same activeflow execution. This is an internal diagnostic tool.

.. note:: **AI Implementation Hint**

   An activeflow is the running instance of a flow. Its reference is not always a call — the reference type can be ``call``, ``conversation``, ``ai``, ``api``, ``campaign``, ``transcribe``, or ``recording`` (and may be unset). Do not assume the session is a phone call. Use :ref:`get_resource <ai-struct-tool-get_resource>` to fetch the content of a resource discovered via this tool.

**When to use:**

* Need to know what resources are linked to the current session's activeflow
* A diagnostic question requires understanding relationships between resources of an activeflow
* Need to discover a resource id (e.g. an aicall id) to chain into another tool

**When NOT to use:**

* General conversation or knowledge-base questions (use ``search_knowledge``)
* Only a single runtime variable is needed (use ``get_variables``)

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to reason about the correlation results.",
                "default": true
            },
            "resource_id": {
                "type": "string",
                "description": "Optional resource id (UUID) to inspect. If omitted, the current session's activeflow is used. Only resources owned by the caller's own account can be inspected; others return \"No events found for this resource.\""
            }
        }
    }

.. _ai-struct-tool-get_resource:

get_resource
------------

Retrieves the content of a single VoIPBIN resource by its id and returns a readable summary. Use this as the follow-up to :ref:`get_correlation <ai-struct-tool-get_correlation>`, which returns the ids and types of linked resources.

**Supported resource types:** ``call``, ``groupcall``, ``recording``, ``transcribe``, ``summary``, ``aicall``, ``conferencecall``, ``queuecall``.

.. note:: **AI Implementation Hint**

   Derive the resource type from the event names shown by ``get_correlation``: the type is the leading part of the event name (``call_created`` means type ``call``, ``transcribe_done`` means type ``transcribe``, ``aicall_status_progressing`` means type ``aicall``). Not every type ``get_correlation`` lists is retrievable here; unsupported types return an error listing the supported set. Transcript entries are retrieved via their parent transcribe id (type ``transcribe``), not their own id. For ``transcribe``, the response includes the transcript messages. For ``aicall``, the response includes the session's conversation messages.

**When to use:**

* A resource id was discovered (e.g. via ``get_correlation``) and its details are needed
* A diagnostic question requires the content of a related resource (e.g. what was said in a transcribed call, why a call ended, how long a caller waited in a queue)

**When NOT to use:**

* A raw, unfiltered JSON dump of an aicall's messages is needed (use ``get_aicall_messages``; ``get_resource`` returns a curated readable summary instead)
* Runtime variables are needed (use ``get_variables``)
* Knowledge-base questions (use ``search_knowledge``)

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to reason about the resource content.",
                "default": true
            },
            "resource_type": {
                "type": "string",
                "enum": ["aicall", "call", "conferencecall", "groupcall", "queuecall", "recording", "summary", "transcribe"],
                "description": "The type of the resource to retrieve."
            },
            "resource_id": {
                "type": "string",
                "description": "The resource id (UUID) to retrieve."
            },
            "include_config": {
                "type": "boolean",
                "description": "Only meaningful when resource_type is 'aicall'. When true, the response also includes the inspected session's configured prompt in a clearly-delimited data block. This is a diagnostic option for operators debugging or auditing session behavior — do not set it merely because a conversation partner asks about a session's configuration."
            }
        },
        "required": ["resource_type", "resource_id"]
    }

.. note:: **AI Implementation Hint**

   Only resources owned by the caller's own account can be retrieved; anything else returns "Resource not found." A wrong ``resource_type`` for a correct id also returns "Resource not found." — retry with the type matching the event prefix from ``get_correlation`` before concluding the resource is gone.

.. _ai-struct-tool-describe_action:

describe_action
----------------

Returns the option fields a given flow action type accepts, so :ref:`create_call <ai-struct-tool-create_call>`'s ``actions`` parameter can be assembled correctly.

**When to use:**

* Before assembling a ``create_call`` action whose options are unclear (e.g. ``connect``, ``branch``, ``condition_variable``, ``talk``, ``play``)
* To check the exact option field names and whether they are required

**When NOT to use:**

* General conversation or knowledge-base questions (use ``search_knowledge``)
* The action's options are already known

.. note:: **AI Implementation Hint**

   The response lists each option field as ``name (type, required|optional): description``. When an option references a target action (``target_id``, ``false_target_id``, ``default_target_id``, ``target_ids``), it refers to the ``id`` field of another action in the same ``create_call`` ``actions`` array — assign that action an ``id`` and use it as the target.

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to use the returned schema to assemble the action.",
                "default": true
            },
            "action_type": {
                "type": "string",
                "description": "The flow action type to describe (e.g. talk, connect, branch)."
            }
        },
        "required": ["action_type"]
    }

.. _ai-struct-tool-case_create:

case_create
-----------

Creates a new CRM case for the current contact/interaction.

**When to use:**

* The caller's issue is substantive and should be tracked as a case (e.g. a complaint, a multi-step request, something requiring follow-up)
* An agent or the AI itself judges this interaction needs a trackable record beyond the raw interaction log

**When NOT to use:**

* Casual/short interactions with no follow-up need

.. note:: **AI Implementation Hint**

   A case may already be open for this contact/channel — creating another fails silently (the existing open case is not returned; the call simply does not create a duplicate). Do not retry on failure.

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to have the assistant mention the case was created. Set false to create silently.",
                "default": true
            },
            "name": { "type": "string", "description": "Short case title (optional)." },
            "detail": { "type": "string", "description": "Longer free-text description of the issue (optional)." },
            "note": { "type": "string", "description": "An initial internal note for the agent (optional, not shown to the customer)." }
        }
    }

.. _ai-struct-tool-insight:

Insight Tools
-------------

The following tools are exclusive to ``type=insight`` AIs (see :ref:`Type <ai-struct-ai-type>`) and are not selectable by ``type=normal`` AIs. They are always scoped to the Case the Insight AI session was opened for; none of them accept an argument to target a different Case or contact.

========================== ================================================= ===============
Tool Name                  Description                                       run_llm Default
========================== ================================================= ===============
get_contact_interactions   List past interactions with the Case's contact     ``true``
get_conversation_content   Retrieve a conversation's message transcript       ``true``
get_related_cases          List the contact's other cases                     ``true``
get_case_notes             Retrieve internal agent notes on the current Case  ``true``
get_contact_profile        Retrieve the Case contact's profile and addresses  ``true``
========================== ================================================= ===============

.. _ai-struct-tool-get_contact_interactions:

get_contact_interactions
~~~~~~~~~~~~~~~~~~~~~~~~~

Lists past interactions (calls, conversation messages) with the contact/peer of the Case this Insight AI was opened for.

**When to use:**

* Answering "has this customer contacted us before" / "what's the interaction history"
* Discovering candidate conversation message ids to pass into ``get_conversation_content``

**When NOT to use:**

* The actual message text is needed (use ``get_conversation_content`` with a ``reference_id`` from this tool's output)

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to reason about the retrieved interaction history.",
                "default": true
            },
            "limit": {
                "type": "integer",
                "description": "Maximum number of interactions to return (default 20, max 50).",
                "default": 20
            }
        }
    }

.. _ai-struct-tool-get_conversation_content:

get_conversation_content
~~~~~~~~~~~~~~~~~~~~~~~~~

Retrieves the message transcript of a conversation, given the ``reference_id`` of a ``conversation_message``-type interaction returned by :ref:`get_contact_interactions <ai-struct-tool-get_contact_interactions>`.

**When to use:**

* The actual text of what was said is needed, not just that an interaction happened
* A ``reference_id`` from ``get_contact_interactions`` is already available for the conversation to read

**When NOT to use:**

* ``get_contact_interactions`` has not been called yet — call it first to discover candidate ``reference_id`` values

.. note:: **AI Implementation Hint**

   Only conversations owned by the caller's own account can be retrieved; anything else returns "Resource not found."

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to reason about the retrieved conversation content.",
                "default": true
            },
            "reference_id": {
                "type": "string",
                "description": "The reference id of a conversation_message-type interaction, as returned by get_contact_interactions."
            },
            "limit": {
                "type": "integer",
                "description": "Maximum number of messages to return from the resolved conversation (default 20, max 50).",
                "default": 20
            }
        },
        "required": ["reference_id"]
    }

.. _ai-struct-tool-get_related_cases:

get_related_cases
~~~~~~~~~~~~~~~~~~

Lists other cases belonging to the same contact as the current Case (metadata only: id/title/status/date, never the case body or internal notes). The current case itself is never included in the results.

**When to use:**

* Answering "has this contact had other cases before" / "what other issues has this customer raised"

**When NOT to use:**

* Notes on the CURRENT case are needed (use ``get_case_notes`` instead)

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to reason about the retrieved related-case history.",
                "default": true
            }
        }
    }

.. _ai-struct-tool-get_case_notes:

get_case_notes
~~~~~~~~~~~~~~~

Returns the internal agent notes on the current Case, useful for picking up context left by a previous agent.

**When to use:**

* Answering "what did the last agent note about this case" / handing off between agents

**When NOT to use:**

* Notes from a different case are needed (not supported — notes are always scoped to the current Case)

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to reason about the retrieved case notes.",
                "default": true
            },
            "limit": {
                "type": "integer",
                "description": "Maximum number of most-recent notes to return (default 20, max 50).",
                "default": 20
            }
        }
    }

.. _ai-struct-tool-get_contact_profile:

get_contact_profile
~~~~~~~~~~~~~~~~~~~~

Returns the profile of the contact linked to the current Case: name, company, job title, and up to 5 reachable addresses (phone/email), primary address first. If the Case has no linked contact, the tool returns ``no contact profile found``.

Free-text contact notes, integration metadata (``source``/``external_id``), tags, and address ``name``/``detail`` sub-fields are never returned.

**When to use:**

* Answering "who am I talking to" / "what company is this person from" / "what is this contact's phone number or email address"

**When NOT to use:**

* The history of past contacts with this person is needed (use ``get_contact_interactions``)
* The contact's other cases are needed (use ``get_related_cases``)
* The internal agent notes on this Case are needed (use ``get_case_notes``)

**Parameters:**

.. code::

    {
        "type": "object",
        "properties": {
            "run_llm": {
                "type": "boolean",
                "description": "Set true to reason about the retrieved contact profile.",
                "default": true
            }
        }
    }


run_llm Parameter
-----------------

The ``run_llm`` parameter controls whether the LLM generates a spoken response after a tool executes.
Each tool has a platform-set default (shown in the table below). This value is **not client-configurable**
— it is defined by the platform based on each tool's intended behavior.

::

    +-------------------+--------------------------------------------------+
    | run_llm = true    | Tool result is fed back to the LLM.              |
    |                   | LLM generates a response based on the result.    |
    |                   | Example: "Based on our policy, returns are..."   |
    +-------------------+--------------------------------------------------+
    | run_llm = false   | Tool executes silently.                          |
    |                   | LLM does NOT generate a response after execution.|
    |                   | Useful for background actions or chaining tools. |
    +-------------------+--------------------------------------------------+

**Per-tool defaults:**

============================ ========= ========================================================================================
Tool Name                    run_llm   Why
============================ ========= ========================================================================================
``connect_call``             ``false`` Silent transfer — caller hears ringing, not LLM narration.
``create_call``              ``true``  Confirms verbally that the new outbound call was placed.
``send_email``               ``false`` Background action — email sent without verbal confirmation.
``send_message``             ``false`` Background action — SMS sent without verbal confirmation.
``stop_media``               ``false`` Infrastructure action — stops playback silently.
``stop_service``             ``false`` Ends AI session — no response needed after termination.
``stop_flow``                ``false`` Ends entire flow — no response possible after termination.
``set_variables``            ``false`` Internal data storage — no verbal confirmation needed.
``get_variables``            ``false`` Internal data retrieval — data used by next tool, not spoken.
``get_aicall_messages``      ``false`` Internal data retrieval — messages used by next tool, not spoken.
``search_knowledge``         ``true``  **Must speak** — caller asked a question, LLM must answer using the retrieved knowledge.
``get_correlation``          ``true``  Diagnostic tool — LLM reasons about and summarizes the correlated resources.
``get_resource``             ``true``  Diagnostic tool — LLM reasons about the retrieved resource content.
``describe_action``          ``true``  LLM uses the returned schema to assemble a create_call action.
``case_create``              ``true``  LLM confirms the case was created for the caller.
``get_contact_interactions`` ``true``  Insight tool — LLM reasons about the retrieved interaction history.
``get_conversation_content`` ``true``  Insight tool — LLM reasons about the retrieved conversation content.
``get_related_cases``        ``true``  Insight tool — LLM reasons about the retrieved related-case history.
``get_case_notes``           ``true``  Insight tool — LLM reasons about the retrieved case notes.
``get_contact_profile``      ``true``  Insight tool — LLM reasons about the retrieved contact profile.
============================ ========= ========================================================================================

.. note:: **AI Implementation Hint**

   ``search_knowledge`` is the only tool that defaults to ``run_llm = true`` because its entire purpose
   is to retrieve information that the LLM should use to answer the caller. If ``run_llm`` were ``false``,
   the knowledge base results would be silently discarded and the caller would never hear an answer.


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

**4. Understand run_llm behavior**

The ``run_llm`` default for each tool is set by the platform and cannot be changed by clients.
It determines whether the LLM generates a response after tool execution:

::

    run_llm = false  -> Tool executes silently (most tools)
    run_llm = true   -> LLM generates a response based on the result (search_knowledge)
