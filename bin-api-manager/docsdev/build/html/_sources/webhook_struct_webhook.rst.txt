.. _webhook-struct-webhook:

Webhook
=======

All webhook events follow a common envelope structure with two fields:

* ``type`` (enum string): The webhook event type indicating what occurred (e.g., ``"call_created"``, ``"activeflow_updated"``).
* ``data`` (Object): The resource-specific payload. Structure varies by event type and matches the corresponding resource struct.

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` in webhook payloads indicate the event has not yet occurred. For example, ``tm_hangup`` with this sentinel value means the call is still in progress. Use the ``type`` field to determine which resource struct to parse from the ``data`` field.

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

* ``type`` (enum string): The webhook type. Value: ``"activeflow_created"``.
* ``data`` (Object): The detail of activeflow. See detail :ref:`here <activeflow-struct-activeflow>`.

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

* ``type`` (enum string): The webhook type. Value: ``"activeflow_updated"``.
* ``data`` (Object): The detail of activeflow. See detail :ref:`here <activeflow-struct-activeflow>`.

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

* ``type`` (enum string): The webhook type. Value: ``"activeflow_deleted"``.
* ``data`` (Object): The detail of activeflow. See detail :ref:`here <activeflow-struct-activeflow>`.

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

* ``type`` (enum string): The webhook type. Value: ``"call_created"``.
* ``data`` (Object): The detail of call. See detail :ref:`here <call-struct-call>`.

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

* ``type`` (enum string): The webhook type. Value: ``"call_ringing"``.
* ``data`` (Object): The detail of call. See detail :ref:`here <call-struct-call>`.

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
                "target": "+15559876543",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "destination": {
                "type": "tel",
                "target": "+15559876543",
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

.. _webhook-struct-webhook-call_progressing:

call_progressing
----------------
The notification message sent when the call is answered and enters the progressing state.

.. code::

    {
        "type": "call_progressing",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"call_progressing"``.
* ``data`` (Object): The detail of call. See detail :ref:`here <call-struct-call>`.

Example
+++++++

.. code::

    {
        "type": "call_progressing",
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

* ``type`` (enum string): The webhook type. Value: ``"call_updated"``.
* ``data`` (Object): The detail of call. See detail :ref:`here <call-struct-call>`.

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

.. _webhook-struct-webhook-call_hangup:

call_hangup
-----------
The notification message for the call hangup.

.. code::

    {
        "type": "call_hangup",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"call_hangup"``.
* ``data`` (Object): The detail of call. See detail :ref:`here <call-struct-call>`.

Example
+++++++

.. code::

    {
        "type": "call_hangup",
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

* ``type`` (enum string): The webhook type. Value: ``"queue_created"``.
* ``data`` (Object): The detail of queue. See detail :ref:`here <queue-struct-queue>`.

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

* ``type`` (enum string): The webhook type. Value: ``"queue_updated"``.
* ``data`` (Object): The detail of queue. See detail :ref:`here <queue-struct-queue>`.

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

* ``type`` (enum string): The webhook type. Value: ``"queue_deleted"``.
* ``data`` (Object): The detail of queue. See detail :ref:`here <queue-struct-queue>`.

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

* ``type`` (enum string): The webhook type. Value: ``"queuecall_created"``.
* ``data`` (Object): The detail of queuecall. See detail :ref:`here <queue-struct-queuecall>`.

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

.. _webhook-struct-webhook-queuecall_connecting:

queuecall_connecting
--------------------
Notification message for queuecall is connecting to the agent's conference room.

.. code::

    {
        "type": "queuecall_connecting",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"queuecall_connecting"``.
* ``data`` (Object): The detail of queuecall. See detail :ref:`here <queue-struct-queuecall>`.

Example
+++++++

.. code::

    {
        "type": "queuecall_connecting",
        "data": {
            "id": "c7c1e226-8c86-4b43-9606-2d5bb2059a09",
            "reference_type": "call",
            "reference_id": "1fe1356f-3f7f-4ff9-9d33-08136b38f506",
            "status": "connecting",
            "service_agent_id": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
            "tm_create": "2022-03-29 15:07:46.111715",
            "tm_service": "2022-03-29 15:08:02.233858",
            "tm_update": "2022-03-29 15:08:02.233858",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }

.. _webhook-struct-webhook-queuecall_waiting:

queuecall_waiting
-----------------
The notification message for the queuecall is waiting for an agent.

.. code::

    {
        "type": "queuecall_waiting",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"queuecall_waiting"``.
* ``data`` (Object): The detail of queuecall. See detail :ref:`here <queue-struct-queuecall>`.

Example
+++++++

.. code::

    {
        "type": "queuecall_waiting",
        "data": {
            "id": "c7c1e226-8c86-4b43-9606-2d5bb2059a09",
            "reference_type": "call",
            "reference_id": "1fe1356f-3f7f-4ff9-9d33-08136b38f506",
            "status": "waiting",
            "service_agent_id": "00000000-0000-0000-0000-000000000000",
            "tm_create": "2022-03-29 15:07:46.111715",
            "tm_service": "9999-01-01 00:00:00.000000",
            "tm_update": "2022-03-29 15:07:46.111715",
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

* ``type`` (enum string): The webhook type. Value: ``"queuecall_serviced"``.
* ``data`` (Object): The detail of queuecall. See detail :ref:`here <queue-struct-queuecall>`.

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

* ``type`` (enum string): The webhook type. Value: ``"queuecall_done"``.
* ``data`` (Object): The detail of queuecall. See detail :ref:`here <queue-struct-queuecall>`.

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

* ``type`` (enum string): The webhook type. Value: ``"queuecall_abandoned"``.
* ``data`` (Object): The detail of queuecall. See detail :ref:`here <queue-struct-queuecall>`.

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

* ``type`` (enum string): The webhook type. Value: ``"agent_created"``.
* ``data`` (Object): The detail of agent. See detail :ref:`here <agent-struct-agent>`.

.. _webhook-struct-webhook-agent_updated:

agent_updated
-------------
The notification message for the agent update.

.. code::

    {
        "type": "agent_updated",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"agent_updated"``.
* ``data`` (Object): The detail of agent. See detail :ref:`here <agent-struct-agent>`.

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

* ``type`` (enum string): The webhook type. Value: ``"agent_deleted"``.
* ``data`` (Object): The detail of agent. See detail :ref:`here <agent-struct-agent>`.

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

* ``type`` (enum string): The webhook type. Value: ``"agent_status_updated"``.
* ``data`` (Object): The detail of agent. See detail :ref:`here <agent-struct-agent>`.

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
                    "target": "+15559876543",
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

.. _webhook-struct-webhook-message_created:

message_created
----------------
The notification message for a new SMS message (sent or received).

.. code::

    {
        "type": "message_created",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"message_created"``.
* ``data`` (Object): The detail of the message. See detail :ref:`here <message-struct-message>`.

Example
+++++++

.. code::

    {
        "type": "message_created",
        "data": {
            "id": "a5d2114a-8e84-48cd-8bb2-c406eeb08cd1",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "type": "sms",
            "source": {
                "type": "tel",
                "target": "+15551234567",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "targets": [
                {
                    "destination": {
                        "type": "tel",
                        "target": "+15559876543",
                        "target_name": "",
                        "name": "",
                        "detail": ""
                    },
                    "status": "queued",
                    "parts": 1,
                    "tm_update": "2022-03-13 15:11:06.497184"
                }
            ],
            "text": "Hello, this is test message.",
            "direction": "outbound",
            "tm_create": "2022-03-13 15:11:05.235717",
            "tm_update": "2022-03-13 15:11:05.235717",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }

.. _webhook-struct-webhook-message_updated:

message_updated
----------------
The notification message for an SMS message's delivery status update.

.. code::

    {
        "type": "message_updated",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"message_updated"``.
* ``data`` (Object): The detail of the message. See detail :ref:`here <message-struct-message>`.

.. _webhook-struct-webhook-email_created:

email_created
--------------
The notification message for a new email (sent or received).

.. code::

    {
        "type": "email_created",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"email_created"``.
* ``data`` (Object): The detail of the email. See detail :ref:`here <email-struct-email>`.

Example
+++++++

.. code::

    {
        "type": "email_created",
        "data": {
            "id": "1f25e6c9-6709-44d1-b93e-a5f1c5f80411",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "source": {
                "type": "email",
                "target": "service@voipbin.net",
                "target_name": "voipbin service",
                "name": "",
                "detail": ""
            },
            "destinations": [
                {
                    "type": "email",
                    "target": "recipient@example.com",
                    "target_name": "",
                    "name": "",
                    "detail": ""
                }
            ],
            "status": "initiated",
            "subject": "Hello from VoIPBIN",
            "content": "This is a test email sent via the VoIPBIN API.",
            "attachments": [],
            "tm_create": "2025-03-14 19:04:01.160250",
            "tm_update": "2025-03-14 19:04:01.160250",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }

.. _webhook-struct-webhook-email_updated:

email_updated
--------------
The notification message for an email's delivery status update (e.g. ``processed``, ``delivered``, ``bounce``).

.. code::

    {
        "type": "email_updated",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"email_updated"``.
* ``data`` (Object): The detail of the email. See detail :ref:`here <email-struct-email>`.

.. _webhook-struct-webhook-email_deleted:

email_deleted
--------------
The notification message for an email delete.

.. code::

    {
        "type": "email_deleted",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"email_deleted"``.
* ``data`` (Object): The detail of the email. See detail :ref:`here <email-struct-email>`.

.. _webhook-struct-webhook-webchat_message_created:

webchat_message_created
------------------------
The notification message for a new webchat message (inbound from the visitor or outbound from Flow/AI/an agent).

.. code::

    {
        "type": "webchat_message_created",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"webchat_message_created"``.
* ``data`` (Object): The detail of the webchat message. See detail :ref:`here <webchat-struct-message>`.

Example
+++++++

.. code::

    {
        "type": "webchat_message_created",
        "data": {
            "id": "b1c2d3e4-f5a6-7890-bcde-f12345678901",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "widget_id": "550e8400-e29b-41d4-a716-446655440000",
            "session_id": "7a1bcb1a-1b8e-4f5c-9a6d-3e2f1a0b4c5d",
            "direction": "inbound",
            "status": "sent",
            "text": "Hi, I need help with my order",
            "tm_create": "2025-03-14 19:04:01.160250",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }

.. _webhook-struct-webhook-webchat_session_ended:

webchat_session_ended
-----------------------
The notification message for a webchat session ending (explicitly or via idle timeout).

.. code::

    {
        "type": "webchat_session_ended",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"webchat_session_ended"``.
* ``data`` (Object): The detail of the webchat session. See detail :ref:`here <webchat-struct-session>`.

.. _webhook-struct-webhook-conversation_created:

conversation_created
----------------------
The notification message for a new conversation.

.. code::

    {
        "type": "conversation_created",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"conversation_created"``.
* ``data`` (Object): The detail of the conversation. See detail :ref:`here <conversation-struct-conversation>`.

Example
+++++++

.. code::

    {
        "type": "conversation_created",
        "data": {
            "id": "bdc9d9f5-706c-4e2d-9be7-7dc1e5fd45a0",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "owner_type": "",
            "owner_id": "00000000-0000-0000-0000-000000000000",
            "account_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
            "name": "conversation",
            "detail": "conversation detail",
            "type": "message",
            "dialog_id": "+673802",
            "self": {
                "type": "tel",
                "target": "+14703298699",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "peer": {
                "type": "tel",
                "target": "+673802",
                "target_name": "",
                "name": "",
                "detail": ""
            },
            "tm_create": "2022-06-23 05:05:40.950834",
            "tm_update": "2022-06-23 05:05:40.950842",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }

.. _webhook-struct-webhook-conversation_updated:

conversation_updated
----------------------
The notification message for a conversation update (e.g. assignment to an agent).

.. code::

    {
        "type": "conversation_updated",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"conversation_updated"``.
* ``data`` (Object): The detail of the conversation. See detail :ref:`here <conversation-struct-conversation>`.

.. _webhook-struct-webhook-conversation_message_created:

conversation_message_created
-------------------------------
The notification message for a new message within a conversation.

.. code::

    {
        "type": "conversation_message_created",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"conversation_message_created"``.
* ``data`` (Object): The detail of the conversation message. See detail :ref:`here <conversation-struct-message>`.

Example
+++++++

.. code::

    {
        "type": "conversation_message_created",
        "data": {
            "id": "cc46341b-f00a-452f-b527-19c85d030eaf",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "conversation_id": "64558b45-40a8-43db-b814-9c0dbf6d47b5",
            "direction": "incoming",
            "status": "done",
            "reference_type": "line",
            "reference_id": "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
            "source": {
                "type": "line",
                "target": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
            },
            "destination": {
                "type": "line",
                "target": ""
            },
            "text": "안녕",
            "medias": [],
            "tm_create": "2022-06-24 04:28:51.558082",
            "tm_update": "2022-06-24 04:28:51.558090",
            "tm_delete": "9999-01-01 00:00:00.000000"
        }
    }

.. _webhook-struct-webhook-conversation_message_updated:

conversation_message_updated
-------------------------------
The notification message for a conversation message's status update.

.. code::

    {
        "type": "conversation_message_updated",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"conversation_message_updated"``.
* ``data`` (Object): The detail of the conversation message. See detail :ref:`here <conversation-struct-message>`.

.. _webhook-struct-webhook-conversation_message_deleted:

conversation_message_deleted
-------------------------------
The notification message for a conversation message delete.

.. code::

    {
        "type": "conversation_message_deleted",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"conversation_message_deleted"``.
* ``data`` (Object): The detail of the conversation message. See detail :ref:`here <conversation-struct-message>`.

.. _webhook-struct-webhook-account_created:

account_created
-----------------
The notification message for a new conversation account.

.. code::

    {
        "type": "account_created",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"account_created"``.
* ``data`` (Object): The detail of the conversation account. See detail :ref:`here <conversation-struct-account>`.

.. _webhook-struct-webhook-account_updated:

account_updated
-----------------
The notification message for a conversation account update.

.. code::

    {
        "type": "account_updated",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"account_updated"``.
* ``data`` (Object): The detail of the conversation account. See detail :ref:`here <conversation-struct-account>`.

.. _webhook-struct-webhook-account_deleted:

account_deleted
-----------------
The notification message for a conversation account delete.

.. code::

    {
        "type": "account_deleted",
        "data": {
            ...
        }
    }

* ``type`` (enum string): The webhook type. Value: ``"account_deleted"``.
* ``data`` (Object): The detail of the conversation account. See detail :ref:`here <conversation-struct-account>`.
