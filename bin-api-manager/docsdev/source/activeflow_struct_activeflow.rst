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
        "reference_activeflow_id": "<string>",
        "on_complete_flow_id": "<string>",
        "webhook_uri": "<string>",
        "webhook_method": "<string>",
        "current_action": {
            ...
        },
        "forward_action_id": "<string>",
        "executed_actions": [
            ...
        ],
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The activeflow's unique identifier. Returned when listing via ``GET /activeflows`` or ``GET /activeflows/{id}``.
* ``customer_id`` (UUID): The customer who owns this activeflow. Obtained from ``GET /customers`` or your authentication context.
* ``flow_id`` (UUID): The flow template this activeflow was created from. Obtained from ``GET /flows``.
* ``status`` (enum string): The activeflow's current status. See detail :ref:`here <activeflow-struct-activeflow-status>`.
* ``reference_type`` (enum string): The resource type that triggered this activeflow. See detail :ref:`here <activeflow-struct-activeflow-reference-type>`.
* ``reference_id`` (UUID): The ID of the resource that triggered this activeflow (e.g., a call ID if ``reference_type`` is ``call``). Obtained from the corresponding resource endpoint (e.g., ``GET /calls/{id}``).
* ``reference_activeflow_id`` (UUID): The parent activeflow's ID if this is a sub-flow. Obtained from ``GET /activeflows``. Set to ``00000000-0000-0000-0000-000000000000`` if this is not a sub-flow.
* ``on_complete_flow_id`` (UUID): Flow to execute when this activeflow completes. Obtained from the ``id`` field of ``GET /flows``. Set to ``00000000-0000-0000-0000-000000000000`` if no completion flow is assigned.
* ``webhook_uri`` (String, optional): Per-activeflow webhook destination URI. When set (at creation via ``POST /activeflows``), activeflow webhook events are delivered to this URI in addition to the customer-level webhook destination. Empty if no per-activeflow webhook is configured.
* ``webhook_method`` (enum string, optional): HTTP method used to deliver the per-activeflow webhook. One of ``POST``, ``GET``, ``PUT`` or ``DELETE``. See detail :ref:`here <activeflow-struct-activeflow-webhook-method>`. Empty if no per-activeflow webhook is configured.
* ``current_action`` (Object): The action currently being executed. See detail :ref:`here <flow-struct-action-action>`.
* ``forward_action_id`` (UUID): The ID of the next action to execute. Set to ``00000000-0000-0000-0000-000000000000`` if sequential (next in array).
* ``executed_actions`` (Array of Object): History of actions that have been executed during this activeflow's lifetime. Each element is an action object. See detail :ref:`here <flow-struct-action-action>`.
* ``tm_create`` (String, ISO 8601): Timestamp when the activeflow was created.
* ``tm_update`` (String, ISO 8601): Timestamp of the last state change.
* ``tm_delete`` (String, ISO 8601): Timestamp when the activeflow was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Activeflows are typically created automatically when a flow is triggered (e.g., by an incoming call). You can also create one directly via ``POST /activeflows`` (with ``reference_type`` set to ``api``), optionally supplying ``webhook_uri`` and ``webhook_method`` to receive activeflow webhook events at a per-activeflow destination, additively to the customer-level webhook. You can list them (``GET /activeflows``), inspect them (``GET /activeflows/{id}``), or stop them (``POST /activeflows/{id}/stop``). Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred.

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
        "reference_activeflow_id": "00000000-0000-0000-0000-000000000000",
        "on_complete_flow_id": "00000000-0000-0000-0000-000000000000",
        "webhook_uri": "https://example.com/webhooks/activeflow",
        "webhook_method": "POST",
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
        "executed_actions": [],
        "tm_create": "2023-04-06 14:53:12.569073",
        "tm_update": "2023-04-06 14:54:24.652558",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _activeflow-struct-activeflow-status:

Status
------
Activeflow's status.

=========== ============
Status      Description
=========== ============
``""``      Initial state. The activeflow has been created but execution has not yet started.
running     Activeflow is running.
ended       Activeflow has stopped.
=========== ============

.. _activeflow-struct-activeflow-reference-type:

Reference type
--------------
The resource type that triggered the activeflow execution.

============ ================================================
Type         Description
============ ================================================
call         Incoming or outgoing call triggered the flow.
message      Incoming SMS/MMS message triggered the flow.
api          Flow started via API call.
campaign     Outbound campaign triggered the flow.
transcribe   Transcription service triggered the flow.
recording    Recording completion triggered the flow.
ai           AI service triggered the flow.
============ ================================================

.. _activeflow-struct-activeflow-webhook-method:

Webhook method
--------------
HTTP method used to deliver the per-activeflow webhook. Applies only when ``webhook_uri`` is set.

=========== ============
Method      Description
=========== ============
``""``      No per-activeflow webhook method configured.
POST        Deliver the webhook using an HTTP POST request.
GET         Deliver the webhook using an HTTP GET request.
PUT         Deliver the webhook using an HTTP PUT request.
DELETE      Deliver the webhook using an HTTP DELETE request.
=========== ============

.. note:: **Additive per-activeflow webhook**

   When ``webhook_uri`` and ``webhook_method`` are supplied at creation (``POST /activeflows``), activeflow webhook events are delivered to that destination in addition to the customer-level webhook destination. The per-activeflow webhook does not replace the customer-level webhook; both receive the events.

