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

Step 1: Create a RAG with Sources
-----------------------------------

Create a new RAG knowledge base with initial document sources. You can provide uploaded files via ``storage_file_ids`` and/or web URLs via ``source_urls`` at creation time.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/rags?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "Product Documentation KB",
            "description": "Knowledge base for product manuals, API docs, and FAQ articles",
            "storage_file_ids": ["d4e5f6a7-b8c9-0123-defa-456789012345"],
            "source_urls": ["https://docs.example.com/faq"]
        }'

Response:

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",   // Save this as rag_id
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Product Documentation KB",
        "description": "Knowledge base for product manuals, API docs, and FAQ articles",
        "status": "processing",
        "sources": [
            {
                "id": "f1a2b3c4-d5e6-7890-abcd-ef1234567891",   // Source ID for the uploaded file
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "storage_file_id": "d4e5f6a7-b8c9-0123-defa-456789012345",
                "status": "pending",
                "status_message": ""
            },
            {
                "id": "f1a2b3c4-d5e6-7890-abcd-ef1234567892",   // Source ID for the URL
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "source_url": "https://docs.example.com/faq",
                "status": "pending",
                "status_message": ""
            }
        ],
        "tm_create": "2026-03-15 09:00:00.000000",
        "tm_update": "2026-03-15 09:00:00.000000"
    }

.. note:: **AI Implementation Hint**

   You can also create a RAG without sources (omit ``storage_file_ids`` and ``source_urls``) and add them later via ``POST /rags/{id}/sources``. This is useful when you want to set up the RAG first and add documents incrementally.

Step 2: List RAGs
-------------------

Retrieve a paginated list of all RAG knowledge bases for the authenticated customer.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/rags?token=<YOUR_AUTH_TOKEN>&page_size=10'

Response:

.. code::

    {
        "next_page_token": "2026-03-15 09:00:00.000000",
        "result": [
            {
                "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "name": "Product Documentation KB",
                "description": "Knowledge base for product manuals, API docs, and FAQ articles",
                "status": "ready",
                "sources": [
                    {
                        "id": "f1a2b3c4-d5e6-7890-abcd-ef1234567891",
                        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                        "storage_file_id": "d4e5f6a7-b8c9-0123-defa-456789012345",
                        "status": "ready",
                        "status_message": "42 chunks created"
                    },
                    {
                        "id": "f1a2b3c4-d5e6-7890-abcd-ef1234567892",
                        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                        "source_url": "https://docs.example.com/faq",
                        "status": "ready",
                        "status_message": "15 chunks created"
                    }
                ],
                "tm_create": "2026-03-15 09:00:00.000000",
                "tm_update": "2026-03-15 09:00:00.000000"
            }
        ]
    }

.. note:: **AI Implementation Hint**

   Use ``page_size`` (Integer) to control how many RAGs are returned per page and ``page_token`` (String, ISO 8601 timestamp) to fetch subsequent pages. The ``next_page_token`` in the response is the cursor for the next page. When ``next_page_token`` is empty, there are no more results.

Step 3: Add More Sources to an Existing RAG
----------------------------------------------

Add additional document sources to an existing RAG via ``POST /rags/{id}/sources``.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/rags/a1b2c3d4-e5f6-7890-abcd-ef1234567890/sources?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source_urls": ["https://docs.example.com/getting-started"]
        }'

Response:

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Product Documentation KB",
        "description": "Knowledge base for product manuals, API docs, and FAQ articles",
        "status": "processing",
        "sources": [
            {
                "id": "f1a2b3c4-d5e6-7890-abcd-ef1234567891",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "storage_file_id": "d4e5f6a7-b8c9-0123-defa-456789012345",
                "status": "ready",
                "status_message": "42 chunks created"
            },
            {
                "id": "f1a2b3c4-d5e6-7890-abcd-ef1234567892",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "source_url": "https://docs.example.com/faq",
                "status": "ready",
                "status_message": "15 chunks created"
            },
            {
                "id": "f1a2b3c4-d5e6-7890-abcd-ef1234567893",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "source_url": "https://docs.example.com/getting-started",
                "status": "pending",
                "status_message": ""
            }
        ],
        "tm_create": "2026-03-15 09:00:00.000000",
        "tm_update": "2026-03-15 09:30:00.000000"
    }

Step 4: Remove a Source from a RAG
-------------------------------------

Remove a single source (document) and its chunks from a RAG. Use the source ``id`` (UUID) from the ``sources`` array in the ``GET /rags/{id}`` response.

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/rags/a1b2c3d4-e5f6-7890-abcd-ef1234567890/sources/f1a2b3c4-d5e6-7890-abcd-ef1234567892?token=<YOUR_AUTH_TOKEN>'

Response (updated RAG with the source removed):

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Product Documentation KB",
        "description": "Knowledge base for product manuals, API docs, and FAQ articles",
        "status": "ready",
        "sources": [
            {
                "id": "f1a2b3c4-d5e6-7890-abcd-ef1234567891",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "storage_file_id": "d4e5f6a7-b8c9-0123-defa-456789012345",
                "status": "ready",
                "status_message": "42 chunks created"
            },
            {
                "id": "f1a2b3c4-d5e6-7890-abcd-ef1234567893",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "source_url": "https://docs.example.com/getting-started",
                "status": "ready",
                "status_message": "28 chunks created"
            }
        ],
        "tm_create": "2026-03-15 09:00:00.000000",
        "tm_update": "2026-03-15 09:40:00.000000"
    }

.. note:: **AI Implementation Hint**

   Removing a source deletes the source document and all its vector chunks from the database. This is useful for replacing outdated content: remove the old source, then add the updated version via ``POST /rags/{id}/sources``. The source ``id`` is returned in the ``sources`` array of any ``GET /rags/{id}``, ``POST /rags``, or ``POST /rags/{id}/sources`` response.

Step 5: Check RAG Status
--------------------------

The RAG's ``status`` field reflects the aggregate processing state of all its sources. Poll the RAG endpoint to check progress.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/rags/a1b2c3d4-e5f6-7890-abcd-ef1234567890?token=<YOUR_AUTH_TOKEN>'

Response (all sources processed):

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Product Documentation KB",
        "description": "Knowledge base for product manuals, API docs, and FAQ articles",
        "status": "ready",
        "sources": [
            {
                "id": "f1a2b3c4-d5e6-7890-abcd-ef1234567891",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "storage_file_id": "d4e5f6a7-b8c9-0123-defa-456789012345",
                "status": "ready",
                "status_message": "42 chunks created"
            },
            {
                "id": "f1a2b3c4-d5e6-7890-abcd-ef1234567893",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "source_url": "https://docs.example.com/getting-started",
                "status": "ready",
                "status_message": "28 chunks created"
            }
        ],
        "tm_create": "2026-03-15 09:00:00.000000",
        "tm_update": "2026-03-15 09:40:00.000000"
    }

.. note:: **AI Implementation Hint**

   Poll with a reasonable interval (e.g., every 5 seconds). Processing time depends on document size: a short FAQ page may complete in seconds, while a large PDF manual may take several minutes. When the RAG ``status`` is ``ready``, at least one source has been successfully processed and the RAG can be used for queries. Check individual source ``status`` fields for per-document details.

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

Step 7: Delete a RAG
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

- Check the RAG ``status`` before relying on it for AI conversations. A RAG with ``status: processing`` still has documents being ingested.
- When source content changes, remove the outdated source via ``DELETE /rags/{id}/sources/{source_id}`` and add the updated version via ``POST /rags/{id}/sources``. Individual sources can be removed, but not updated in place.
- For ``url`` sources, ensure the URL is publicly accessible. Private URLs behind authentication will fail during the fetch step.
- For uploaded file sources, upload the file to storage first via ``POST /storage-files``, verify the upload succeeded, then provide the ``storage_file_id`` when creating or adding sources to a RAG.

**Performance:**

- Smaller, well-structured documents produce better retrieval results than large unstructured documents.
- Add multiple targeted documents rather than one massive document covering everything.

Troubleshooting
---------------

* **400 Bad Request:**
    * **Cause:** Missing required field (e.g., ``name`` when creating a RAG).
    * **Fix:** Include all required fields in the request body. Verify field names match the API schema exactly.

* **400 Bad Request:**
    * **Cause:** Neither ``storage_file_ids`` nor ``source_urls`` provided when adding sources.
    * **Fix:** Provide at least one ``storage_file_ids`` UUID or one ``source_urls`` URL.

* **404 Not Found:**
    * **Cause:** The RAG or source UUID does not exist or belongs to a different customer.
    * **Fix:** Verify the UUID was obtained from a recent ``GET /rags`` or ``GET /rags/{id}`` call.

* **409 Conflict:**
    * **Cause:** A source with the same ``storage_file_id`` or ``source_url`` already exists in this RAG. Duplicate sources are rejected.
    * **Fix:** Check the existing ``sources`` array via ``GET /rags/{id}`` before adding. If you need to refresh a source, remove the existing one first via ``DELETE /rags/{id}/sources/{source_id}``, then re-add it.

* **RAG stuck in ``processing``:**
    * **Cause:** One or more source documents are still being processed, or a source is temporarily unavailable.
    * **Fix:** Check individual source statuses in the ``sources`` array of the ``GET /rags/{id}`` response. If a source has ``status: error``, check its ``status_message`` for details.

* **Source status is ``error``:**
    * **Cause:** The system failed to fetch, parse, or process the source content. Common reasons include unreachable URLs, unsupported file formats, or empty documents.
    * **Fix:** Check the ``status_message`` field for details. Fix the underlying issue (e.g., make the URL accessible, use a supported format). Remove the failed source via ``DELETE /rags/{id}/sources/{source_id}`` and re-add the corrected source via ``POST /rags/{id}/sources``.
