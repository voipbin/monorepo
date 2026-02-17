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

* ``id`` (UUID): The campaign's unique identifier. Returned when creating via ``POST /campaigns`` or listing via ``GET /campaigns``.
* ``type`` (enum string): Campaign's type. See :ref:`Type <campaign-struct-campaign-type>`.
* ``name`` (String): Human-readable name for the campaign.
* ``detail`` (String): Detailed description of the campaign.
* ``status`` (enum string): Campaign's current status. See :ref:`Status <campaign-struct-campaign-status>`.
* ``service_level`` (Integer): Campaign's service level percentage. Controls the dialing rate relative to available agents. See :ref:`Service Level <campaign-struct-campaign-service_level>`.
* ``end_handle`` (enum string): What happens when the outdial target list is exhausted. See :ref:`End Handle <campaign-struct-campaign-end_handle>`.
* ``actions`` (Array of Object): List of flow actions executed when a target answers. Each action follows the :ref:`Action <flow-struct-action-action>` structure.
* ``outplan_id`` (UUID): The outplan controlling dialing strategy. Obtained from the ``id`` field of ``GET /outplans``. Set to ``00000000-0000-0000-0000-000000000000`` if not assigned.
* ``outdial_id`` (UUID): The outdial containing target destinations. Obtained from the ``id`` field of ``GET /outdials``. Set to ``00000000-0000-0000-0000-000000000000`` if not assigned.
* ``queue_id`` (UUID): The queue for routing answered calls to agents. Obtained from the ``id`` field of ``GET /queues``. Set to ``00000000-0000-0000-0000-000000000000`` if not assigned.
* ``next_campaign_id`` (UUID): The campaign to chain after this one finishes. Obtained from the ``id`` field of ``GET /campaigns``. Set to ``00000000-0000-0000-0000-000000000000`` if not assigned.
* ``tm_create`` (string, ISO 8601): Timestamp when the campaign was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any campaign property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the campaign was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   A ``tm_delete`` value of ``9999-01-01 00:00:00.000000`` means the resource has **not** been deleted. This is a sentinel value, not a real timestamp. When filtering active resources, check for this value.

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
Campaign's type. Determines how the campaign communicates with targets.

=========== ============
Type        Description
=========== ============
call        The campaign will make a voice call to each target destination and execute the configured flow actions upon answer.
flow        The campaign will execute the configured flow actions directed at each target destination without an explicit voice call setup.
=========== ============

.. _campaign-struct-campaign-status:

Status
------
Campaign's current operational status. Use ``PUT /campaigns/{id}`` with ``{"status": "run"}`` to start and ``{"status": "stop"}`` to stop the campaign.

=========== ============
Type        Description
=========== ============
stop        The campaign is stopped. No dialing is occurring. This is the initial state after creation.
stopping    The campaign is transitioning to stopped. Active calls are being terminated before the campaign fully stops. This is a transient state.
run         The campaign is actively running. It will create new calls or flow executions based on the outplan and outdial configuration.
=========== ============

.. _campaign-struct-campaign-service_level:

Service level
-------------
The service level controls the amount of campaigncalls. It affects the campaign's campaigncall creation.

The campaign creates a new campaigncall when the following condition is met:

.. code::

    Available agent > Current dialing campaign calls * Service level / 100

This is valid only if the campaign has a valid queue_id.

.. _campaign-struct-campaign-end_handle:

End handle
----------
Determines what the campaign does when all targets in the outdial have been attempted.

=========== ============
Type        Description
=========== ============
stop        The campaign will transition to stopped status when the outdial has no more targets to dial. This is the typical setting for one-time campaigns.
continue    The campaign will remain in running status after all outdial targets have been attempted. Useful for campaigns where new targets may be added to the outdial dynamically.
=========== ============

