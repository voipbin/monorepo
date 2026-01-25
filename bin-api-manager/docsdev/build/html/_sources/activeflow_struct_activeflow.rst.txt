.. _activeflow-struct-activeflow:

Activeflow
==========

.. _activeflow-struct-activeflow-activeflow:

Activeflow
----------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "flow_id": "<string>",
        "status": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "current_action": {
            ...
        },
        "forward_action_id": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* id: Activeflow's ID.
* customer_id: Customer's ID.
* flow_id: Flow's ID.
* *status*: Activeflow's status. See detail :ref:`here <activeflow-struct-activeflow-status>`.
* *reference_type*: Represent which resource started activeflow.
* reference_id: Referenced type's ID.
* current_action: Currently running action on this activeflow. See detail :ref:`here <flow-struct-action-action>`.
* forward_action_id: Forward action id.

Example
+++++++

.. code::

    {
        "id": "6f18ae1c-ddf8-413b-9572-ad30574604ef",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "flow_id": "93993ae1-0408-4639-ad5f-1288aa8d4325",
        "status": "ended",
        "reference_type": "call",
        "reference_id": "fd581a20-2606-47fd-a7e8-6bba7c294170",
        "current_action": {
            "id": "93ebcadb-ecae-4291-8d49-ca81a926b8b3",
            "next_id": "00000000-0000-0000-0000-000000000000",
            "type": "digits_receive",
            "option": {
                "length": 1,
                "duration": 5000
            }
        },
        "forward_action_id": "00000000-0000-0000-0000-000000000000",
        "tm_create": "2023-04-06 14:53:12.569073",
        "tm_update": "2023-04-06 14:54:24.652558",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _activeflow-struct-activeflow-status:

Status
------
Activeflow's status.

=========== ============
Type        Description
=========== ============
EMPTY       None
running     Activeflow is running.
ended       Activeflow has stopped.
=========== ============

Reference type
--------------
Triggered resource.

=========== ============
Type        Description
=========== ============
EMPTY       None
call        Call resource started the activeflow.
sms         SMS resource started the activeflow.
=========== ============

