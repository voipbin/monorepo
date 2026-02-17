.. _talk-struct-message:

Message
=======

.. _talk-struct-message-message:

Message
-------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "owner_type": "<string>",
        "owner_id": "<string>",
        "chat_id": "<string>",
        "parent_id": "<string>",
        "type": "<string>",
        "text": "<string>",
        "medias": [
            {
                "type": "<string>"
            }
        ],
        "metadata": {
            "reactions": [
                {
                    "emoji": "<string>",
                    "owner_type": "<string>",
                    "owner_id": "<string>",
                    "tm_create": "<string>"
                }
            ]
        },
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The message's unique identifier. Returned when creating via ``POST /service_agents/talk_messages`` or listing via ``GET /service_agents/talk_messages``.
* ``customer_id`` (UUID): The customer's unique identifier. Obtained from ``GET /customers``.
* ``owner_type`` (enum string): Type of the message owner. Currently only ``"agent"`` for user-sent messages, or ``"system"`` for system-generated messages.
* ``owner_id`` (UUID): The agent's unique identifier who sent the message. Obtained from ``GET /agents``.
* ``chat_id`` (UUID): The talk's unique identifier that this message belongs to. Obtained from ``GET /service_agents/talk_chats``.
* ``parent_id`` (UUID, optional): Parent message ID for threaded replies. Must be the ``id`` of an existing message in the same talk. Omit or set to empty string for top-level messages.
* ``type`` (enum string): Message type. See :ref:`Type <talk-struct-message-type>`.
* ``text`` (String): Message text content.
* ``medias`` (Array of Object): Array of media attachments. Each object contains ``type`` (MIME type string), ``url`` (String), and ``name`` (String).
* ``metadata`` (Object): Message metadata including reactions. Contains a ``reactions`` array.
* ``tm_create`` (string, ISO 8601): Timestamp when the message was created.
* ``tm_update`` (string, ISO 8601): Timestamp when the message was last updated.
* ``tm_delete`` (string, ISO 8601): Timestamp when the message was deleted (soft delete).

.. note:: **AI Implementation Hint**

   A ``tm_delete`` value of empty string ``""`` means the message has not been deleted. To create a threaded reply, set ``parent_id`` to the ``id`` of the message you want to reply to. The parent message must exist in the same talk.

Example
+++++++

.. code::

    {
        "id": "a3c5e8f2-4d3a-4b5c-9e7f-1a2b3c4d5e6f",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "owner_type": "agent",
        "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
        "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "parent_id": "b2d4f7a1-3c2e-4f5a-8d9c-2e3f4a5b6c7d",
        "type": "normal",
        "text": "That's a great idea! Let's proceed with that approach.",
        "medias": [],
        "metadata": {
            "reactions": [
                {
                    "emoji": "👍",
                    "owner_type": "agent",
                    "owner_id": "47fe0b7c-7333-46cf-8b23-61e14e62490a",
                    "tm_create": "2024-01-17 10:35:00.000000"
                }
            ]
        },
        "tm_create": "2024-01-17 10:32:00.000000",
        "tm_update": "2024-01-17 10:32:00.000000",
        "tm_delete": ""
    }

.. _talk-struct-message-type:

Type
----
Message type indicates whether it's a regular message or system notification.

=========== ============================================================
Type        Description
=========== ============================================================
normal      Regular message sent by an agent. Contains user-authored text
            and optional media attachments.
system      System-generated notification message. Automatically created
            when participants join or leave the talk.
=========== ============================================================

Threading
---------
Messages can form conversation threads by specifying a parent message ID. When a message includes a ``parent_id``, it appears as a reply to the parent message, creating a nested conversation structure within the talk.

**Threading Rules:**

* Parent message must exist in the same talk.
* Parent message can be deleted (soft delete) - the thread structure is preserved.
* Replies can have their own replies, creating multi-level threads.

Reactions
---------
Agents can add emoji reactions to messages. Reactions are stored in the message metadata and include:

* ``emoji`` (String): The emoji character (e.g., "thumbsup", "heart").
* ``owner_type`` (enum string): Type of the reactor. Currently only ``"agent"``.
* ``owner_id`` (UUID): Unique identifier of the agent who reacted. Obtained from ``GET /agents``.
* ``tm_create`` (string, ISO 8601): Timestamp when the reaction was added.

Multiple agents can react to the same message, and the same agent can add multiple different emoji reactions. Reactions provide quick feedback without sending a full message response.
