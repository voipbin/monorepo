.. _outplan-tutorial:

Tutorial
========

Get list of outplans
--------------------

Example

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/outplans?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e",
                "name": "test outplan",
                "detail": "outplan for test use.",
                "source": {
                    "type": "tel",
                    "target": "+15559876543",
                    "target_name": "",
                    "name": "",
                    "detail": ""
                },
                "dial_timeout": 30000,
                "try_interval": 60000,
                "max_try_count_0": 5,
                "max_try_count_1": 5,
                "max_try_count_2": 5,
                "max_try_count_3": 5,
                "max_try_count_4": 5,
                "tm_create": "2022-04-28 01:50:23.414000",
                "tm_update": "2022-04-30 12:01:13.780469",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2022-04-28 01:50:23.414000"
    }

Get detail of outplan
---------------------

Example

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/outplans/d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e",
        "name": "test outplan",
        "detail": "outplan for test use.",
        "source": {
            "type": "tel",
            "target": "+15559876543",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "dial_timeout": 30000,
        "try_interval": 60000,
        "max_try_count_0": 5,
        "max_try_count_1": 5,
        "max_try_count_2": 5,
        "max_try_count_3": 5,
        "max_try_count_4": 5,
        "tm_create": "2022-04-28 01:50:23.414000",
        "tm_update": "2022-04-30 12:01:13.780469",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


Create a new outplan
--------------------

Example

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/outplans?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "test outplan",
            "detail": "outplan for test use.",
            "source": {
                "type": "tel",
                "target": "+15559876543"
            },
            "dial_timeout": 30000,
            "try_interval": 600000,
            "max_try_count_0": 5,
            "max_try_count_1": 5,
            "max_try_count_2": 5,
            "max_try_count_3": 5,
            "max_try_count_4": 5
        }'

Update outplan's dial info
--------------------------

Example

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/outplans/d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e/dial_info?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "+15559876543"
            },
            "dial_timeout": 30000,
            "try_interval": 60000,
            "max_try_count_0": 5,
            "max_try_count_1": 5,
            "max_try_count_2": 5,
            "max_try_count_3": 5,
            "max_try_count_4": 5
        }'

Delete outplan
--------------

Example

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/outplans/88334a03-bc6b-40b6-878f-46df2d9865db?token=<YOUR_AUTH_TOKEN>'

Update outplan's basic info
---------------------------

Example

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/outplans/d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "test outplan",
            "detail": "outplan for test use"
        }'

    {
        "id": "d5fb7357-7ddb-4f2d-87b5-8ccbfd6c039e",
        "name": "test outplan",
        "detail": "outplan for test use",
        "source": {
            "type": "tel",
            "target": "+15559876543",
            "target_name": "",
            "name": "",
            "detail": ""
        },
        "dial_timeout": 30000,
        "try_interval": 60000,
        "max_try_count_0": 5,
        "max_try_count_1": 5,
        "max_try_count_2": 5,
        "max_try_count_3": 5,
        "max_try_count_4": 5,
        "tm_create": "2022-04-28 01:50:23.414000",
        "tm_update": "2022-05-02 05:59:44.290658",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }
