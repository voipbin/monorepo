.. _number-tutorial: number-tutorial

Tutorial
========

Get list of available numbers
-----------------------------

Example

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/available_numbers?token=<YOUR_AUTH_TOKEN>&country_code=US&page_size=5'

    {
    "result": [
        {
            "number": "+12182558711",
            "country": "US",
            "region": "MN",
            "postal_code": "",
            "features": [
                "emergency",
                "fax",
                "voice",
                "sms",
                "mms"
            ]
        },
        ...
    ]



Get list of available virtual numbers
-------------------------------------

Example

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/available_numbers?token=<YOUR_AUTH_TOKEN>&type=virtual&page_size=5'

    {
    "result": [
        {
            "number": "+899100000001",
            "country": "",
            "region": "",
            "postal_code": "",
            "features": []
        },
        ...
    ]


Create virtual number
---------------------

Virtual numbers use the +899 prefix and do not require a provider purchase.

Example

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/numbers?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "number": "+899100000001"
    }'

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "number": "+899100000001",
        "type": "virtual",
        "call_flow_id": "00000000-0000-0000-0000-000000000000",
        "message_flow_id": "00000000-0000-0000-0000-000000000000",
        "name": "",
        "detail": "",
        "status": "active",
        "t38_enabled": false,
        "emergency_enabled": false,
        "tm_create": "2024-01-15 10:30:00.000000",
        "tm_update": "",
        "tm_delete": ""
    }


Get list of numbers
-------------------------

Example

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/numbers?token=<YOUR_AUTH_TOKEN>&page_size=10'

    {
        "result": [
            {
                "id": "b7ee1086-fcbc-4f6f-96e5-7f9271e25279",
                "number": "+16062067563",
                "flow_id": "00000000-0000-0000-0000-000000000000",
                "status": "purchase-pending",
                "t38_enabled": true,
                "emergency_enabled": false,
                "tm_purchase": "2021-03-03 06:34:09.000000",
                "tm_create": "2021-03-03 06:34:09.733751",
                "tm_update": "",
                "tm_delete": ""
            },
            {
                "id": "d5532488-0b2d-11eb-b18c-172ab8f2d3d8",
                "number": "+16195734778",
                "flow_id": "decc2634-0b2a-11eb-b38d-87a8f1051188",
                "status": "active",
                "t38_enabled": false,
                "emergency_enabled": false,
                "tm_purchase": "",
                "tm_create": "2020-10-11 01:00:00.000001",
                "tm_update": "",
                "tm_delete": ""
            }
        ],
        "next_page_token": "2020-10-11 01:00:00.000001"
    }


Get detail of number
--------------------

Example

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/numbers/d5532488-0b2d-11eb-b18c-172ab8f2d3d8?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "d5532488-0b2d-11eb-b18c-172ab8f2d3d8",
        "number": "+16195734778",
        "flow_id": "decc2634-0b2a-11eb-b38d-87a8f1051188",
        "status": "active",
        "t38_enabled": false,
        "emergency_enabled": false,
        "tm_purchase": "",
        "tm_create": "2020-10-11 01:00:00.000001",
        "tm_update": "",
        "tm_delete": ""
    }

Delete number
-------------

Example

.. code::

    $ curl -k --location --request DELETE 'https://api.voipbin.net/v1.0/numbers/b7ee1086-fcbc-4f6f-96e5-7f9271e25279?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "b7ee1086-fcbc-4f6f-96e5-7f9271e25279",
        "number": "+16062067563",
        "flow_id": "00000000-0000-0000-0000-000000000000",
        "status": "deleted",
        "t38_enabled": true,
        "emergency_enabled": false,
        "tm_purchase": "2021-03-03 06:34:09.000000",
        "tm_create": "2021-03-03 06:34:09.733751",
        "tm_update": "2021-03-03 06:52:53.848439",
        "tm_delete": "2021-03-03 06:52:53.848439"
    }

Create number
-------------

Example

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/numbers?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "number": "+16062067563"
    }'

    {
        "id": "b7ee1086-fcbc-4f6f-96e5-7f9271e25279",
        "number": "+16062067563",
        "flow_id": "00000000-0000-0000-0000-000000000000",
        "status": "active",
        "t38_enabled": true,
        "emergency_enabled": false,
        "tm_purchase": "2021-03-03 18:41:23.000000",
        "tm_create": "2021-03-03 18:41:24.657788",
        "tm_update": "",
        "tm_delete": ""
    }
