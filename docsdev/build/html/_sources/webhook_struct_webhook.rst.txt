.. _webhook-struct-webhook:

Webhook
=======

.. _webhook-struct-webhook-activeflow_created:

activeflow_created
------------------
The notification message for the activeflow create.

.. code::

    {
        "type": "activeflow_created",
        "data": {
            ...
        }
    }

* type: The webhook type.
* data: The detail of activeflow. See detail :ref:`here <flow-struct-activeflow>`.

Example
+++++++

.. code::

    {
        "type": "activeflow_created",
        "data": {
            "id": "74ac5405-7c70-4184-9388-1c9f8f8ce25f",
            "flow_id": "d157ce07-0360-4cad-9007-c8ab89fccf9c",
            "customer_id": "00000000-0000-0000-0000-000000000000",
            "reference_type": "call",
            "reference_id": "5371e9db-d035-4db6-a8d6-0994d33e744e",
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
            "tm_create": "2022-04-11 00:23:54.724620",
            "tm_update": "2022-04-11 00:23:54.724620",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }

.. _webhook-struct-webhook-activeflow_updated:

activeflow_updated
------------------
The notification message for the activeflow update.

.. code::

    {
        "type": "activeflow_updated",
        "data": {
            ...
        }
    }

* type: The webhook type.
* data: The detail of activeflow. See detail :ref:`here <flow-struct-activeflow>`.

Example
+++++++

.. code::

    {
        "type": "activeflow_updated",
        "data": {
            "id": "74ac5405-7c70-4184-9388-1c9f8f8ce25f",
            "flow_id": "d157ce07-0360-4cad-9007-c8ab89fccf9c",
            "customer_id": "00000000-0000-0000-0000-000000000000",
            "reference_type": "call",
            "reference_id": "5371e9db-d035-4db6-a8d6-0994d33e744e",
            "current_action": {
                "id": "df25724f-e308-4c89-9325-cf56cd09249e",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "answer"
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
            "tm_create": "2022-04-11 00:23:54.724620",
            "tm_update": "2022-04-11 00:23:54.840938",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }

.. _webhook-struct-webhook-activeflow_deleted:

activeflow_deleted
------------------
The notification message for the activeflow delete.

.. code::

    {
        "type": "activeflow_deleted",
        "data": {
            ...
        }
    }

* type: The webhook type.
* data: The detail of activeflow. See detail :ref:`here <flow-struct-activeflow>`.

Example
+++++++

.. code::

    {
        "type": "activeflow_deleted",
        "data": {
            "id": "74ac5405-7c70-4184-9388-1c9f8f8ce25f",
            "flow_id": "d157ce07-0360-4cad-9007-c8ab89fccf9c",
            "customer_id": "00000000-0000-0000-0000-000000000000",
            "reference_type": "call",
            "reference_id": "5371e9db-d035-4db6-a8d6-0994d33e744e",
            "current_action": {
                "id": "2e7ec294-fc66-4039-8446-6590b82ed54f",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "talk",
                "option": {
                    "text": "hello. welcome to voipbin. This is test message. Please enjoy the voipbin's service. thank you.",
                    "gender": "female",
                    "language": "en-US"
                }
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
            "tm_create": "2022-04-11 00:23:54.724620",
            "tm_update": "2022-04-11 00:23:55.134500",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }

.. _webhook-struct-webhook-call_created:

call_created
------------
The notification message for the call create.

.. code::

    {
        "type": "call_created",
        "data": {
            ...
        }
    }

* type: The webhook type.
* data: The detail of call. See detail :ref:`here <call-struct-call>`.

Example
+++++++

.. code::

    {
        "type": "call_created",
        "data": {
            "id": "5371e9db-d035-4db6-a8d6-0994d33e744e",
            "flow_id": "d157ce07-0360-4cad-9007-c8ab89fccf9c",
            "type": "flow",
            "master_call_id": "00000000-0000-0000-0000-000000000000",
            "chained_call_ids": [],
            "recording_id": "00000000-0000-0000-0000-000000000000",
            "recording_ids": [],
            "source": {
                "type": "tel",
                "target": "+821100000002",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "destination": {
                "type": "tel",
                "target": "+821100000001",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "status": "ringing",
            "action": {
                "id": "00000000-0000-0000-0000-000000000001",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": ""
            },
            "direction": "incoming",
            "hangup_by": "",
            "hangup_reason": "",
            "tm_create": "2022-04-11 00:23:53.636000",
            "tm_update": "9999-01-01 00:00:00.000000",
            "tm_progressing": "9999-01-01 00:00:00.000000",
            "tm_ringing": "9999-01-01 00:00:00.000000",
            "tm_hangup": "9999-01-01 00:00:00.000000"
        }
    }

.. _webhook-struct-webhook-call_ringing:

call_ringing
-------------
The notification message for the call ringing.

.. code::

    {
        "type": "call_ringing",
        "data": {
            ...
        }
    }

* type: The webhook type.
* data: The detail of call. See detail :ref:`here <call-struct-call>`.

Example
+++++++

.. code::

    {
        "type": "call_ringing",
        "data": {
            "id": "ad132775-1ab2-485e-856f-72c2e383cdc6",
            "flow_id": "6da52ef9-7d7d-48e4-8bca-921e7b78e47c",
            "type": "flow",
            "master_call_id": "1fe1356f-3f7f-4ff9-9d33-08136b38f506",
            "chained_call_ids": [],
            "recording_id": "00000000-0000-0000-0000-000000000000",
            "recording_ids": [],
            "source": {
                "type": "tel",
                "target": "+821021656521",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "destination": {
                "type": "tel",
                "target": "+821021656521",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "status": "ringing",
            "action": {
                "id": "00000000-0000-0000-0000-000000000001",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": ""
            },
            "direction": "outgoing",
            "hangup_by": "",
            "hangup_reason": "",
            "tm_create": "2022-03-29 15:08:01.815004",
            "tm_update": "2022-03-29 15:08:03.421646",
            "tm_progressing": "9999-01-01 00:00:00.000000",
            "tm_ringing": "2022-03-29 15:08:03.314000",
            "tm_hangup": "9999-01-01 00:00:00.000000"
        }
    }

.. _webhook-struct-webhook-call_answered:

call_answered
-------------
The notification message for the call answer.

.. code::

    {
        "type": "call_answered",
        "data": {
            ...
        }
    }

* type: The webhook type.
* data: The detail of call. See detail :ref:`here <call-struct-call>`.

Example
+++++++

.. code::

    {
        "type": "call_answered",
        "data": {
            "id": "5371e9db-d035-4db6-a8d6-0994d33e744e",
            "flow_id": "d157ce07-0360-4cad-9007-c8ab89fccf9c",
            "type": "flow",
            "master_call_id": "00000000-0000-0000-0000-000000000000",
            "chained_call_ids": [],
            "recording_id": "00000000-0000-0000-0000-000000000000",
            "recording_ids": [],
            "source": {
                "type": "tel",
                "target": "+821100000002",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "destination": {
                "type": "tel",
                "target": "+821100000001",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "status": "progressing",
            "action": {
                "id": "df25724f-e308-4c89-9325-cf56cd09249e",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "answer",
                "tm_execute": "2022-04-11 00:23:55.012416032"
            },
            "direction": "incoming",
            "hangup_by": "",
            "hangup_reason": "",
            "tm_create": "2022-04-11 00:23:53.636000",
            "tm_update": "2022-04-11 00:23:55.130190",
            "tm_progressing": "2022-04-11 00:23:55.026000",
            "tm_ringing": "9999-01-01 00:00:00.000000",
            "tm_hangup": "9999-01-01 00:00:00.000000"
        }
    }

.. _webhook-struct-webhook-call_updated:

call_updated
------------
The notification message for the call update.

.. code::

    {
        "type": "call_updated",
        "data": {
            ...
        }
    }

* type: The webhook type.
* data: The detail of call. See detail :ref:`here <call-struct-call>`.

Example
+++++++

.. code::

    {
        "type": "call_updated",
        "data": {
            "id": "bf682a17-6b3f-412c-bbac-faa81fb9ada3",
            "flow_id": "70875796-0497-4ff9-acd0-e226a14495a9",
            "type": "flow",
            "master_call_id": "00000000-0000-0000-0000-000000000000",
            "chained_call_ids": [
                "a876f057-bb20-4b87-824c-d7afa3e71af5"
            ],
            "recording_id": "00000000-0000-0000-0000-000000000000",
            "recording_ids": [],
            "source": {
                "type": "tel",
                "target": "test11",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "destination": {
                "type": "tel",
                "target": "+821100000004",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "status": "hangup",
            "action": {
                "id": "4aae4342-d702-4e23-9c14-64dc20d2075d",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "confbridge_join",
                "option": {
                    "confbridge_id": "821fc304-0ed8-4e93-8a0a-c23312c062be"
                },
                "tm_execute": "2022-03-29 14:10:06.409155828"
            },
            "direction": "incoming",
            "hangup_by": "remote",
            "hangup_reason": "normal",
            "tm_create": "2022-03-29 14:09:52.886000",
            "tm_update": "2022-03-29 14:10:33.709605",
            "tm_progressing": "2022-03-29 14:09:54.629000",
            "tm_ringing": "9999-01-01 00:00:00.000000",
            "tm_hangup": "2022-03-29 14:10:33.105000"
        }
    }

.. _webhook-struct-webhook-call_hungup:

call_hungup
-----------
The notification message for the call hangup.

.. code::

    {
        "type": "call_hungup",
        "data": {
            ...
        }
    }

* type: The webhook type.
* data: The detail of call. See detail :ref:`here <call-struct-call>`.

Example
+++++++

.. code::

    {
        "type": "call_hungup",
        "data": {
            "id": "593555d2-787e-4b06-862f-407bb2e43be1",
            "flow_id": "d157ce07-0360-4cad-9007-c8ab89fccf9c",
            "type": "flow",
            "master_call_id": "00000000-0000-0000-0000-000000000000",
            "chained_call_ids": [],
            "recording_id": "00000000-0000-0000-0000-000000000000",
            "recording_ids": [],
            "source": {
                "type": "tel",
                "target": "+821100000002",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "destination": {
                "type": "tel",
                "target": "+821100000001",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "status": "hangup",
            "action": {
                "id": "2e7ec294-fc66-4039-8446-6590b82ed54f",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "talk",
                "option": {
                    "text": "hello. welcome to voipbin. This is test message. Please enjoy the voipbin's service. thank you.",
                    "gender": "female",
                    "language": "en-US"
                },
                "tm_execute": "2022-04-11 06:10:55.918010931"
            },
            "direction": "incoming",
            "hangup_by": "remote",
            "hangup_reason": "normal",
            "tm_create": "2022-04-11 06:10:54.788000",
            "tm_update": "2022-04-11 06:10:58.431000",
            "tm_progressing": "2022-04-11 06:10:55.765000",
            "tm_ringing": "9999-01-01 00:00:00.000000",
            "tm_hangup": "2022-04-11 06:10:58.431000"
        }
    }

.. _webhook-struct-webhook-queue_created:

queue_created
-------------
Notification message for queue create.

.. code::

    {
        "type": "queue_created",
        "data": {
            ...
        }
    }

* type: The webhook type.
* data: The detail of call. See detail :ref:`here <queue-struct-queue>`.

.. _webhook-struct-webhook-queue_updated:

queue_updated
-------------
The notification message for the queue update.

.. code::

    {
        "type": "queue_updated",
        "data": {
            ...
        }
    }

* type: The webhook type.
* data: The detail of call. See detail :ref:`here <queue-struct-queue>`.

.. _webhook-struct-webhook-queue_deleted:

queue_deleted
-------------
The notification message for the queue delete.

.. code::

    {
        "type": "queue_deleted",
        "data": {
            ...
        }
    }

* type: The webhook type.
* data: The detail of call. See detail :ref:`here <queue-struct-queue>`.

.. _webhook-struct-webhook-queuecall_created:

queuecall_created
-----------------
The notification message for the queuecall create.

.. code::

    {
        "type": "queuecall_created",
        "data": {
            ...
        }
    }

* type: The webhook type.
* data: The detail of queuecall. See detail :ref:`here <queue-struct-queuecall>`.

Example
+++++++

.. code::

    {
        "type": "queuecall_created",
        "data": {
            "id": "c7c1e226-8c86-4b43-9606-2d5bb2059a09",
            "reference_type": "call",
            "reference_id": "1fe1356f-3f7f-4ff9-9d33-08136b38f506",
            "status": "wait",
            "service_agent_id": "00000000-0000-0000-0000-000000000000",
            "tm_create": "2022-03-29 15:07:46.111715",
            "tm_service": "9999-01-01 00:00:00.000000",
            "tm_update": "9999-01-01 00:00:00.000000",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }

.. _webhook-struct-webhook-queuecall_entering:

queuecall_entering
------------------
Notification message for queuecall is entering to the agent's conference room.

.. code::

    {
        "type": "queuecall_entering",
        "data": {
            ...
        }
    }

* type: The webhook type.
* data: The detail of queuecall. See detail :ref:`here <queue-struct-queuecall>`.

Example
+++++++

.. code::

    {
        "type": "queuecall_entering",
        "data": {
            "id": "c7c1e226-8c86-4b43-9606-2d5bb2059a09",
            "reference_type": "call",
            "reference_id": "1fe1356f-3f7f-4ff9-9d33-08136b38f506",
            "status": "entering",
            "service_agent_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
            "tm_create": "2022-03-29 15:07:46.111715",
            "tm_service": "2022-03-29 15:08:02.233858",
            "tm_update": "2022-03-29 15:08:02.233858",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }

.. _webhook-struct-webhook-queuecall_kicking:

queuecall_kicking
-----------------
The notification message for the queuecall is being kicked.

.. code::

    {
        "type": "queuecall_kicking",
        "data": {
            ...
        }
    }

* type: The webhook type.
* *data*: The detail of queuecall. See detail :ref:`here <queue-struct-queuecall>`.

Example
+++++++

.. code::

    {
        "type": "queuecall_kicking",
        "data": {
            "id": "c7c1e226-8c86-4b43-9606-2d5bb2059a09",
            "reference_type": "call",
            "reference_id": "1fe1356f-3f7f-4ff9-9d33-08136b38f506",
            "status": "entering",
            "service_agent_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
            "tm_create": "2022-03-29 15:07:46.111715",
            "tm_service": "2022-03-29 15:08:02.233858",
            "tm_update": "2022-03-29 15:08:02.233858",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }


.. _webhook-struct-webhook-queuecall_serviced:

queuecall_serviced
------------------
The notification message for the queuecall is serviced.

.. code::

    {
        "type": "queuecall_serviced",
        "data": {
            ...
        }
    }

* type: The webhook type.
* *data*: The detail of queuecall. See detail :ref:`here <queue-struct-queuecall>`.

Example
+++++++

.. code::

    {
        "type": "queuecall_serviced",
        "data": {
            "id": "c7c1e226-8c86-4b43-9606-2d5bb2059a09",
            "reference_type": "call",
            "reference_id": "1fe1356f-3f7f-4ff9-9d33-08136b38f506",
            "status": "service",
            "service_agent_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
            "tm_create": "2022-03-29 15:07:46.111715",
            "tm_service": "2022-03-29 15:08:04.811442",
            "tm_update": "2022-03-29 15:08:04.811442",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }

.. _webhook-struct-webhook-queuecall_done:

queuecall_done
-----------------
The notification message for the queuecall is done.

.. code::

    {
        "type": "queuecall_done",
        "data": {
            ...
        }
    }

* type: The webhook type.
* *data*: The detail of queuecall. See detail :ref:`here <queue-struct-queuecall>`.

Example
+++++++

.. code::

    {
        "type": "queuecall_done",
        "data": {
            "id": "c7c1e226-8c86-4b43-9606-2d5bb2059a09",
            "reference_type": "call",
            "reference_id": "1fe1356f-3f7f-4ff9-9d33-08136b38f506",
            "status": "done",
            "service_agent_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
            "tm_create": "2022-03-29 15:07:46.111715",
            "tm_service": "2022-03-29 15:08:04.811442",
            "tm_update": "2022-03-29 15:08:25.814885",
            "tm_delete": "2022-03-29 15:08:25.814885"
        }
    }

.. _webhook-struct-webhook-queuecall_abandoned:

queuecall_abandoned
-------------------
The notification message for the queuecall is abandoned.

.. code::

    {
        "type": "queuecall_abandoned",
        "data": {
            ...
        }
    }

* type: The webhook type.
* *data*: The detail of queuecall. See detail :ref:`here <queue-struct-queuecall>`.

.. _webhook-struct-webhook-agent_created:

agent_created
-------------
The notification message for the agent create.

.. code::

    {
        "type": "agent_created",
        "data": {
            ...
        }
    }

* type: The webhook type.
* *data*: The detail of agent. See detail :ref:`here <agent-struct-agent>`.

.. _webhook-struct-webhook-agent_updated:

agent_updated
-------------
Notification message for agent update.

The notification message for the agent update.

.. code::

    {
        "type": "agent_updated",
        "data": {
            ...
        }
    }

* type: The webhook type.
* *data*: The detail of agent. See detail :ref:`here <agent-struct-agent>`.

.. _webhook-struct-webhook-agent_deleted:

agent_deleted
-------------
The notification message for the agent delete.

.. code::

    {
        "type": "agent_deleted",
        "data": {
            ...
        }
    }

* type: The webhook type.
* *data*: The detail of agent. See detail :ref:`here <agent-struct-agent>`.

.. _webhook-struct-webhook-agent_status_updated:

agent_status_updated
--------------------
The notification message for the agent's status update.

.. code::

    {
        "type": "agent_status_updated",
        "data": {
            ...
        }
    }

* type: The webhook type.
* *data*: The detail of agent. See detail :ref:`here <agent-struct-agent>`.

Example
+++++++

.. code::

    {
        "type": "agent_status_updated",
        "data": {
            "id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
            "username": "test1",
            "name": "test agent 1",
            "detail": "test agent. username test1",
            "ring_method": "ringall",
            "status": "available",
            "permission": 0,
            "tag_ids": [
                "d7450dda-21e0-4611-b09a-8d771c50a5e6"
            ],
            "addresses": [
                {
                    "type": "tel",
                    "target": "+821021656521",
                    "target_name": "",
                    "name": "",
                    "detail": ""
                }
            ],
            "tm_create": "2021-11-29 06:09:07.263846",
            "tm_update": "2022-03-29 15:08:00.814900",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }
