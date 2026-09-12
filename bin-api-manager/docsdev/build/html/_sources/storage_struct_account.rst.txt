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

* ``id`` (UUID): The storage account's unique identifier. Returned when listing storage accounts via ``GET /storage_accounts``.
* ``customer_id`` (UUID): The customer who owns this storage account. Obtained from ``GET /customers``.
* ``total_file_count`` (Integer): Total number of files stored in this account.
* ``total_file_size`` (Integer): Total size of all files in bytes. Divide by 1,073,741,824 to get size in GB.
* ``tm_create`` (string, ISO 8601): Timestamp when the storage account was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update (e.g., file upload or deletion changed totals).
* ``tm_delete`` (string, ISO 8601): Timestamp when the storage account was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the storage account has not been deleted. The ``total_file_size`` is in bytes; compare against the quota (default 10 GB = 10,737,418,240 bytes) to check remaining capacity.

Example
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
