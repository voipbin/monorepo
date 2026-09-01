.. _contact-case-overview:

Case Overview
=============

.. note:: **AI Context**

   * **Complexity:** Medium
   * **Cost:** Free (case management is a record-keeping layer with no per-operation charges; sending a message from a case bills the same as an ordinary conversation message)
   * **Async:** No. All ``/contact_cases`` operations are synchronous and return the result immediately.

A Case is a thin, per-channel session header that groups related activity (calls, conversation messages) on a single peer address into a start/end unit an agent can pick up, work, and close. Cases sit above the raw :ref:`Peer Events <contact-peer-event-overview>` log: where Peer Events is an unfiltered activity feed, a Case is the CRM-style "ticket" an agent actually works, with a lifecycle (``open`` / ``closed``), an owner, optional attribution to a resolved :ref:`Contact <contact-struct-contact-contact>`, private internal notes, and the ability to send outbound messages tied to the case.

With the Case API you can:

- List and filter cases by status, owner, contact, or origin reference
- List unresolved cases (open, with no contact attributed yet) for triage
- Get a single case, close it, or continue a closed case as a new linked case
- Attach or detach a case's resolved contact
- Add, list, and delete internal (agent-only) notes on a case
- Send an outbound message from an open case to its associated peer address

.. note:: **AI Implementation Hint**

   There is no direct customer-facing endpoint to *create* a case. Cases are created automatically by the platform — most commonly via a Flow's ``case_create`` action (see :ref:`Flow Actions <flow-struct-action>`) or an AI tool call — when handling an inbound call or conversation message. Use this API to list, inspect, and manage the lifecycle of cases the platform has already created; to make case creation part of a call/message flow, add a ``case_create`` action to the relevant Flow.

   These endpoints require JWT-authenticated access (customer admin or manager permission) and do not accept access-key/direct authentication — a direct-auth call returns ``DIRECT_ACCESS_NOT_SUPPORTED``. See :ref:`Error Reason Codes <error-reason-catalog>`.


Case Lifecycle
--------------

::

    (created by case_create Flow action / AI tool)
                        |
                        v
                    +--------+
                    |  open  |<---------------------------+
                    +--------+                             |
                        |                                  |
              POST .../close                       POST .../continue
                        |                          (creates a NEW case,
                        v                           chained via previous_case_id)
                    +--------+                             |
                    | closed |-----------------------------+
                    +--------+

* ``open``: The case is active. Notes can be added, its contact attribution can change, and messages can be sent through it.
* ``closed``: The case has been resolved. ``closed_at``, ``closed_reason``, ``closed_by_type``, and ``closed_by_id`` record how and by whom it was closed.

``POST /contact_cases/{id}/close`` closes an open case. ``closed_by_id`` is always derived server-side from the authenticated caller's own agent identity — it cannot be forged by supplying an arbitrary agent ID. Closing an already-closed case is not an error: the response reflects the case's actual persisted closed state (which may have been closed by a different agent).

``POST /contact_cases/{id}/continue`` creates a new, open case that continues a previously closed one, linked via ``previous_case_id``. This models a customer re-contacting about the same matter after their case was closed.


Filtering and Listing
----------------------

``GET /contact_cases`` supports these optional filters (combinable):

.. list-table::
   :header-rows: 1

   * - Parameter
     - Description
   * - status
     - ``open`` or ``closed``
   * - reference_id
     - The internal resource ID (e.g. a call ID) that ``reference_type`` points at
   * - owner_type
     - e.g. ``agent``
   * - owner_id
     - UUID of the owning agent
   * - contact_id
     - Only cases attributed to this Contact

``GET /contact_cases/unresolved`` is a dedicated triage view: it returns open cases with no ``contact_id`` attributed yet (accepts no filters other than pagination).


Attaching a Contact
--------------------

::

    PUT https://api.voipbin.net/v1.0/contact_cases/<case-id>
    { "contact_id": "<contact-id>" }   -- attach
    { "contact_id": "" }               -- detach (empty string, not omitted)

The target ``contact_id`` must belong to the same customer as the case; a cross-tenant ``contact_id`` is rejected as not found. Every attach/detach is recorded as a ``case_contact_attributed``/``case_contact_detached`` audit event.


Case Notes (Internal Only)
-----------------------------

Notes (``GET``/``POST /contact_cases/{id}/notes``, ``DELETE /contact_cases/{id}/notes/{note_id}``) are agent-facing annotations on a case. They are **never** exposed in any customer-facing webhook or channel — they exist purely for internal agent handoff and audit (e.g. "Called the customer back, no answer.").

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/contact_cases/<case-id>/notes?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data '{
            "author_type": "agent",
            "author_id": "2a2ec0ba-8004-11ec-aea5-439829c92a7c",
            "text": "Called the customer back, no answer."
        }'


Sending a Message From a Case
--------------------------------

``POST /contact_cases/{id}/messages`` sends an outbound message through the case's associated channel. The case must be ``open`` (a closed case must be reopened via ``POST .../continue`` first). Validation, in order:

1. **Case validation** — the case belongs to the caller's customer and is ``status: open``.
2. **Destination-to-case binding** — ``destination`` must be attributable to this specific case: either one of the resolved Contact's addresses, or the case's own ``peer.target`` if unresolved. Both failure modes return the same generic error, by design, so ``case_id`` cannot be used as a probe to determine which check failed.
3. **Source-ownership validation** — ``source`` must be an active, normal PSTN number owned by the case's customer.

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/contact_cases/<case-id>/messages?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data '{
            "source": "+15551234567",
            "destination": "+15559876543",
            "text": "Thanks for reaching out -- following up on your request."
        }'

The response is a :ref:`Message <conversation-struct-message-message>` object (the same shape returned by the Conversation API), since the case is sent through the underlying conversation-message pipeline.

.. note:: **AI Implementation Hint**

   ``source``/``destination`` ownership validation currently only supports ``tel`` (PSTN number) channels. Cases whose peer channel is WhatsApp or LINE cannot yet send messages through this endpoint.


Agent-Facing Surface
----------------------

The same case management capabilities are also available under ``/service_agents/contact_cases``, gated by regular agent permission rather than admin/manager. This surface additionally exposes ``POST /service_agents/contact_cases/{id}/assign``, which assigns a case's owner to a given agent (``owner_type`` is always fixed to ``agent`` server-side) — there is no equivalent top-level admin endpoint for assignment.


Troubleshooting
----------------

* **409 Conflict when sending a case message:**
    * **Cause:** The case is not ``open`` (it was already closed), or the conversation is in a state that rejects the send.
    * **Fix:** Call ``POST /contact_cases/{id}/continue`` to reopen a linked case, then retry the send.

* **400 Bad Request on destination/source validation:**
    * **Cause:** ``destination`` is not one of the resolved contact's addresses (or the case's own peer target), or ``source`` is not an active PSTN number owned by the case's customer.
    * **Fix:** Verify the addresses via ``GET /contacts/{id}`` (for destination) and ``GET /numbers`` (for source).

* **404 Not Found when accessing a case:**
    * **Cause:** The case UUID does not exist or belongs to another customer.
    * **Fix:** Verify the UUID was obtained from a recent ``GET https://api.voipbin.net/v1.0/contact_cases`` list call.


Related Documentation
-----------------------

- :ref:`Contact Overview <contact-overview>` - Contacts a case may be attributed to
- :ref:`Peer Events Overview <contact-peer-event-overview>` - Raw activity log a case groups
- :ref:`Flow Actions <flow-struct-action>` - The ``case_create`` action that opens new cases
- :ref:`Conversation Overview <conversation-overview>` - The messaging pipeline case messages are sent through
