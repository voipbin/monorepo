.. _recording-struct-recording:

Recording
=========

.. _recording-struct-recording-recording:

Recording
---------

.. code::

    {
        "id": "<string>",
        "type": "<string>",
        "reference_id": "<string>",
        "status": "<string>",
        "format": "<string>",
        "tm_start": "<string>",
        "tm_end": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The recording's unique identifier. Returned when creating a recording via ``POST /calls/{id}/recording_start`` or listing via ``GET /recordings``.
* ``type`` (enum string): The recording's type. See :ref:`Type <recording-struct-type>`.
* ``reference_id`` (UUID): The ID of the call or conference being recorded. Obtained from ``GET /calls`` or ``GET /conferences``.
* ``status`` (enum string): The recording's current status. See :ref:`Status <recording-struct-recording-status>`.
* ``format`` (enum string): The recording file format. See :ref:`Format <recording-struct-recording-format>`.
* ``tm_start`` (string, ISO 8601): Timestamp when the recording started capturing audio.
* ``tm_end`` (string, ISO 8601): Timestamp when the recording stopped capturing audio.
* ``tm_create`` (string, ISO 8601): Timestamp when the recording resource was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any recording property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the recording was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the recording has not been deleted. An empty ``tm_delete`` (``""``) in older recordings also indicates not deleted.

.. _recording-struct-type:

Type
----

All possible values for the ``type`` field:

========== ===========
Type       Description
========== ===========
call       Recording of a single call. Audio from both parties is mixed into one channel. The ``reference_id`` corresponds to a call ID from ``GET /calls``.
conference Recording of a conference. Audio from all participants is mixed together. The ``reference_id`` corresponds to a conference ID from ``GET /conferences``.
========== ===========

.. _recording-struct-recording-status:

Status
------

All possible values for the ``status`` field:

========== ===========
Status     Description
========== ===========
initiating Preparing the recording. Audio capture is being set up. The recording file is not yet available.
recording  Actively capturing audio. The file is being written to storage in real-time.
ended      Recording is complete. The file is ready for download via ``GET /recordingfiles/{id}``.
========== ===========

.. _recording-struct-recording-format:

Format
------

All possible values for the ``format`` field:

========== ===========
Format     Description
========== ===========
wav        WAV format (PCM). 8 kHz sample rate, 16-bit, mono. Approximately 1 MB per minute of audio.
========== ===========
