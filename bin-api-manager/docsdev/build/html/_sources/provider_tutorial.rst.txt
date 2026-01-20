.. _provider-tutorial:

Tutorial
========

Get list of providers
---------------------

Example

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/providers?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
                "type": "sip",
                "hostname": "sip.telnyx.com",
                "tech_prefix": "",
                "tech_postfix": "",
                "tech_headers": {},
                "name": "telnyx basic",
                "detail": "telnyx basic",
                "tm_create": "2022-10-22 16:16:16.874761",
                "tm_update": "2022-10-24 04:53:14.171374",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
            ...
        ],
        "next_page_token": "2022-10-22 16:16:16.874761"
    }

Get detail of provider
----------------------

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


Create a new provider
---------------------

Example

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/providers?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "type": "sip",
            "hostname": "test.com",
            "tech_prefix": "",
            "tech_postfix": "",
            "tech_headers": {},
            "name": "test domain",
            "detail": "test domain creation"
        }'


Update provider
--------------------------

Example

.. code::

    $ curl --location --request PUT 'https://api.voipbin.net/v1.0/providers/4dbeabd6-f397-4375-95d2-a38411e07ed1?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=<YOUR_AUTH_TOKEN>' \
        --data-raw '{
            "type": "sip",
            "hostname": "sip.telnyx.com",
            "tech_prefix": "",
            "tech_postfix": "",
            "tech_headers": {},
            "name": "telnyx basic",
            "detail": "telnyx basic"
        }'


Delete provider
---------------

Example

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/providers/7efc9379-2d3e-4e54-9d36-23cff676a83e?token=<YOUR_AUTH_TOKEN>'

