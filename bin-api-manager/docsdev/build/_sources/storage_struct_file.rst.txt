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

* id: File's ID.
* customer_id: Customer's ID.
* owner_id: Owner(Agent)'s ID. If the owner does not exist, this will be empty.
* reference_type: Reference type.
* reference_id: Reference ID.
* name: File's name.
* detail: File's detail.
* filename: File's filename.
* filesize: File's size(Byte).
* uri_download: File's download URI.
* tm_download_expire: File's download expiration time.

example
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

Type
----

============== ==============================
Reference Type       Description
============== ==============================
none           Has no reference information.
recording      Recording type
============== ==============================
