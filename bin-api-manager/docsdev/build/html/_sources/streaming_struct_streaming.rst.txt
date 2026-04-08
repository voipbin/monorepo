.. _streaming-struct-streaming:

Streaming
=========

.. _streaming-struct-streaming-streaming:

Streaming
---------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "streaming_id": "<string>",
        "transcribe_id": "<string>",
        "direction": "<string>",
        "message": "<string>",
        "tm_event": "<string>",
        "tm_create": "<string>"
    }

* ``id`` (UUID): The streaming event's unique identifier.
* ``customer_id`` (UUID): The customer who owns this streaming event. Obtained from the ``id`` field of ``GET /customers``.
* ``streaming_id`` (UUID): The streaming session that produced this event.
* ``transcribe_id`` (UUID): The transcription session associated with this streaming event. Obtained from the ``id`` field of ``GET /transcribes``.
* ``direction`` (enum string): The audio direction being streamed. Possible values: ``in`` (inbound audio), ``out`` (outbound audio), ``both`` (both directions).
* ``message`` (string): The real-time transcribed text content of this streaming event. May be empty for intermediate events.
* ``tm_event`` (string, ISO 8601): Timestamp when this streaming speech event occurred.
* ``tm_create`` (string, ISO 8601): Timestamp when this streaming record was created.

Example
-------

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "streaming_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
        "transcribe_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "direction": "in",
        "message": "I need help with my order",
        "tm_event": "2024-03-01T10:00:03.500000Z",
        "tm_create": "2024-03-01T10:00:03.600000Z"
    }
