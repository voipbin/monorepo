.. _ai-struct-aiprompthistory:

AIPromptHistory
===============

.. _ai-struct-aiprompthistory-aiprompthistory:

AIPromptHistory
---------------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "ai_id": "<string>",
        "prompt": "<string>",
        "tm_create": "<string>"
    }

* ``id`` (UUID): Unique identifier of this prompt history entry. Returned when listing via ``GET /ais/{ai_id}/prompt_histories`` or retrieving a single entry via ``GET /ais/{ai_id}/prompt_histories/{history_id}``.
* ``customer_id`` (UUID): The customer that owns this entry. Obtained from the ``id`` field of ``GET /customers``.
* ``ai_id`` (UUID): The AI configuration this entry belongs to. Obtained from the ``id`` field of ``GET /ais``.
* ``prompt`` (string): The ``init_prompt`` value recorded at the time of this history entry. Created automatically whenever the ``init_prompt`` of an AI changes via ``POST /ais`` or ``PUT /ais/{id}``.
* ``tm_create`` (string, ISO 8601): When this history entry was recorded.

.. note:: **AI Implementation Hint**

   Prompt history is read-only. Entries are created automatically whenever ``init_prompt`` changes; they cannot be created, updated, or deleted via the API. To restore a previous prompt, read the desired entry and submit it via ``PUT /ais/{id}``.

Example
+++++++

.. code::

    {
        "id": "b1c2d3e4-f5a6-7890-bcde-f12345678901",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "ai_id": "a092c5d9-632c-48d7-b70b-499f2ca084b1",
        "prompt": "You are a friendly sales assistant. Help customers find the right products.",
        "tm_create": "2024-03-15 10:22:45.123456"
    }
