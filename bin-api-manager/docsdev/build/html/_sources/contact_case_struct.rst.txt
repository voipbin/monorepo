.. _contact-case-struct:

Structures
==========

.. _contact-case-struct-case:

Case
----

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "peer": {<Address>},
        "local": {<Address>},
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
        "name": "<string>",
        "detail": "<string>",
        "previous_case_id": "<string>",
        "tag_ids": ["<string>"],
        "tm_create": "<string>",
        "tm_update": "<string>"
    }

* ``id`` (UUID): The case's unique identifier. Returned from ``GET /contact_cases``.
* ``customer_id`` (UUID): The customer who owns this case.
* ``peer`` (Object): The remote party this case is scoped to, structurally identical to ``commonaddress.Address`` (``type``/``target``/``target_name``/``name``/``detail``) — the same shape as :ref:`Peer Event's peer field <contact-peer-event-struct-peerevent>`, distinct from :ref:`Contact's Address <contact-struct-contact-address>`.
* ``local`` (Object): The customer's own endpoint the case's interactions arrived on or were placed from, same shape as ``peer``. Always present as an object; individual fields are empty when no local endpoint was known at creation time.
* ``reference_type`` (String): Origin channel type, e.g. ``call``, ``conversation_message``.
* ``reference_id`` (String): The internal VoIPBin resource ID that ``reference_type`` points at (the call ID when ``reference_type`` is ``call``, the conversation ID when it is ``conversation_message``). Set automatically at creation time, never customer- or agent-supplied. Empty when no such internal resource ID applies.
* ``contact_id`` (UUID): The resolved contact this case is attributed to. Nullable until resolved via ``PUT /contact_cases/{id}``.
* ``owner_type`` (String): Type of the case owner, e.g. ``agent``.
* ``owner_id`` (UUID): ID of the case owner. Not cleared when a case is closed.
* ``status`` (enum string): Case lifecycle status. See :ref:`Status <contact-case-struct-status>`.
* ``opened_at`` (string, ISO 8601): Timestamp when the case was opened. Nullable.
* ``closed_at`` (string, ISO 8601): Timestamp when the case was closed. Nullable.
* ``closed_reason`` (String): Reason the case was closed. See :ref:`ClosedReason <contact-case-struct-closedreason>`.
* ``closed_by_type`` (String): Type of the actor that closed the case. See :ref:`ClosedByType <contact-case-struct-closedbytype>`.
* ``closed_by_id`` (UUID): ID of the actor that closed the case. Nullable.
* ``name`` (String): Optional freeform case name/title, settable only at creation time (e.g. by the ``case_create`` Flow action).
* ``detail`` (String): Optional freeform case detail, settable only at creation time.
* ``previous_case_id`` (UUID): The prior (now-closed) case this case continues from, if created via ``POST /contact_cases/{id}/continue``. Nil for the first case on a given peer.
* ``tag_ids`` (Array of UUID, omitted when empty): Tags attached to the case, mirroring the same tag-based grouping model used by :ref:`Queue.tag_ids <queue-struct-queue>`.
* ``tm_create`` (string, ISO 8601): Timestamp when the case was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update.

.. _contact-case-struct-status:

Status
^^^^^^

.. list-table::
   :header-rows: 1

   * - Value
     - Description
   * - open
     - The case is active
   * - closed
     - The case has been resolved

.. _contact-case-struct-closedreason:

ClosedReason
^^^^^^^^^^^^

.. list-table::
   :header-rows: 1

   * - Value
     - Description
   * - agent_closed
     - Closed explicitly by an agent via ``POST /contact_cases/{id}/close``

.. _contact-case-struct-closedbytype:

ClosedByType
^^^^^^^^^^^^

.. list-table::
   :header-rows: 1

   * - Value
     - Description
   * - agent
     - Closed by an authenticated agent

.. _contact-case-struct-casenote:

CaseNote
--------

Internal, agent-facing annotation on a case. Never surfaced in any customer-facing webhook or channel.

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
* ``customer_id`` (UUID): The customer who owns this note.
* ``case_id`` (UUID): The case this note belongs to.
* ``author_type`` (enum string): Type of the note's author. See :ref:`AuthorType <contact-case-struct-authortype>`.
* ``author_id`` (UUID): ID of the agent authoring this note. Nullable for system-authored notes.
* ``text`` (String): The note's text content.
* ``tm_create`` (string, ISO 8601): Timestamp when the note was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update.
* ``tm_delete`` (string, ISO 8601): Timestamp of soft deletion. ``null`` if active.

.. _contact-case-struct-authortype:

AuthorType
^^^^^^^^^^

.. list-table::
   :header-rows: 1

   * - Value
     - Description
   * - agent
     - Authored by an agent
   * - system
     - Authored automatically by the platform
