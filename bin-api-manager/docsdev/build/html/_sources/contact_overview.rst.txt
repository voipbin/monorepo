.. _contact-overview:

Overview
========
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
- **Phone Numbers**: Multiple numbers per contact with type (mobile, work, home, fax, other) and E.164 normalization
- **Emails**: Multiple addresses per contact with type (work, personal, other)
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
    | GET /contacts/lookup?phone=+15551234567    |
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
    | GET /contacts/lookup?email=john@acme.com   |
    +--------------------------------------------+
         |
         v
    Contact matched → enrich conversation context

Phone numbers are matched in E.164 format for reliable international matching. Email addresses are matched case-insensitively.


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
    | Required agent tags: [enterprise, spanish] |
    +--------------------------------------------+

**Multi-Channel Contact History**

::

    Contact: John Smith
    +--------------------------------------------+
    | Phone Numbers:                             |
    |   +15551234567 (mobile, primary)          |
    |   +15559876543 (work)                     |
    |                                            |
    | Emails:                                    |
    |   john@acme.com (work, primary)           |
    |   john.smith@gmail.com (personal)         |
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
- Assign email types (work, personal) for proper channel selection
- Remove outdated phone numbers and emails regularly


Related Documentation
---------------------

- :ref:`Tag Overview <tag-overview>` - Tag management for contact categorization
- :ref:`Agent Overview <agent_overview>` - Agents that interact with contacts
- :ref:`Queue Overview <queue-overview>` - Routing based on contact attributes
- :ref:`Conversation Overview <conversation-overview>` - Conversations with contacts
