.. _conversation-struct-conversation:

Conversation
===============

.. _conversation-struct-conversation-conversation:

Conversation
------------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "owner_type": "<string>",
        "owner_id": "<string>",
        "account_id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "type": "<string>",
        "dialog_id": "<string>",
        "self": {
            ...
        },
        "peer": {
            ...
        },
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The conversation's unique identifier. Returned when creating via ``POST /conversations`` or listing via ``GET /conversations``.
* ``customer_id`` (UUID): The customer's ID. Obtained from the ``id`` field of ``GET /customers``.
* ``owner_type`` (enum string): The type of agent currently assigned to this conversation. When set (e.g., ``agent``), the conversation is currently owned by an agent and inbound messages on this conversation **skip the registered flow trigger** until the assignment is cleared. Empty string means no agent is assigned and the registered flow runs as usual on inbound messages. The server derives this field from ``owner_id``; clients must not set it directly. See :ref:`Assigning a Conversation to an Agent <conversation-overview-assigning-conversation-to-agent>`.
* ``owner_id`` (UUID): The unique identifier of the agent currently assigned to this conversation. Obtained from the ``id`` field of ``GET /agents``. The nil UUID ``00000000-0000-0000-0000-000000000000`` (or empty string in webhook payloads) means the conversation is unassigned. When populated, the conversation is currently owned by that agent and inbound messages **skip the registered flow trigger** for the duration of the assignment. Already-running activeflows are unaffected. See :ref:`Assigning a Conversation to an Agent <conversation-overview-assigning-conversation-to-agent>`.
* ``account_id`` (UUID): The messaging account ID associated with this conversation.
* ``name`` (String): A human-readable name for the conversation (e.g., "Customer Support #1234").
* ``detail`` (String): Additional description or context for the conversation.
* ``type`` (enum string): The channel type of this conversation. See :ref:`Type <conversation-struct-conversation-type>`.
* ``dialog_id`` (String): An identifier associated with the channel dialog (e.g., a phone number or Line chatroom ID).
* ``self`` (Object): The local party's address in this conversation. See :ref:`Address <common-struct-address>`.
* ``peer`` (Object): The remote party's address in this conversation. See :ref:`Address <common-struct-address>`.
* ``tm_create`` (string, ISO 8601): Timestamp when the conversation was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update.
* ``tm_delete`` (string, ISO 8601): Timestamp of deletion (soft delete).

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the conversation has not been deleted.

Example
+++++++

.. code::

    {
        "id": "bdc9d9f5-706c-4e2d-9be7-7dc1e5fd45a0",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "owner_type": "agent",
        "owner_id": "3a4b5c6d-7e8f-9012-3456-789abcdef012",
        "account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "name": "conversation",
        "detail": "conversation detail",
        "type": "message",
        "dialog_id": "+673802",
        "self": {
            "type": "tel",
            "target": "+14703298699",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "peer": {
            "type": "tel",
            "target": "+673802",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "tm_create": "2022-06-23 05:05:40.950834",
        "tm_update": "2022-06-23 05:05:40.950842",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _conversation-struct-conversation-type:

Type
----
Conversation's type.

+------------------+------------------------------------------------------------------+
| Type             | Description                                                      |
+==================+==================================================================+
| message          | Conversation initiated via SMS/MMS messaging channel.            |
+------------------+------------------------------------------------------------------+
| line             | Conversation initiated via Line messaging platform.              |
+------------------+------------------------------------------------------------------+
| whatsapp         | Conversation initiated via WhatsApp messaging platform.          |
+------------------+------------------------------------------------------------------+
| email            | Conversation initiated via email channel.                        |
+------------------+------------------------------------------------------------------+
