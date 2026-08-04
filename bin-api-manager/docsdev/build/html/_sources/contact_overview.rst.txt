.. _contact-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free (contacts are organizational records with no per-operation charges)
   * **Async:** No. All contact CRUD operations are synchronous and return the result immediately.

VoIPBIN's Contact API provides CRM-style contact management for organizing and enriching communication workflows. Each contact can hold multiple phone numbers, email addresses, and tags, enabling caller ID enrichment, routing decisions, and integration with external CRM systems.

With the Contact API you can:

- Create and manage contacts with structured personal and company information
- Store multiple phone numbers and email addresses per contact
- Look up contacts by phone number (E.164) or email address
- Assign tags for categorization and routing
- Track contact origin and link to external CRM systems


How Contacts Work
-----------------
Contacts act as the central identity record for people your platform communicates with.

**Contact Architecture**

::

    +-----------------------------------------------------------------------+
    |                        Contact System                                 |
    +-----------------------------------------------------------------------+

    +-------------------+
    |     Contact       |
    |  (identity record)|
    +--------+----------+
             |
             | has many
             v
    +--------+----------+--------+----------+--------+----------+
    |                   |                   |                   |
    v                   v                   v                   v
    +----------+   +----------+   +----------+   +----------+
    |  Phone   |   |  Email   |   |   Tags   |   | External |
    | Numbers  |   | Addrs    |   |          |   |   ID     |
    +----------+   +----------+   +----------+   +----------+
         |              |              |              |
         v              v              v              v
    +---------+   +---------+   +---------+   +-----------+
    | Lookup  |   | Lookup  |   | Filter  |   | CRM Sync  |
    | by phone|   | by email|   | & route |   | & dedup   |
    +---------+   +---------+   +---------+   +-----------+

**Key Components**

- **Contact**: A person or entity with name, company, and job title
- **Addresses**: A unified list of phone numbers and email addresses per contact. Each address has a ``type`` of either ``tel`` (E.164-normalized phone number) or ``email``, an optional free-form ``name`` label (e.g. "Mobile", "Work") and ``detail`` note, and an ``is_primary`` flag. There is no fixed enum of address sub-types beyond ``tel``/``email`` — labels like "mobile" or "work" are just optional free text, not a constrained category.
- **Tags**: Labels for categorization and skill-based routing
- **Source**: How the contact was created (manual, import, api, sync)
- **External ID**: Reference to an external CRM record for deduplication


Contact Lookup
--------------
Find contacts by phone number or email address. This is useful for enriching incoming calls or messages with caller identity.

**Lookup by Phone Number**

::

    Incoming Call: +15551234567
         |
         v
    +--------------------------------------------+
    | GET https://api.voipbin.net/v1.0/contacts/lookup?phone=+15551234567 |
    +--------------------------------------------+
         |
         v
    +--------------------------------------------+
    | Contact Found:                             |
    |   Name: John Smith                         |
    |   Company: Acme Corp                       |
    |   Tags: [vip, enterprise]                  |
    +--------------------------------------------+
         |
         v
    Route to VIP queue / display caller info

**Lookup by Email**

::

    Incoming Email: john@acme.com
         |
         v
    +--------------------------------------------+
    | GET https://api.voipbin.net/v1.0/contacts/lookup?email=john@acme.com |
    +--------------------------------------------+
         |
         v
    Contact matched → enrich conversation context

Phone numbers are matched in E.164 format for reliable international matching. Email addresses are matched case-insensitively.

.. note:: **AI Implementation Hint**

   When using phone number lookup, the ``+`` character must be URL-encoded as ``%2B``. For example: ``GET https://api.voipbin.net/v1.0/contacts/lookup?phone=%2B15551234567``. Omitting this encoding will result in the ``+`` being interpreted as a space and the lookup will fail.


Source Tracking
---------------
Each contact records how it was created, enabling analytics and deduplication.

**Source Values**

+----------+------------------------------------------------------------------+
| Source   | Description                                                      |
+==========+==================================================================+
| manual   | Created by a user through the admin console or agent interface   |
+----------+------------------------------------------------------------------+
| import   | Imported from a CSV file or bulk upload                          |
+----------+------------------------------------------------------------------+
| api      | Created programmatically via the API                             |
+----------+------------------------------------------------------------------+
| sync     | Synchronized from an external CRM system                         |
+----------+------------------------------------------------------------------+


External CRM Integration
-------------------------
The ``external_id`` field links a contact to its record in an external CRM system such as Salesforce, HubSpot, or Zoho.

**Integration Pattern**

::

    External CRM                     VoIPBIN
    +-------------------+            +-------------------+
    | Salesforce        |            | Contact           |
    | Contact ID:       |   sync     | external_id:      |
    | 003Dn00000X1234   |----------->| 003Dn00000X1234   |
    | Name: John Smith  |            | source: sync      |
    +-------------------+            +-------------------+

**Use Cases**

- **Deduplication**: Prevent duplicate contacts during re-imports by matching on ``external_id``
- **Two-Way Sync**: Keep contact data consistent between VoIPBIN and your CRM
- **Referential Integrity**: Maintain links between VoIPBIN contacts and CRM records


.. _contact-overview-key_features:

Key Features and Use Cases
--------------------------
Contacts support various communication and organizational workflows.

**Caller ID Enrichment**

::

    Incoming Call
         |
         v
    Lookup contact by phone number
         |
         v
    +--------------------------------------------+
    | Display to agent:                          |
    |   Caller: John Smith                       |
    |   Company: Acme Corp                       |
    |   Tags: enterprise, vip                    |
    |   Previous interactions: 12                |
    +--------------------------------------------+

**Contact-Based Routing**

::

    Incoming Call from known contact
         |
         v
    Lookup contact → check tags
         |
         v
    +--------------------------------------------+
    | Contact Tags: [enterprise, spanish]        |
    |                                            |
    | Route to: Enterprise Spanish Support Queue |
    +--------------------------------------------+

.. note:: **Two independent tag mechanisms**

   You can use a contact's tags in your own flow logic (e.g. a ``condition_variable`` action) to route the call to a *specific queue* dedicated to that skill combination, as shown above. Once the call reaches a queue, agent selection is a separate step that filters by *agent* tags against the *queue's* ``tag_ids`` (see :ref:`Agent Searching <queue-overview>`) -- contact tags themselves are not consulted at agent-selection time. To guarantee skill-matched agents, either route to a queue whose ``tag_ids`` require the matching skill, or use one queue per skill combination staffed only with agents who have that skill.

**Multi-Channel Contact History**

::

    Contact: John Smith
    +--------------------------------------------+
    | Addresses:                                  |
    |   +15551234567 (tel, "Mobile", primary)    |
    |   +15559876543 (tel, "Work")               |
    |   john@acme.com (email, "Work", primary)   |
    |   john.smith@gmail.com (email, "Personal") |
    |                                            |
    | All channels linked to one identity        |
    +--------------------------------------------+


Best Practices
--------------

**1. Phone Number Format**

- Always provide phone numbers in E.164 format (e.g., +15551234567)
- The system normalizes numbers to E.164 for consistent matching
- Mark one number as primary for outbound communication

**2. Contact Organization**

- Use tags to categorize contacts (e.g., vip, enterprise, partner)
- Set ``source`` accurately to track where contacts originate
- Use ``external_id`` when integrating with external CRM systems

**3. Lookup Best Practices**

- Use phone lookup for incoming call enrichment
- Use email lookup for incoming message enrichment
- Lookup returns the first matching contact

**4. Data Quality**

- Keep display names consistent (first + last name)
- Use the address ``name`` label (e.g. "Work", "Personal") to help distinguish multiple addresses of the same ``type``
- Remove outdated phone numbers and emails regularly


Troubleshooting
---------------

* **Phone number lookup returns no results:**
    * **Cause:** The phone number is not in E.164 format, or the ``+`` is not URL-encoded as ``%2B``.
    * **Fix:** Ensure the number starts with ``+`` followed by country code and subscriber number (e.g., ``+15551234567``). URL-encode the ``+`` as ``%2B`` in the query string.

* **Duplicate contacts created during re-import:**
    * **Cause:** The ``external_id`` field was not set during the initial import.
    * **Fix:** Always set ``external_id`` when importing contacts from external systems. Use ``external_id`` to check for existing records before creating new ones.

* **Contact lookup by email returns wrong contact:**
    * **Cause:** Multiple contacts share the same email address.
    * **Fix:** Lookup returns the first matching contact. Ensure email addresses are unique per contact, or use the contact ID directly if you know the specific record.

* **404 Not Found when accessing a contact:**
    * **Cause:** The contact UUID does not exist or belongs to another customer.
    * **Fix:** Verify the UUID was obtained from a recent ``GET https://api.voipbin.net/v1.0/contacts`` list call with your authentication token.


Related Documentation
---------------------

- :ref:`Contact Addresses Overview <contact-address-overview>` - Standalone address resource: unresolved addresses, cross-contact search, and claiming
- :ref:`Contact Cases Overview <contact-case-overview>` - Case management for grouping and tracking related interactions with a contact
- :ref:`Peer Events Overview <contact-peer-event-overview>` - Raw interaction/activity log for a contact's addresses
- :ref:`Tag Overview <tag-overview>` - Tag management for contact categorization
- :ref:`Agent Overview <agent-overview>` - Agents that interact with contacts
- :ref:`Queue Overview <queue-overview>` - Routing based on contact attributes
- :ref:`Conversation Overview <conversation-overview>` - Conversations with contacts
