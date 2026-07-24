.. _contact-peer-event-overview:

Peer Events Overview
=====================

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free (read-only query against an existing log)
   * **Async:** No. ``GET /contact_peer_events`` is synchronous and returns results immediately.

VoIPBIN's Peer Events API returns the raw, unfiltered log of peer/local address activity recorded by the platform. Unlike :ref:`Contact Interactions <contact-overview>`, this endpoint applies **no identity resolution and no CRM eligibility filtering**: it may include rows for internal-resource peer types (agent extensions, AI participants, conference legs, SIP trunks) that Contact Interactions deliberately excludes.

Use this endpoint when you need to see everything that happened for an address (or a contact's registered addresses), including internal/system activity, and are prepared to filter or group that activity yourself for presentation.

With the Peer Events API you can:

- Search all recorded activity for a single ``peer_type`` + ``peer_target`` pair (e.g. a specific phone number)
- Search all recorded activity across every address registered to a contact
- Page through results using a cursor-based ``next_page_token``


How Peer Events Differ From Contact Interactions
-------------------------------------------------

::

    +-----------------------------------------------------------------------+
    |                  Two Read Paths, Two Purposes                          |
    +-----------------------------------------------------------------------+

    GET /contact_interactions                    GET /contact_peer_events
    +--------------------------+                  +--------------------------+
    | Identity-resolved        |                  | Raw, unfiltered          |
    | CRM-eligible only        |                  | Includes internal noise |
    | (customer-facing rows)   |                  | (agent/ai/conference)   |
    +--------------------------+                  +--------------------------+
              |                                             |
              v                                             v
        Customer-facing history                    Full activity audit
        for a contact                               for an address or contact

Both endpoints can be filtered by ``contact_id`` or by a single ``peer_type`` + ``peer_target`` pair. The difference is entirely in what rows are returned, not how you ask for them: Contact Interactions returns a curated, customer-facing subset, while Peer Events returns everything the platform recorded, verbatim.

.. note:: **AI Implementation Hint**

   If your application shows this data directly to end customers, use ``GET /contact_interactions`` instead — it is already filtered for that purpose. Use ``GET /contact_peer_events`` only when you need the complete picture (e.g. an internal operations/debugging view) and will filter or label internal-resource rows yourself before display.


Filtering
---------

Exactly one filter is required per request:

**By Contact**

::

    GET https://api.voipbin.net/v1.0/contact_peer_events?contact_id=<contact_id>

Resolves every address registered to the contact and returns activity matching any of them.

**By Peer Address**

::

    GET https://api.voipbin.net/v1.0/contact_peer_events?peer_type=tel&peer_target=%2B155****4567

Returns activity for a single address directly, without needing a resolved contact record. Useful for ad-hoc lookups of a specific phone number or email address.

.. note:: **AI Implementation Hint**

   As with Contact lookups, URL-encode ``+`` as ``%2B`` in ``peer_target`` when the type is ``tel``.


Internal-Resource Noise
------------------------

Because this endpoint applies no filtering, ``peer_type`` may be a value your application does not otherwise expect from customer-facing endpoints, such as:

+-------------+------------------------------------------------------------------+
| peer_type   | Description                                                       |
+=============+====================================================================+
| tel         | A phone number (customer-facing, same as Contact Interactions)   |
+-------------+------------------------------------------------------------------+
| email       | An email address (customer-facing, same as Contact Interactions) |
+-------------+------------------------------------------------------------------+
| agent       | An internal agent extension leg                                  |
+-------------+------------------------------------------------------------------+
| ai          | An AI participant leg                                            |
+-------------+------------------------------------------------------------------+
| conference  | A conference bridge leg                                          |
+-------------+------------------------------------------------------------------+
| sip         | A raw SIP trunk leg                                               |
+-------------+------------------------------------------------------------------+

Applications displaying this data to end users are responsible for filtering or clearly labeling non-customer-facing rows themselves.


Pagination
----------

Results are paginated with a cursor token, newest first.

::

    GET https://api.voipbin.net/v1.0/contact_peer_events?peer_type=tel&peer_target=%2B155****4567&page_size=50
         |
         v
    { "result": [...], "next_page_token": "2026-01-15T10:30:00.123000Z" }
         |
         v
    GET https://api.voipbin.net/v1.0/contact_peer_events?peer_type=tel&peer_target=%2B155****4567&page_size=50&page_token=2026-01-15T10:30:00.123000Z


Troubleshooting
----------------

* **400 Bad Request when filtering:**
    * **Cause:** Zero filters, or more than one filter (``contact_id`` together with ``peer_type``/``peer_target``), were provided.
    * **Fix:** Provide exactly one of ``contact_id`` or ``peer_type``+``peer_target``.

* **Unexpected agent/ai/conference rows in results:**
    * **Cause:** This is expected behavior — Peer Events returns raw, unfiltered activity by design.
    * **Fix:** If you only want customer-facing history, use ``GET /contact_interactions`` instead, or filter ``peer_type`` client-side.

* **Empty result for a contact with known activity:**
    * **Cause:** The contact has zero registered addresses, so there was nothing to search.
    * **Fix:** Verify the contact has addresses via ``GET /contacts/{id}``.


Related Documentation
----------------------

- :ref:`Contact Overview <contact-overview>` - Contact records and their registered addresses
- :ref:`Contact Interactions <contact-struct-contact>` - The identity-resolved, CRM-eligible counterpart to this endpoint
