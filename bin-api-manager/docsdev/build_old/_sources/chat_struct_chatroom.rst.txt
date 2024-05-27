.. _chat-struct-chatroom:

Chatroom
========

.. _chat-struct-chatroom-chatroom:

Chatroom
--------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "type": "<string>",
        "chat_id": "<string>",
        "owner_id": "<string>",
        "participant_ids": [
            "<string>",
            ...
        ],
        "name": "<string>",
        "detail": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* id: Chat's ID.
* customer_id: Customer's ID.
* *type*: Chat's type. See detail :ref:`here <chat-struct-chatroom-type>`.
* chat_id: Master chat's ID.
* owner_id: Owner's ID.
* participant_ids: list of participate ids.
* name: Chat's name.
* detail: Chat's detail.

Example
+++++++

.. code::

    {
        "id": "1e385680-0f41-4e2a-b154-a61c62bf830a",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "type": "normal",
        "chat_id": "e8b2e976-f043-44c8-bb89-e214e225e813",
        "owner_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
        "participant_ids": [
            "47fe0b7c-7333-46cf-8b23-61e14e62490a",
            "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"
        ],
        "name": "test chat normal",
        "detail": "test chat with agent 1 and agent2",
        "tm_create": "2022-09-22 02:41:45.237021",
        "tm_update": "2022-09-22 02:41:45.237021",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


.. _chat-struct-chatroom-type:

Type
----
Chat's type.

=========== ============
Type        Description
=========== ============
unknown     unknown.
normal      1:1 chat.
group       n:n group chat.
=========== ============
