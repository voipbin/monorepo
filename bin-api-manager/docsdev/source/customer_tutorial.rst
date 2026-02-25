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
                "name": "Acme Corporation",
                "detail": "Enterprise customer account",
                "email": "admin@acme-corp.com",
                "phone_number": "+15551234567",
                "address": "123 Main St, San Francisco, CA 94105",
                "webhook_method": "POST",
                "webhook_uri": "https://webhooks.acme-corp.com/voipbin",
                "billing_account_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
                "email_verified": true,
                "status": "active",
                "tm_deletion_scheduled": null,
                "tm_create": "2024-01-15T10:30:00Z",
                "tm_update": "2024-06-20T14:22:35Z",
                "tm_delete": null
            }
        ],
        "next_page_token": "2024-01-15T10:30:00Z"
    }

Get detail of customer
----------------------

Example

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/customers/5e4a0680-804e-11ec-8477-2fea5968d85b?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Acme Corporation",
        "detail": "Enterprise customer account",
        "email": "admin@acme-corp.com",
        "phone_number": "+15551234567",
        "address": "123 Main St, San Francisco, CA 94105",
        "webhook_method": "POST",
        "webhook_uri": "https://webhooks.acme-corp.com/voipbin",
        "billing_account_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
        "email_verified": true,
        "status": "active",
        "tm_deletion_scheduled": null,
        "tm_create": "2024-01-15T10:30:00Z",
        "tm_update": "2024-06-20T14:22:35Z",
        "tm_delete": null
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
            "name": "Test Company",
            "detail": "Test customer account",
            "email": "admin@test-company.com",
            "webhook_method": "POST",
            "webhook_uri": "https://webhooks.test-company.com/voipbin"
        }'

    {
        "id": "ff424526-f65d-483f-bc36-3b2357c6c6a9",
        "name": "Test Company",
        "detail": "Test customer account",
        "email": "admin@test-company.com",
        "phone_number": "",
        "address": "",
        "webhook_method": "POST",
        "webhook_uri": "https://webhooks.test-company.com/voipbin",
        "billing_account_id": "00000000-0000-0000-0000-000000000000",
        "email_verified": false,
        "status": "initial",
        "tm_deletion_scheduled": null,
        "tm_create": "2024-03-10T15:57:08Z",
        "tm_update": "2024-03-10T15:57:08Z",
        "tm_delete": null
    }

Delete customer
---------------

.. note:: **AI Implementation Hint**

   Deleting a customer is a destructive operation that removes the customer account and all associated resources (agents, numbers, flows, etc.). This cannot be undone. The API returns the deleted customer object with ``tm_delete`` set to the deletion timestamp.

Example

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/customers/ff424526-f65d-483f-bc36-3b2357c6c6a9?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "ff424526-f65d-483f-bc36-3b2357c6c6a9",
        "name": "Test Company",
        "detail": "Test customer account",
        "email": "admin@test-company.com",
        "phone_number": "",
        "address": "",
        "webhook_method": "POST",
        "webhook_uri": "https://webhooks.test-company.com/voipbin",
        "billing_account_id": "00000000-0000-0000-0000-000000000000",
        "email_verified": false,
        "status": "deleted",
        "tm_deletion_scheduled": null,
        "tm_create": "2024-03-10T15:57:08Z",
        "tm_update": "2024-03-10T15:57:08Z",
        "tm_delete": "2024-03-10T15:59:08Z"
    }

Unregister account (schedule deletion)
---------------------------------------

Request account deletion with a 30-day grace period. The account transitions to ``frozen`` status and a deletion date is scheduled.

Example

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/auth/unregister' \
        --header 'Content-Type: application/json' \
        --header 'Authorization: Bearer <YOUR_AUTH_TOKEN>' \
        --data-raw '{
            "password": "yourPassword"
        }'

    {
        "id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Acme Corporation",
        "detail": "Enterprise customer account",
        "email": "admin@acme-corp.com",
        "phone_number": "+15551234567",
        "address": "123 Main St, San Francisco, CA 94105",
        "webhook_method": "POST",
        "webhook_uri": "https://webhooks.acme-corp.com/voipbin",
        "billing_account_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
        "email_verified": true,
        "status": "frozen",
        "tm_deletion_scheduled": "2024-07-20T14:22:35Z",
        "tm_create": "2024-01-15T10:30:00Z",
        "tm_update": "2024-06-20T14:22:35Z",
        "tm_delete": null
    }

Unregister account immediately
------------------------------

Skip the 30-day grace period and permanently delete the account immediately. Requires a confirmation phrase and password.

Example

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/auth/unregister' \
        --header 'Content-Type: application/json' \
        --header 'Authorization: Bearer <YOUR_AUTH_TOKEN>' \
        --data-raw '{
            "password": "yourPassword",
            "confirmation_phrase": "DELETE",
            "immediate": true
        }'

    {
        "id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "",
        "detail": "",
        "email": "",
        "phone_number": "",
        "address": "",
        "webhook_method": "",
        "webhook_uri": "",
        "billing_account_id": "00000000-0000-0000-0000-000000000000",
        "email_verified": false,
        "status": "deleted",
        "tm_deletion_scheduled": null,
        "tm_create": "2024-01-15T10:30:00Z",
        "tm_update": "2024-06-20T14:25:00Z",
        "tm_delete": "2024-06-20T14:25:00Z"
    }

.. note:: **AI Implementation Hint**

   Immediate deletion cannot be undone. All customer resources are cascade-deleted: agents, numbers, flows, queues, trunks, extensions, files, billing accounts, tags, transcriptions, and contacts. PII is anonymized -- notice how name, detail, email, and other personal fields are empty in the response. The ``confirmation_phrase`` must be exactly ``"DELETE"`` (case-sensitive). Do not call this endpoint unless the user has explicitly confirmed permanent, irreversible deletion.

Cancel unregistration (recover account)
----------------------------------------

During the 30-day grace period, cancel the scheduled deletion and restore the account to ``active`` status.

Example

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/auth/unregister' \
        --header 'Authorization: Bearer <YOUR_AUTH_TOKEN>'

    {
        "id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Acme Corporation",
        "detail": "Enterprise customer account",
        "email": "admin@acme-corp.com",
        "phone_number": "+15551234567",
        "address": "123 Main St, San Francisco, CA 94105",
        "webhook_method": "POST",
        "webhook_uri": "https://webhooks.acme-corp.com/voipbin",
        "billing_account_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
        "email_verified": true,
        "status": "active",
        "tm_deletion_scheduled": null,
        "tm_create": "2024-01-15T10:30:00Z",
        "tm_update": "2024-06-20T14:30:00Z",
        "tm_delete": null
    }
