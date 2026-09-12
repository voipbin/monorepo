.. _talk-struct-participant:

Participant
===========

.. _talk-struct-participant-participant:

Participant
-----------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "owner_type": "<string>",
        "owner_id": "<string>",
        "chat_id": "<string>",
        "tm_joined": "<string>"
    }

* ``id`` (UUID): The participant's unique identifier. Returned when adding a participant via ``POST /service_agents/talk_chats/{id}/participants``.
* ``customer_id`` (UUID): The customer's unique identifier. Obtained from ``GET /customers``.
* ``owner_type`` (enum string): Type of the participant. Currently only ``"agent"`` is supported.
* ``owner_id`` (UUID): The agent's unique identifier. Obtained from ``GET /agents``.
* ``chat_id`` (UUID): The talk's unique identifier that this participant belongs to. Obtained from ``GET /service_agents/talk_chats``.
* ``tm_joined`` (string, ISO 8601): Timestamp when the participant joined the talk.

.. note:: **AI Implementation Hint**

   When a participant is removed and later re-added, a new participant record is created with a new ``id`` and updated ``tm_joined`` timestamp. The previous participant record is hard-deleted from the database.

Example
+++++++

.. code::

    {
        "id": "f4d6e9b3-5c4a-4d5e-9f8a-2b3c4d5e6f7g",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "owner_type": "agent",
        "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
        "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "tm_joined": "2024-01-17 10:30:00.000000"
    }

Participant Management
----------------------
Participants can be added to or removed from talks dynamically. When a participant is removed and later re-added, their join timestamp is updated to reflect the most recent join time.

**Participant Rules:**

* Each talk can have multiple participants.
* The same agent cannot be added as a participant twice (enforced by unique constraint).
* When a participant is removed, they are hard-deleted from the database.
* Re-adding a participant creates a new record with a new join timestamp.
* Only participants of a talk can view messages and send new messages.

**Permissions:**

* Participants can view all messages in the talk.
* Participants can send messages to the talk.
* Participants can add reactions to any message in the talk.
* Non-participants cannot access talk messages or send messages.
