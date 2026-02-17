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
        "phone_numbers": [<PhoneNumber>, ...],
        "emails": [<Email>, ...],
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
* ``phone_numbers`` (Array of Object): Array of phone numbers associated with this contact. See :ref:`PhoneNumber <contact-struct-contact-phonenumber>`.
* ``emails`` (Array of Object): Array of email addresses associated with this contact. See :ref:`Email <contact-struct-contact-email>`.
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

.. _contact-struct-contact-phonenumber:

PhoneNumber
-----------

.. code::

    {
        "id": "<string>",
        "number": "<string>",
        "number_e164": "<string>",
        "type": "<string>",
        "is_primary": <boolean>,
        "tm_create": "<string>"
    }

* ``id`` (UUID): The phone number entry's unique identifier. Returned when adding via ``POST /contacts/{id}/phone-numbers``.
* ``number`` (String): Phone number as originally entered by the user.
* ``number_e164`` (String): Phone number normalized to E.164 format (e.g., ``+15551234567``). Used for lookup matching.
* ``type`` (enum string): Phone number type. See :ref:`PhoneNumberType <contact-struct-contact-phonenumbertype>`.
* ``is_primary`` (Boolean): Whether this is the primary phone number for the contact. Only one number should be marked primary.
* ``tm_create`` (string, ISO 8601): Timestamp when the phone number was added.

.. _contact-struct-contact-phonenumbertype:

PhoneNumberType
^^^^^^^^^^^^^^^

+----------+------------------------------------------------------------------+
| Value    | Description                                                      |
+==========+==================================================================+
| mobile   | Mobile/cell phone number                                         |
+----------+------------------------------------------------------------------+
| work     | Work/office phone number                                         |
+----------+------------------------------------------------------------------+
| home     | Home phone number                                                |
+----------+------------------------------------------------------------------+
| fax      | Fax number                                                       |
+----------+------------------------------------------------------------------+
| other    | Other phone number type                                          |
+----------+------------------------------------------------------------------+

.. _contact-struct-contact-email:

Email
-----

.. code::

    {
        "id": "<string>",
        "address": "<string>",
        "type": "<string>",
        "is_primary": <boolean>,
        "tm_create": "<string>"
    }

* ``id`` (UUID): The email entry's unique identifier. Returned when adding via ``POST /contacts/{id}/emails``.
* ``address`` (String): Email address (stored and matched in lowercase).
* ``type`` (enum string): Email address type. See :ref:`EmailType <contact-struct-contact-emailtype>`.
* ``is_primary`` (Boolean): Whether this is the primary email for the contact. Only one email should be marked primary.
* ``tm_create`` (string, ISO 8601): Timestamp when the email address was added.

.. _contact-struct-contact-emailtype:

EmailType
^^^^^^^^^

+----------+------------------------------------------------------------------+
| Value    | Description                                                      |
+==========+==================================================================+
| work     | Work/business email address                                      |
+----------+------------------------------------------------------------------+
| personal | Personal email address                                           |
+----------+------------------------------------------------------------------+
| other    | Other email address type                                         |
+----------+------------------------------------------------------------------+
