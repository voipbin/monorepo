.. _ai-struct-ai:

AI
========

.. _ai-struct-ai-ai:

AI
--------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "engine_type": "<string>",
        "init_prompt": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* id: AI's ID.
* customer_id: Customer's ID.
* name: AI's name.
* detail: AI's detail.
* *engine_type*: AI's engine type. See detail :ref:`here <ai-struct-ai-engine_type>`
* init_prompt: Defines AI's initial prompt. It will define the AI engine's behavior.

Example
+++++++

.. code::

    {
        "id": "a092c5d9-632c-48d7-b70b-499f2ca084b1",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "test AI",
        "detail": "test AI for simple scenario",
        "engine_type": "chatGPT",
        "tm_create": "2023-02-09 07:01:35.666687",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _ai-struct-ai-engine_type:

Type
----
AI's type.

=========== ============
Type        Description
=========== ============
chatGPT     Openai's Chat AI. https://chat.openai.com/chat
clova       Naver's Clova AI(WIP). https://clova.ai/
=========== ============
