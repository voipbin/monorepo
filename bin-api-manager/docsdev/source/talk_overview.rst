.. _talk-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free -- Talk messaging does not incur per-message charges.
   * **Async:** No. ``POST https://api.voipbin.net/v1.0/service_agents/talk_chats`` and ``POST https://api.voipbin.net/v1.0/service_agents/talk_messages`` return synchronously with the created resource. Real-time delivery to other participants is handled via WebSocket push events.

VoIPBIN's Talk API provides a modern messaging platform for real-time communication between agents. With support for threading, reactions, and group conversations, Talk enables efficient team collaboration and internal communication.

With the Talk API you can:

- Create one-on-one and group conversations
- Send messages with threading support
- Add emoji reactions to messages
- Attach media files to messages
- Manage participants dynamically


How Talk Works
--------------
Talk provides a real-time messaging system where agents communicate through conversations called "talks."

**Talk Architecture**

::

    +----------+        +----------------+        +-----------+
    | Agent A  |--API-->|    VoIPBIN     |--push->| Agent B   |
    +----------+        |   Talk Hub     |        +-----------+
                        +----------------+
                               |
                        +------+------+
                        |  WebSocket  |
                        |  (real-time)|
                        +-------------+
                               |
              +----------------+----------------+
              v                v                v
         +---------+      +---------+      +---------+
         | Agent C |      | Agent D |      | Agent E |
         +---------+      +---------+      +---------+

**Key Components**

- **Talk**: A conversation container with participants and messages
- **Participant**: An agent who can send and receive messages in a talk
- **Message**: Text, media, or system notifications within a talk
- **Thread**: Messages grouped as replies to a parent message


Chat Types
----------
VoIPBIN supports different chat types for various communication needs.

**Chat Type Values**

+------------+------------------------------------------------------------------+
| Type       | Description                                                      |
+============+==================================================================+
| direct     | 1:1 Direct Message -- private between two users                  |
+------------+------------------------------------------------------------------+
| group      | Group Direct Message -- private multi-user chat, invite-only     |
+------------+------------------------------------------------------------------+
| talk       | Public Open Channel -- topic-based, searchable (e.g., #general)  |
+------------+------------------------------------------------------------------+

**Direct Chat (1:1)**

Private conversation between two agents.

::

    +-------------------------+
    |      Talk (1:1)         |
    +-------------------------+
    |                         |
    |  +---------+            |
    |  | Agent A |<--------+  |
    |  +---------+         |  |
    |       |              |  |
    |       v              |  |
    |  +---------+         |  |
    |  | Agent B |---------+  |
    |  +---------+            |
    |                         |
    +-------------------------+

**Group Chat**

Multi-participant conversation for team discussions.

::

    +---------------------------------------+
    |            Talk (Group)               |
    +---------------------------------------+
    |                                       |
    |  +---------+  +---------+  +-------+  |
    |  | Agent A |  | Agent B |  | Agent |  |
    |  +---------+  +---------+  |   C   |  |
    |       |           |        +-------+  |
    |       |           |            |      |
    |       +-----+-----+------------+      |
    |             |                         |
    |             v                         |
    |      +-------------+                  |
    |      | All receive |                  |
    |      |  messages   |                  |
    |      +-------------+                  |
    |                                       |
    +---------------------------------------+


Chat Lifecycle
--------------
Chats are created and managed through the Talk API. Participants can be added and removed.

.. note:: **AI Implementation Hint**

   The Talk API uses ``/service_agents/talk_chats`` for chat management and ``/service_agents/talk_messages`` for messaging. The ``owner_id`` for participants must be a valid agent UUID obtained from ``GET https://api.voipbin.net/v1.0/agents``. Only agents who are participants of a chat can send or view messages.

**Chat Flow**

::

    POST /service_agents/talk_chats
              |
              v
       +------------+
       |   Chat     |
       |  created   |
       +-----+------+
             |
             v
      Add participants,
      send messages,
      manage threads

**Participant Management**

::

    POST /service_agents/talk_chats/{id}/participants
              |
              v
       +------------+
       | Participant|
       |   added    |
       +-----+------+
             |
             | DELETE /service_agents/talk_chats/{id}/participants/{pid}
             v
       +------------+
       | Participant|
       |  removed   |
       +------------+


Creating and Managing Chats
---------------------------
Create chats and manage participants through the Talk API.

**Create a Chat**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/service_agents/talk_chats?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "type": "group",
            "name": "Project Discussion",
            "participants": [
                {
                    "owner_type": "agent",
                    "owner_id": "agent-uuid-1"
                },
                {
                    "owner_type": "agent",
                    "owner_id": "agent-uuid-2"
                }
            ]
        }'

**Add Participant**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/service_agents/talk_chats/<chat-id>/participants?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "owner_type": "agent",
            "owner_id": "agent-uuid-3"
        }'

**Remove Participant**

.. code::

    $ curl -X DELETE 'https://api.voipbin.net/v1.0/service_agents/talk_chats/<chat-id>/participants/<participant-id>?token=<token>'


Sending Messages
----------------
Send messages within talks with support for threading and media.

**Send a Message**

::

    Agent                                   VoIPBIN              Other Participants
       |                                       |                        |
       | POST /service_agents/talk_messages    |                        |
       +-------------------------------------->|                        |
       |                                       | Broadcast via WebSocket|
       |                                       +----------------------->|
       |  message_id                           |                        |
       |<--------------------------------------+                        |
       |                                       |                        |

**Basic Message Example:**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/service_agents/talk_messages?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "chat_id": "<chat-id>",
            "type": "normal",
            "text": "Hello team, let'\''s discuss the project status."
        }'

**Reply to Message (Threading):**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/service_agents/talk_messages?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "chat_id": "<chat-id>",
            "type": "normal",
            "text": "I agree, we should prioritize the API work.",
            "parent_id": "parent-message-id"
        }'

**Message with Media:**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/service_agents/talk_messages?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "chat_id": "<chat-id>",
            "type": "normal",
            "text": "Here is the design document.",
            "medias": [
                {
                    "type": "file",
                    "file_id": "<file-uuid>"
                }
            ]
        }'


Message Threading
-----------------
Threading organizes conversations by grouping related messages.

**Thread Structure**

::

    +---------------------------------------------------------------+
    |  Talk: "Project Discussion"                                   |
    +---------------------------------------------------------------+
    |                                                               |
    |  [Message 1] "What's the status of the API integration?"      |
    |       |                                                       |
    |       +-- [Reply 1.1] "Working on authentication now"         |
    |       |                                                       |
    |       +-- [Reply 1.2] "Should be done by Friday"              |
    |       |                                                       |
    |       +-- [Reply 1.3] "Great, let me know if you need help"   |
    |                                                               |
    |  [Message 2] "Meeting tomorrow at 2pm"                        |
    |       |                                                       |
    |       +-- [Reply 2.1] "I'll be there"                         |
    |                                                               |
    +---------------------------------------------------------------+

**Thread Benefits**

- Keeps related discussions organized
- Easy to follow conversation context
- Reduces noise in main talk view


Message Reactions
-----------------
Add emoji reactions to messages for quick feedback.

**Add Reaction**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/service_agents/talk_messages/<message-id>/reactions?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "emoji": "thumbsup"
        }'

**Reaction Display**

::

    +-----------------------------------------------+
    | [Agent A] "The deployment was successful!"    |
    |                                               |
    |  👍 3   🎉 2   ❤️ 1                           |
    +-----------------------------------------------+

**Common Reactions**

+------------+----------------------------------+
| Emoji      | Common Use                       |
+============+==================================+
| 👍         | Agreement, acknowledgment        |
+------------+----------------------------------+
| 👎         | Disagreement                     |
+------------+----------------------------------+
| ❤️         | Appreciation, love               |
+------------+----------------------------------+
| 🎉         | Celebration, success             |
+------------+----------------------------------+
| 👀         | Looking into it                  |
+------------+----------------------------------+
| ✅         | Done, completed                  |
+------------+----------------------------------+


Message Types
-------------
Messages can be different types based on their origin.

**Normal Messages**

Regular messages sent by agents.

::

    {
        "type": "normal",
        "agent_id": "agent-123",
        "text": "Let's schedule a call tomorrow."
    }

**System Messages**

Automated notifications about talk events.

::

    {
        "type": "system",
        "text": "Agent B joined the conversation."
    }

    {
        "type": "system",
        "text": "Agent C left the conversation."
    }


Real-Time Updates
-----------------
Receive real-time message updates via WebSocket.

**WebSocket Connection**

::

    Agent                          VoIPBIN
       |                              |
       | WebSocket connect            |
       +----------------------------->|
       |                              |
       | Subscribe to talk events     |
       +----------------------------->|
       |                              |
       |<==== chatmessage_created ====|
       |<==== chat_created ===========|
       |<==== chatparticipant_added ==|
       |                              |

**Event Types**

+--------------------------------------+------------------------------------------------+
| Event                                | When it fires                                  |
+======================================+================================================+
| chat_created                         | New chat session created                       |
+--------------------------------------+------------------------------------------------+
| chat_updated                         | Chat session details updated                   |
+--------------------------------------+------------------------------------------------+
| chat_deleted                         | Chat session deleted                           |
+--------------------------------------+------------------------------------------------+
| chatmessage_created                  | New message sent to chat                       |
+--------------------------------------+------------------------------------------------+
| chatmessage_deleted                  | Message removed from chat                      |
+--------------------------------------+------------------------------------------------+
| chatmessage_reaction_updated         | Reaction added or removed on a message         |
+--------------------------------------+------------------------------------------------+
| chatparticipant_added                | Participant joined the chat                    |
+--------------------------------------+------------------------------------------------+
| chatparticipant_removed              | Participant left the chat                      |
+--------------------------------------+------------------------------------------------+


Common Scenarios
----------------

**Scenario 1: Team Project Discussion**

Create a group talk for project collaboration.

::

    1. Create chat with team members
       POST /service_agents/talk_chats
       { "type": "group", "name": "Q1 Project", "participants": [...] }

    2. Send updates as messages
       POST /service_agents/talk_messages
       { "chat_id": "<chat-id>", "type": "normal", "text": "Sprint 1 completed!" }

    3. Team reacts and replies in threads
       - Reactions: 🎉 👍
       - Thread: "Great work! What's next?"

**Scenario 2: Quick Decision**

Use reactions for quick polls.

::

    [Agent A] "Should we deploy today? 👍 for yes, 👎 for no"
         |
         +-- 👍 Agent B
         +-- 👍 Agent C
         +-- 👍 Agent D
         +-- 👎 Agent E
         |
    Result: 3-1, deploy approved

**Scenario 3: Support Escalation**

Internal discussion about customer issue.

::

    +------------------------------------------------+
    | Talk: "Customer Issue #12345"                  |
    +------------------------------------------------+
    |                                                |
    | [Agent A] "Customer reporting payment failure" |
    |                                                |
    | [Agent B] "Let me check the logs"              |
    |     |                                          |
    |     +-- [Reply] "Found it - gateway timeout"   |
    |     +-- [Reply] "Retrying the transaction"     |
    |                                                |
    | [Agent A] "Customer confirmed it works now"    |
    |     👍 Agent B                                 |
    |                                                |
    +------------------------------------------------+


Best Practices
--------------

**1. Talk Organization**

- Use descriptive talk names
- Create separate talks for different topics
- Archive inactive talks

**2. Threading**

- Reply in threads to keep discussions organized
- Use threads for detailed discussions
- Keep main talk for announcements and new topics

**3. Reactions**

- Use reactions for quick acknowledgments
- Avoid over-reacting to every message
- Use consistent reaction meanings in your team

**4. Media Sharing**

- Use descriptive filenames
- Keep file sizes reasonable
- Reference shared files in message text


Troubleshooting
---------------

**Message Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Message not delivered     | Check agent is participant in talk; verify     |
|                           | WebSocket connection                           |
+---------------------------+------------------------------------------------+
| Thread not showing        | Verify parent_id is correct; check message     |
|                           | exists                                         |
+---------------------------+------------------------------------------------+
| Media not loading         | Check URL accessibility; verify file format    |
+---------------------------+------------------------------------------------+

**Participant Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Can't add participant     | Verify agent exists; check talk is active      |
+---------------------------+------------------------------------------------+
| Not receiving messages    | Check participant status; verify WebSocket     |
|                           | subscription                                   |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Agent Overview <agent-overview>` - Agent management
- :ref:`WebSocket Overview <websocket-overview>` - Real-time connections
- :ref:`Conversation Overview <conversation-overview>` - Multi-channel messaging
