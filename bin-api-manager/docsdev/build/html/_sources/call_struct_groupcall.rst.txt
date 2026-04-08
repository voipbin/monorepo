.. _call-struct-groupcall:

Struct Groupcall
================

.. _call-struct-groupcall-groupcall:

Groupcall
---------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "owner_type": "<string>",
        "owner_id": "<string>",
        "status": "<string>",
        "flow_id": "<string>",
        "source": {
            ...
        },
        "destinations": [
            {
                ...
            },
            ...
        ],
        "master_call_id": "<string>",
        "master_groupcall_id": "<string>",
        "ring_method": "<string>",
        "answer_method": "<string>",
        "answer_call_id": "<string>",
        "call_ids": [
           "<string>",
           ...
        ],
        "answer_groupcall_id": "<string>",
        "groupcall_ids": [
           "<string>",
           ...
        ],
        "call_count": <integer>,
        "groupcall_count": <integer>,
        "dial_index": <integer>,
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The groupcall's unique identifier. Returned when creating a groupcall or when listing groupcalls via ``GET /groupcalls``.
* ``customer_id`` (UUID): The customer who owns this groupcall. Obtained from the ``id`` field of ``GET /customers``.
* ``owner_type`` (enum string): The type of owner for this groupcall. Possible values: ``agent`` (owned by a specific agent), or empty string (no specific owner).
* ``owner_id`` (UUID): The ID of the owner. When ``owner_type`` is ``agent``, this is an agent UUID from ``GET /agents``. Set to ``00000000-0000-0000-0000-000000000000`` if no owner.
* ``status`` (enum string): The groupcall's current status. See :ref:`Status <call-struct-groupcall-status>`.
* ``flow_id`` (UUID): The flow associated with this groupcall. Obtained from the ``id`` field of ``GET /flows``. Set to ``00000000-0000-0000-0000-000000000000`` if no flow is assigned.
* ``source`` (Object): Source address info. See :ref:`Address <common-struct-address-address>`.
* ``destinations`` (Array of Object): List of destination addresses to ring. Each entry follows the :ref:`Address <common-struct-address-address>` structure. A destination can also be another groupcall for nested groupcalls.
* ``master_call_id`` (UUID): The ID of the master call that initiated this groupcall. Obtained from ``GET /calls``. Set to ``00000000-0000-0000-0000-000000000000`` if none.
* ``master_groupcall_id`` (UUID): The ID of the parent groupcall if this is a nested groupcall. Obtained from ``GET /groupcalls``. Set to ``00000000-0000-0000-0000-000000000000`` if this is a top-level groupcall.
* ``ring_method`` (enum string): How destinations are rung. See :ref:`Ring method <call-struct-groupcall-ring_method>`.
* ``answer_method`` (enum string): What happens when a destination answers. See :ref:`Answer method <call-struct-groupcall-answer_method>`.
* ``answer_call_id`` (UUID): The call ID of the destination that answered. Set to ``00000000-0000-0000-0000-000000000000`` until a destination answers. Obtained from ``GET /calls``.
* ``call_ids`` (Array of UUID): List of call IDs created for each destination. Each ID can be used with ``GET /calls/{id}`` to check individual call status.
* ``answer_groupcall_id`` (UUID): The ID of the nested groupcall that answered, if applicable. Set to ``00000000-0000-0000-0000-000000000000`` until a nested groupcall answers. Obtained from ``GET /groupcalls``.
* ``groupcall_ids`` (Array of UUID): List of nested groupcall IDs created for this groupcall. Each ID can be used with ``GET /groupcalls/{id}`` to check status.
* ``call_count`` (Integer): The number of remaining calls for the current dial attempt.
* ``groupcall_count`` (Integer): The number of remaining nested groupcalls for the current dial attempt.
* ``dial_index`` (Integer): The current dial index. Only meaningful when ``ring_method`` is ``ring_all``.
* ``tm_create`` (string, ISO 8601): Timestamp when the groupcall was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any groupcall property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the groupcall was deleted.

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the groupcall has not been deleted.

.. note:: **AI Implementation Hint**

   Groupcalls involve active calls to real destinations, which are chargeable. Each destination in the ``destinations`` array results in an individual call being created. With ``ring_all``, all calls are placed simultaneously. Monitor ``call_ids`` to track the status of each individual call.

Example
+++++++

.. code::

    {
        "id": "d8596b14-4d8e-4a86-afde-642b46d59ac7",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "owner_type": "agent",
        "owner_id": "62005165-7592-4ff7-9076-55bf491023f2",
        "status": "progressing",
        "flow_id": "00000000-0000-0000-0000-000000000000",
        "source": {
            "type": "tel",
            "target": "+15551234567",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "destinations": [
            {
                "type": "extension",
                "target": "test11@test",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            {
                "type": "extension",
                "target": "test12@test",
                "target_name": "",
                "name": "",
                "detail": ""
            }
        ],
        "master_call_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "master_groupcall_id": "00000000-0000-0000-0000-000000000000",
        "ring_method": "ring_all",
        "answer_method": "hangup_others",
        "answer_call_id": "00000000-0000-0000-0000-000000000000",
        "call_ids": [
            "3c77eb43-2098-4890-bb6c-5af0707ba4a6"
        ],
        "answer_groupcall_id": "00000000-0000-0000-0000-000000000000",
        "groupcall_ids": [],
        "call_count": 2,
        "groupcall_count": 0,
        "dial_index": 0,
        "tm_create": "2023-04-21 15:33:28.569053",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _call-struct-groupcall-status:

Status
------

All possible values for the ``status`` field:

============ ===========
Status       Description
============ ===========
progressing  The groupcall is actively dialing destinations and waiting for answers.
hangingup    The groupcall is in the process of hanging up all calls.
hangup       The groupcall has ended. All calls have been terminated. Final state.
============ ===========

.. _call-struct-groupcall-ring_method:

Ring method
-----------
Groupcall's ringing method (enum string).

=========== ============
Type        Description
=========== ============
ring_all    Make a call to all destinations at once. The first destination to answer wins; all other calls are cancelled.
linear      Make a call to each destination one-by-one in order. If a destination does not answer, the next one is tried.
=========== ============

.. _call-struct-groupcall-answer_method:

Answer method
-------------
What happens when a destination answers (enum string).

============= ===================
Type          Description
============= ===================
hangup_others Hang up all other unanswered calls when one destination answers.
============= ===================
