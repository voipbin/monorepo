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

* id: Groupcall's ID.
* customer_id: Customer's ID
* *source*: Source address info. See detail :ref:`here <common-struct-address-address>`.
* *destinations*: List of destination addresses info. See detail :ref:`here <common-struct-address-address>`.
* *ring_method*: Ring method. See detail :ref:`here <call-struct-groupcall-ring_method>`
* *answer_method*: Answering method. See detail :ref:`here <call-struct-groupcall-answer_method>`
* answer_call_id: Represents answered call id.
* call_ids: List of created call ids.

Example
+++++++

.. code::

    {
        "id": "d8596b14-4d8e-4a86-afde-642b46d59ac7",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "source": {
            "type": "tel",
            "target": "+821028286521",
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
Groupcall's ringing method.

=========== ============
Type        Description
=========== ============
ring_all    Make a call to the all destinations at once.
linear      Make a call to the destination one-by-one in a linear.
=========== ============

.. _call-struct-groupcall-answer_method:

Answer method
-------------
Call's status.

============= ===================
Type          Description
============= ===================
hangup_others Hang up the other calls.
============= ===================
