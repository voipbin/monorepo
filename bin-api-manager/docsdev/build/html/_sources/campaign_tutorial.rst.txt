.. _campaign-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with campaigns, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* An outdial ID (UUID) with at least one target. Create an outdial via ``POST /outdials`` and add targets via ``POST /outdials/{id}/targets``.
* An outplan ID (UUID) defining the dialing strategy. Create one via ``POST /outplans``.
* A queue ID (UUID) with agents assigned (for call-type campaigns that connect to agents). Create one via ``POST /queues``.
* (Optional) A flow ID (UUID) defining call actions. Create one via ``POST /flows``, or define inline ``actions`` in the campaign creation request.

.. note:: **AI Implementation Hint**

   Campaigns are created in ``stop`` status. You must explicitly set the status to ``running`` via ``PUT /campaigns/{id}`` to start dialing. Running a campaign incurs charges for each outbound call or message made. Always verify your outdial targets and outplan settings before starting.

Get list of campaigns
---------------------

Example
+++++++

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/campaigns?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
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
        ],
        "next_page_token": "2022-04-28 02:16:39.712142"
    }


Get detail of campaign
----------------------

Example
+++++++

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/campaigns/183c0d5c-691e-42f3-af2b-9bffc2740f83?token=<YOUR_AUTH_TOKEN>'

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

Create a new campaign
---------------------

Example
+++++++

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/campaigns?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "name": "test campaign",
        "detail": "test campaign detail",
        "type": "call",
        "service_level": 100,
        "end_handle": "stop",
        "actions": [
            {
                "type": "talk",
                "option": {
                    "text": "Hello. This is outbound campaign's test calling. Please wait until the agent answer the call. Thank you.",
                    "language": "en-US"
                }
            }
        ],
        "outplan_id": "d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e",
        "outdial_id": "40bea034-1d17-474d-a5de-da00d0861c69",
        "queue_id": "99bf739a-932f-433c-b1bf-103d33d7e9bb"
    }'

    {
        "id": "c1d2e3f4-a5b6-7890-cdef-123456789012",
        "name": "test campaign",
        "detail": "test campaign detail",
        "type": "call",
        "status": "stop",
        "service_level": 100,
        "end_handle": "stop",
        "outplan_id": "d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e",
        "outdial_id": "40bea034-1d17-474d-a5de-da00d0861c69",
        "queue_id": "99bf739a-932f-433c-b1bf-103d33d7e9bb",
        "tm_create": "2022-10-22 16:16:16.874761",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }
