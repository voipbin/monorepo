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

* id: Contact's unique identifier.
* customer_id: Customer who owns this contact.
* first_name: Contact's first name.
* last_name: Contact's last name.
* display_name: Contact's display name.
* company: Contact's company or organization.
* job_title: Contact's job title.
* source: How the contact was created. See :ref:`Source <contact-struct-contact-source>`.
* external_id: Reference ID in an external CRM system (Salesforce, HubSpot, Zoho, etc.).
* phone_numbers: Array of phone numbers. See :ref:`PhoneNumber <contact-struct-contact-phonenumber>`.
* emails: Array of email addresses. See :ref:`Email <contact-struct-contact-email>`.
* tag_ids: Array of tag UUIDs assigned to this contact.
* tm_create: Timestamp of creation.
* tm_update: Timestamp of last update.
* tm_delete: Timestamp of deletion (soft delete).

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

* id: Phone number's unique identifier.
* number: Phone number as originally entered.
* number_e164: Phone number normalized to E.164 format (e.g., "+15551234567").
* type: Phone number type. See :ref:`PhoneNumberType <contact-struct-contact-phonenumbertype>`.
* is_primary: Whether this is the primary phone number for the contact.
* tm_create: Timestamp of creation.

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

* id: Email's unique identifier.
* address: Email address (stored in lowercase).
* type: Email address type. See :ref:`EmailType <contact-struct-contact-emailtype>`.
* is_primary: Whether this is the primary email for the contact.
* tm_create: Timestamp of creation.

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
