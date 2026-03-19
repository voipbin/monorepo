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
        "status": "<string>",
        "sources": [<object>],
        "tm_create": "<string>",
        "tm_update": "<string>"
    }

* ``id`` (UUID): The RAG's unique identifier. Returned when creating a RAG via ``POST /rags`` or when listing RAGs via ``GET /rags``.
* ``customer_id`` (UUID): The customer that owns this RAG. Obtained from the ``id`` field of ``GET /customer``.
* ``name`` (String, Required): A human-readable name for the knowledge base (e.g., ``"Product Documentation KB"``).
* ``description`` (String, Optional): A description of the RAG's purpose or the type of content it contains (e.g., ``"Contains all customer-facing product documentation and FAQs"``).
* ``status`` (String enum): The aggregate processing status derived from all source documents. One of:

  * ``pending`` — No documents have been processed yet (or no documents exist).
  * ``processing`` — At least one document is still being processed (pending or in-progress).
  * ``ready`` — At least one document processed successfully. Individual source errors are visible in the ``sources`` list.
  * ``error`` — All documents failed processing. Check individual ``sources`` for details.

* ``sources`` (Array of :ref:`Source <rag-struct-rag-source>`): List of document sources with their individual ingestion status. Each source corresponds to a file or URL provided when creating or adding sources to the RAG.
* ``tm_create`` (String, ISO 8601): Timestamp when the RAG was created.
* ``tm_update`` (String, ISO 8601): Timestamp when the RAG was last updated.

.. note:: **AI Implementation Hint**

   The ``status`` field is computed from the individual document statuses:

   * If **any** document is ``pending`` or ``processing``, the RAG status is ``processing``.
   * If **all** documents are terminal (``ready`` or ``error``) and at least one is ``ready``, the RAG status is ``ready``.
   * Only when **all** documents have ``error`` status does the RAG show ``error``.

   Poll ``GET /rags/{id}`` to check ingestion progress. When ``status`` becomes ``ready``, the RAG can be queried.

Example
+++++++

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Product Documentation KB",
        "description": "Knowledge base containing product manuals, API docs, and FAQ articles",
        "status": "ready",
        "sources": [
            {
                "storage_file_id": "d4e5f6a7-b8c9-0123-defa-456789012345",
                "status": "ready",
                "status_message": "42 chunks created"
            },
            {
                "source_url": "https://docs.example.com/faq",
                "status": "ready",
                "status_message": "15 chunks created"
            }
        ],
        "tm_create": "2026-03-15 09:00:00.000000",
        "tm_update": "2026-03-15 09:00:00.000000"
    }

.. _rag-struct-rag-source:

Source
--------

Each source represents a document that was provided to the RAG for ingestion.

.. code::

    {
        "storage_file_id": "<string>",
        "source_url": "<string>",
        "status": "<string>",
        "status_message": "<string>"
    }

* ``storage_file_id`` (UUID, Optional): The storage file ID if the source is an uploaded file. Obtained from the ``id`` field of ``POST /files``. Present only for uploaded file sources.
* ``source_url`` (String URI, Optional): The URL if the source is a web document. Present only for URL sources.
* ``status`` (String enum): The processing status of this individual source. One of: ``pending``, ``processing``, ``ready``, ``error``.
* ``status_message`` (String): Details about the current status. Contains error details when ``status`` is ``error``.
