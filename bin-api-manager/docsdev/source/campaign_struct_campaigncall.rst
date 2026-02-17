.. _campaign-struct-campaigncall:

Campaigncall
===================

.. _campaign-struct-campaigncall-campaigncall:

Campaigncall
------------

.. code::

    {
        "id": "<string>",
        "campaign_id": "<string>",
        "outplan_id": "<string>",
        "outdial_id": "<string>",
        "outdial_target_id": "<string>",
        "queue_id": "<string>",
        "activeflow_id": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "status": "<string>",
        "result": "<string>",
        "source": {
            ...
        },
        "destination": {
            ...
        },
        "destination_index": <number>,
        "try_count": <number>,
        "tm_create": "2022-04-29 07:01:45.808944",
        "tm_update": "2022-04-29 07:02:48.304704"
    }

* ``id`` (UUID): The campaigncall's unique identifier. Returned when listing via ``GET /campaigncalls``.
* ``campaign_id`` (UUID): The parent campaign. Obtained from the ``id`` field of ``GET /campaigns``.
* ``outplan_id`` (UUID): The outplan used for this dial attempt. Obtained from the ``id`` field of ``GET /outplans``.
* ``outdial_id`` (UUID): The outdial containing the target. Obtained from the ``id`` field of ``GET /outdials``.
* ``outdial_target_id`` (UUID): The specific outdialtarget being dialed. Obtained from the ``id`` field of ``GET /outdials/{id}/targets``.
* ``queue_id`` (UUID): The queue for routing the answered call. Obtained from the ``id`` field of ``GET /queues``.
* ``activeflow_id`` (UUID): The activeflow executing the call actions. Obtained from the ``id`` field of ``GET /activeflows``.
* ``reference_type`` (enum string): The type of resource this campaigncall is linked to. See :ref:`Reference Type <campaign-struct-campaigncall-reference_type>`.
* ``reference_id`` (UUID): The ID of the referenced resource (e.g., the call ID when ``reference_type`` is ``call``).
* ``status`` (enum string): The campaigncall's current status. See :ref:`Status <campaign-struct-campaigncall-status>`.
* ``result`` (enum string): The campaigncall's outcome after completion. See :ref:`Result <campaign-struct-campaigncall-result>`.
* ``source`` (Object): Source address used for the dial attempt. See :ref:`Address <common-struct-address-address>`.
* ``destination`` (Object): Destination address being dialed. See :ref:`Address <common-struct-address-address>`.
* ``destination_index`` (Integer): Index of the destination within the outdialtarget (0-4), corresponding to ``destination_0`` through ``destination_4``.
* ``try_count`` (Integer): The current attempt number for this destination.
* ``tm_create`` (string, ISO 8601): Timestamp when the campaigncall was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to the campaigncall.

Example
+++++++

.. code::

    {
        "id": "56347901-5bb9-422d-add5-5a2ca47fa737",
        "campaign_id": "183c0d5c-691e-42f3-af2b-9bffc2740f83",
        "outplan_id": "d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e",
        "outdial_id": "40bea034-1d17-474d-a5de-da00d0861c69",
        "outdial_target_id": "f50b169d-ce02-4bc9-a6e7-bb632c71e450",
        "queue_id": "99bf739a-932f-433c-b1bf-103d33d7e9bb",
        "activeflow_id": "02aad54d-270e-43c7-82c5-bf42502c8bc6",
        "reference_type": "call",
        "reference_id": "a69189aa-7295-4c3a-b51f-df1dbbded5f6",
        "status": "done",
        "result": "success",
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
        "destination_index": 0,
        "try_count": 1,
        "tm_create": "2022-04-29 07:01:45.808944",
        "tm_update": "2022-04-29 07:02:48.304704"
    }

.. _campaign-struct-campaigncall-reference_type:

Reference type
--------------
The type of resource this campaigncall is associated with. The ``reference_id`` field contains the ID of this resource.

=========== ============
Type        Description
=========== ============
none        No associated resource. The campaigncall has not yet created a reference.
call        The campaigncall is associated with a voice call. The ``reference_id`` is the call's UUID, retrievable via ``GET /calls/{reference_id}``.
=========== ============

.. _campaign-struct-campaigncall-status:

Status
------
The campaigncall's current operational status. This is a read-only field managed by the system.

=========== ============
Type        Description
=========== ============
dialing     The campaigncall is dialing the target. The call has not been answered yet.
progressing The campaigncall is in progress. The target has answered and the flow actions are executing.
done        The campaigncall has completed. The call has been hung up. Check the ``result`` field for the outcome.
=========== ============

.. _campaign-struct-campaigncall-result:

Result
------
The campaigncall's outcome, calculated from the final status of the referenced resource (call, SMS, etc.).

For example, if the call ended with ``no_answer``, the result is calculated as ``fail``.

=========== ============
Type        Description
=========== ============
""          No result yet. The campaigncall is still in progress (``status`` is ``dialing`` or ``progressing``).
success     The campaigncall ended successfully. The outdialtarget's status is set to ``done`` and no retry will be made.
fail        The campaigncall ended unsuccessfully. The outdialtarget's status is set to ``idle`` and a retry will be scheduled if retries remain per the outplan configuration.
=========== ============

.. note:: **AI Implementation Hint**

   Only a ``normal`` call hangup reason maps to ``success``. All other hangup reasons (busy, no_answer, failed, etc.) map to ``fail``. A ``fail`` result triggers a retry if the outdialtarget has not exceeded its ``max_try_count`` for the current destination index.

The call hangup reason - result mapping table.

================== ============
Call hangup reason Calculated result
================== ============
normal             success
All others         fail
================== ============
