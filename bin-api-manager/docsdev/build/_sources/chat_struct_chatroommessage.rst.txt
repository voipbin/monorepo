.. _chat-struct-chatroommessage:

Chatroommessage
===============

.. _chat-struct-chatroommessage-chatroommessage:

Chatroommessage
---------------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "chatroom_id": "<string>",
        "messagechat_id": "<string>",
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
        "tm_delete": "<string>"
    }


* id: Chatroommessage's ID.
* customer_id: Customer's ID.
* chatroom_id: Chatroom's ID.
* messagechat_id: Chatmessage's ID.
* source: Source info. See detail  :ref:`here <common-struct-address-address>`.
* *type*: Chatmessage's type. See detail :ref:`here <chat-struct-chatroommessage-type>`.
* text: Text message.
* medias: List of medias.

Example
+++++++

.. code::

    {
        "id": "04f90d46-2d51-4c8e-ba7d-e181f48bc925",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "chatroom_id": "1e385680-0f41-4e2a-b154-a61c62bf830a",
        "messagechat_id": "2b4acb7b-f1ba-43c5-ae43-0435a07d55ea",
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
        "tm_create": "2022-09-25 13:11:59.274200",
        "tm_update": "2022-09-25 13:11:59.274200",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _chat-struct-chatroommessage-type:

Type
----
Chatroommessage's type.

=========== ============
Type        Description
=========== ============
""(empty)   Unknown type.
system      System message.
normal      Normal message.
=========== ============
