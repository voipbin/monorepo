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
        "source": {
            ...
        },
        "destinations": [
            {
                ...
            },
            ...
        ],
        "ring_method": "<string>",
        "answer_method": "<string>",
        "answer_call_id": "<string>",
        "call_ids": [
           "<string>",
           ...
        ],
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The groupcall's unique identifier. Returned when creating a groupcall or when listing groupcalls via ``GET /groupcalls``.
* ``customer_id`` (UUID): The customer who owns this groupcall. Obtained from the ``id`` field of ``GET /customers``.
* ``source`` (Object): Source address info. See :ref:`Address <common-struct-address-address>`.
* ``destinations`` (Array of Object): List of destination addresses to ring. Each entry follows the :ref:`Address <common-struct-address-address>` structure. A destination can also be another groupcall for nested groupcalls.
* ``ring_method`` (enum string): How destinations are rung. See :ref:`Ring method <call-struct-groupcall-ring_method>`.
* ``answer_method`` (enum string): What happens when a destination answers. See :ref:`Answer method <call-struct-groupcall-answer_method>`.
* ``answer_call_id`` (UUID): The call ID of the destination that answered. Set to ``00000000-0000-0000-0000-000000000000`` until a destination answers. Obtained from ``GET /calls``.
* ``call_ids`` (Array of UUID): List of call IDs created for each destination. Each ID can be used with ``GET /calls/{id}`` to check individual call status.
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
        "source": {
            "type": "tel",
            "target": "+15551234567",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "destinations": [
            {
                "type": "endpoint",
                "target": "test11@test",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            {
                "type": "endpoint",
                "target": "test12@test",
                "target_name": "",
                "name": "",
                "detail": ""
            }
        ],
        "ring_method": "",
        "answer_method": "",
        "answer_call_id": "00000000-0000-0000-0000-000000000000",
        "call_ids": [
            "3c77eb43-2098-4890-bb6c-5af0707ba4a6"
        ],
        "tm_create": "2023-04-21 15:33:28.569053",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

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
