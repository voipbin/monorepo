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

* id: Campaigncall's ID.
* campaign_id: Campaign's ID.
* outplan_id: Outplan's ID.
* outdial_id: Outdial's ID.
* outdial_target_id: outdialtarget's ID.
* queue_id: Queue's ID.
* activeflow_id: Activeflow's ID.
* *reference_type*: Reference's type. See detail :ref:`here <campaign-struct-campaigncall-reference_type>`.
* reference_id: Reference's ID.
* *status*: Campaigncall's status. See detail :ref:`here <campaign-struct-campaigncall-status>`.
* *result*: Campaigncall's result. See detail :ref:`here <campaign-struct-campaigncall-result>`.
* *source*: Source address info. See detail :ref:`here <common-struct-address-address>`.
* *destination*: Destination address info. See detail :ref:`here <common-struct-address-address>`.
* destination_index: Destination's index.
* try_count: Try count.

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
        "destination_index": 0,
        "try_count": 1,
        "tm_create": "2022-04-29 07:01:45.808944",
        "tm_update": "2022-04-29 07:02:48.304704"
    }

.. _campaign-struct-campaigncall-reference_type:

Reference type
--------------
Campaigncall's reference type.

=========== ============
Type        Description
=========== ============
none        Has no reference type.
call        The reference type is call. Reference id is call's ID.
=========== ============

.. _campaign-struct-campaigncall-status:

Status
------
Campaigncall's status.

=========== ============
Type        Description
=========== ============
dialing     The campaigncall is dialing(not answered yet)
progressing The campaigncall is progressing(the call answered)
done        The campaigncall is hungup
=========== ============

.. _campaign-struct-campaigncall-result:

Result
------
Campaigncall's result. The result is calculated by the final status/result of the referenced resource(call/sms/...).

For example, if the call ended with no_answer, the result will be calculated to the fail.

=========== ============
Type        Description
=========== ============
""          Have no result yet.
success     The campaigncall ended successfully. The target's status will be set to the done and will not make retry.
fail        The campaigncall ended unsuccesfully. The target's status will be set to the idle and will make a retry.
=========== ============

The call hangup reason - result mapping table.

================== ============
Call hangup reason Calculated result
================== ============
normal             success
All others         fail
================== ============
