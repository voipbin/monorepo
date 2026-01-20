.. _route-tutorial:

Tutorial
========

Get list of routes
---------------------

Example

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/routes?customer_id=00000000-0000-0000-0000-000000000001&token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "491b6858-5357-11ed-b753-8fd49cd36340",
                "customer_id": "00000000-0000-0000-0000-000000000001",
                "provider_id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
                "priority": 1,
                "target": "all",
                "tm_create": "2022-10-22 16:16:16.874761",
                "tm_update": "2022-10-22 16:16:16.874761",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2022-10-22 16:16:16.874761"
    }

Get detail of route
---------------------

Example

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/routes/dccfd81b-5f11-4c49-8e3f-70730ef1a4d3?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "dccfd81b-5f11-4c49-8e3f-70730ef1a4d3",
        "customer_id": "00000000-0000-0000-0000-000000000001",
        "provider_id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
        "priority": 1,
        "target": "+82",
        "tm_create": "2022-10-26 11:41:18.124909",
        "tm_update": "2022-10-29 14:50:33.477405",
        "tm_delete": "2022-10-29 14:50:33.477405"
    }

Create a new route
---------------------

Example

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/routes?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "customer_id": "00000000-0000-0000-0000-000000000001",
        "provider_id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
        "priority": 1,
        "target": "+82"
    }'

    {
        "id": "b972b61c-59d2-4217-8fbb-a32304be5c3b",
        "customer_id": "00000000-0000-0000-0000-000000000001",
        "provider_id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
        "priority": 1,
        "target": "+82",
        "tm_create": "2022-11-02 15:36:32.174346",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


Update route
------------

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

    {
        "id": "b972b61c-59d2-4217-8fbb-a32304be5c3b",
        "customer_id": "00000000-0000-0000-0000-000000000001",
        "provider_id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
        "priority": 1,
        "target": "+82",
        "tm_create": "2022-11-02 15:36:32.174346",
        "tm_update": "2022-11-02 15:43:09.190169",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }


Delete route
---------------

Example

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/routes/b972b61c-59d2-4217-8fbb-a32304be5c3b?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "b972b61c-59d2-4217-8fbb-a32304be5c3b",
        "customer_id": "00000000-0000-0000-0000-000000000001",
        "provider_id": "4dbeabd6-f397-4375-95d2-a38411e07ed1",
        "priority": 1,
        "target": "+82",
        "tm_create": "2022-11-02 15:36:32.174346",
        "tm_update": "2022-11-02 15:44:09.686612",
        "tm_delete": "2022-11-02 15:44:09.686612"
    }

