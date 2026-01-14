.. _flow-struct-activeflow:

Actvieflow
==========

.. _flow-struct-activeflow-activeflow:

Activeflow
----------

.. code::

    {
        "id": "<string>",
        "flow_id": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "current_action": {
            ...
        },
        "forward_action_id": "<string>",
        "actions": [
            ...
        ],
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* id: Activeflow's ID.
* flow_id: Based flow id.
* *reference_type*: Referenced resource type. See detail :ref:`here <flow-struct-activeflow-type>`.
* reference_id: Referenced resource's ID.
* current_action: Currently executing action. See detail :ref:`here <flow-struct-action>`
* forward_action_id: Forward action ID. This action_id will be executed in the next action execution.
* actions: List of actions. See detail :ref:`here <flow-struct-action>`.

Example
+++++++

.. code::

    {
        "type": "activeflow_created",
        "data": {
            "id": "7daa1750-f0a1-4674-a266-b95a68b27b7c",
            "flow_id": "d157ce07-0360-4cad-9007-c8ab89fccf9c",
            "reference_type": "call",
            "reference_id": "8c0c92c8-b5e6-42b2-80f3-5785b639eb3a",
            "current_action": {
                "id": "00000000-0000-0000-0000-000000000001",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": ""
            },
            "forward_action_id": "00000000-0000-0000-0000-000000000000",
            "actions": [
                {
                    "id": "df25724f-e308-4c89-9325-cf56cd09249e",
                    "next_id": "00000000-0000-0000-0000-000000000000",
                    "type": "answer"
                },
                {
                    "id": "2e7ec294-fc66-4039-8446-6590b82ed54f",
                    "next_id": "00000000-0000-0000-0000-000000000000",
                    "type": "talk",
                    "option": {
                        "text": "hello. welcome to voipbin. This is test message. Please enjoy the voipbin's service. thank you.",
                        "gender": "female",
                        "language": "en-US"
                    }
                }
            ],
            "tm_create": "2022-04-10 09:06:12.332217",
            "tm_update": "2022-04-10 09:06:12.332217",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }

.. _flow-struct-activeflow-type:

Reference type
--------------
Represent referenced resource's type.

======================= ==================
type                    Description
======================= ==================
call                    Call resource reference.
message                 Message resource reference.
======================= ==================

