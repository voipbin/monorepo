.. _ai-struct-participant:

AI Call Participant
===================

.. _ai-struct-participant-participant:

Participant
-----------

.. code::

    {
        "ai_id": "<string>",
        "aicall_id": "<string>",
        "tm_create": "<string>"
    }

* ``ai_id`` (UUID): The AI configuration's unique identifier. Obtained from the ``id`` field of ``GET /ais``.
* ``aicall_id`` (UUID): The AI call session's unique identifier. Obtained from the ``id`` field of ``GET /aicalls``.
* ``tm_create`` (string, ISO 8601): Timestamp when this participant record was created.

Example
-------

.. code::

    {
        "ai_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "aicall_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "tm_create": "2024-03-01T10:00:00.000000Z"
    }
