.. _speaking-struct-speaking:

Speaking
========

Speaking
--------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "language": "<string>",
        "provider": "<string>",
        "voice_id": "<string>",
        "direction": "<string>",
        "status": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>",
    }

* ``id`` (UUID): The speaking session's unique identifier. Returned when creating a TTS session via ``POST /speakings`` or listing via ``GET /speakings``.
* ``customer_id`` (UUID): The customer who owns this speaking session. Obtained from ``GET /customer``.
* ``reference_type`` (enum string): The type of resource receiving TTS audio. See :ref:`Reference Type <speaking-struct-speaking-reference_type>`.
* ``reference_id`` (UUID): The ID of the resource receiving TTS audio. Depending on ``reference_type``, obtained from ``GET /calls`` or ``GET /conferences``.
* ``language`` (String, BCP47): The language and locale for TTS synthesis (e.g., ``en-US``, ``ko-KR``). Must match the provider's supported languages.
* ``provider`` (enum string, optional): The TTS provider used for synthesis. See :ref:`Provider <speaking-struct-speaking-provider>`. If omitted, defaults to ``elevenlabs``.
* ``voice_id`` (String, optional): A provider-specific voice identifier. If omitted, the provider's default voice for the specified language is used. Obtain available voices from the provider's documentation.
* ``direction`` (enum string): The audio routing direction. See :ref:`Direction <speaking-struct-speaking-direction>`.
* ``status`` (enum string): The speaking session's current status. See :ref:`Status <speaking-struct-speaking-status>`.
* ``tm_create`` (string, ISO 8601): Timestamp when the speaking session was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any speaking property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the speaking session was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the speaking session has not been deleted.

Example
+++++++

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "reference_type": "call",
        "reference_id": "12f8f1c9-a6c3-4f81-93db-ae445dcf188f",
        "language": "en-US",
        "provider": "elevenlabs",
        "voice_id": "",
        "direction": "both",
        "status": "active",
        "tm_create": "2025-06-15 14:30:00.123456",
        "tm_update": "2025-06-15 14:30:02.456789",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _speaking-struct-speaking-reference_type:

reference_type
--------------

All possible values for the ``reference_type`` field:

=========== ============
Type        Description
=========== ============
call        Attach TTS to a live call. The ``reference_id`` is a call ID from ``GET /calls``.
confbridge  Attach TTS to a live conference. The ``reference_id`` is a conference ID from ``GET /conferences``.
=========== ============

.. _speaking-struct-speaking-provider:

provider
--------

All possible values for the ``provider`` field:

=========== ============
Provider    Description
=========== ============
elevenlabs  ElevenLabs TTS. High-quality neural voices. Default provider if omitted.
gcp         Google Cloud Text-to-Speech. Wide language support with WaveNet and Neural2 voices.
aws         Amazon Polly. Neural and standard voices with SSML support.
=========== ============

When creating a speaking session, the ``provider`` field is optional. If omitted, VoIPBIN defaults to ``elevenlabs``.

.. _speaking-struct-speaking-status:

status
------

All possible values for the ``status`` field:

=========== ============
Status      Description
=========== ============
initiating  TTS session is being set up. Provider connection is being established. Do not call ``POST /speakings/{id}/say`` in this state.
active      TTS session is ready. Send text via ``POST /speakings/{id}/say``. Audio is being injected into the call.
stopped     TTS session has ended. Stopped via ``POST /speakings/{id}/stop`` or the call was hung up.
=========== ============

.. _speaking-struct-speaking-direction:

direction
---------

All possible values for the ``direction`` field:

=========== ============
Direction   Description
=========== ============
in          Audio injected toward the caller (remote party hears it, local party does not).
out         Audio injected toward the callee/local side (local party hears it, remote party does not).
both        Audio injected to both sides of the call. Both parties hear the synthesized speech.
=========== ============
