.. _contact-address-struct:

Structures
==========

.. _contact-address-struct-address:

Address
-------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "contact_id": "<string>",
        "type": "<string>",
        "target": "<string>",
        "target_name": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "is_primary": <boolean>,
        "tm_create": "<string>"
    }

* ``id`` (UUID): The address's unique identifier. Returned when creating via ``POST /contact_addresses``.
* ``customer_id`` (UUID): The customer who owns this address. Obtained from ``GET /customers``.
* ``contact_id`` (UUID): The contact this address is attached to. Absent/empty for an unresolved address that has not yet been claimed onto a contact. See :ref:`The Unresolved Pool <contact-address-overview>`.
* ``type`` (enum string): Address type. See :ref:`AddressType <contact-address-struct-addresstype>`.
* ``target`` (String): The address value. E.164 format for ``tel``, lowercase email for ``email``, an opaque continuity token for ``web_session``.
* ``target_name`` (String): Optional display name associated with the address (e.g., a caller-ID name reported by the network). Empty string if not set.
* ``name`` (String): Optional human-readable label for this address (e.g., "Main Office"). Free-form text, not a constrained category.
* ``detail`` (String): Optional free-form notes about this address. Empty string if not set.
* ``is_primary`` (Boolean): Whether this is the primary address for the contact. Always ``false`` for an unresolved address (cannot be set until the address is claimed onto a contact).
* ``tm_create`` (string, ISO 8601): Timestamp when the address was created.

.. _contact-address-struct-addresstype:

AddressType
^^^^^^^^^^^

.. list-table::
   :header-rows: 1

   * - Value
     - Description
   * - tel
     - Phone number in E.164 format
   * - email
     - Email address
   * - web_session
     - Webchat visitor continuity token (standalone ``/contact_addresses`` only; never appears on a Contact's own ``addresses`` field)
