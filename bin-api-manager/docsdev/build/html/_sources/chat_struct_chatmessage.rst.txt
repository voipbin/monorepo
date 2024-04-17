.. _chat-struct-chatmessage:

Chatmessage
===========

.. _chat-struct-chatmessage-chatmessage:

Chatmessage
-----------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "chat_id": "<string>",
        "source": {
            ...
        },
        "type": "<string>",
        "text": "<string>",
        "medias": [
            ...
        ],
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>",
    }

* id: Chat's ID.
* customer_id: Customer's ID.
* chat_id: Master chat's ID.
* source: Source info. See detail  :ref:`here <common-struct-address-address>`.
* *type*: Chatmessage's type. See detail :ref:`here <chat-struct-chatmessage-type>`.
* text: Text message.
* medias: List of medias.

Example
+++++++

.. code::

    {
        "id": "2b4acb7b-f1ba-43c5-ae43-0435a07d55ea",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "source": {
            "type": "agent",
            "target": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "type": "normal",
        "text": "test message",
        "medias": [],
        "tm_create": "2022-09-25 13:11:59.075363",
        "tm_update": "2022-09-25 13:11:59.075363",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


.. _chat-struct-chatmessage-type:

Type
----
Chatmessage's type.

=========== ============
Type        Description
=========== ============
system      System message.
normal      Normal message.
=========== ============
