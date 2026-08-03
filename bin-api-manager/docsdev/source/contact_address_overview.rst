.. _contact-address-overview:

Contact Addresses Overview
============================

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free (addresses are organizational records with no per-operation charges)
   * **Async:** No. All ``/contact_addresses`` operations are synchronous and return the result immediately.

VoIPBIN's Contact Addresses API is the standalone resource for managing individual phone-number/email/web-session entries independently of a specific contact. It backs the ``addresses`` array embedded on :ref:`Contact <contact-struct-contact-contact>`, but adds capabilities the nested ``/contacts/{id}/addresses`` endpoints do not have: creating an address before it is attached to any contact (an "unresolved" address), searching/filtering addresses across all contacts, and explicitly claiming an unresolved address onto a contact.

With the Contact Addresses API you can:

- List and filter addresses across all contacts for the customer, or just the unresolved pool
- Get, update, or delete a single address by ID
- Create an address already attached to a contact, or an "unresolved" address with no contact yet
- Claim an unresolved address onto a specific contact, optionally overriding an existing claim

.. note:: **AI Implementation Hint**

   These endpoints require JWT-authenticated access (customer admin or manager permission) and do not accept access-key/direct authentication — a direct-auth call returns ``DIRECT_ACCESS_NOT_SUPPORTED``. See :ref:`Error Reason Codes <error-reason-catalog>`.


Relationship to Contact's Nested Address Endpoints
-----------------------------------------------------

::

    POST /contacts/{id}/addresses           POST /contact_addresses
    (nested under a known contact)          (standalone resource)
    +---------------------------+           +---------------------------+
    | contact_id always in path |           | contact_id optional in    |
    | Address is always attached|           | body -- omit it to create |
    | to that contact           |           | an "unresolved" address   |
    +---------------------------+           +---------------------------+

Both surfaces write to the same underlying ``contact_addresses`` table and return the same :ref:`Address <contact-address-struct-address>` shape. Use the nested ``/contacts/{id}/addresses`` endpoints (see :ref:`Tutorial <contact-tutorial>`) when you already know the contact. Use the standalone ``/contact_addresses`` resource when you need to search across contacts, work with addresses that are not yet attached to a contact, or explicitly claim an address onto a contact.


Address Types
-------------

.. list-table::
   :header-rows: 1

   * - Value
     - Description
   * - tel
     - A phone number, normalized to E.164
   * - email
     - An email address, normalized to lowercase
   * - web_session
     - A webchat visitor's continuity token -- a temporary/internal attribution address. Can be created and managed through ``/contact_addresses``, but is never surfaced on a Contact's own ``addresses`` field.

.. note:: **AI Implementation Hint**

   ``web_session`` is only valid on the standalone ``/contact_addresses`` resource. The nested ``/contacts/{id}/addresses`` endpoints only accept ``tel``/``email``, and a Contact's own ``addresses`` array never includes ``web_session`` rows even if one is later claimed onto that contact.


The Unresolved Pool
--------------------

An address created via ``POST /contact_addresses`` without a ``contact_id`` becomes "unresolved": it exists in the customer's address pool but is not attached to any contact, and it cannot be marked ``is_primary``. This is useful for capturing an inbound caller's or webchat visitor's address before identity resolution has determined which contact (if any) it belongs to.

**List only unresolved addresses**

::

    GET https://api.voipbin.net/v1.0/contact_addresses?unresolved=true

``unresolved=true`` is mutually exclusive with ``contact_id``; if both are supplied, ``unresolved=true`` wins and ``contact_id`` is ignored.

**Claim an unresolved address onto a contact**

::

    POST https://api.voipbin.net/v1.0/contact_addresses/<address-id>/claim
    { "contact_id": "<contact-id>" }

Claiming is idempotent if the address is already claimed by the same contact. By default, claiming an address already owned by a different, still-active contact returns ``409 Conflict``. Pass ``"force": true`` in the request body to overwrite that ownership instead — the previous owner's claim is closed and a new one opens for the requesting contact. Addresses owned by a soft-deleted (tombstoned) contact are always repaired in place onto the new contact, regardless of ``force``.


Filtering and Search
---------------------

``GET /contact_addresses`` supports these optional filters (combinable):

.. list-table::
   :header-rows: 1

   * - Parameter
     - Description
   * - contact_id
     - Only addresses belonging to this contact
   * - type
     - Only addresses of this type (``tel``, ``email``, or ``web_session``)
   * - unresolved
     - When ``true``, only unresolved (no ``contact_id``) addresses. Overrides ``contact_id`` when both are given.
   * - target
     - Exact match on the address value (E.164 number or email address)


Troubleshooting
----------------

* **409 Conflict when claiming an address:**
    * **Cause:** The address is already claimed by a different, still-active contact, and ``force`` was omitted or ``false``.
    * **Fix:** Confirm the reassignment is intentional, then retry with ``"force": true``.

* **is_primary rejected on creation:**
    * **Cause:** The address was created without a ``contact_id`` (unresolved), and unresolved addresses cannot be primary.
    * **Fix:** Either supply a ``contact_id`` at creation time, or claim the address onto a contact first and update ``is_primary`` afterward via ``PUT /contact_addresses/{id}``.

* **404 Not Found when accessing an address:**
    * **Cause:** The address UUID does not exist or belongs to another customer.
    * **Fix:** Verify the UUID was obtained from a recent ``GET https://api.voipbin.net/v1.0/contact_addresses`` list call with your authentication token.


Related Documentation
-----------------------

- :ref:`Contact Overview <contact-overview>` - Contacts and their embedded ``addresses`` array
- :ref:`Contact Tutorial <contact-tutorial>` - Nested ``/contacts/{id}/addresses`` endpoints
- :ref:`Peer Events Overview <contact-peer-event-overview>` - Activity log keyed by address
