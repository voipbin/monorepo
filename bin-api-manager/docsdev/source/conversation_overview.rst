.. _conversation-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Medium
   * **Cost:** Free (conversations are organizational containers; message delivery costs apply per channel)
   * **Async:** Yes. Messages sent within a conversation are delivered asynchronously. Use webhooks to receive delivery status and inbound message events.

.. note:: **Recent change (additive, non-breaking)**

   As of this release, ``owner_type`` and ``owner_id`` in conversation webhook payloads will start carrying real values for conversations that have been explicitly assigned to an agent. Existing unassigned conversations continue to read empty values for both fields, so no client-side change is required. See :ref:`Assigning a Conversation to an Agent <conversation-overview-assigning-conversation-to-agent>`.

VoIPBIN's Conversation API provides a unified multi-channel messaging platform that enables seamless communication across SMS, MMS, email, chat, and social networking channels. Users can start a conversation through one channel and continue it through another without losing context.

With the Conversation API you can:

- Create unified conversations across multiple channels
- Switch channels seamlessly within the same conversation
- Track message history across all channels
- Manage participants dynamically
- Receive real-time updates via webhooks


How Conversations Work
----------------------
VoIPBIN Conversations acts as a unified hub that routes messages across different communication channels while maintaining conversation context.

**Conversation Architecture**

::

    +----------+        +----------------+        +---------------+
    |   SMS    |------->|                |------->|    SMS/MMS    |
    +----------+        |                |        +---------------+
                        |                |
    +----------+        |    VoIPBIN     |        +---------------+
    |  Email   |------->|  Conversation  |------->|     Email     |
    +----------+        |      Hub       |        +---------------+
                        |                |
    +----------+        |                |        +---------------+
    |   Chat   |------->|                |------->|   Chat/SNS    |
    +----------+        +-------+--------+        +---------------+
                                |
                         +------+------+
                         |   Webhook   |
                         |  (events)   |
                         +-------------+

**Key Components**

- **Conversation**: A container that groups related messages across channels
- **Participant**: An endpoint (phone number, email, chat ID) in the conversation
- **Message**: Content sent within a conversation via any channel
- **Channel**: The communication method (SMS, MMS, email, chat, SNS)

**Unified Conversation Flow**

::

    User                    VoIPBIN                     Recipient
      |                        |                            |
      | SMS: "Hello"           |                            |
      +----------------------->| Route to conversation      |
      |                        | (auto-detect or create)    |
      |                        +--------------------------->|
      |                        |              SMS delivered |
      |                        |                            |
      |                        |<---------------------------+
      |                        |   Email reply: "Hi there"  |
      |                        |                            |
      |<-----------------------+                            |
      | Webhook: message       |                            |
      | received in same       |                            |
      | conversation           |                            |


Channel Types
-------------
VoIPBIN supports multiple communication channels within a single conversation.

**Supported Channels**

.. list-table::
   :header-rows: 1

   * - Channel
     - Description
   * - message
     - SMS/MMS text messages to mobile phones
   * - line
     - LINE messaging platform


**Channel Selection**

::

                      Which channel?
                            |
              +-------------+-------------+
              |                           |
              v                           v
        +-----------+               +-----------+
        |  message  |               |   line    |
        | (SMS/MMS) |               | messaging |
        +-----+-----+               +-----+-----+
              |                           |
              v                           v
        Short,                      Real-time,
        immediate                   interactive
        (< 160 chars)               chat-based


Conversation Lifecycle
----------------------
A conversation is created automatically when VoIPBIN receives an inbound message (SMS/MMS or LINE) or when a message is sent via the API. Conversations persist as long as they are not deleted.

**Conversation Message States**

Messages within a conversation move through predictable states.

::

    Message sent/received
           |
           v
    +--------------+
    | progressing  |
    +------+-------+
           |
      +----+----+
      |         |
      v         v
    +------+ +--------+
    | done | | failed |
    +------+ +--------+

**Message Status Descriptions**

.. list-table::
   :header-rows: 1

   * - Status
     - What's happening
   * - progressing
     - Message is being processed and delivered
   * - done
     - Message was successfully delivered
   * - failed
     - Message delivery failed



Conversation Rooms
------------------
VoIPBIN automatically organizes messages into distinct conversation rooms based on participants and channels.

**Room Matching Logic**

::

    New message arrives
           |
           v
    +--------------------+
    | Check participants |
    | and channel        |
    +--------+-----------+
             |
             v
    +--------------------+     Yes    +--------------------+
    | Match existing     |----------->| Add message to     |
    | conversation?      |            | existing room      |
    +--------+-----------+            +--------------------+
             |
             | No
             v
    +--------------------+
    | Create new         |
    | conversation room  |
    +--------------------+

**Room Benefits**

- Messages automatically grouped by context
- No manual conversation management needed
- Full history preserved across channel switches
- Participants can be added or removed dynamically


How Conversations Are Created
-----------------------------
Conversations are created automatically by VoIPBIN when inbound messages arrive or when messages are sent through flows. You can list existing conversations with ``GET https://api.voipbin.net/v1.0/conversations``.

.. note:: **AI Implementation Hint**

   There is no ``POST /conversations`` endpoint. Conversations are created automatically based on incoming messages (SMS/MMS or LINE) or outbound message flow actions. To view conversations, use ``GET https://api.voipbin.net/v1.0/conversations``. Each conversation links a ``self`` address (your number) to a ``peer`` address (the external party). The ``type`` field is ``message`` for SMS/MMS or ``line`` for LINE messaging.


Sending Messages
----------------
Send messages to a conversation and VoIPBIN routes to appropriate channels.

**Send Message Flow**

::

    Your App                    VoIPBIN                 Participants
        |                          |                         |
        | POST /conversations/     |                         |
        |   {id}/messages          |                         |
        +------------------------->|                         |
        |                          | Determine best channel  |
        |                          | for each participant    |
        |                          |                         |
        |                          +-- SMS ----------------->|
        |                          +-- Email --------------->|
        |                          +-- Chat ---------------->|
        |  message_id              |                         |
        |<-------------------------+                         |
        |                          |                         |

**Send Message Example**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/conversations/<conversation-id>/messages?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "text": "Your order has been shipped!"
        }'

**Channel Selection Priority**

When sending to a conversation, VoIPBIN selects the best channel based on:

1. Participant's last active channel
2. Message content (media requires MMS/email)
3. Participant preferences
4. Channel availability


Receiving Messages
------------------
VoIPBIN delivers inbound messages to your application via webhooks.

**Webhook Delivery**

::

    Participant             VoIPBIN                      Your App
         |                     |                            |
         | SMS reply           |                            |
         +-------------------->|                            |
         |                     | Match to conversation      |
         |                     |                            |
         |                     | POST /your-webhook         |
         |                     | {conversation_message}     |
         |                     +--------------------------->|
         |                     |                            |
         |                     |            200 OK          |
         |                     |<---------------------------+
         |                     |                            |

**Inbound Message Webhook**

.. code::

    {
        "type": "conversation_message_received",
        "data": {
            "conversation_id": "conv-abc-123",
            "message": {
                "id": "msg-xyz-789",
                "participant": {
                    "type": "tel",
                    "target": "+15559876543"
                },
                "channel": "sms",
                "text": "Thanks for the update!",
                "direction": "inbound",
                "tm_create": "2024-01-15T10:30:00Z"
            }
        }
    }


.. _conversation-overview-assigning-conversation-to-agent:

Assigning a Conversation to an Agent
------------------------------------
A conversation can be explicitly assigned to a specific agent so that inbound messages on that conversation are routed to the agent for manual handling instead of running through the registered flow. Assignment is an additive feature: conversations that have never been assigned continue to behave exactly as before.

**How assignment works**

Assignment is performed via a partial-update on the conversation. This operation requires admin or manager permission:

.. code:: bash

    PUT https://api.voipbin.net/v1.0/conversations/<conversation-id>?token=<admin-or-manager-token>
    Content-Type: application/json

    {
        "owner_id": "<agent-uuid>"
    }

The server derives ``owner_type`` from ``owner_id``: when ``owner_id`` is a real agent UUID, ``owner_type`` is set to ``agent``; when ``owner_id`` is the nil UUID, ``owner_type`` is cleared to an empty string. Clients must not set ``owner_type`` directly — any value supplied for ``owner_type`` in the request body is ignored.

**How unassignment works**

There are two ways to unassign a conversation:

1. **Admin/manager**: Use ``PUT /v1.0/conversations/<id>`` with ``owner_id`` set to the nil UUID.
2. **Owning agent** (or admin/manager): Use the dedicated ``POST /v1.0/conversations/<id>/unassign`` endpoint. No request body is needed.

.. code:: bash

    # Option 1: Admin/manager via PUT (also works for reassignment)
    PUT https://api.voipbin.net/v1.0/conversations/<conversation-id>?token=<admin-or-manager-token>
    Content-Type: application/json

    {
        "owner_id": "00000000-0000-0000-0000-000000000000"
    }

.. code:: bash

    # Option 2: Owning agent (or admin/manager) via dedicated unassign endpoint
    POST https://api.voipbin.net/v1.0/conversations/<conversation-id>/unassign?token=<agent-or-admin-token>

The ``POST /unassign`` endpoint takes no request body and returns the updated conversation object.

**Permission Semantics**

.. list-table::
   :header-rows: 1

   * - Caller
     - Permitted operations
   * - Customer admin / manager
     - ``PUT /v1.0/conversations/<id>``: assign, reassign, or unassign the conversation. All fields (owner, name, detail) may be updated. ``POST /v1.0/conversations/<id>/unassign``: unassign the conversation (no body required).
   * - Owning agent
     - ``POST /v1.0/conversations/<id>/unassign`` only — self-unassign without a request body. ``PUT /v1.0/conversations/<id>`` is **not** permitted for agents (returns 403 even for the owning agent).
   * - Any other agent
     - No assignment-related changes permitted. 403 returned.


.. note:: **Breaking Change**

   As of this release, ``PUT /v1.0/conversations/<id>`` requires **admin or manager** permission. Owning agents that previously used this endpoint to self-unassign (by sending the nil UUID) **must migrate** to ``POST /v1.0/conversations/<id>/unassign`` instead. The ``/unassign`` endpoint is the supported path for agent-initiated unassignment.

**Service-Agents Surface**

The same endpoints are also available under the ``/service_agents/`` path prefix, which is admin/manager-only:

* ``PUT https://api.voipbin.net/v1.0/service_agents/conversations/<id>`` — same as ``PUT /v1.0/conversations/<id>`` but requires admin or manager permission (no agent access).
* ``POST https://api.voipbin.net/v1.0/service_agents/conversations/<id>/unassign`` — same unassign semantics as ``POST /v1.0/conversations/<id>/unassign`` (admin/manager + owning agent).

**Error Responses**

* **403 Forbidden:**
    * **Cause (PUT):** Caller is not an admin or manager. Agents (including the owning agent) are not permitted to call ``PUT /v1.0/conversations/<id>``.
    * **Cause (POST /unassign):** Caller is neither an admin/manager nor the current owner of the conversation.
    * **Fix:** Use a customer admin or manager token for ``PUT``. For unassignment, use ``POST /conversations/<id>/unassign`` with the owning agent's token or any admin/manager token.

* **400 Bad Request:**
    * **Cause:** ``owner_id`` could not be validated — either it does not reference an existing agent, or the referenced agent does not belong to the same customer as the conversation. The two cases are intentionally indistinguishable in the response.
    * **Fix:** Verify the agent UUID via ``GET https://api.voipbin.net/v1.0/agents`` under the same customer and retry.

**Behavior Change for Inbound Messages**

When a conversation is assigned (``owner_id`` is a real agent UUID):

* New inbound messages do **not** trigger the conversation's registered flow. No new activeflow is created.
* Outbound message delivery via ``POST /v1.0/conversations/<id>/messages`` continues to work normally — the assigned agent (or admin/manager) can reply through the standard message-send API.
* Any activeflow that was already running before the assignment continues to run to completion; assignment does not interrupt in-flight flows.

When a conversation is unassigned (``owner_id`` is the nil UUID or empty):

* The next inbound message resumes the standard behavior and triggers the registered flow as usual.

.. note:: **AI Implementation Hint**

   To unassign a conversation as the owning agent, use ``POST https://api.voipbin.net/v1.0/conversations/<id>/unassign`` with no request body. Do **not** use ``PUT /conversations/<id>`` — that endpoint now requires admin or manager permission and will return 403 for agent callers. Admin and manager callers may use either endpoint.

**Listing Conversations by Owner**

Filter conversations by their currently assigned agent using the ``owner_id`` query parameter on ``GET https://api.voipbin.net/v1.0/conversations``:

.. code:: bash

    GET https://api.voipbin.net/v1.0/conversations?owner_id=<agent-uuid>&page_token=<token>

This is the supported way to build a "my conversations" view for an agent.

Permission rule: an agent caller (non-admin, non-manager) MUST set ``owner_id`` to their own agent UUID; otherwise the request is rejected with ``403 Forbidden``. Admin and manager callers may pass any ``owner_id`` or omit the filter entirely. Omitting the filter — or passing the nil UUID — disables the filter and returns all conversations the caller can see.

**Webhook Updates**

When a conversation is assigned or unassigned, a ``conversation_updated`` event fires with the new ``owner_type`` and ``owner_id`` values. See :ref:`Conversation <conversation-struct-conversation>` for the field definitions.


Cross-Channel Continuity
------------------------
The key feature of VoIPBIN Conversations is seamless channel switching.

**Cross-Channel Example**

::

    +---------------------------------------------------------------+
    | Conversation: "Order Support #5678"                           |
    +---------------------------------------------------------------+
    |                                                               |
    | [10:00] Customer via SMS:                                     |
    |         "When will my order arrive?"                          |
    |                                                               |
    | [10:05] Support via Email:                                    |
    |         "Your order is scheduled for Friday delivery.         |
    |          Here's the tracking link: ..."                       |
    |         [attachment: tracking-details.pdf]                    |
    |                                                               |
    | [10:10] Customer via Chat:                                    |
    |         "Can I change the delivery address?"                  |
    |                                                               |
    | [10:12] Support via Chat:                                     |
    |         "Yes, I've updated it. Sending confirmation..."       |
    |                                                               |
    | [10:13] Support via SMS:                                      |
    |         "Address updated! Confirmation sent to your email."   |
    |                                                               |
    +---------------------------------------------------------------+

**Benefits**

- Single conversation ID tracks all interactions
- Full history visible regardless of channel
- Participants can use their preferred channel
- Agents see unified view of all messages


Event Types
-----------
VoIPBIN sends webhook events for conversation activities.

.. list-table::
   :header-rows: 1

   * - Event
     - When it fires
   * - conversation_created
     - New conversation started
   * - conversation_updated
     - Conversation metadata changed
   * - conversation_deleted
     - Conversation deleted
   * - conversation_message_created
     - New message created in conversation
   * - conversation_message_updated
     - Message status or content updated
   * - conversation_message_deleted
     - Message deleted from conversation



Common Scenarios
----------------

**Scenario 1: Customer Support Ticket**

Unified support across channels.

::

    Customer: SMS "Having login issues"
         |
         v
    +---------------------------+
    | VoIPBIN creates           |
    | conversation              |
    +---------------------------+
         |
         v
    Support agent responds via email
    (includes detailed instructions + screenshots)
         |
         v
    Customer follows up via chat
    (real-time troubleshooting)
         |
         v
    Issue resolved - conversation closed
    (full history in one place)

**Scenario 2: Order Notifications**

Multi-channel order updates.

::

    +--------------------------------------------+
    | Order placed                               |
    | -> SMS: "Order confirmed! #12345"          |
    +--------------------------------------------+
                       |
                       v
    +--------------------------------------------+
    | Order shipped                              |
    | -> Email: Tracking details + invoice       |
    | -> SMS: "Your order shipped!"              |
    +--------------------------------------------+
                       |
                       v
    +--------------------------------------------+
    | Out for delivery                           |
    | -> SMS: "Arriving today by 5pm"            |
    +--------------------------------------------+
                       |
                       v
    +--------------------------------------------+
    | Delivered                                  |
    | -> SMS: "Delivered! Rate your experience"  |
    +--------------------------------------------+

**Scenario 3: Appointment Reminders**

Escalating reminders across channels.

::

    3 days before:
        Email -> Detailed appointment info

    1 day before:
        SMS -> "Reminder: Appointment tomorrow at 2pm"

    2 hours before:
        SMS -> "Your appointment is in 2 hours"

    Customer replies via any channel
        -> All responses in same conversation


Auto-Generated Titles
---------------------

When a conversation is created automatically — on an inbound SMS or a LINE follow event — the platform
generates a ``name`` and ``detail`` using the peer's identity and the channel type.

**Format**

* ``name``: ``{channel} · {peer}`` — the middle dot (·) is U+00B7.
* ``detail``: ``{channel} conversation``

**Examples**

.. list-table::
   :header-rows: 1

   * - Scenario
     - name
     - detail
   * - Inbound SMS, contact known
     - ``SMS · Alice (+14155551234)``
     - ``SMS conversation``
   * - Inbound SMS, number only (no contact name)
     - ``SMS · +14155551234``
     - ``SMS conversation``
   * - Inbound SMS, no peer information available
     - ``SMS · Unknown``
     - ``SMS conversation``
   * - LINE follow event, display name known
     - ``LINE · Alice``
     - ``LINE conversation``
   * - LINE follow event, no display name
     - ``LINE · Unknown``
     - ``LINE conversation``


.. note:: **AI Implementation Hint**

   LINE user IDs (e.g., ``Uabcdef1234567890``) are opaque platform identifiers and are intentionally
   **not** shown in the generated title even when no display name is available. Only human-readable
   identifiers (phone numbers, email addresses, SIP URIs, extension numbers) appear in parentheses
   after the contact name.

   The ``name`` field can be updated at any time via ``PUT https://api.voipbin.net/v1.0/conversations/{id}``.
   Auto-generated titles apply only to conversations created automatically; conversations created via
   ``POST /v1.0/conversations`` use the ``name`` and ``detail`` values provided in the request body.


Best Practices
--------------

**1. Channel Selection**

- Use SMS for urgent, short notifications
- Use email for detailed information with attachments
- Use chat for real-time, interactive conversations
- Respect participant channel preferences

**2. Conversation Organization**

- Use descriptive conversation names
- Set appropriate conversation timeouts
- Archive completed conversations
- Tag conversations for easy filtering

**3. Message Content**

- Keep messages channel-appropriate
- Include context when switching channels
- Use consistent tone across channels
- Avoid duplicate notifications

**4. Participant Management**

- Verify participant endpoints before adding
- Remove inactive participants
- Handle bounce-backs and failures gracefully


Troubleshooting
---------------

**Message Issues**

.. list-table::
   :header-rows: 1

   * - Symptom
     - Solution
   * - Message not delivered
     - Check participant endpoint validity; verify channel is available for participant
   * - Wrong channel selected
     - Check channel selection priority; verify participant preferences
   * - Duplicate messages
     - Check for retry logic; ensure idempotency using message IDs


**Conversation Issues**

.. list-table::
   :header-rows: 1

   * - Symptom
     - Solution
   * - Messages in wrong conversation
     - Check participant matching; verify conversation is active (not closed)
   * - New conversation created unexpectedly
     - Previous conversation may have timed out; check conversation state
   * - Participant can't receive messages
     - Verify endpoint; check channel availability; review delivery errors


**Webhook Issues**

.. list-table::
   :header-rows: 1

   * - Symptom
     - Solution
   * - Missing events
     - Verify webhook URL configuration; check endpoint returns 200 OK within 5 seconds
   * - Delayed events
     - Check webhook endpoint performance; review retry queue



Related Documentation
---------------------

- :ref:`Message Overview <message-overview>` - SMS/MMS messaging
- :ref:`Email Overview <email-overview>` - Email integration
- :ref:`Talk Overview <talk-overview>` - Internal team messaging
- :ref:`Webhook Overview <webhook-overview>` - Webhook configuration
- :ref:`Using AI in Conversations <ai-overview-conversation-ai>` - Run an AI agent inside an SMS or LINE conversation via the ``ai_talk`` flow action

