.. _storage-struct-file:

File
=========

.. _storage-struct-file-file:

File
---------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "owner_id": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "filename": "<string>",
        "filesize": <number>,
        "uri_download": "<string>",
        "tm_download_expire": "<string>",
        "tm_create": "2024-05-27 07:55:16.515315",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

* ``id`` (UUID): The file's unique identifier. Returned when uploading a file via ``POST /files`` or listing via ``GET /files``.
* ``customer_id`` (UUID): The customer who owns this file. Obtained from ``GET /customers``.
* ``owner_id`` (UUID): The agent who uploaded or owns this file. Obtained from ``GET /agents``. Empty string if no specific owner.
* ``reference_type`` (enum string): The type of resource this file is associated with. See :ref:`Reference Type <storage-struct-file-reference-type>`.
* ``reference_id`` (UUID): The ID of the associated resource. Depending on ``reference_type``, obtained from ``GET /recordings`` or other endpoints. Set to ``00000000-0000-0000-0000-000000000000`` if no reference.
* ``name`` (String): A human-readable name for the file. May be empty if not explicitly set.
* ``detail`` (String): A longer description of the file's purpose or content. May be empty.
* ``filename`` (String): The original filename of the uploaded file (e.g., ``screenshot.png``, ``call_recording.wav``).
* ``filesize`` (Integer): The file size in bytes.
* ``uri_download`` (String): A time-limited signed URL for downloading the file. Check ``tm_download_expire`` before using. Fetch fresh details via ``GET /files/{id}`` if expired.
* ``tm_download_expire`` (string, ISO 8601): Expiration time of the ``uri_download``. After this time, the URL is no longer valid.
* ``tm_create`` (string, ISO 8601): Timestamp when the file was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any file property. Set to ``9999-01-01 00:00:00.000000`` if never updated after creation.
* ``tm_delete`` (string, ISO 8601): Timestamp when the file was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the file has not been deleted. The ``uri_download`` is a time-limited signed URL; always check ``tm_download_expire`` before using it. If expired, call ``GET /files/{id}`` to obtain a fresh URL.

Example
+++++++

.. code::

    {
        "id": "0c0c305c-7a55-4395-85a8-ae4860f01393",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "owner_id": "62005165-7592-4ff7-9076-55bf491023f2",
        "reference_type": "",
        "reference_id": "00000000-0000-0000-0000-000000000000",
        "name": "",
        "detail": "",
        "filename": "Screenshot from 2024-05-26 21-58-51.png",
        "filesize": 110881,
        "uri_download": "https://example.com/storage_url",
        "tm_download_expire": "2034-05-25 07:55:16.115684",
        "tm_create": "2024-05-27 07:55:16.515315",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _storage-struct-file-reference-type:

Reference Type
--------------

All possible values for the ``reference_type`` field:

============== ==============================
Reference Type Description
============== ==============================
none           The file has no associated resource. Typically for manually uploaded files. The ``reference_id`` will be ``00000000-0000-0000-0000-000000000000``.
recording      The file is associated with a recording. The ``reference_id`` is a recording ID from ``GET /recordings``.
============== ==============================
