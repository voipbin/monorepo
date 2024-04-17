.. _domain-tutorial: domain-tutorial

Tutorial
========

Get list of domains
-------------------

Gets the list of registered domains.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/domains?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQzMzE0OTgsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.Gg1WsbrKDnQQh7Pvi5y5CV51NVQBz7pgU_T9TxshXPs'

    {
        "result": [
            {
                "id": "5a8437de-abc4-4685-a532-3c494d50c065",
                "user_id": 1,
                "domain_name": "test1.sip.voipbin.net",
                "name": "test domain example",
                "detail": "test domain creation example",
                "tm_create": "2021-02-22 05:02:04.591070",
                "tm_update": "",
                "tm_delete": ""
            },
            ...
        ],
        "next_page_token": "2021-02-15 02:34:29.592973"
    }

Get detail of specified domain
------------------------------

Gets the detail of registered domain.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/domains/5a8437de-abc4-4685-a532-3c494d50c065?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQzMzE0OTgsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.Gg1WsbrKDnQQh7Pvi5y5CV51NVQBz7pgU_T9TxshXPs'

    {
        "id": "5a8437de-abc4-4685-a532-3c494d50c065",
        "user_id": 1,
        "domain_name": "test1.sip.voipbin.net",
        "name": "test domain example",
        "detail": "test domain creation example",
        "tm_create": "2021-02-22 05:02:04.591070",
        "tm_update": "",
        "tm_delete": ""
    }


Create a domain
---------------

Create a new domain.

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/domains?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQzMzE0OTgsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.Gg1WsbrKDnQQh7Pvi5y5CV51NVQBz7pgU_T9TxshXPs' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "name": "test domain",
        "detail": "test domain creation",
        "domain_name": "test2.sip.voipbin.net"
    }'

    {
        "id": "8a66a234-67d6-4722-9482-ac7866708994",
        "user_id": 1,
        "domain_name": "test2.sip.voipbin.net",
        "name": "test domain",
        "detail": "test domain creation",
        "tm_create": "2021-02-23 01:47:34.852092",
        "tm_update": "",
        "tm_delete": ""
    }

Update the domain
-----------------

Update the existed domain with given info.

.. code::

    $ curl -k --location --request PUT 'https://api.voipbin.net/v1.0/domains/8a66a234-67d6-4722-9482-ac7866708994?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQzMzE0OTgsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.Gg1WsbrKDnQQh7Pvi5y5CV51NVQBz7pgU_T9TxshXPs' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "name": "update name",
        "detail": "update detail"
    }'

    {
        "id": "8a66a234-67d6-4722-9482-ac7866708994",
        "user_id": 1,
        "domain_name": "test2.sip.voipbin.net",
        "name": "update name",
        "detail": "update detail",
        "tm_create": "2021-02-23 01:47:34.852092",
        "tm_update": "2021-02-23 01:48:41.521605",
        "tm_delete": ""
    }

Delete the domain
-----------------

Delete the existed domain of given domain id.
This will deletes all of related extensions of the domain.

.. code::

    $ curl -k --location --request DELETE 'https://api.voipbin.net/v1.0/domains/8a66a234-67d6-4722-9482-ac7866708994?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQzMzE0OTgsInVzZXIiOnsiaWQiOjEsInBlcm1pc3Npb24iOjEsInVzZXJuYW1lIjoiYWRtaW4ifX0.Gg1WsbrKDnQQh7Pvi5y5CV51NVQBz7pgU_T9TxshXPs'

