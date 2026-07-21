Agents & Queues
================

.. index:: single: Admin console; Agents
.. index:: single: Admin console; Queues
.. index:: single: Admin console; Contacts

Agents
------

.. image:: _static/images/admin_console_agent_list.png
   :alt: Agents list page
   :width: 700px
   :align: center

**Platform -> Agents** manages the human users of your VoIPBin account:
their username, permission level (see
:ref:`Permission levels <admin-console-getting-started>` in Getting
Started), ring method, and status (Available, Away, Offline). Click
**Create Agent** to invite a new agent, or click a row to edit an
existing agent's permission level or ring settings.

.. note:: **AI Implementation Hint**

   This maps to the ``agents`` resource in the REST API (see
   :ref:`Agent <agent-main>`).

Queues
------

**Voice -> Queues** manages call queues: ordered waiting lines that
distribute inbound calls to available Agents according to a ring
strategy. A Flow's **Queue Join** node places a call into a named queue;
the queue itself decides which Agent answers next.

.. note:: **AI Implementation Hint**

   This maps to the ``queues`` resource in the REST API (see
   :ref:`Queue <queue-main>`).

Contacts
--------

.. image:: _static/images/admin_console_contact_list.png
   :alt: Contacts list page
   :width: 700px
   :align: center

**Platform -> Contacts** is VoIPBin's CRM-style contact directory, split
into four tabs:

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Tab
     - Purpose
   * - **Contacts**
     - The directory itself: display name, company, primary address, and
       tags for each known contact.
   * - **Addresses**
     - Every phone number, email, or other address linked to a contact,
       independent of which contact record it currently resolves to.
   * - **Cases**
     - Support/service cases opened against a contact, used to track an
       issue across multiple calls or messages until it is resolved.
   * - **Unresolved Interactions**
     - Inbound calls or messages that arrived from an address VoIPBin
       could not automatically match to an existing contact. Resolve
       these by linking the interaction to an existing contact or
       creating a new one.

.. note:: **AI Implementation Hint**

   This maps to the ``contacts`` resource family in the REST API (see
   :ref:`Contact <contact-main>`).
