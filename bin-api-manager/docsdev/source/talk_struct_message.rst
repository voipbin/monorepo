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

* id: Message's unique identifier.
* customer_id: Customer's unique identifier.
* owner_type: Type of the message owner (e.g., "agent").
* owner_id: Owner's unique identifier.
* chat_id: Talk's unique identifier that this message belongs to.
* parent_id: Parent message ID for threaded replies (optional).
* type: Message type. See detail :ref:`here <talk-struct-message-type>`.
* text: Message text content.
* medias: Array of media attachments.
* metadata: Message metadata including reactions.
* tm_create: Timestamp when the message was created.
* tm_update: Timestamp when the message was last updated.
* tm_delete: Timestamp when the message was deleted (soft delete).

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

=========== ============
Type        Description
=========== ============
normal      Regular message sent by an agent.
system      System-generated notification message.
=========== ============

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

* **emoji**: The emoji character (e.g., "👍", "❤️", "😊").
* **owner_type**: Type of the reactor (e.g., "agent").
* **owner_id**: Unique identifier of the agent who reacted.
* **tm_create**: Timestamp when the reaction was added.

Multiple agents can react to the same message, and the same agent can add multiple different emoji reactions. Reactions provide quick feedback without sending a full message response.
