.. _extension-tutorial: extension-tutorial

Tutorial
========

Get list of extensions
----------------------

Gets the list of registered extensions of the given domain.

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/extensions?token=<YOUR_AUTH_TOKEN>&domain_id=cc6a05eb-33a4-444b-bf8a-359de7d95499'

    {
        "result": [
            {
                "id": "0e7f8158-c770-4930-a98e-f2165b189c1f",
                "user_id": 1,
                "name": "test domain",
                "detail": "test domain creation",
                "domain_id": "cc6a05eb-33a4-444b-bf8a-359de7d95499",
                "extension": "test11",
                "password": "bad79bd2-71e6-11eb-9577-c756bf092a88",
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
        "user_id": 1,
        "name": "test domain",
        "detail": "test domain creation",
        "domain_id": "cc6a05eb-33a4-444b-bf8a-359de7d95499",
        "extension": "test11",
        "password": "bad79bd2-71e6-11eb-9577-c756bf092a88",
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
        "domain_id": "cc6a05eb-33a4-444b-bf8a-359de7d95499",

        "extension": "test12",
        "password": "27a4d0f2-757c-11eb-bc8f-4f045857b89c"
    }'

    {
        "id": "6a7934ff-0e1c-4857-857b-23c9e27d267b",
        "user_id": 1,
        "name": "test domain",
        "detail": "test domain creation",
        "domain_id": "cc6a05eb-33a4-444b-bf8a-359de7d95499",
        "extension": "test12",
        "password": "27a4d0f2-757c-11eb-bc8f-4f045857b89c",
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
        "user_id": 1,
        "name": "update test extension name",
        "detail": "update test extension detail",
        "domain_id": "cc6a05eb-33a4-444b-bf8a-359de7d95499",
        "extension": "test12",
        "password": "5316382a-757c-11eb-9348-bb32547e99c4",
        "tm_create": "2021-02-23 02:09:39.701458",
        "tm_update": "2021-02-23 02:11:03.992067",
        "tm_delete": ""
    }

Enable direct extension
-----------------------

Enable direct external access for an extension. This generates a unique hash that allows the extension to be reached at ``sip:direct.<hash>@sip.voipbin.net``.

.. code::

    $ curl -k --location --request PUT 'https://api.voipbin.net/v1.0/extensions/6a7934ff-0e1c-4857-857b-23c9e27d267b?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "direct": true
    }'

    {
        "id": "6a7934ff-0e1c-4857-857b-23c9e27d267b",
        "customer_id": "cc6a05eb-33a4-444b-bf8a-359de7d95499",
        "name": "update test extension name",
        "detail": "update test extension detail",
        "extension": "test12",
        "domain_name": "cc6a05eb-33a4-444b-bf8a-359de7d95499.registrar.voipbin.net",
        "username": "test12",
        "password": "5316382a-757c-11eb-9348-bb32547e99c4",
        "direct_hash": "a1b2c3d4e5f6",
        "tm_create": "2021-02-23 02:09:39.701458",
        "tm_update": "2021-02-23 02:15:00.000000",
        "tm_delete": ""
    }

Regenerate direct extension hash
---------------------------------

Regenerate the direct hash for an extension that already has direct access enabled. The old hash is invalidated and a new one is assigned.

.. code::

    $ curl -k --location --request PUT 'https://api.voipbin.net/v1.0/extensions/6a7934ff-0e1c-4857-857b-23c9e27d267b?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "direct_regenerate": true
    }'

    {
        "id": "6a7934ff-0e1c-4857-857b-23c9e27d267b",
        "customer_id": "cc6a05eb-33a4-444b-bf8a-359de7d95499",
        "name": "update test extension name",
        "detail": "update test extension detail",
        "extension": "test12",
        "domain_name": "cc6a05eb-33a4-444b-bf8a-359de7d95499.registrar.voipbin.net",
        "username": "test12",
        "password": "5316382a-757c-11eb-9348-bb32547e99c4",
        "direct_hash": "f6e5d4c3b2a1",
        "tm_create": "2021-02-23 02:09:39.701458",
        "tm_update": "2021-02-23 02:20:00.000000",
        "tm_delete": ""
    }

Disable direct extension
------------------------

Disable direct external access for an extension. The hash is removed and the extension is no longer reachable via the direct SIP URI.

.. code::

    $ curl -k --location --request PUT 'https://api.voipbin.net/v1.0/extensions/6a7934ff-0e1c-4857-857b-23c9e27d267b?token=<YOUR_AUTH_TOKEN>' \
    --header 'Content-Type: application/json' \
    --data-raw '{
        "direct": false
    }'

    {
        "id": "6a7934ff-0e1c-4857-857b-23c9e27d267b",
        "customer_id": "cc6a05eb-33a4-444b-bf8a-359de7d95499",
        "name": "update test extension name",
        "detail": "update test extension detail",
        "extension": "test12",
        "domain_name": "cc6a05eb-33a4-444b-bf8a-359de7d95499.registrar.voipbin.net",
        "username": "test12",
        "password": "5316382a-757c-11eb-9348-bb32547e99c4",
        "direct_hash": "",
        "tm_create": "2021-02-23 02:09:39.701458",
        "tm_update": "2021-02-23 02:25:00.000000",
        "tm_delete": ""
    }

Delete the extension
--------------------

Delete the existing extension of given id.

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/extensions/6a7934ff-0e1c-4857-857b-23c9e27d267b?token=<YOUR_AUTH_TOKEN>'

