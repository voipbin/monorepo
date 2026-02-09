.. _contact-tutorial:

Tutorial
========

Create a contact
----------------
Create a new contact with phone numbers, emails, and tags.

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
            "phone_numbers": [
                {
                    "number": "+15551234567",
                    "type": "mobile",
                    "is_primary": true
                },
                {
                    "number": "+15559876543",
                    "type": "work",
                    "is_primary": false
                }
            ],
            "emails": [
                {
                    "address": "john@acme.com",
                    "type": "work",
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
        "phone_numbers": [
            {
                "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
                "number": "+15551234567",
                "number_e164": "+15551234567",
                "type": "mobile",
                "is_primary": true,
                "tm_create": "2026-02-07T14:45:59.038962Z"
            },
            {
                "id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
                "number": "+15559876543",
                "number_e164": "+15559876543",
                "type": "work",
                "is_primary": false,
                "tm_create": "2026-02-07T14:45:59.038962Z"
            }
        ],
        "emails": [
            {
                "id": "d4e5f6a7-b8c9-0123-defa-234567890123",
                "address": "john@acme.com",
                "type": "work",
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
                "phone_numbers": [
                    {
                        "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
                        "number": "+15551234567",
                        "number_e164": "+15551234567",
                        "type": "mobile",
                        "is_primary": true,
                        "tm_create": "2026-02-07T14:45:59.038962Z"
                    }
                ],
                "emails": [
                    {
                        "id": "d4e5f6a7-b8c9-0123-defa-234567890123",
                        "address": "john@acme.com",
                        "type": "work",
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

Add a phone number
------------------
Add a new phone number to an existing contact.

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/contacts/<contact-id>/phone-numbers?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data '{
            "number": "+15553334444",
            "type": "home",
            "is_primary": false
        }'

Update a phone number
---------------------
Update an existing phone number's details.

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/contacts/<contact-id>/phone-numbers/<phone-number-id>?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data '{
            "number": "+15553334444",
            "type": "mobile",
            "is_primary": true
        }'

Remove a phone number
---------------------

.. code::

    $ curl -X DELETE 'https://api.voipbin.net/v1.0/contacts/<contact-id>/phone-numbers/<phone-number-id>?token=<YOUR_AUTH_TOKEN>'

Add an email address
--------------------
Add a new email address to an existing contact.

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/contacts/<contact-id>/emails?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data '{
            "address": "john.smith@gmail.com",
            "type": "personal",
            "is_primary": false
        }'

Update an email address
-----------------------
Update an existing email address's details.

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/contacts/<contact-id>/emails/<email-id>?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data '{
            "address": "john.smith@gmail.com",
            "type": "personal",
            "is_primary": true
        }'

Remove an email address
-----------------------

.. code::

    $ curl -X DELETE 'https://api.voipbin.net/v1.0/contacts/<contact-id>/emails/<email-id>?token=<YOUR_AUTH_TOKEN>'

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
