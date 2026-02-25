.. _transcribe-struct-transcribe:

Transcribe
==========

Transcribe
----------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "status": "<string>",
        "language": "<string>",
        "provider": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>",
    }

* ``id`` (UUID): The transcribe session's unique identifier. Returned when creating a transcription via ``POST /transcribes`` or listing via ``GET /transcribes``.
* ``customer_id`` (UUID): The customer who owns this transcription. Obtained from ``GET /customers``.
* ``reference_type`` (enum string): The type of resource being transcribed. See :ref:`Reference Type <transcribe-struct-transcribe-reference_type>`.
* ``reference_id`` (UUID): The ID of the resource being transcribed. Depending on ``reference_type``, obtained from ``GET /calls``, ``GET /recordings``, or ``GET /conferences``.
* ``status`` (enum string): The transcription session's current status. See :ref:`Status <transcribe-struct-transcribe-status>`.
* ``language`` (String, BCP47): The language code for transcription (e.g., ``en-US``, ``ko-KR``, ``ja-JP``). See :ref:`Supported Languages <transcribe-overview-supported_languages>`.
* ``provider`` (enum string, optional): The STT provider used for this transcription. See :ref:`Provider <transcribe-struct-transcribe-provider>`.
* ``tm_create`` (string, ISO 8601): Timestamp when the transcribe session was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any transcribe property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the transcribe session was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the transcription has not been deleted.

Example
+++++++

.. code::

    {
        "id": "bbf08426-3979-41bc-a544-5fc92c237848",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "reference_type": "call",
        "reference_id": "12f8f1c9-a6c3-4f81-93db-ae445dcf188f",
        "status": "done",
        "language": "en-US",
        "provider": "gcp",
        "tm_create": "2024-04-01 07:17:04.091019",
        "tm_update": "2024-04-01 13:25:32.428602",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _transcribe-struct-transcribe-reference_type:

reference_type
--------------

All possible values for the ``reference_type`` field:

=========== ============
Type        Description
=========== ============
call        Transcribing a live call in real-time. The ``reference_id`` is a call ID from ``GET /calls``.
recording   Transcribing a previously recorded audio file. The ``reference_id`` is a recording ID from ``GET /recordings``.
confbridge  Transcribing a live conference. The ``reference_id`` is a conference ID from ``GET /conferences``.
=========== ============

.. _transcribe-struct-transcribe-provider:

provider
--------------

All possible values for the ``provider`` field:

=========== ============
Provider    Description
=========== ============
gcp         Google Cloud Speech-to-Text
aws         Amazon Transcribe
=========== ============

When creating a transcription, the ``provider`` field is optional. If omitted, VoIPBIN selects the best available provider automatically (default order: GCP, then AWS). If a specific provider is requested but unavailable, the system falls back to the default order.

.. _transcribe-struct-transcribe-status:

status
--------------

All possible values for the ``status`` field:

=========== ============
Status      Description
=========== ============
progressing Transcription is actively in progress. New transcript segments are being generated and delivered via webhook or WebSocket.
done        Transcription is complete. No more transcript segments will be generated. All transcripts are available via ``GET /transcripts?transcribe_id={id}``.
=========== ============

.. _transcribe-struct-transcription:

Transcription
=============

Transcription
-------------

.. code::

    {
        "id": "<string>",
        "transcribe_id": "<string>",
        "direction": "<string>",
        "message": "<string>",
        "tm_transcript": "<string>",
        "tm_create": "<string>",
    },

* ``id`` (UUID): The individual transcript segment's unique identifier.
* ``transcribe_id`` (UUID): The parent transcribe session's ID. Obtained from ``GET /transcribes`` or the response of ``POST /transcribes``.
* ``direction`` (enum string): Whether the speech was incoming or outgoing. See :ref:`Direction <transcribe-struct-transcription-direction>`.
* ``message`` (String): The transcribed text content of this speech segment.
* ``tm_transcript`` (String): Time offset within the call when this speech occurred. Uses ``0001-01-01 00:00:00`` as epoch; the time portion represents the offset from the start of the transcription session (e.g., ``0001-01-01 00:01:04.441160`` means 1 minute and 4 seconds into the call). Sort by this field to reconstruct conversation order.
* ``tm_create`` (string, ISO 8601): Absolute timestamp when this transcript segment was created.

.. note:: **AI Implementation Hint**

   The ``tm_transcript`` field is a time offset, not an absolute timestamp. Its date part (``0001-01-01``) is a sentinel value meaning "relative to the start of the transcription session." To reconstruct a conversation in order, sort all transcript segments by ``tm_transcript``, not by ``tm_create`` (which reflects delivery time, not speech time).

Example
+++++++

.. code::

    {
        "id": "06af78f0-b063-48c0-b22d-d31a5af0aa88",
        "transcribe_id": "bbf08426-3979-41bc-a544-5fc92c237848",
        "direction": "in",
        "message": "Hi, good to see you. How are you today.",
        "tm_transcript": "0001-01-01 00:05:04.441160",
        "tm_create": "2024-04-01 07:22:07.229309"
    }

.. _transcribe-struct-transcription-direction:

direction
---------

All possible values for the ``direction`` field:

=========== ============
Direction   Description
=========== ============
in          Incoming speech toward VoIPBIN (i.e., what the caller/remote party said).
out         Outgoing speech from VoIPBIN (i.e., TTS audio, recorded prompts, or the connected party's speech sent from VoIPBIN).
=========== ============

.. _transcribe-struct-speech-webhook:

Speech Webhook Message
======================

Speech Webhook Message
----------------------
The speech webhook message is the payload delivered for ``transcribe_speech_started``, ``transcribe_speech_interim``, and ``transcribe_speech_ended`` events. These events are generated during a real-time streaming transcription session when voice activity is detected.

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

* ``id`` (UUID): The unique identifier of the speech event.
* ``customer_id`` (UUID): The customer who owns this transcription session. Obtained from ``GET /customers``.
* ``streaming_id`` (UUID): The unique identifier of the audio streaming session that produced this event.
* ``transcribe_id`` (UUID): The parent transcribe session's ID. Obtained from ``GET /transcribes`` or the response of ``POST /transcribes``.
* ``direction`` (enum string): Whether the speech was incoming or outgoing. See :ref:`Direction <transcribe-struct-transcription-direction>`.
* ``message`` (String): The interim transcribed text. Present for ``transcribe_speech_interim`` events. Empty for ``transcribe_speech_started`` and ``transcribe_speech_ended`` events.
* ``tm_event`` (string, ISO 8601): Timestamp when the speech event occurred.
* ``tm_create`` (string, ISO 8601): Timestamp when the speech event record was created.

Example
+++++++

**transcribe_speech_started:**

.. code::

    {
        "type": "transcribe_speech_started",
        "data": {
            "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "streaming_id": "c0d1e2f3-a4b5-6c7d-8e9f-0a1b2c3d4e5f",
            "transcribe_id": "bbf08426-3979-41bc-a544-5fc92c237848",
            "direction": "in",
            "tm_event": "2024-04-01 07:22:07.229309",
            "tm_create": "2024-04-01 07:22:07.229309"
        }
    }

**transcribe_speech_interim:**

.. code::

    {
        "type": "transcribe_speech_interim",
        "data": {
            "id": "b2c3d4e5-f6a7-8901-bcde-f23456789012",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "streaming_id": "c0d1e2f3-a4b5-6c7d-8e9f-0a1b2c3d4e5f",
            "transcribe_id": "bbf08426-3979-41bc-a544-5fc92c237848",
            "direction": "in",
            "message": "Hello, I need help with my account",
            "tm_event": "2024-04-01 07:22:08.115000",
            "tm_create": "2024-04-01 07:22:08.115000"
        }
    }

**transcribe_speech_ended:**

.. code::

    {
        "type": "transcribe_speech_ended",
        "data": {
            "id": "c3d4e5f6-a7b8-9012-cdef-345678901234",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "streaming_id": "c0d1e2f3-a4b5-6c7d-8e9f-0a1b2c3d4e5f",
            "transcribe_id": "bbf08426-3979-41bc-a544-5fc92c237848",
            "direction": "in",
            "tm_event": "2024-04-01 07:22:12.500000",
            "tm_create": "2024-04-01 07:22:12.500000"
        }
    }
