.. _contact-struct-contact:

Structures
==========

.. _contact-struct-contact-contact:

Contact
-------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "first_name": "<string>",
        "last_name": "<string>",
        "display_name": "<string>",
        "company": "<string>",
        "job_title": "<string>",
        "source": "<string>",
        "external_id": "<string>",
        "addresses": [<Address>, ...],
        "tag_ids": ["<string>", ...],
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The contact's unique identifier. Returned when creating via ``POST /contacts`` or listing via ``GET /contacts``.
* ``customer_id`` (UUID): The customer who owns this contact. Obtained from ``GET /customers``.
* ``first_name`` (String): Contact's first name.
* ``last_name`` (String): Contact's last name.
* ``display_name`` (String): Contact's display name, typically "First Last". Used for caller ID enrichment.
* ``company`` (String): Contact's company or organization name.
* ``job_title`` (String): Contact's job title or role.
* ``source`` (enum string): How the contact was created. See :ref:`Source <contact-struct-contact-source>`.
* ``external_id`` (String): Reference ID in an external CRM system (Salesforce, HubSpot, Zoho, etc.). Used for deduplication and two-way sync.
* ``addresses`` (Array of Object): Array of addresses (tel or email) associated with this contact. See :ref:`Address <contact-struct-contact-address>`.
* ``tag_ids`` (Array of UUID): Array of tag UUIDs assigned to this contact. Each tag ID is obtained from ``GET /tags``.
* ``tm_create`` (string, ISO 8601): Timestamp when the contact was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update. ``null`` if never updated.
* ``tm_delete`` (string, ISO 8601): Timestamp of deletion (soft delete). ``null`` if not deleted.

.. _contact-struct-contact-source:

Source
------
How the contact was created.

+----------+------------------------------------------------------------------+
| Value    | Description                                                      |
+==========+==================================================================+
| manual   | Created by a user through the UI                                 |
+----------+------------------------------------------------------------------+
| import   | Imported from a file or bulk upload                              |
+----------+------------------------------------------------------------------+
| api      | Created programmatically via the API                             |
+----------+------------------------------------------------------------------+
| sync     | Synchronized from an external CRM system                         |
+----------+------------------------------------------------------------------+

.. _contact-struct-contact-address:

Address
-------

.. code::

    {
        "id": "<string>",
        "type": "<string>",
        "target": "<string>",
        "is_primary": <boolean>,
        "tm_create": "<string>"
    }

* ``id`` (UUID): The address entry's unique identifier. Returned when adding via ``POST /contacts/{id}/addresses``.
* ``type`` (enum string): Address type. See :ref:`AddressType <contact-struct-contact-addresstype>`.
* ``target`` (String): The address value. E.164 format for ``tel`` (e.g., ``+155****4567``); email address for ``email`` (e.g., ``user@example.com``).
* ``is_primary`` (Boolean): Whether this is the primary address for the given type. Only one address per type should be marked primary.
* ``tm_create`` (string, ISO 8601): Timestamp when the address was added.

.. _contact-struct-contact-addresstype:

AddressType
^^^^^^^^^^^

+----------+------------------------------------------------------------------+
| Value    | Description                                                      |
+==========+==================================================================+
| tel      | Phone number in E.164 format                                     |
+----------+------------------------------------------------------------------+
| email    | Email address                                                    |
+----------+------------------------------------------------------------------+
