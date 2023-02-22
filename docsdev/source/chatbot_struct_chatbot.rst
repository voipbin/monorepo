.. _chatbot-struct-chatbot:

Chatbot
========

.. _chat-struct-chatroom-chatroom:

Chatroom
--------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "engine_type": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* id: Chatbot's ID.
* customer_id: Customer's ID.
* name: Chatbot's name.
* detail: Chatbot's detail.
* *engine_type*: Chatbot's engine type. See detail :ref:`here <chatbot-struct-chatbot-engine_type>`

Example
+++++++

.. code::

    {
        "id": "a092c5d9-632c-48d7-b70b-499f2ca084b1",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "test chatbot",
        "detail": "test chatbot for simple scenario",
        "engine_type": "chatGPT",
        "tm_create": "2023-02-09 07:01:35.666687",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _chatbot-struct-chatbot-engine_type:

Type
----
Chat's type.

=========== ============
Type        Description
=========== ============
chatGPT     Openai's Chat AI. https://chat.openai.com/chat
clova       Naver's Clova AI(WIP). https://clova.ai/
=========== ============
