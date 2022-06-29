.. _customer-struct-customer:

Customer
========

.. _customer-struct-customer-customer:

Customer
--------

.. code::

    {
        "id": "<string>",
        "username": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "webhook_method": "<string>",
        "webhook_uri": "<string>",
        "line_secret": "<string>",
        "line_token": "<string>",
        "permission_ids": [
            "<string>",
            ...
        ],
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* id: Customer's ID.
* username: Customer's username.
* name: Name.
* detail: Detail.
* webhook_method: Webhook method.
* webhook_uri: Webhook URI.
* line_secret: Line's secret.
* line_token: Line's token.
* permission_ids: List of permission ids.

Example
+++++++

.. code::

    {
        "id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "username": "admin",
        "name": "admin",
        "detail": "admin account",
        "webhook_method": "POST",
        "webhook_uri": "https://en7evajwhmqbt.x.pipedream.net",
        "line_secret": "ba5f0575d826d5b4a052a43145ef1391",
        "line_token": "tsfIiDB/2cGI5sHRMIop7S3SS4KsbElJ/ukQKs6LpHY1XoG2pTMHqdiyLNu8aMda2pi3vTXscCKp8XGEvfl6dmIT1nfTTdMkmY84iRLIOIAl85iG/XZueI1WBRvchfV8TlZwDmECbSSzL+Wuv+jO+gdB04t89/1O/w1cDnyilFU=",
        "permission_ids": [
            "03796e14-7cb4-11ec-9dba-e72023efd1c6"
        ],
        "tm_create": "2022-02-01 00:00:00.000000",
        "tm_update": "2022-06-16 08:37:16.952738",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }
