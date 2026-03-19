.. _rag-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free for CRUD operations. Chargeable for document processing (parsing, chunking, and embedding consume credits).
   * **Async:** Yes. Document processing runs asynchronously after creation. Check status via ``GET https://api.voipbin.net/v1.0/rag-documents/{id}``.

A **RAG** (Retrieval-Augmented Generation) is a knowledge base that lets your AI agents answer questions using your own content. Instead of relying solely on the LLM's training data, the AI retrieves relevant passages from your uploaded documents and URLs at query time, grounding its responses in your authoritative content.

.. note:: **AI Implementation Hint**

   A RAG is a configuration resource, not a runtime entity. You create a RAG via ``POST /rags``, add documents to it via ``POST /rag-documents``, then reference the RAG's ``id`` in an AI configuration (``rag_id`` field on ``POST /ais`` or ``PUT /ais/{id}``). The RAG itself does not answer questions — it provides the knowledge base that the AI agent queries during conversations.

How it works
============

Architecture
------------

::

    +-----------------------------------------------------------------------+
    |                     RAG Architecture                                  |
    +-----------------------------------------------------------------------+

    1. SETUP (design-time)

       User creates a RAG
       +---------------------+
       |       RAG            |
       | name: "Product KB"   |
       | id: <rag-uuid>       |
       +----------+----------+
                  |
                  | Add documents
                  v
       +---------------------+     +---------------------+
       |   Document A         |     |   Document B         |
       |   type: url          |     |   type: uploaded      |
       |   source_url: https: |     |   storage_file_id:    |
       |     //docs.example.. |     |     <file-uuid>       |
       +----------+----------+     +----------+----------+
                  |                            |
                  v                            v
       +-------------------------------------------------------+
       |              Document Processing Pipeline              |
       |                                                        |
       |   1. Fetch/read content                                |
       |   2. Parse into text                                   |
       |   3. Chunk into passages                               |
       |   4. Generate embeddings (vectorize)                   |
       |   5. Store chunks in vector database                   |
       +-------------------------------------------------------+

    2. RUNTIME (during a conversation)

       Caller asks a question
            |
            v
       AI extracts query from conversation
            |
            v
       Query embedding generated
            |
            v
       Vector DB searched for similar chunks
            |
            v
       Top-k relevant passages retrieved
            |
            v
       Passages injected into LLM context
            |
            v
       AI responds with grounded answer

Document Processing
-------------------
When you add a document to a RAG, the system processes it asynchronously through a pipeline:

1. **Fetch** — For ``url`` documents, the system fetches the content from ``source_url``. For ``uploaded`` documents, it reads the file from storage (``storage_file_id``).
2. **Parse** — The raw content is converted to plain text. Supported formats include PDF, HTML, plain text, and common document formats.
3. **Chunk** — The text is split into overlapping passages (chunks) optimized for retrieval. Chunk sizes are tuned to balance context completeness with retrieval precision.
4. **Embed** — Each chunk is converted to a vector embedding using an embedding model.
5. **Store** — The chunk text and its embedding are stored in a vector database, indexed by the RAG's ``id``.

Document Lifecycle
------------------
Documents progress through a status lifecycle during processing:

* ``pending`` — Document created but processing has not started yet.
* ``processing`` — Document is being fetched, parsed, chunked, and embedded.
* ``ready`` — Processing complete. Chunks are stored and available for retrieval.
* ``error`` — Processing failed. Check ``status_message`` for the error reason.

::

    +-----------------------------------------------------------------------+
    |                     Document Status Lifecycle                         |
    +-----------------------------------------------------------------------+

    Created via POST
         |
         v
    +-----------+       +--------------+       +-----------+
    | pending   | ----> | processing   | ----> |   ready   |
    +-----------+       +--------------+       +-----------+
                              |
                              | (failure)
                              v
                        +-----------+
                        |   error   |
                        +-----------+

.. note:: **AI Implementation Hint**

   After creating a document via ``POST /rag-documents``, poll ``GET /rag-documents/{id}`` to check the ``status`` field. Do not assume the document is ready for queries immediately — processing can take seconds to minutes depending on document size. Only documents with ``status: ready`` contribute to RAG query results.

Use Cases
=========

* **Customer support knowledge base** — Upload FAQ documents, support articles, and troubleshooting guides. The AI agent retrieves relevant answers during live support calls.
* **Product documentation** — Add product manuals, API docs, and feature guides. The AI answers customer questions with accurate, up-to-date product information.
* **Internal training** — Build knowledge bases from training materials, SOPs, and policy documents for AI-assisted employee onboarding calls.
* **Sales enablement** — Upload pricing sheets, competitive analysis, and product comparisons. The AI assists sales agents with accurate data during prospect calls.

Related Documentation
=====================

- :ref:`AI Configuration <ai-overview>` — How AI voice conversations work and how to reference a RAG in an AI configuration
- :ref:`Team <team-overview>` — Multi-agent conversations that can leverage RAG-backed AI members
- :ref:`Storage <storage-main>` — File storage for uploaded documents (``storage_file_id``)
