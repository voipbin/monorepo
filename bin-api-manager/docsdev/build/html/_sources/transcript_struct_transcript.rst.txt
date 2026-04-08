.. _transcript-struct-transcript:

Transcript
==========

.. _transcript-struct-transcript-transcript:

Transcript
----------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "transcribe_id": "<string>",
        "direction": "<string>",
        "message": "<string>",
        "tm_transcript": "<string>",
        "tm_create": "<string>"
    }

* ``id`` (UUID): The transcript's unique identifier. Returned when listing via ``GET /transcripts``.
* ``customer_id`` (UUID): The customer who owns this transcript. Obtained from the ``id`` field of ``GET /customers``.
* ``transcribe_id`` (UUID): The transcription session that produced this transcript. Obtained from the ``id`` field of ``GET /transcribes``.
* ``direction`` (enum string): The audio direction that was transcribed. See :ref:`Direction <transcript-struct-transcript-direction>`.
* ``message`` (string): The transcribed text content.
* ``tm_transcript`` (string, ISO 8601): Timestamp when the speech was captured.
* ``tm_create`` (string, ISO 8601): Timestamp when this transcript record was created.

.. _transcript-struct-transcript-direction:

Direction
---------

All possible values for the ``direction`` field:

========= ===========
Direction Description
========= ===========
in        Inbound audio (caller's speech)
out       Outbound audio (callee's speech)
both      Both directions combined
========= ===========

Example
-------

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "transcribe_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "direction": "in",
        "message": "Hello, I would like to schedule an appointment.",
        "tm_transcript": "2024-03-01T10:00:05.123456Z",
        "tm_create": "2024-03-01T10:00:05.200000Z"
    }
