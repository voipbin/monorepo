.. _outdial-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with outdials, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* Target phone numbers in E.164 format (e.g., ``+15551234567``) for telephone destinations.
* (Optional) SIP URIs for SIP destinations or email addresses for email campaigns.

.. note:: **AI Implementation Hint**

   Outdialtargets support up to 5 destinations per target (``destination_0`` through ``destination_4``). The campaign dials destinations in order, moving to the next only when all retries on the current destination are exhausted. Phone numbers must be in E.164 format with the ``+`` prefix.

Get list of outdials
--------------------

Example

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/outdials?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "40bea034-1d17-474d-a5de-da00d0861c69",
                "campaign_id": "00000000-0000-0000-0000-000000000000",
                "name": "test outdial",
                "detail": "outdial for test use.",
                "data": "",
                "tm_create": "2022-04-28 01:41:40.503790",
                "tm_update": "9999-01-01 00:00:00.000000",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2022-04-28 01:41:40.503790"
    }

Get a detail of outdial
-----------------------

.. code::

    curl --location --request GET 'https://api.voipbin.net/v1.0/outdials/40bea034-1d17-474d-a5de-da00d0861c69?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "40bea034-1d17-474d-a5de-da00d0861c69",
        "campaign_id": "00000000-0000-0000-0000-000000000000",
        "name": "test outdial",
        "detail": "outdial for test use.",
        "data": "",
        "tm_create": "2022-04-28 01:41:40.503790",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Create a new outdial
---------------------

Example

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/outdials?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=<YOUR_AUTH_TOKEN>' \
        --data-raw '{
            "name": "test outdial",
            "detail": "outdial for test use.",
            "data": "test data"
        }'


Create a new outdialtarget
--------------------------

Example

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/outdials/40bea034-1d17-474d-a5de-da00d0861c69/targets?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=<YOUR_AUTH_TOKEN>' \
        --data-raw '{
            "name": "test destination 0",
            "detail": "test detatination 0 detail",
            "data": "test data",
            "destination_0": {
                "type": "tel",
                "target": "+15559876543"
            }
        }'

    {
        "id": "1b3d7a92-7146-466d-90f5-4bc701ada4c0",
        "outdial_id": "40bea034-1d17-474d-a5de-da00d0861c69",
        "name": "test destination 0",
        "detail": "test detatination 0 detail",
        "data": "test data",
        "status": "idle",
        "destination_0": {
            "type": "tel",
            "target": "+15559876543",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "destination_1": null,
        "destination_2": null,
        "destination_3": null,
        "destination_4": null,
        "try_count_0": 0,
        "try_count_1": 0,
        "try_count_2": 0,
        "try_count_3": 0,
        "try_count_4": 0,
        "tm_create": "2022-04-30 17:52:16.484341",
        "tm_update": "2022-04-30 17:52:16.484341",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Get list of outdialtargets
--------------------------

Example

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/outdials/40bea034-1d17-474d-a5de-da00d0861c69/targets?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "1b3d7a92-7146-466d-90f5-4bc701ada4c0",
                "outdial_id": "40bea034-1d17-474d-a5de-da00d0861c69",
                "name": "test destination 0",
                "detail": "test detatination 0 detail",
                "data": "test data",
                "status": "done",
                "destination_0": {
                    "type": "tel",
                    "target": "+15559876543",
                    "target_name": "",
                    "name": "",
                    "detail": ""
                },
                "destination_1": null,
                "destination_2": null,
                "destination_3": null,
                "destination_4": null,
                "try_count_0": 1,
                "try_count_1": 0,
                "try_count_2": 0,
                "try_count_3": 0,
                "try_count_4": 0,
                "tm_create": "2022-04-30 17:52:16.484341",
                "tm_update": "2022-04-30 17:53:51.183345",
                "tm_delete": "9999-01-01 00:00:00.000000"
            },
            ...
        ],
        "next_page_token": "2022-04-28 01:44:07.212667"
    }
