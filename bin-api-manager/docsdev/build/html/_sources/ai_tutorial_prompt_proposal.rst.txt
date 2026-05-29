.. _ai-tutorial-prompt-proposal:

Tutorial: Improve an AI Prompt from Audits
==========================================

This tutorial walks through proposing an improved system prompt for an AI
based on N completed audits, then accepting the proposal so the AI uses the
new prompt on subsequent calls.

Prerequisites
+++++++++++++

* A valid authentication token (String). Obtain via ``POST /auth/login`` or
  use an accesskey from ``GET /accesskeys``.
* An AI configuration (UUID) you own. Create one via ``POST /ais`` or look
  one up via ``GET /ais``.
* At least 1 (recommended 3–10) completed audits for that AI with
  ``status = "completed"``. Trigger audits via ``POST /aiaudits`` against
  completed AI calls.
* All audits must be for the AI's **current** prompt version. If you have
  updated the AI's ``init_prompt`` since some audits were taken, those
  audits cannot be used in the same proposal.

.. note:: **Cost note**

   Each proposal calls Gemini 2.5 Pro once and is billed against your
   VoIPBIN balance. Generation typically completes in 10–30 seconds.

Step 1: Submit a proposal
-------------------------

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/aipromptproposals?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "ai_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
            "audit_ids": [
                "11111111-1111-1111-1111-111111111111",
                "22222222-2222-2222-2222-222222222222",
                "33333333-3333-3333-3333-333333333333"
            ],
            "language": "en-US"
        }'

Response (``202 Accepted``):

.. code::

    {
        "id": "p1234567-89ab-cdef-0123-456789abcdef",
        "customer_id": "<uuid>",
        "ai_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "audit_ids": [
            "11111111-1111-1111-1111-111111111111",
            "22222222-2222-2222-2222-222222222222",
            "33333333-3333-3333-3333-333333333333"
        ],
        "basis_prompt_history_id": "<uuid>",
        "original_prompt": "You are a helpful assistant...",
        "proposed_prompt": "",
        "rationale": "",
        "status": "progressing",
        "applied_prompt_history_id": "00000000-0000-0000-0000-000000000000",
        "error": "",
        "tm_create": "2026-05-29T10:30:00.000000",
        "tm_update": "9999-01-01T00:00:00.000000",
        "tm_delete": "9999-01-01T00:00:00.000000"
    }

Save the ``id`` from the response — that is the ``proposal_id`` used in
subsequent steps.

Step 2: Poll until completed
----------------------------

Generation runs in the background. Poll the proposal record until ``status``
becomes ``completed``:

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/aipromptproposals/p1234567-89ab-cdef-0123-456789abcdef?token=<YOUR_AUTH_TOKEN>'

Once ``status`` is ``completed``, the response contains the generated
content:

.. code::

    {
        "id": "p1234567-89ab-cdef-0123-456789abcdef",
        "status": "completed",
        "original_prompt": "You are a helpful assistant...",
        "proposed_prompt": "You are a helpful, concise customer-service assistant. When the caller asks about pricing, restate the plan name before quoting the price...",
        "rationale": "The audits show the AI frequently quoted prices without naming the plan, leading to caller confusion. The proposed prompt adds an explicit instruction to restate the plan name before any price.",
        ...
    }

If ``status`` becomes ``failed``, inspect the ``error`` field for the
canonicalized failure reason and create a new proposal once the underlying
cause is addressed.

Step 3: Render the diff
-----------------------

The server returns both ``original_prompt`` and ``proposed_prompt``. Compute
the diff client-side using any standard library (``diff``, ``jsdiff``, etc.)
and present it for human review.

Step 4: Accept (apply) or Reject
--------------------------------

To apply the proposed prompt to the AI:

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/aipromptproposals/p1234567-89ab-cdef-0123-456789abcdef/accept?token=<YOUR_AUTH_TOKEN>'

Response (``200 OK``): the updated proposal record with
``status = "accepted"`` and ``applied_prompt_history_id`` populated. The
AI's ``init_prompt`` and ``current_prompt_history_id`` are updated in the
same transaction; subsequent calls will use the new prompt.

To dismiss the proposal without applying it:

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/aipromptproposals/p1234567-89ab-cdef-0123-456789abcdef/reject?token=<YOUR_AUTH_TOKEN>'

Response (``200 OK``): the updated proposal record with
``status = "rejected"``. The AI is unchanged.

Failure modes
-------------

* **400** ``audit prompt version mismatch`` — at least one selected audit
  was for an older prompt version. Re-run audits on the current prompt or
  select only matching audits.
* **409** ``prompt version drifted`` — the AI's prompt changed between
  create and accept. The proposal is marked ``expired``. Create a new
  proposal.
* **409** ``audit set invalidated`` — at least one source audit was deleted
  between create and accept. Create a new proposal with valid audits.
* **429** — too many in-flight proposals for this customer (max 3). Wait
  for one to complete or reject one that is no longer needed.

See :ref:`AI Prompt Proposal Structure <ai-struct-aipromptproposal>` for the
full field reference and :ref:`ai-overview` for the API surface.
