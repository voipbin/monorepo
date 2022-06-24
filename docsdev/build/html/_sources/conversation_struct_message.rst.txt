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
* direction: Message's direction.
* status: Message's status.
* reference_type: Conversation's reference type. See detail :ref:`here <conversation-struct-conversation-reference_type>`.
* reference_id: Conversation's reference id.
* source: Conversation's source address.
* text: Message's text.
* medias: Message's medias.

.. _conversation-struct-message-direction:

Direction
--------------
Message's direction.

================ ============
Direction type   Description
================ ============
incoming         Incoming message.
outgoing         Outgoing message.
================ ============
