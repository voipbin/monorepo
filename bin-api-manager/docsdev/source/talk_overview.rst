.. _talk-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free -- Talk messaging does not incur per-message charges.
   * **Async:** No. ``POST /talks`` and ``POST /talks/{id}/messages`` return synchronously with the created resource. Real-time delivery to other participants is handled via WebSocket push events.

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


Talk Types
----------
VoIPBIN supports different talk types for various communication needs.

**One-on-One Talk**

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

**Group Talk**

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


Talk Lifecycle
--------------
Talks and participants move through predictable states.

**Talk States**

::

    POST /talks
          |
          v
    +------------+
    |   active   |<-----------------+
    +-----+------+                  |
          |                         |
          | close or               | reopen
          | all leave              |
          v                         |
    +------------+                  |
    |   closed   |------------------+
    +------------+

**Participant States**

::

    POST /talks/{id}/participants
              |
              v
       +------------+
       |   active   |
       +-----+------+
             |
             | leave or removed
             v
       +------------+
       |    left    |
       +------------+


Creating and Managing Talks
---------------------------
Create talks and manage participants through the API.

**Create a Talk**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/talks?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "Project Discussion",
            "participant_ids": [
                "agent-id-1",
                "agent-id-2",
                "agent-id-3"
            ]
        }'

.. note:: **AI Implementation Hint**

   Talk uses the ``/service_agents/talk_chats`` endpoint (not ``/talks``) for agent-facing operations. The participant's ``owner_id`` must be a valid agent UUID obtained from ``GET /agents``. Only agents who are participants of a talk can send or view messages in that talk.

**Add Participant**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/talks/<talk-id>/participants?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "agent_id": "agent-id-4"
        }'

**Remove Participant**

.. code::

    $ curl -X DELETE 'https://api.voipbin.net/v1.0/talks/<talk-id>/participants/<participant-id>?token=<token>'


Sending Messages
----------------
Send messages within talks with support for threading and media.

**Send a Message**

::

    Agent                       VoIPBIN                    Other Participants
       |                           |                              |
       | POST /talks/{id}/messages |                              |
       +-------------------------->|                              |
       |                           | Broadcast to participants    |
       |                           +----------------------------->|
       |  message_id               |                              |
       |<--------------------------+                              |
       |                           |                              |

**Basic Message Example:**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/talks/<talk-id>/messages?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "text": "Hello team, let'\''s discuss the project status."
        }'

**Reply to Message (Threading):**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/talks/<talk-id>/messages?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "text": "I agree, we should prioritize the API work.",
            "parent_id": "parent-message-id"
        }'

**Message with Media:**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/talks/<talk-id>/messages?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "text": "Here is the design document.",
            "medias": [
                {
                    "type": "application/pdf",
                    "url": "https://storage.example.com/design-doc.pdf",
                    "name": "design-document.pdf"
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

    $ curl -X POST 'https://api.voipbin.net/v1.0/messages/<message-id>/reactions?token=<token>' \
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
       |<==== message_created ========|
       |<==== message_updated ========|
       |<==== participant_joined =====|
       |<==== reaction_added =========|
       |                              |

**Event Types**

+----------------------+------------------------------------------------+
| Event                | When it fires                                  |
+======================+================================================+
| message_created      | New message sent to talk                       |
+----------------------+------------------------------------------------+
| message_updated      | Message edited                                 |
+----------------------+------------------------------------------------+
| message_deleted      | Message removed                                |
+----------------------+------------------------------------------------+
| reaction_added       | Reaction added to message                      |
+----------------------+------------------------------------------------+
| reaction_removed     | Reaction removed from message                  |
+----------------------+------------------------------------------------+
| participant_joined   | Agent joined the talk                          |
+----------------------+------------------------------------------------+
| participant_left     | Agent left the talk                            |
+----------------------+------------------------------------------------+


Common Scenarios
----------------

**Scenario 1: Team Project Discussion**

Create a group talk for project collaboration.

::

    1. Create talk with team members
       POST /talks
       { "name": "Q1 Project", "participant_ids": [...] }

    2. Send updates as messages
       POST /talks/{id}/messages
       { "text": "Sprint 1 completed!" }

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

- :ref:`Agent Overview <agent_overview>` - Agent management
- :ref:`WebSocket Overview <websocket-overview>` - Real-time connections
- :ref:`Conversation Overview <conversation-overview>` - Multi-channel messaging
