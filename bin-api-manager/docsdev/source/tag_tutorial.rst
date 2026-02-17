.. _tag-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with tags, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.

.. note:: **AI Implementation Hint**

   Tag names must be unique within a customer account. After creating a tag, use its ``id`` (UUID) when assigning it to agents via ``PUT /agents/{id}/tag_ids`` or when configuring queue requirements via ``POST /queues``. Tags by themselves have no effect until assigned to agents and referenced by queues.

Create a new tag
----------------

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/tags?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --header 'Cookie: token=<YOUR_AUTH_TOKEN>' \
        --data-raw '{
            "name": "test tag",
            "detail": "test tag example"
        }'

    {
        "id": "d7450dda-21e0-4611-b09a-8d771c50a5e6",
        "name": "test tag",
        "detail": "test tag example",
        "tm_create": "2022-10-22 16:16:16.874761",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }
