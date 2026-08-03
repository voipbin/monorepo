.. _conference-struct-conference:

Conference
==========

.. _conference-struct-conference-conference:

Conference
----------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "type": "<string>",
        "status": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "data": {},
        "timeout": <integer>,
        "pre_flow_id": "<string>",
        "post_flow_id": "<string>",
        "conferencecall_ids": [
            ...
        ],
        "recording_id": "<string>",
        "recording_ids": [
            ...
        ],
        "transcribe_id": "<string>",
        "transcribe_ids": [
            ...
        ],
        "direct_hash": "<string>",
        "tm_end": "<string or null>",
        "tm_create": "<string>",
        "tm_update": "<string or null>",
        "tm_delete": "<string or null>"
    }

* ``id`` (UUID): The conference's unique identifier. Returned when creating via ``POST /conferences`` or listing via ``GET /conferences``.
* ``customer_id`` (UUID): The customer who owns this conference. Obtained from ``GET /customers``.
* ``type`` (enum string): The conference's type. Immutable after creation. See :ref:`Type <conference-struct-conference-type>`.
* ``status`` (enum string): The conference's current status. See :ref:`Status <conference-struct-conference-status>`.
* ``name`` (String): Human-readable name for the conference.
* ``detail`` (String): Detailed description of the conference.
* ``data`` (Object): Reserved for future use.
* ``timeout`` (Integer): Conference auto-termination timeout in seconds. Set to ``0`` for no timeout. If a value between ``1`` and ``59`` is given, it is silently replaced with the default timeout of ``86400`` seconds (24 hours) instead of being honored as-is; use ``0`` or a value of ``60`` or greater to get the timeout you expect.
* ``pre_flow_id`` (UUID): The flow to execute before the conference starts (e.g., greeting message). Obtained from the ``id`` field of ``GET /flows``. Set to ``00000000-0000-0000-0000-000000000000`` if no pre-conference flow is assigned.
* ``post_flow_id`` (UUID): The flow to execute after the conference ends. Obtained from the ``id`` field of ``GET /flows``. Set to ``00000000-0000-0000-0000-000000000000`` if no post-conference flow is assigned.
* ``conferencecall_ids`` (Array of UUID): List of participant IDs currently in the conference. Each ID can be used with ``GET /conferencecalls/{id}`` to retrieve participant details.
* ``recording_id`` (UUID): The currently active recording's ID. Obtained from ``GET /recordings``. Set to ``00000000-0000-0000-0000-000000000000`` if no recording is active.
* ``recording_ids`` (Array of UUID): List of all recording IDs created during this conference's lifetime. Each ID can be used with ``GET /recordings/{id}`` to retrieve the recording.
* ``transcribe_id`` (UUID): The currently active transcription's ID. Obtained from ``GET /transcribes``. Set to ``00000000-0000-0000-0000-000000000000`` if no transcription is active.
* ``transcribe_ids`` (Array of UUID): List of all transcription IDs created during this conference's lifetime. Each ID can be used with ``GET /transcribes/{id}`` to retrieve the transcription.
* ``direct_hash`` (String): Hash for direct conference access. Empty string when direct access is disabled. When enabled, this hash forms the direct SIP URI: ``sip:direct.<hash>@sip.voipbin.net``. Regenerate via ``POST /conferences/{id}/direct-hash-regenerate``.
* ``tm_end`` (String, ISO 8601, nullable): Timestamp when the conference ended. ``null`` if the conference is still active.
* ``tm_create`` (String, ISO 8601): Timestamp when the conference was created.
* ``tm_update`` (String, ISO 8601, nullable): Timestamp of the last update to any conference property. ``null`` if never updated.
* ``tm_delete`` (String, ISO 8601, nullable): Timestamp when the conference was deleted. ``null`` if not deleted.

.. note:: **AI Implementation Hint**

   ``tm_update``/``tm_delete``/``tm_end`` are ``null`` when the corresponding event has not yet occurred, not a sentinel timestamp.


Example
+++++++

.. code::

    {
        "id": "99accfb7-c0dd-4a54-997d-dd18af7bc280",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "type": "conference",
        "status": "progressing",
        "name": "test conference",
        "detail": "test conference for example.",
        "data": {},
        "timeout": 0,
        "pre_flow_id": "00000000-0000-0000-0000-000000000000",
        "post_flow_id": "00000000-0000-0000-0000-000000000000",
        "conferencecall_ids": [],
        "recording_id": "00000000-0000-0000-0000-000000000000",
        "recording_ids": [],
        "transcribe_id": "00000000-0000-0000-0000-000000000000",
        "transcribe_ids": [],
        "direct_hash": "",
        "tm_end": null,
        "tm_create": "2022-02-03 06:08:56.672025",
        "tm_update": "2022-08-06 19:11:13.040418",
        "tm_delete": null
    }

.. _conference-struct-conference-type:

Type
----
Conference's type (enum string). Immutable after creation.

========== ==============
Type       Description
========== ==============
conference Multi-party conference room. Supports 2+ participants. Remains active even with 0 or 1 participant. Only terminates when explicitly deleted or timeout expires.
connect    Two-party bridge. Designed for exactly 2 participants (e.g., customer-agent). Auto-ejects the remaining participant when one leaves, then terminates.
queue      Conference room backed by a queue. Created and managed internally by the queue system (see :ref:`Queue Overview <queue-overview>`) rather than created directly by API users.
========== ==============

.. _conference-struct-conference-status:

Status
------
Conference's current status (enum string). States only move forward, never backward.

=========== ==============
Status      Description
=========== ==============
starting    Conference is being initialized. Brief transitional state. No operations (recording, transcription) are possible yet.
progressing Conference is active. Participants can join, recording and transcription can start. This is the main operational state.
terminating Conference is closing. No new participants can join. Waiting for existing participants to leave.
terminated  Conference is completely closed. No further operations are possible. This is the final state.
=========== ==============
