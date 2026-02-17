.. _customer-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before managing customers, you need:

* An authentication token with admin permissions. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* (For creation) A unique ``username`` and ``password`` for the new customer account.
* (Optional) A webhook URI to receive event notifications for the customer.

.. note:: **AI Implementation Hint**

   Customer creation requires admin-level permissions. Regular agents cannot create or delete customers. The ``password`` field is required when creating a customer but is write-only and never returned in API responses. When a customer is created, a guest agent with admin permissions is automatically created for that account.

Get list of customers
----------------------

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/customers?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "username": "admin",
                "name": "admin",
                "detail": "admin account",
                "webhook_method": "POST",
                "webhook_uri": "https://en7evasdjwhmqbt.x.pipedream.net",
                "line_secret": "ba5fsf0575d826d5b4asdf052a43145ef1391",
                "line_token": "tsfIiDB/2cGI5sHRMIop7S3SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdisyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTsdfsfTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
                "permission_ids": [
                    "03796e14-7cb4-11ec-9dba-e72023efd1c6"
                ],
                "tm_create": "2022-02-01 00:00:00.000000",
                "tm_update": "2022-06-16 08:37:16.952738",
                "tm_delete": "9999-01-01 00:00:00.000000"
            }
        ],
        "next_page_token": "2022-02-01 00:00:00.000000"
    }

Get detail of customer
----------------------

Example

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/customers/5e4a0680-804e-11ec-8477-2fea5968d85b?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "username": "admin",
        "name": "admin",
        "detail": "admin account",
        "webhook_method": "POST",
        "webhook_uri": "https://en7evajwhmqbt.x.pipedream.net",
        "line_secret": "ba5f0575d826d5b4a0512a145ef1391",
        "line_token": "tsfIiDB/2cGI5sssaIop7S3SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
        "permission_ids": [
            "03796e14-7cb4-11ec-9dba-e72023efd1c6"
        ],
        "tm_create": "2022-02-01 00:00:00.000000",
        "tm_update": "2022-06-16 08:37:16.952738",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Create a new customer
---------------------

Example

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/customers?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "username": "test1",
            "password": "ee5f3d14-5ac6-11ed-808e-6f7d676a444b",
            "name": "test 1",
            "detail": "test user 1",
            "webhook_method": "POST",
            "webhook_uri": "https://en7evajwhm11qbt.x.pipedream.net",
            "line_secret": "ba5f0575d826d5b4a051112a145ef1391",
            "line_token": "tsfIiDB/2cGI5sssaIop7S311SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
            "permission_ids": []
        }'

    {
        "id": "ff424526-f65d-483f-bc36-3b2357c6c6a9",
        "username": "test1",
        "name": "test 1",
        "detail": "test user 1",
        "webhook_method": "POST",
        "webhook_uri": "https://en7evajwhm11qbt.x.pipedream.net",
        "line_secret": "ba5f0575d826d5b4a051112a145ef1391",
        "line_token": "tsfIiDB/2cGI5sssaIop7S311SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
        "permission_ids": [],
        "tm_create": "2022-11-02 15:57:08.368093",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Delete customer
---------------

.. note:: **AI Implementation Hint**

   Deleting a customer is a destructive operation that removes the customer account and all associated resources (agents, numbers, flows, etc.). This cannot be undone. The API returns the deleted customer object with ``tm_delete`` set to the deletion timestamp.

Example

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/customers/ff424526-f65d-483f-bc36-3b2357c6c6a9?token=<YOUR_AUTH_TOKEN>' \

    {
        "id": "ff424526-f65d-483f-bc36-3b2357c6c6a9",
        "username": "test1",
        "name": "test 1",
        "detail": "test user 1",
        "webhook_method": "POST",
        "webhook_uri": "https://en7evajwhm11qbt.x.pipedream.net",
        "line_secret": "ba5f0575d826d5b4a051112a145ef1391",
        "line_token": "tsfIiDB/2cGI5sssaIop7S311SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
        "permission_ids": [],
        "tm_create": "2022-11-02 15:57:08.368093",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "2022-11-02 15:59:08.368093"
    }
