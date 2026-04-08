.. _ai-struct-summary:

AI Summary
==========

.. _ai-struct-summary-summary:

Summary
-------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "activeflow_id": "<string>",
        "on_end_flow_id": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "status": "<string>",
        "language": "<string>",
        "content": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The summary's unique identifier. Returned when creating via ``POST /summaries`` or listing via ``GET /summaries``.
* ``customer_id`` (UUID): The customer who owns this summary. Obtained from the ``id`` field of ``GET /customers``.
* ``activeflow_id`` (UUID): The ID of the active flow associated with this summary. Obtained from the ``id`` field of ``GET /activeflows``. Set to ``00000000-0000-0000-0000-000000000000`` if no active flow.
* ``on_end_flow_id`` (UUID): The flow to execute when the summary generation completes. Obtained from the ``id`` field of ``GET /flows``. Set to ``00000000-0000-0000-0000-000000000000`` if no flow is assigned.
* ``reference_type`` (enum string): The type of resource being summarized. See :ref:`Reference Type <ai-struct-summary-reference-type>`.
* ``reference_id`` (UUID): The ID of the resource being summarized (e.g., a call ID, conference ID, or transcribe ID).
* ``status`` (enum string): The summary's current processing status. See :ref:`Status <ai-struct-summary-status>`.
* ``language`` (string): The BCP47 language code for the summary output (e.g., ``en-US``, ``ko-KR``).
* ``content`` (string): The generated summary text. Empty while status is ``progressing``.
* ``tm_create`` (string, ISO 8601): Timestamp when this summary was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to this summary.
* ``tm_delete`` (string, ISO 8601): Timestamp when this summary was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Summary generation is asynchronous. After creating a summary via ``POST /summaries``, poll ``GET /summaries/{id}`` until the ``status`` changes from ``progressing`` to ``done``. The ``content`` field will be empty until processing completes.

.. _ai-struct-summary-reference-type:

Reference Type
--------------

All possible values for the ``reference_type`` field:

============ ===========
Type         Description
============ ===========
call         Summarize a phone call conversation
conference   Summarize a conference session
transcribe   Summarize a transcription session
recording    Summarize a recording
============ ===========

.. _ai-struct-summary-status:

Status
------

All possible values for the ``status`` field:

============= ===========
Status        Description
============= ===========
progressing   The summary is being generated
done          The summary has been completed and content is available
============= ===========

Example
-------

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "activeflow_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "on_end_flow_id": "00000000-0000-0000-0000-000000000000",
        "reference_type": "call",
        "reference_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
        "status": "done",
        "language": "en-US",
        "content": "The customer called to inquire about their account balance and requested a callback from the billing department.",
        "tm_create": "2024-03-01T10:05:00.000000Z",
        "tm_update": "2024-03-01T10:05:30.000000Z",
        "tm_delete": "9999-01-01T00:00:00.000000Z"
    }
