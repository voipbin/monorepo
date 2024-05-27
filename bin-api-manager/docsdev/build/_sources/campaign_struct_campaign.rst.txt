.. _campaign-struct-campaign:

Campaign
===============

.. _campaign-struct-campaign-campaign:

Campaign
--------

.. code::

    {
        "id": "<string>",
        "type": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "status": "<string>",
        "service_level": <number>,
        "end_handle": "<string>",
        "actions": [
            ...
        ],
        "outplan_id": "<string>",
        "outdial_id": "<string>",
        "queue_id": "<string>",
        "next_campaign_id": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* id: Campaign's ID.
* *type*: Campaign's type. See detail :ref:`here <campaign-struct-campaign-type>`.
* name: Campaign's name.
* detail: Campaign's detail.
* *status*: Campaign's status. See detail :ref:`here <campaign-struct-campaign-status>`.
* *service_level*: Campaign's service level. See detail :ref:`here <campaign-struct-campaign-service_level>`.
* *end_handle*: Campaign's outdial list end handle. See detail :ref:`here <campaign-struct-campaign-end_handle>`.
* *actions*: Campaign's list of actions. See detail :ref:`here <flow-struct-action-action>`.
* outplan_id: Outplan's ID.
* outdial_id: Outdial's ID.
* queue_id: Queue's ID.
* next_campaign_id: Next campaign's ID.

Example
+++++++

.. code::

    {
        "id": "183c0d5c-691e-42f3-af2b-9bffc2740f83",
        "type": "call",
        "name": "test campaign",
        "detail": "test campaign detail",
        "status": "stop",
        "service_level": 100,
        "end_handle": "stop",
        "actions": [
            {
                "id": "00000000-0000-0000-0000-000000000000",
                "next_id": "00000000-0000-0000-0000-000000000000",
                "type": "talk",
                "option": {
                    "text": "Hello. This is outbound campaign's test calling. Please wait until the agent answer the call. Thank you.",
                    "gender": "female",
                    "language": "en-US"
                }
            }
        ],
        "outplan_id": "d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e",
        "outdial_id": "40bea034-1d17-474d-a5de-da00d0861c69",
        "queue_id": "99bf739a-932f-433c-b1bf-103d33d7e9bb",
        "next_campaign_id": "00000000-0000-0000-0000-000000000000",
        "tm_create": "2022-04-28 02:16:39.712142",
        "tm_update": "2022-04-30 17:53:51.685259",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _campaign-struct-campaign-type:

Type
----
Campaign's type.

=========== ============
Type        Description
=========== ============
call        The campaign will make a call to the destination with a flow.
flow        The campaign will execute flow with a destination.
=========== ============

.. _campaign-struct-campaign-status:

Status
------
Campaign's status.

=========== ============
Type        Description
=========== ============
stop        The campaign stopped.
stopping    The campaign is being stop. Waiting for dialing/process call's termination.
run         The campaign is running. It will create a new call or flow execution.
=========== ============

.. _campaign-struct-campaign-service_level:

Service level
-------------
The service level control the amount of campaigncalls. It appects to the campaign's campaigncall creation.

The campaign makes a new campaigncall when...

.. code::

    Available agent > Current dialing campaign calls * Service level / 100

It valid only if the campaign has a valid queue_id.

.. _campaign-struct-campaign-end_handle:

End handle
----------
Campaign's outdial list end handle.

=========== ============
Type        Description
=========== ============
stop        The campaign will stop if the outdial has no more outdial target
continue    The campaign will continue to run after outdial has no more outdial target.
=========== ============

