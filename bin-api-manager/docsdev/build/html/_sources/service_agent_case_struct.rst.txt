.. _service_agent-case-struct:

Structures
==========

.. _service_agent-case-struct-case:

Case
----

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "peer": { ... },
        "local": { ... },
        "name": "<string>",
        "detail": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "contact_id": "<string>",
        "owner_type": "<string>",
        "owner_id": "<string>",
        "status": "<string>",
        "opened_at": "<string>",
        "closed_at": "<string>",
        "closed_reason": "<string>",
        "closed_by_type": "<string>",
        "closed_by_id": "<string>",
        "previous_case_id": "<string>",
        "tag_ids": [
            "<string>"
        ],
        "tm_create": "<string>",
        "tm_update": "<string>"
    }

* ``id`` (UUID): The case's unique identifier. Returned from ``GET /service_agents/contact_cases``.
* ``customer_id`` (UUID): The customer this case belongs to.
* ``peer`` (Object): The remote party this case is scoped to (e.g. the caller's phone number). See :ref:`Address <common-struct-address-address>`.
* ``local`` (Object): The customer's own endpoint (number/channel/account) the interaction arrived on or was placed from. Always present; a zero value serializes as ``{}`` when no local endpoint is known. See :ref:`Address <common-struct-address-address>`.
* ``name`` (String, optional): Freeform case label, settable only at creation time by the system.
* ``detail`` (String, optional): Freeform case description, settable only at creation time by the system.
* ``reference_type`` (String): The type of the resource that originated this case (e.g. ``call``, ``conversation_message``), reusing the same vocabulary as Contact Interactions.
* ``reference_id`` (String, optional): The ID of the originating resource, derived automatically by the system when the case is created. Never client-supplied.
* ``contact_id`` (UUID, nullable): The resolved :ref:`Contact <contact-struct-contact-contact>` attached to this case, if any. Set or cleared via ``PUT /service_agents/contact_cases/{id}``.
* ``owner_type`` (String): Always ``"agent"`` once assigned; empty before assignment.
* ``owner_id`` (UUID, nullable): The agent currently (or most recently) responsible for the case. Set via ``POST /service_agents/contact_cases/{id}/assign``. Never cleared by closing the case.
* ``status`` (enum string): ``open`` or ``closed``. See :ref:`Status <service_agent-case-struct-case-status>`.
* ``opened_at`` (string, ISO 8601, nullable): When the case was opened.
* ``closed_at`` (string, ISO 8601, nullable): When the case was closed, if applicable.
* ``closed_reason`` (String, optional): Why the case was closed. See :ref:`Closed reason <service_agent-case-struct-case-closed_reason>`.
* ``closed_by_type`` (String, optional): ``agent`` or ``system``. Derived server-side from the caller that closed the case.
* ``closed_by_id`` (UUID, nullable): The agent that closed the case, when ``closed_by_type`` is ``agent``.
* ``previous_case_id`` (UUID, nullable): The prior, now-closed case for the same peer, set automatically on re-contact.
* ``tag_ids`` (Array of UUID, optional): Tag IDs assigned to the case.
* ``tm_create`` (string, ISO 8601): Timestamp when the case was created.
* ``tm_update`` (string, ISO 8601): Timestamp when the case was last updated.

.. _service_agent-case-struct-case-status:

Status
^^^^^^

========== ============
Type       Description
========== ============
open       The case is active and awaiting or being worked.
closed     The case has been resolved and closed.
========== ============

.. _service_agent-case-struct-case-closed_reason:

Closed reason
^^^^^^^^^^^^^

============== ============
Type           Description
============== ============
agent_closed   An agent explicitly closed the case via ``POST .../close``.
timeout        The case was closed automatically after a period of inactivity.
merged         Reserved for a future same-channel case-merge feature. Not yet used.
============== ============


.. _service_agent-case-struct-casenote:

CaseNote
--------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "case_id": "<string>",
        "author_type": "<string>",
        "author_id": "<string>",
        "text": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The note's unique identifier.
* ``customer_id`` (UUID): The customer this note belongs to.
* ``case_id`` (UUID): The case this note is attached to.
* ``author_type`` (enum string): ``agent`` or ``system``. See :ref:`Author type <service_agent-case-struct-casenote-author_type>`.
* ``author_id`` (UUID, nullable): The authoring agent's ID, when ``author_type`` is ``agent``. Null for system-authored notes.
* ``text`` (String): The note's free-text content.
* ``tm_create`` (string, ISO 8601): Timestamp when the note was created.
* ``tm_update`` (string, ISO 8601): Timestamp when the note was last updated.
* ``tm_delete`` (string, ISO 8601): Timestamp when the note was deleted, if applicable.

.. note:: **AI Implementation Hint**

   ``CaseNote`` is never included in any customer-facing webhook or response -- it only ever appears in ``GET/POST /service_agents/contact_cases/{id}/notes`` responses to authenticated agents of the same customer.

.. _service_agent-case-struct-casenote-author_type:

Author type
^^^^^^^^^^^

========== ============
Type       Description
========== ============
agent      Authored by the agent identified in ``author_id``.
system     Authored automatically by the platform. ``author_id`` is null.
========== ============
