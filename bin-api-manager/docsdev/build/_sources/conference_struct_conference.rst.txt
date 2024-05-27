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

* id: Conference's ID.
* customer_id: Customer's ID.
* *type*: Conference's type. See detail :ref:`here <conference-struct-conference-type>`.
* *status*: Conference's status. See detail :ref:`here <conference-struct-conference-status>`.
* name: Conference's name.
* detail: Conference's detail description.
* data: Reserved.
* timeout: Timeout(second).
* pre_actions: Set of actions for entering calls.
* post_action: Set of actions for leaving calls.
* conferencecall_ids: List of conferencecall ids.
* recording_id: Currently recoriding id.
* recording_ids: List of recording ids.


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
                    "gender": "female",
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
                    "gender": "female",
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
Conference's type.

========== ==============
Type       Description
========== ==============
conference Conference.
connect    Connect.
========== ==============

.. _conference-struct-conference-status:

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
