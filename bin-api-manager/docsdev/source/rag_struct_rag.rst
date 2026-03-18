.. _rag-struct-rag:

RAG
========

.. _rag-struct-rag-rag:

RAG
--------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "name": "<string>",
        "description": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>"
    }

* ``id`` (UUID): The RAG's unique identifier. Returned when creating a RAG via ``POST /rags`` or when listing RAGs via ``GET /rags``.
* ``customer_id`` (UUID): The customer that owns this RAG. Obtained from the ``id`` field of ``GET /customer``.
* ``name`` (String, Required): A human-readable name for the knowledge base (e.g., ``"Product Documentation KB"``).
* ``description`` (String, Optional): A description of the RAG's purpose or the type of content it contains (e.g., ``"Contains all customer-facing product documentation and FAQs"``).
* ``tm_create`` (String, ISO 8601): Timestamp when the RAG was created.
* ``tm_update`` (String, ISO 8601): Timestamp when the RAG was last updated.

.. note:: **AI Implementation Hint**

   The ``name`` and ``description`` fields are for organizational purposes only — they do not affect how the RAG retrieves or ranks content. Use descriptive names to help identify knowledge bases when you have multiple RAGs (e.g., ``"Support KB - English"`` vs ``"Support KB - Korean"``).

Example
+++++++

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Product Documentation KB",
        "description": "Knowledge base containing product manuals, API docs, and FAQ articles",
        "tm_create": "2026-03-15 09:00:00.000000",
        "tm_update": "2026-03-15 09:00:00.000000"
    }
