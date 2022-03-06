.. _conference-struct:

Struct
======

.. _conference-struct-conference:

Conference
----------

.. code::

    {
        "id": "99accfb7-c0dd-4a54-997d-dd18af7bc280",
        "type": "conference",
        "status": "progressing",
        "name": "test conference",
        "detail": "test conference for example.",
        "data": {},
        "timeout": 0,
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
        "call_ids": [],
        "recording_id": "00000000-0000-0000-0000-000000000000",
        "recording_ids": [],
        "tm_create": "2022-02-03 06:08:56.672025",
        "tm_update": "2022-02-03 06:29:48.601658",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

* *id*: Conference's ID.
* *type*: Conference's type. See detail :ref:`here <conference-struct-type>`.
* *status*: Conference's status. See detail :ref:`here <conference-struct-status>`.
* *name*: Conference's name.
* *detail*: Conference's detail description.
* *data*: Reserved.
* *timeout*: Timeout(second).
* *pre_actions*: Set of actions for entering calls.
* *post_action*: Set of actions for leaving calls.
* *call_ids*: List of conferencing call ids.
* *recording_id*: Currently recoriding id.
* *recording_ids*: List of recording ids.

.. _conference-struct-type:

Type
----
Conference's type.

========== ==============
Type       Description
========== ==============
conference Conference.
connect    Connect.
========== ==============

.. _conference-struct-status:

Status
------
Conference's status.

=========== ==============
Type        Description
=========== ==============
starting    Conference is starting.
progressing Conference is progressing.
terminating Conference is terminating.
terminated  Conference is terminated.
=========== ==============
