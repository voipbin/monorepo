.. _extension-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with extensions, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* An extension number (string) and password for the new SIP device.

.. note:: **AI Implementation Hint**

   When creating an extension, the ``extension`` field and ``username`` field are typically set to the same value. The ``password`` is used for SIP device authentication. After creation, configure SIP devices with the ``username``, ``password``, and the domain ``{customer-id}.registrar.voipbin.net``.

Get list of extensions
----------------------

Gets the list of registered extensions for your account.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/extensions?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "0e7f8158-c770-4930-a98e-f2165b189c1f",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "name": "test domain",
                "detail": "test domain creation",
                "extension": "test11",
                "domain_name": "5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net",
                "username": "test11",
                "password": "bad79bd2-71e6-11eb-9577-c756bf092a88",
                "direct_hash": "",
                "tm_create": "2021-02-18 12:42:27.688282",
                "tm_update": "",
                "tm_delete": ""
            }
        ],
        "next_page_token": "2021-02-18 12:42:27.688282"
    }

Get detail of specified extension
---------------------------------

Gets the detail of registered extension.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/extensions/0e7f8158-c770-4930-a98e-f2165b189c1f?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "0e7f8158-c770-4930-a98e-f2165b189c1f",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "test domain",
        "detail": "test domain creation",
        "extension": "test11",
        "domain_name": "5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net",
        "username": "test11",
        "password": "bad79bd2-71e6-11eb-9577-c756bf092a88",
        "direct_hash": "",
        "tm_create": "2021-02-18 12:42:27.688282",
        "tm_update": "",
        "tm_delete": ""
    }


Create a extension
------------------

Create a new extension.

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/extensions?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "name": "test domain",
        "detail": "test domain creation",
        "extension": "test12",
        "password": "27a4d0f2-757c-11eb-bc8f-4f045857b89c"
    }'

    {
        "id": "6a7934ff-0e1c-4857-857b-23c9e27d267b",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "test domain",
        "detail": "test domain creation",
        "extension": "test12",
        "domain_name": "5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net",
        "username": "test12",
        "password": "27a4d0f2-757c-11eb-bc8f-4f045857b89c",
        "direct_hash": "",
        "tm_create": "2021-02-23 02:09:39.701458",
        "tm_update": "",
        "tm_delete": ""
    }

Update the extension
--------------------

Update the existing extension with given info.

.. code::

    $ curl -k --location --request PUT 'https://api.voipbin.net/v1.0/extensions/6a7934ff-0e1c-4857-857b-23c9e27d267b?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "name": "update test extension name",
        "detail": "update test extension detail",
        "password": "5316382a-757c-11eb-9348-bb32547e99c4"
    }'

    {
        "id": "6a7934ff-0e1c-4857-857b-23c9e27d267b",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "update test extension name",
        "detail": "update test extension detail",
        "extension": "test12",
        "domain_name": "5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net",
        "username": "test12",
        "password": "5316382a-757c-11eb-9348-bb32547e99c4",
        "direct_hash": "",
        "tm_create": "2021-02-23 02:09:39.701458",
        "tm_update": "2021-02-23 02:11:03.992067",
        "tm_delete": ""
    }

Regenerate direct extension hash
---------------------------------

Regenerate the direct extension hash. This invalidates the previous SIP URI and creates a new one. If the extension has no existing direct hash, one is created automatically. Useful when the existing hash has been compromised or shared unintentionally.

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/extensions/6a7934ff-0e1c-4857-857b-23c9e27d267b/direct-hash-regenerate?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "6a7934ff-0e1c-4857-857b-23c9e27d267b",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "update test extension name",
        "detail": "update test extension detail",
        "extension": "test12",
        "domain_name": "5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net",
        "username": "test12",
        "password": "5316382a-757c-11eb-9348-bb32547e99c4",
        "direct_hash": "f7e8d9c0b1a2",
        "tm_create": "2021-02-23 02:09:39.701458",
        "tm_update": "2021-02-23 02:18:45.334567",
        "tm_delete": ""
    }

.. note:: **AI Implementation Hint**

   This endpoint requires no request body. The ``direct_hash`` in the response is the new hash — the previous hash is permanently invalidated. Update any stored SIP URIs that reference the old hash.

Delete the extension
--------------------

Delete the existing extension of given id.

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/extensions/6a7934ff-0e1c-4857-857b-23c9e27d267b?token=<YOUR_AUTH_TOKEN>'

