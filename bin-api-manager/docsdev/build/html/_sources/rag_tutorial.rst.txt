.. _rag-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before creating a RAG, you need:

* A valid authentication token (String). Obtain via ``POST /auth/login`` or use an accesskey from ``GET /accesskeys``.
* (Optional) A storage file ID (UUID) if you plan to upload documents. Upload files first via ``POST /storage-files``.

.. note:: **AI Implementation Hint**

   A RAG knowledge base is most useful when connected to an AI agent. After completing this tutorial, reference the RAG's ``id`` in an AI configuration via ``POST /ais`` or ``PUT /ais/{id}`` (set the ``rag_id`` field). The AI agent will then query this knowledge base during conversations.

Step 1: Create a RAG
---------------------

Create a new RAG knowledge base to hold your documents.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/rags?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "Product Documentation KB",
            "description": "Knowledge base for product manuals, API docs, and FAQ articles"
        }'

Response:

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",   // Save this as rag_id
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Product Documentation KB",
        "description": "Knowledge base for product manuals, API docs, and FAQ articles",
        "tm_create": "2026-03-15 09:00:00.000000",
        "tm_update": "2026-03-15 09:00:00.000000"
    }

Step 2: Add a URL Document
---------------------------

Add a web page to the RAG. The system will fetch the URL, parse the content, and create searchable chunks.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/rag-documents?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "rag_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
            "name": "Product FAQ Page",
            "doc_type": "url",
            "source_url": "https://docs.example.com/faq"
        }'

Response:

.. code::

    {
        "id": "b2c3d4e5-f6a7-8901-bcde-f23456789012",   // Save this as document_id
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "rag_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "name": "Product FAQ Page",
        "doc_type": "url",
        "storage_file_id": "00000000-0000-0000-0000-000000000000",
        "source_url": "https://docs.example.com/faq",
        "status": "pending",                              // Processing has not started yet
        "status_message": "",
        "tm_create": "2026-03-15 09:10:00.000000",
        "tm_update": "2026-03-15 09:10:00.000000"
    }

Step 3: Add an Uploaded Document
---------------------------------

Add a file that you previously uploaded to VoIPBIN storage. Upload the file first via ``POST /storage-files``, then reference its ``id`` as ``storage_file_id``.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/rag-documents?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "rag_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
            "name": "Product Manual v2.1 (PDF)",
            "doc_type": "uploaded",
            "storage_file_id": "d4e5f6a7-b8c9-0123-defa-456789012345"
        }'

Response:

.. code::

    {
        "id": "c3d4e5f6-a7b8-9012-cdef-345678901234",   // Save this as uploaded_document_id
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "rag_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "name": "Product Manual v2.1 (PDF)",
        "doc_type": "uploaded",
        "storage_file_id": "d4e5f6a7-b8c9-0123-defa-456789012345",
        "source_url": "",
        "status": "pending",
        "status_message": "",
        "tm_create": "2026-03-15 09:15:00.000000",
        "tm_update": "2026-03-15 09:15:00.000000"
    }

Step 4: Check Document Status
------------------------------

Documents are processed asynchronously. Poll the document endpoint to check processing progress.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/rag-documents/b2c3d4e5-f6a7-8901-bcde-f23456789012?token=<YOUR_AUTH_TOKEN>'

Response (processing complete):

.. code::

    {
        "id": "b2c3d4e5-f6a7-8901-bcde-f23456789012",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "rag_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "name": "Product FAQ Page",
        "doc_type": "url",
        "storage_file_id": "00000000-0000-0000-0000-000000000000",
        "source_url": "https://docs.example.com/faq",
        "status": "ready",                                // Processing complete
        "status_message": "",
        "tm_create": "2026-03-15 09:10:00.000000",
        "tm_update": "2026-03-15 09:12:30.000000"
    }

.. note:: **AI Implementation Hint**

   Poll with a reasonable interval (e.g., every 5 seconds). Processing time depends on document size: a short FAQ page may complete in seconds, while a large PDF manual may take several minutes. Only documents with ``status: ready`` are included in RAG query results. Documents with ``status: error`` should be investigated and re-created.

Step 5: List Documents by RAG
-------------------------------

List all documents belonging to a specific RAG by filtering on ``rag_id``.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/rag-documents?rag_id=a1b2c3d4-e5f6-7890-abcd-ef1234567890&token=<YOUR_AUTH_TOKEN>'

Response:

.. code::

    {
        "result": [
            {
                "id": "b2c3d4e5-f6a7-8901-bcde-f23456789012",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "rag_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
                "name": "Product FAQ Page",
                "doc_type": "url",
                "storage_file_id": "00000000-0000-0000-0000-000000000000",
                "source_url": "https://docs.example.com/faq",
                "status": "ready",
                "status_message": "",
                "tm_create": "2026-03-15 09:10:00.000000",
                "tm_update": "2026-03-15 09:12:30.000000"
            },
            {
                "id": "c3d4e5f6-a7b8-9012-cdef-345678901234",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "rag_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
                "name": "Product Manual v2.1 (PDF)",
                "doc_type": "uploaded",
                "storage_file_id": "d4e5f6a7-b8c9-0123-defa-456789012345",
                "source_url": "",
                "status": "ready",
                "status_message": "",
                "tm_create": "2026-03-15 09:15:00.000000",
                "tm_update": "2026-03-15 09:17:45.000000"
            }
        ],
        "next_page_token": ""
    }

Step 6: Update RAG
--------------------

Update the RAG's name or description.

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/rags/a1b2c3d4-e5f6-7890-abcd-ef1234567890?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "Product Documentation KB (v2)",
            "description": "Updated knowledge base with product manuals, API docs, FAQ articles, and troubleshooting guides"
        }'

Response:

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Product Documentation KB (v2)",
        "description": "Updated knowledge base with product manuals, API docs, FAQ articles, and troubleshooting guides",
        "tm_create": "2026-03-15 09:00:00.000000",
        "tm_update": "2026-03-15 10:00:00.000000"
    }

Step 7: Delete a Document
--------------------------

Delete a single document from the RAG. This removes the document and all its chunks from the vector database.

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/rag-documents/b2c3d4e5-f6a7-8901-bcde-f23456789012?token=<YOUR_AUTH_TOKEN>'

Response:

.. code::

    {
        "id": "b2c3d4e5-f6a7-8901-bcde-f23456789012",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "rag_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "name": "Product FAQ Page",
        "doc_type": "url",
        "storage_file_id": "00000000-0000-0000-0000-000000000000",
        "source_url": "https://docs.example.com/faq",
        "status": "ready",
        "status_message": "",
        "tm_create": "2026-03-15 09:10:00.000000",
        "tm_update": "2026-03-15 09:12:30.000000"
    }

.. note:: **AI Implementation Hint**

   Deleting a document does not affect the RAG itself or other documents in the RAG. The deleted document's chunks are removed from the vector database, so subsequent RAG queries will no longer return results from that document. The storage file (if ``doc_type`` is ``uploaded``) is NOT deleted — manage storage files separately via ``DELETE /storage-files/{id}``.

Step 8: Delete a RAG
----------------------

Delete a RAG and all its documents. This is a cascade delete — all documents belonging to the RAG are deleted along with their vector database chunks.

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/rags/a1b2c3d4-e5f6-7890-abcd-ef1234567890?token=<YOUR_AUTH_TOKEN>'

Response:

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Product Documentation KB (v2)",
        "description": "Updated knowledge base with product manuals, API docs, FAQ articles, and troubleshooting guides",
        "tm_create": "2026-03-15 09:00:00.000000",
        "tm_update": "2026-03-15 10:00:00.000000"
    }

.. note:: **AI Implementation Hint**

   Deleting a RAG performs a cascade delete — all documents in the RAG are automatically deleted, and their chunks are removed from the vector database. If any AI configuration references this RAG (via ``rag_id``), the AI will no longer have access to this knowledge base. Update or remove the ``rag_id`` from affected AI configurations via ``PUT /ais/{id}`` to avoid runtime errors.

Best Practices
--------------

**Knowledge Base Design:**

- Create separate RAGs for distinct knowledge domains (e.g., one for product docs, another for internal policies). This keeps retrieval focused and relevant.
- Use descriptive ``name`` and ``description`` fields to make RAGs easy to identify when managing multiple knowledge bases.
- Keep documents focused on a single topic when possible. A 5-page FAQ document retrieves more precisely than a 500-page monolithic manual.

**Document Management:**

- Check document ``status`` before relying on a RAG for AI conversations. Documents with ``status: pending`` or ``status: processing`` are not yet available for queries.
- When source content changes, delete the old document and create a new one. Documents are immutable — there is no update endpoint.
- For ``url`` documents, ensure the URL is publicly accessible. Private URLs behind authentication will fail during the fetch step.
- For ``uploaded`` documents, upload the file to storage first via ``POST /storage-files``, verify the upload succeeded, then create the document.

**Performance:**

- Smaller, well-structured documents produce better retrieval results than large unstructured documents.
- Add multiple targeted documents rather than one massive document covering everything.

Troubleshooting
---------------

* **400 Bad Request:**
    * **Cause:** Missing required field (e.g., ``name``, ``rag_id``, or ``doc_type``).
    * **Fix:** Include all required fields in the request body. Verify field names match the API schema exactly.

* **400 Bad Request:**
    * **Cause:** ``doc_type`` is ``uploaded`` but ``storage_file_id`` is missing or set to the zero UUID.
    * **Fix:** Upload a file via ``POST /storage-files`` first, then set ``storage_file_id`` to the returned ``id``.

* **400 Bad Request:**
    * **Cause:** ``doc_type`` is ``url`` but ``source_url`` is missing or not a valid URL.
    * **Fix:** Provide a fully qualified URL starting with ``http://`` or ``https://``.

* **404 Not Found:**
    * **Cause:** The RAG or document UUID does not exist or belongs to a different customer.
    * **Fix:** Verify the UUID was obtained from a recent ``GET /rags`` or ``GET /rag-documents`` call.

* **404 Not Found:**
    * **Cause:** The ``rag_id`` specified when creating a document does not exist.
    * **Fix:** Create the RAG first via ``POST /rags``, then use the returned ``id`` as ``rag_id``.

* **Document stuck in ``pending`` or ``processing``:**
    * **Cause:** Processing pipeline delay or the document source is temporarily unavailable.
    * **Fix:** Wait and retry ``GET /rag-documents/{id}``. If the status does not change after several minutes, delete and re-create the document.

* **Document status is ``error``:**
    * **Cause:** The system failed to fetch, parse, or process the document content. Common reasons include unreachable URLs, unsupported file formats, or empty documents.
    * **Fix:** Check the ``status_message`` field for details. Fix the underlying issue (e.g., make the URL accessible, use a supported format) and create a new document.
