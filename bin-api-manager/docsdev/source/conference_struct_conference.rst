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
        "pre_actions": [
            {
                ...
            }
        ],
        "post_actions": [
            {
                ...
            }
        ],
        "conferencecall_ids": [
            ...
        ],
        "recording_id": "<string>",
        "recording_ids": [
            ...
        ],
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The conference's unique identifier. Returned when creating via ``POST /conferences`` or listing via ``GET /conferences``.
* ``customer_id`` (UUID): The customer who owns this conference. Obtained from ``GET /customers``.
* ``type`` (enum string): The conference's type. Immutable after creation. See :ref:`Type <conference-struct-conference-type>`.
* ``status`` (enum string): The conference's current status. See :ref:`Status <conference-struct-conference-status>`.
* ``name`` (String): Human-readable name for the conference.
* ``detail`` (String): Detailed description of the conference.
* ``data`` (Object): Reserved for future use.
* ``timeout`` (Integer): Conference auto-termination timeout in seconds. Set to ``0`` for no timeout.
* ``pre_actions`` (Array of Object): Flow actions executed when a participant joins (e.g., greeting message). Each element follows the :ref:`Action <flow-struct-action>` structure.
* ``post_actions`` (Array of Object): Flow actions executed when a participant leaves. Each element follows the :ref:`Action <flow-struct-action>` structure.
* ``conferencecall_ids`` (Array of UUID): List of participant IDs currently in the conference. Each ID can be used with ``GET /conferencecalls/{id}`` to retrieve participant details.
* ``recording_id`` (UUID): The currently active recording's ID. Obtained from ``GET /recordings``. Set to ``00000000-0000-0000-0000-000000000000`` if no recording is active.
* ``recording_ids`` (Array of UUID): List of all recording IDs created during this conference's lifetime. Each ID can be used with ``GET /recordings/{id}`` to retrieve the recording.
* ``tm_create`` (String, ISO 8601): Timestamp when the conference was created.
* ``tm_update`` (String, ISO 8601): Timestamp of the last update to any conference property.
* ``tm_delete`` (String, ISO 8601): Timestamp when the conference was deleted.

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the conference is still active.


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
        "pre_actions": [
            {
                "id": "00000000-0000-0000-0000-000000000000",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "talk",
                "option": {
                    "text": "Hello. Welcome to the test conference.",
                    "language": "en-US"
                }
            }
        ],
        "post_actions": [
            {
                "id": "00000000-0000-0000-0000-000000000000",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "talk",
                "option": {
                    "text": "The conference has closed. Thank you. Good bye.",
                    "language": "en-US"
                }
            }
        ],
        "conferencecall_ids": [],
        "recording_id": "00000000-0000-0000-0000-000000000000",
        "recording_ids": [],
        "tm_create": "2022-02-03 06:08:56.672025",
        "tm_update": "2022-08-06 19:11:13.040418",
        "tm_delete": "9999-01-01 00:00:00.000000"
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
