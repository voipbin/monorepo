.. _recording-struct-recording:

Recording
=========

.. _recording-struct-recording-recording:

Recording
---------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "owner_type": "<string>",
        "owner_id": "<string>",
        "activeflow_id": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "status": "<string>",
        "format": "<string>",
        "on_end_flow_id": "<string>",
        "tm_start": "<string>",
        "tm_end": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The recording's unique identifier. Returned when creating a recording via ``POST /calls/{id}/recording_start`` or listing via ``GET /recordings``.
* ``customer_id`` (UUID): The customer who owns this recording. Obtained from the ``id`` field of ``GET /customers``.
* ``owner_type`` (enum string): The type of owner for this recording. Possible values: ``agent`` (owned by a specific agent), or empty string (no specific owner).
* ``owner_id`` (UUID): The ID of the owner. When ``owner_type`` is ``agent``, this is an agent UUID from ``GET /agents``. Set to ``00000000-0000-0000-0000-000000000000`` if no owner.
* ``activeflow_id`` (UUID): The ID of the active flow associated with this recording. Obtained from the ``id`` field of ``GET /activeflows``. Set to ``00000000-0000-0000-0000-000000000000`` if no active flow.
* ``reference_type`` (enum string): The type of resource being recorded. See :ref:`Reference Type <recording-struct-reference-type>`.
* ``reference_id`` (UUID): The ID of the call or conference being recorded. Obtained from ``GET /calls`` or ``GET /conferences``.
* ``status`` (enum string): The recording's current status. See :ref:`Status <recording-struct-recording-status>`.
* ``format`` (enum string): The recording file format. See :ref:`Format <recording-struct-recording-format>`.
* ``on_end_flow_id`` (UUID): The flow to execute when the recording ends. Obtained from the ``id`` field of ``GET /flows``. Set to ``00000000-0000-0000-0000-000000000000`` if no flow is assigned.
* ``tm_start`` (string, ISO 8601): Timestamp when the recording started capturing audio.
* ``tm_end`` (string, ISO 8601): Timestamp when the recording stopped capturing audio.
* ``tm_create`` (string, ISO 8601): Timestamp when the recording resource was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any recording property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the recording was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the recording has not been deleted. An empty ``tm_delete`` (``""``) in older recordings also indicates not deleted.

.. _recording-struct-reference-type:

Reference Type
--------------

All possible values for the ``reference_type`` field:

========== ===========
Type       Description
========== ===========
call       Recording of a single call. Audio from both parties is mixed into one channel. The ``reference_id`` corresponds to a call ID from ``GET /calls``.
confbridge Recording of a conference bridge. Audio from all participants is mixed together. The ``reference_id`` corresponds to a conference ID from ``GET /conferences``.
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
stopping   Recording stop has been requested. Audio capture is being finalized.
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
