.. _campaign_struct:

Struct
======

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
* *type*: Campaign's type. See detail :ref:`here <campaign-struct-type>`.
* name: Campaign's name.
* detail: Campaign's detail.
* *status*: Campaign's status. See detail :ref:`here <campaign-struct-status>`.
* service_level: Campaign's service level.
* *end_handle*: Campaign's outdial list end handle. See detail :ref:`here <campaign-struct-end_handle>`.
* *actions*: Campaign's list of actions. See detail :ref:`here <flow-action-action>`.
* outplan_id: Outplan's ID.
* outdial_id: Outdial's ID.
* queue_id: Queue's ID.
* next_campaign_id: Next campaign's ID.

.. _campaign-struct-type:

Type
----
Campaign's type.

=========== ============
Type        Description
=========== ============
call        The campaign will make a call to the destination with a flow.
flow        The campaign will execute flow with a destination.
=========== ============

.. _campaign-struct-status:

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

.. _campaign-struct-end_handle:

End handle
----------
Campaign's outdial list end handle.

=========== ============
Type        Description
=========== ============
stop        The campaign will stop if the outdial has no more outdial target
continue    The campaign will continue to run after outdial has no more outdial target.
=========== ============

