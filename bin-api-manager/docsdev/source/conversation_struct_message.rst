.. _conversation-struct-message:

Message
===============

.. _conversation-struct-message-message:

Message
------------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "conversation_id": "<string>",
        "direction": "<string>",
        "status": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "text": "<string>",
        "medias": [],
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The message's unique identifier within the conversation.
* ``customer_id`` (UUID): The customer's ID. Obtained from the ``id`` field of ``GET /customers``.
* ``conversation_id`` (UUID): The parent conversation's ID. Obtained from ``GET /conversations`` or from the URL path when sending messages.
* ``direction`` (enum string): Whether the message is incoming or outgoing. See :ref:`Direction <conversation-struct-message-direction>`.
* ``status`` (enum string): The message's delivery status (e.g., ``sent``, ``received``, ``failed``).
* ``reference_type`` (enum string): The channel used for this message. See the parent conversation's ``reference_type``.
* ``reference_id`` (UUID): An identifier associated with the channel reference.
* ``text`` (String): The message body text content.
* ``medias`` (Array of Object): List of media attachments (images, videos, etc.) included with the message.
* ``tm_create`` (string, ISO 8601): Timestamp when the message was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last status update.
* ``tm_delete`` (string, ISO 8601): Timestamp of deletion (soft delete).

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the message has not been deleted.

Example
+++++++

.. code::

    {
        "id": "cc46341b-f00a-452f-b527-19c85d030eaf",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "conversation_id": "64558b45-40a8-43db-b814-9c0dbf6d47b5",
        "direction": "incoming",
        "status": "received",
        "reference_type": "line",
        "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
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

+------------------+------------------------------------------------------------------+
| Direction        | Description                                                      |
+==================+==================================================================+
| incoming         | Incoming message from a participant towards VoIPBIN. Delivered   |
|                  | to your application via webhook.                                 |
+------------------+------------------------------------------------------------------+
| outgoing         | Outgoing message sent from your application via VoIPBIN to a     |
|                  | conversation participant.                                        |
+------------------+------------------------------------------------------------------+
