.. _conversation-overview:

Overview
========
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

+------------+------------------------------------------------------------------+
| Channel    | Description                                                      |
+============+==================================================================+
| SMS        | Standard text messages to mobile phones                          |
+------------+------------------------------------------------------------------+
| MMS        | Multimedia messages with images, videos, audio                   |
+------------+------------------------------------------------------------------+
| Email      | Email messages with attachments                                  |
+------------+------------------------------------------------------------------+
| Chat       | Real-time web/mobile chat                                        |
+------------+------------------------------------------------------------------+
| SNS        | Social networking services (WhatsApp, Facebook Messenger, etc.)  |
+------------+------------------------------------------------------------------+

**Channel Selection**

::

                      Which channel?
                            |
          +-----------------+------------------+
          |                 |                  |
          v                 v                  v
    +----------+      +----------+       +----------+
    |   SMS    |      |  Email   |       |   Chat   |
    +----+-----+      +----+-----+       +----+-----+
         |                 |                  |
         v                 v                  v
    Short,            Formal,            Real-time,
    immediate         detailed           interactive
    (< 160 chars)     with attachments   presence-aware


Conversation Lifecycle
----------------------
Conversations move through predictable states.

**Conversation States**

::

    Message received/sent
           |
           v
    +------------+
    |   active   |<-----------------+
    +-----+------+                  |
          |                         |
          | no activity             | new message
          | (timeout)               |
          v                         |
    +------------+                  |
    |   idle     |------------------+
    +-----+------+
          |
          | explicit close
          | or long timeout
          v
    +------------+
    |   closed   |
    +------------+

**State Descriptions**

+-------------+------------------------------------------------------------------+
| State       | What's happening                                                 |
+=============+==================================================================+
| active      | Conversation has recent activity, participants engaged           |
+-------------+------------------------------------------------------------------+
| idle        | No recent messages, but conversation still open                  |
+-------------+------------------------------------------------------------------+
| closed      | Conversation ended, new messages create new conversation         |
+-------------+------------------------------------------------------------------+


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


Creating Conversations
----------------------
Create conversations explicitly or let VoIPBIN auto-create them.

**Create a Conversation**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/conversations?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "Customer Support #1234",
            "participants": [
                {
                    "type": "tel",
                    "target": "+15551234567"
                },
                {
                    "type": "email",
                    "target": "support@company.com"
                }
            ]
        }'

**Add Participant**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/conversations/<conversation-id>/participants?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "type": "tel",
            "target": "+15559876543"
        }'


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
            "text": "Your order has been shipped!",
            "medias": [
                {
                    "type": "image/png",
                    "url": "https://example.com/tracking-map.png"
                }
            ]
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

+----------------------------------+------------------------------------------------+
| Event                            | When it fires                                  |
+==================================+================================================+
| conversation_created             | New conversation started                       |
+----------------------------------+------------------------------------------------+
| conversation_updated             | Conversation metadata changed                  |
+----------------------------------+------------------------------------------------+
| conversation_closed              | Conversation ended                             |
+----------------------------------+------------------------------------------------+
| conversation_message_received    | Inbound message from participant               |
+----------------------------------+------------------------------------------------+
| conversation_message_sent        | Outbound message delivered                     |
+----------------------------------+------------------------------------------------+
| conversation_participant_added   | New participant joined                         |
+----------------------------------+------------------------------------------------+
| conversation_participant_removed | Participant left conversation                  |
+----------------------------------+------------------------------------------------+


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

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Message not delivered     | Check participant endpoint validity; verify    |
|                           | channel is available for participant           |
+---------------------------+------------------------------------------------+
| Wrong channel selected    | Check channel selection priority; verify       |
|                           | participant preferences                        |
+---------------------------+------------------------------------------------+
| Duplicate messages        | Check for retry logic; ensure idempotency      |
|                           | using message IDs                              |
+---------------------------+------------------------------------------------+

**Conversation Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Messages in wrong         | Check participant matching; verify             |
| conversation              | conversation is active (not closed)            |
+---------------------------+------------------------------------------------+
| New conversation created  | Previous conversation may have timed out;      |
| unexpectedly              | check conversation state                       |
+---------------------------+------------------------------------------------+
| Participant can't receive | Verify endpoint; check channel availability;   |
| messages                  | review delivery errors                         |
+---------------------------+------------------------------------------------+

**Webhook Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Missing events            | Verify webhook URL configuration; check        |
|                           | endpoint returns 200 OK within 5 seconds       |
+---------------------------+------------------------------------------------+
| Delayed events            | Check webhook endpoint performance; review     |
|                           | retry queue                                    |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Message Overview <message-overview>` - SMS/MMS messaging
- :ref:`Email Overview <email-overview>` - Email integration
- :ref:`Talk Overview <talk-overview>` - Internal team messaging
- :ref:`Webhook Overview <webhook-overview>` - Webhook configuration

