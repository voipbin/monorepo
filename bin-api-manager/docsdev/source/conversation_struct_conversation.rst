.. _conversation-struct-conversation:

Conversation
===============

.. _conversation-struct-conversation-conversation:

Conversation
------------

.. code::


    {
        "id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "source": {
            ...
        },
        "participants": [
            ...
        ],
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The conversation's unique identifier. Returned when creating via ``POST /conversations`` or listing via ``GET /conversations``.
* ``name`` (String): A human-readable name for the conversation (e.g., "Customer Support #1234").
* ``detail`` (String): Additional description or context for the conversation.
* ``reference_type`` (enum string): The channel type that initiated this conversation. See :ref:`Reference type <conversation-struct-conversation-reference_type>`.
* ``reference_id`` (String): An identifier associated with the reference channel (e.g., a phone number or Line user ID).
* ``source`` (Object): The conversation's source address. See :ref:`Address <common-struct-address>`.
* ``participants`` (Array of Object): List of participant addresses in this conversation. See :ref:`Address <common-struct-address>`.
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
        "name": "conversation",
        "detail": "conversation detail",
        "reference_type": "message",
        "reference_id": "+673802",
        "source": {
            "type": "tel",
            "target": "+14703298699",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "participants": [
            {
                "type": "tel",
                "target": "+14703298699",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            {
                "type": "tel",
                "target": "+673802",
                "target_name": "",
                "name": "",
                "detail": ""
            }
        ],
        "tm_create": "2022-06-23 05:05:40.950834",
        "tm_update": "2022-06-23 05:05:40.950842",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _conversation-struct-conversation-reference_type:

Reference type
--------------
Conversation's reference type.

+------------------+------------------------------------------------------------------+
| Reference type   | Description                                                      |
+==================+==================================================================+
| message          | Conversation initiated via SMS/MMS messaging channel.            |
+------------------+------------------------------------------------------------------+
| line             | Conversation initiated via Line messaging platform.              |
+------------------+------------------------------------------------------------------+
