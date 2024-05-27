.. _conversation-struct-message:

Message
===============

.. _conversation-struct-message-message:

Message
------------

.. code::

    {
        "id": "<string>",
        "conversation_id": "<string>",
        "direction": "<string>",
        "status": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "source": {
            ...
        },
        "text": "<string>",
        "medias": [],
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* id: Message's ID.
* conversation_id: Conversation's ID.
* direction: Message's direction. See detail :ref:`here <conversation-struct-message-direction>`.
* status: Message's status.
* reference_type: Conversation's reference type. See detail :ref:`here <conversation-struct-conversation-reference_type>`.
* reference_id: Conversation's reference id.
* source: Conversation's source address. See detail :ref:`here <common-struct-address>`.
* text: Message's text.
* medias: Message's medias.

Example
+++++++

.. code::

    {
        "id": "cc46341b-f00a-452f-b527-19c85d030eaf",
        "conversation_id": "64558b45-40a8-43db-b814-9c0dbf6d47b5",
        "direction": "incoming",
        "status": "received",
        "reference_type": "line",
        "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
        "source": {
            "type": "line",
            "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "text": "안녕",
        "medias": [],
        "tm_create": "2022-06-24 04:28:51.558082",
        "tm_update": "2022-06-24 04:28:51.558090",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _conversation-struct-message-direction:

Direction
--------------
Message's direction.

================ ============
Direction type   Description
================ ============
incoming         Incoming message(Towards voipbin).
outgoing         Outgoing message(From voipbin).
================ ============
