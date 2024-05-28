.. _storage-struct-account:

Account
=========

.. _storage-struct-account-account:

Account
---------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "total_file_count": <number>,
        "total_file_size": <number>,
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* id: Account's ID.
* customer_id: Customer's ID.
* total_file_count: Total file count.
* total_file_size: Total file size(Byte).

example
+++++++

.. code::

    {
        "id": "1f8ccab9-1b64-11ef-8530-42010a7e5015",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "total_file_count": 2,
        "total_file_size": 221762,
        "tm_create": "2024-05-26 13:30:31.176034",
        "tm_update": "2024-05-27 07:55:17.016150",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }
