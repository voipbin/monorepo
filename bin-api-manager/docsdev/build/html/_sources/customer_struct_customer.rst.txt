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

* ``id`` (UUID): The customer's unique identifier. Returned when creating a customer via ``POST /customers`` or when listing customers via ``GET /customers``.
* ``username`` (String): The customer's login username. Must be unique across the platform.
* ``name`` (String): The display name of the customer organization.
* ``detail`` (String): An optional description or notes about the customer account.
* ``webhook_method`` (enum string): The HTTP method used for webhook notifications. Typically ``POST``.
* ``webhook_uri`` (String): The URI where webhook event notifications are sent for this customer.
* ``line_secret`` (String): The LINE messaging platform channel secret for LINE integration.
* ``line_token`` (String): The LINE messaging platform channel access token for LINE integration.
* ``permission_ids`` (Array of UUID): List of default permission IDs for this customer's resources. Each ID is obtained from ``GET /permissions``.
* ``tm_create`` (string, ISO 8601): Timestamp when the customer was created.
* ``tm_update`` (string, ISO 8601): Timestamp when the customer was last updated.
* ``tm_delete`` (string, ISO 8601): Timestamp when the customer was deleted, if applicable.

.. note:: **AI Implementation Hint**

   A ``tm_delete`` value of ``9999-01-01 00:00:00.000000`` indicates the customer has not been deleted. The ``line_secret`` and ``line_token`` fields are only needed if the customer uses LINE messaging integration; they can be left empty for voice-only customers.

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
