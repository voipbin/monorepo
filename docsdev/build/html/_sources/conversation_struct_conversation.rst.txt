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
        "participants": [
            {
                "type": "line",
                "target": "",
                "target_name": "me",
                "name": "",
                "detail": ""
            },
            {
                "type": "line",
                "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
                "target_name": "Unknown",
                "name": "",
                "detail": ""
            }
        ],
        "tm_create": "2022-06-17 06:06:14.446158",
        "tm_update": "2022-06-17 06:06:14.446167",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

* id: Conversation's ID.
* name: Conversation's name.
* detail: Conversation's detail.
* reference_type: Conversation's reference type. See detail :ref:
* reference_id: Conversation's reference id.
* participants: List of participants.

Reference type
--------------
Conversation's reference type.

================ ============
Reference type   Description
================ ============
message          Message.
line             Line.
================ ============
