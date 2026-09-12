.. _contact-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with contacts, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* (Optional) Tag IDs (UUIDs) for categorization. Create tags via ``POST /tags`` or obtain existing ones from ``GET /tags``.
* (Optional) Phone numbers in E.164 format (e.g., ``+15551234567``) for phone number entries.

.. note:: **AI Implementation Hint**

   Contacts store a single unified ``addresses`` array (each entry has ``type``: ``tel`` or ``email``) rather than separate phone-number and email collections. Phone numbers are automatically normalized to E.164 format and stored in the address's ``target`` field. When using the phone lookup endpoint (``GET /contacts/lookup?phone=...``), the ``+`` character must be URL-encoded as ``%2B``. Contact operations are free and synchronous.

Create a contact
----------------
Create a new contact with addresses (phone numbers and/or emails) and tags.

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/contacts?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data '{
            "first_name": "John",
            "last_name": "Smith",
            "display_name": "John Smith",
            "company": "Acme Corp",
            "job_title": "Account Manager",
            "source": "manual",
            "notes": "Key enterprise customer contact",
            "addresses": [
                {
                    "type": "tel",
                    "target": "+15551234567",
                    "name": "Mobile",
                    "is_primary": true
                },
                {
                    "type": "tel",
                    "target": "+15559876543",
                    "name": "Work",
                    "is_primary": false
                },
                {
                    "type": "email",
                    "target": "john@acme.com",
                    "name": "Work",
                    "is_primary": true
                }
            ],
            "tag_ids": [
                "uuid-for-enterprise-tag",
                "uuid-for-vip-tag"
            ]
        }'

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "first_name": "John",
        "last_name": "Smith",
        "display_name": "John Smith",
        "company": "Acme Corp",
        "job_title": "Account Manager",
        "source": "manual",
        "external_id": "",
        "notes": "Key enterprise customer contact",
        "addresses": [
            {
                "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "contact_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
                "type": "tel",
                "target": "+15551234567",
                "target_name": "",
                "name": "Mobile",
                "detail": "",
                "is_primary": true,
                "tm_create": "2026-02-07T14:45:59.038962Z"
            },
            {
                "id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "contact_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
                "type": "tel",
                "target": "+15559876543",
                "target_name": "",
                "name": "Work",
                "detail": "",
                "is_primary": false,
                "tm_create": "2026-02-07T14:45:59.038962Z"
            },
            {
                "id": "d4e5f6a7-b8c9-0123-defa-234567890123",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "contact_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
                "type": "email",
                "target": "john@acme.com",
                "target_name": "",
                "name": "Work",
                "detail": "",
                "is_primary": true,
                "tm_create": "2026-02-07T14:45:59.038962Z"
            }
        ],
        "tag_ids": [
            "uuid-for-enterprise-tag",
            "uuid-for-vip-tag"
        ],
        "tm_create": "2026-02-07T14:45:59.038962Z",
        "tm_update": null,
        "tm_delete": null
    }

Get a list of contacts
----------------------
Retrieve all contacts. Supports pagination with ``page_size`` and ``page_token``.

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/contacts?token=<YOUR_AUTH_TOKEN>&page_size=10'

    {
        "result": [
            {
                "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "first_name": "John",
                "last_name": "Smith",
                "display_name": "John Smith",
                "company": "Acme Corp",
                "job_title": "Account Manager",
                "source": "manual",
                "external_id": "",
                "notes": "Key enterprise customer contact",
                "addresses": [
                    {
                        "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
                        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                        "contact_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
                        "type": "tel",
                        "target": "+15551234567",
                        "target_name": "",
                        "name": "Mobile",
                        "detail": "",
                        "is_primary": true,
                        "tm_create": "2026-02-07T14:45:59.038962Z"
                    },
                    {
                        "id": "d4e5f6a7-b8c9-0123-defa-234567890123",
                        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                        "contact_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
                        "type": "email",
                        "target": "john@acme.com",
                        "target_name": "",
                        "name": "Work",
                        "detail": "",
                        "is_primary": true,
                        "tm_create": "2026-02-07T14:45:59.038962Z"
                    }
                ],
                "tag_ids": [
                    "uuid-for-enterprise-tag"
                ],
                "tm_create": "2026-02-07T14:45:59.038962Z",
                "tm_update": "2026-02-07T14:45:59.038962Z",
                "tm_delete": null
            },
            ...
        ],
        "next_page_token": "2026-02-07T14:30:00.038962Z"
    }

Get a specific contact
----------------------

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/contacts/<contact-id>?token=<YOUR_AUTH_TOKEN>'

Update a contact
----------------
Update contact fields. Only the provided fields are changed.

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/contacts/<contact-id>?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data '{
            "first_name": "John",
            "last_name": "Smith",
            "display_name": "John Smith (VIP)",
            "company": "Acme Corporation",
            "job_title": "Senior Account Manager",
            "notes": "Promoted to senior role in January"
        }'

Delete a contact
----------------
Soft-deletes the contact. The record is not permanently removed.

.. code::

    $ curl -X DELETE 'https://api.voipbin.net/v1.0/contacts/<contact-id>?token=<YOUR_AUTH_TOKEN>'

Lookup a contact by phone number
---------------------------------
Find a contact by phone number in E.164 format.

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/contacts/lookup?token=<YOUR_AUTH_TOKEN>&phone=%2B15551234567'

Note: The ``+`` in the phone number must be URL-encoded as ``%2B``.

Lookup a contact by email
-------------------------
Find a contact by email address. Matching is case-insensitive.

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/contacts/lookup?token=<YOUR_AUTH_TOKEN>&email=john@acme.com'

Add an address to a contact
----------------------------
Add a new phone number or email address to an existing contact. ``type`` must be ``tel`` or ``email``.

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/contacts/<contact-id>/addresses?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data '{
            "type": "tel",
            "target": "+15553334444",
            "name": "Home",
            "is_primary": false
        }'

The response is the full updated :ref:`Contact <contact-struct-contact-contact>` object, including the newly added entry in ``addresses``.

Update an address on a contact
--------------------------------
Update an existing address's details. The address ID comes from a prior ``POST /contacts/{id}/addresses`` call or from the contact's ``addresses`` array.

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/contacts/<contact-id>/addresses/<address-id>?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data '{
            "target": "+15553334444",
            "name": "Mobile",
            "is_primary": true
        }'

Remove an address from a contact
-----------------------------------

.. code::

    $ curl -X DELETE 'https://api.voipbin.net/v1.0/contacts/<contact-id>/addresses/<address-id>?token=<YOUR_AUTH_TOKEN>'

.. note:: **AI Implementation Hint**

   These ``/contacts/{id}/addresses`` endpoints always require an existing, known ``contact_id`` in the path. To create or manage addresses that are not yet attached to any contact (an "unresolved" pool, e.g. for inbound-call address capture before identity resolution), or to search/filter addresses across all contacts, use the standalone :ref:`Contact Addresses <contact-address-overview>` resource instead.

Add a tag
---------
Assign a tag to a contact for categorization or routing.

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/contacts/<contact-id>/tags?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data '{
            "tag_id": "uuid-for-enterprise-tag"
        }'

Remove a tag
------------

.. code::

    $ curl -X DELETE 'https://api.voipbin.net/v1.0/contacts/<contact-id>/tags/<tag-id>?token=<YOUR_AUTH_TOKEN>'
