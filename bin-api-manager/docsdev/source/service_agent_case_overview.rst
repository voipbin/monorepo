.. _service_agent-case-overview:

Case Management
================

.. note:: **AI Context**

   * **Complexity:** Medium
   * **Cost:** Free -- case management has no per-operation charge.
   * **Async:** No. All ``/service_agents/contact_cases*`` endpoints are synchronous.

A **Case** is a lightweight, per-channel session header that groups the interactions with a single remote party (a phone number, an email address, ...) into a start/end unit an agent can pick up, work, and close. Where a :ref:`Contact <contact-main>` is a durable identity record and :ref:`Interactions <contact-overview>` are the raw event log, a Case is the working unit an agent-facing console builds its queue/inbox UI around: "here is everything related to this customer's current issue, and who owns it."

With the Case API you can:

- List and inspect the cases open for your customer
- Assign a case to a specific agent, or close it once resolved
- Attach or detach the case's resolved :ref:`Contact <contact-main>`
- Attach internal, agent-only notes that never leak into customer-facing webhooks or responses

Cases are created automatically by the platform (e.g. by a Flow action or an AI tool when a new call/conversation arrives for a peer with no open Case) -- there is no ``POST /service_agents/contact_cases`` endpoint. The Agent Console surface only reads, assigns, annotates, and closes cases.


How Cases Work
---------------

**Case Lifecycle**

::

    +------------------------------------------------------------------+
    |                          Case Lifecycle                          |
    +------------------------------------------------------------------+

     New interaction arrives
     for a peer with no open Case
              |
              v
       +-------------+      assign      +-------------+
       |    open     |----------------->|    open     |
       | (unowned)   |                  | (owned by   |
       +------+------+                  |   agent)    |
              |                         +------+------+
              | close                          | close
              v                                v
       +-------------+                  +-------------+
       |   closed    |<-----------------|   closed    |
       +------+------+                  +-------------+
              |
              | re-contact (new interaction for same peer)
              v
       +-------------+
       | new open    |
       | Case, chained via previous_case_id
       +-------------+

- **Status** is either ``open`` or ``closed``. Cases start ``open`` and unowned.
- **Owner** (``owner_type`` / ``owner_id``) identifies which agent is working the case. Assignment does not change status -- an assigned case stays ``open`` until explicitly closed.
- **Closing** a case is permanent for that Case instance; a subsequent interaction from the same peer creates a **new** Case with ``previous_case_id`` pointing at the just-closed one, so the console can show re-contact history.
- The owner is **never cleared** by closing a case -- ``owner_id`` remains a durable record of who last worked it, even after ``status`` becomes ``closed``.


Endpoints
---------

.. list-table::
   :header-rows: 1

   * - Method
     - Path
     - Description
   * - GET
     - ``/service_agents/contact_cases``
     - List cases for the caller's customer (paginated, no server-side owner filter -- filter client-side).
   * - GET
     - ``/service_agents/contact_cases/{id}``
     - Get a single case by ID. Any agent of the customer may view any case.
   * - PUT
     - ``/service_agents/contact_cases/{id}``
     - Attach (non-empty ``contact_id``) or detach (empty string) the case's Contact.
   * - POST
     - ``/service_agents/contact_cases/{id}/assign``
     - Assign the case to an owner agent (``owner_type`` is always ``"agent"``, fixed server-side).
   * - POST
     - ``/service_agents/contact_cases/{id}/close``
     - Close the case. ``closed_by_id`` is derived from the caller's own identity.
   * - GET
     - ``/service_agents/contact_cases/{id}/notes``
     - List notes on the case, in creation order.
   * - POST
     - ``/service_agents/contact_cases/{id}/notes``
     - Create a note on the case, authored by the caller.
   * - DELETE
     - ``/service_agents/contact_cases/{id}/notes/{note_id}``
     - Delete a note. Only the note's own author may delete it.

See :ref:`Structures <service_agent-case-struct>` for the ``Case`` and ``CaseNote`` field reference.


Listing and Viewing Cases
---------------------------

**List cases**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/service_agents/contact_cases?page_size=50&token=<token>'

**Get a case**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/service_agents/contact_cases/<case-id>?token=<token>'

.. note:: **AI Implementation Hint**

   ``GET /service_agents/contact_cases`` returns every open and closed case for the customer with no server-side ``status``/``owner_id`` filter -- console UIs (e.g. an agent's "My Cases" view) are expected to filter the result client-side. There is no ``status=open`` or ``owner_id=<my-agent-id>`` query parameter on this endpoint today.


Assigning and Closing
------------------------
Assignment and closing are separate operations: assigning a case to an agent does not close it, and closing a case does not require it to have an owner.

**Assign a case to an agent**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/service_agents/contact_cases/<case-id>/assign?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "owner_id": "<agent-id>"
        }'

``owner_id`` must reference an existing agent of the **same customer** as the caller. A cross-tenant or nonexistent agent ID and a nonexistent case ID both return the same ``404`` -- the API deliberately does not reveal which one failed.

**Close a case**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/service_agents/contact_cases/<case-id>/close?token=<token>'

``closed_by_type``/``closed_by_id`` are always derived from the caller's own authenticated agent identity -- the request body carries no fields, so a case can never be closed "as" another agent.

**Attach or detach the case's Contact**

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/service_agents/contact_cases/<case-id>?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "contact_id": "<contact-id>"
        }'

Send an empty string for ``contact_id`` to detach:

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/service_agents/contact_cases/<case-id>?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "contact_id": ""
        }'

Every attach/detach is recorded as a ``case_contact_attributed``/``case_contact_detached`` audit event, queryable via bin-timeline-manager.


Case Notes
-----------
Notes are an **internal, agent-facing annotation** on a Case -- free-text scratchpad entries like "called back, no answer" that help agents hand a case off to each other. Notes are physically and transport-isolated from customer-facing data: they never appear in any customer webhook, in ``GET /contact_cases`` (the admin/manager surface), or in any other customer-visible response.

**List notes**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/service_agents/contact_cases/<case-id>/notes?token=<token>'

**Create a note**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/service_agents/contact_cases/<case-id>/notes?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "text": "Called the customer back, no answer."
        }'

The note's ``author_type``/``author_id`` are always derived server-side from the caller's authenticated agent identity -- there is no way to author a note as another agent or as the system through this endpoint.

**Delete a note**

.. code::

    $ curl -X DELETE 'https://api.voipbin.net/v1.0/service_agents/contact_cases/<case-id>/notes/<note-id>?token=<token>'

.. note:: **AI Implementation Hint**

   An agent may only delete a note it authored itself. Attempting to delete a note authored by another agent, or a system-authored note (``author_id`` is null), returns ``403 PermissionDenied`` -- not ``404``, so the caller can distinguish "not my note" from "note doesn't exist".


Relationship to the Admin/Manager Case API
----------------------------------------------
The same underlying Case resource is also exposed at the top level (``/contact_cases``, gated by admin/manager permission) -- see :ref:`Case Overview <contact-case-overview>`. That surface additionally supports ``POST /contact_cases/{id}/continue`` (reopen a closed case) and sending an outbound conversation message tied to the case, neither of which exists under ``/service_agents/contact_cases``. Conversely, ``POST /service_agents/contact_cases/{id}/assign`` (owner assignment) exists only on this Agent Console surface -- there is no top-level equivalent.


Related Documentation
-----------------------

- :ref:`Case Overview <contact-case-overview>` -- the admin/manager-facing surface for the same resource
- :ref:`Agent Console Overview <service_agent-overview>` -- authentication and authorization model for this surface
- :ref:`Contact Overview <contact-overview>` -- contacts a case may be attributed to
- :ref:`Peer Events Overview <contact-peer-event-overview>` -- the raw activity log a case groups
