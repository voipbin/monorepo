.. _ai-struct-aiaudit:

AI Audit
========

.. _ai-struct-aiaudit-aiaudit:

AIAudit
-------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "aicall_id": "<string>",
        "ai_id": "<string>",
        "prompt_history_id": "<string>",
        "status": "<string>",
        "overall_score": <integer or null>,
        "evaluation": <object or null>,
        "language": "<string>",
        "error": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The audit record's unique identifier.
* ``customer_id`` (UUID): The customer who owns this audit. Obtained from the ``id`` field of ``GET /customers``.
* ``aicall_id`` (UUID): The AI call session this audit evaluates. Obtained from the ``id`` field of ``GET /aicalls``.
* ``ai_id`` (UUID): The AI configuration used during the call. Obtained from the ``id`` field of ``GET /ais``.
* ``prompt_history_id`` (UUID): The prompt snapshot version evaluated by this audit. Obtained from the ``id`` field of ``GET /ai-prompt-histories``. Set to ``00000000-0000-0000-0000-000000000000`` if no prompt history is associated.
* ``status`` (enum string): The audit's current processing status. See :ref:`Status <ai-struct-aiaudit-status>`.
* ``overall_score`` (integer, nullable): A numeric score summarising the evaluation, typically 1–5. Null while status is ``progressing``.
* ``evaluation`` (object, nullable): Structured evaluation detail produced by the evaluator. The exact schema depends on the configured evaluator. Null while status is ``progressing``.
* ``language`` (string): The BCP47 language code used for the evaluation (e.g., ``en-US``, ``ko-KR``).
* ``error`` (enum string): Machine-readable error code set when status is ``failed``. Empty otherwise. See :ref:`Error <ai-struct-aiaudit-error>`.
* ``tm_create`` (string, ISO 8601): Timestamp when this audit record was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to this audit.
* ``tm_delete`` (string, ISO 8601): Timestamp when this audit was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **Audit Processing**

   Audit evaluation is asynchronous. After triggering an audit, poll the record until ``status`` changes from ``progressing`` to ``completed`` or ``failed``. The ``overall_score`` and ``evaluation`` fields will be populated only when status is ``completed``.

.. _ai-struct-aiaudit-status:

Status
------

All possible values for the ``status`` field:

============= ===========
Status        Description
============= ===========
progressing   The audit evaluation is currently running
completed     The audit has finished successfully; scores are available
failed        The audit could not be completed; see ``error`` for the reason
============= ===========

.. _ai-struct-aiaudit-error:

Error
-----

All possible values for the ``error`` field (non-empty only when ``status`` is ``failed``):

====================================== ===========
Error                                  Description
====================================== ===========
invalid_call_metadata                  The call metadata required for evaluation was missing or malformed
prompt_snapshot_not_found              The prompt snapshot referenced by ``prompt_history_id`` could not be found
prompt_snapshot_has_no_history_id      The associated AI configuration has no recorded prompt history
invalid_evaluator_response             The evaluator returned an unrecognisable response
evaluator_unavailable                  The external evaluator service was unreachable
cancelled                              The audit was cancelled before completion
====================================== ===========

Example
-------

.. code::

    {
        "id": "d4e5f6a7-b8c9-0123-defa-b12345678901",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "aicall_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "ai_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "prompt_history_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
        "status": "completed",
        "overall_score": 4,
        "evaluation": {
            "tone": 5,
            "accuracy": 4,
            "resolution": 3
        },
        "language": "en-US",
        "error": "",
        "tm_create": "2024-03-01T10:05:00.000000Z",
        "tm_update": "2024-03-01T10:05:45.000000Z",
        "tm_delete": "9999-01-01T00:00:00.000000Z"
    }
