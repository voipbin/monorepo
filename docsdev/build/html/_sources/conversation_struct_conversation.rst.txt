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

* id: Conversation's ID.
* name: Conversation's name.
* detail: Conversation's detail.
* reference_type: Conversation's reference type. See detail :ref:`here <conversation-struct-conversation-reference_type>`.
* reference_id: Conversation's reference id.
* source: Conversation's source address. See detail :ref:`here <common-struct-address>`.
* participants: List of participants. See detail :ref:`here <common-struct-address>`.

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

================ ============
Reference type   Description
================ ============
message          Message(SMS/MMS).
line             Line.
================ ============
