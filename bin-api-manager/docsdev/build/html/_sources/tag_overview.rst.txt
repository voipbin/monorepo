.. _tag-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free. Creating and managing tags incurs no charges.
   * **Async:** No. ``POST /tags`` returns immediately with the created tag.

VoIPBIN's Tag API provides a flexible labeling system for organizing and categorizing resources. Tags are primarily used for skill-based routing in queues, but can also categorize agents by teams, departments, languages, or any custom attribute.

With the Tag API you can:

- Create and manage tags for categorization
- Assign tags to agents for skill-based routing
- Configure queue requirements based on tags
- Filter and organize resources by tags
- Build flexible routing strategies


How Tags Work
-------------
Tags create a matching system between agents and queues.

**Tag Architecture**

::

    +-----------------------------------------------------------------------+
    |                           Tag System                                  |
    +-----------------------------------------------------------------------+

    +-------------------+
    |       Tags        |
    |  (skill labels)   |
    +--------+----------+
             |
             | assigned to
             v
    +--------+----------+--------+----------+
    |                   |                   |
    v                   v                   v
    +----------+   +----------+   +----------+
    |  Agents  |   |  Queues  |   |  Other   |
    |  (have)  |   | (require)|   | resources|
    +----+-----+   +----+-----+   +----------+
         |              |
         v              v
    +---------+    +---------+
    | Skills  |    | Filter  |
    | Match   |--->| Agents  |
    +---------+    +---------+

**Key Components**

- **Tag**: A label representing a skill, team, or category
- **Agent Tags**: Skills/attributes an agent possesses
- **Queue Tags**: Requirements for agents to handle calls
- **Tag Matching**: Process of finding qualified agents

.. note:: **AI Implementation Hint**

   Queue tag matching uses **AND logic**: an agent must have **all** tags listed in the queue's ``tag_ids`` to be eligible. If a queue requires ``[english, billing]``, an agent with only ``[english]`` will not match. Tag names must be unique per customer account.


Tag Matching
------------
Queue routing uses tags to find qualified agents.

**Matching Rules**

::

    Queue Requirements:          Agent Skills:
    +-------------------+        +-------------------+
    | Tags:             |        | Tags:             |
    | o english         |        | o english         |
    | o billing         |        | o billing         |
    +-------------------+        | o vip_support     |
              |                  +-------------------+
              |                          |
              v                          v
         +------------------------------------------+
         |  MATCH: Agent has ALL required tags      |
         |  (extra tags like vip_support are OK)    |
         +------------------------------------------+
                          |
                          v
                  Agent is eligible!

**Matching Examples**

::

    Example 1: MATCH
    +--------------------------------------------+
    | Queue requires: [english, billing]         |
    | Agent has: [english, billing, spanish]     |
    | Result: MATCH (has all required)           |
    +--------------------------------------------+

    Example 2: NO MATCH
    +--------------------------------------------+
    | Queue requires: [english, billing]         |
    | Agent has: [english, tech_support]         |
    | Result: NO MATCH (missing "billing")       |
    +--------------------------------------------+

    Example 3: MATCH
    +--------------------------------------------+
    | Queue requires: [english]                  |
    | Agent has: [english, spanish, french]      |
    | Result: MATCH (has required tag)           |
    +--------------------------------------------+


Creating and Managing Tags
--------------------------
Create tags to define skills and categories.

**Create a Tag**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/tags?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "billing",
            "detail": "Agent can handle billing inquiries"
        }'

**List Tags**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/tags?token=<token>'

**Assign Tags to Agent**

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/agents/<agent-id>/tag_ids?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "tag_ids": [
                "uuid-for-english-tag",
                "uuid-for-billing-tag"
            ]
        }'


.. _tag-overview-key_features_and_use_cases:

Key Features and Use Cases
--------------------------
Tags enable various organizational and routing strategies.

**Agent Skills**

::

    Agent: John Smith
    +--------------------------------------------+
    | Skills (Tags):                             |
    | o english - Can communicate in English    |
    | o spanish - Can communicate in Spanish    |
    | o billing - Trained on billing issues     |
    | o tier2   - Senior support level          |
    +--------------------------------------------+

**Team Assignment**

::

    Team Structure:
    +--------------------------------------------+
    | Sales Team:     tag = "sales"             |
    | Support Team:   tag = "support"           |
    | Billing Team:   tag = "billing"           |
    | VIP Team:       tag = "vip"               |
    +--------------------------------------------+

**Language Routing**

::

    Language Tags:
    +--------------------------------------------+
    | english  | spanish  | french  | german   |
    | japanese | korean   | chinese | hindi    |
    +--------------------------------------------+

    Queue: Spanish Support
    Required Tags: [spanish, support]
    -> Routes to agents with both tags

**Skill-Based Routing**

::

    Incoming Call: Technical Issue (Spanish)
         |
         v
    Queue: Tech Support Spanish
    Required: [spanish, tech_support]
         |
         v
    +--------------------------------------------+
    | Available Agents:                          |
    | Agent A: [english, tech_support] - NO     |
    | Agent B: [spanish, sales] - NO            |
    | Agent C: [spanish, tech_support] - YES    |
    +--------------------------------------------+
         |
         v
    Route to Agent C


Tag Categories
--------------
Organize tags into logical categories.

**Recommended Tag Structure**

::

    +-----------------------------------------------------------------------+
    |                        Tag Categories                                 |
    +-----------------------------------------------------------------------+

    Language Skills:
    +-------------------+-------------------+-------------------+
    | lang_english      | lang_spanish      | lang_french       |
    | lang_german       | lang_japanese     | lang_korean       |
    +-------------------+-------------------+-------------------+

    Department/Team:
    +-------------------+-------------------+-------------------+
    | team_sales        | team_support      | team_billing      |
    | team_technical    | team_retention    | team_vip          |
    +-------------------+-------------------+-------------------+

    Skill Level:
    +-------------------+-------------------+-------------------+
    | tier_1            | tier_2            | tier_3            |
    | supervisor        | manager           |                   |
    +-------------------+-------------------+-------------------+

    Product Knowledge:
    +-------------------+-------------------+-------------------+
    | product_basic     | product_premium   | product_enterprise|
    +-------------------+-------------------+-------------------+


Common Scenarios
----------------

**Scenario 1: Multilingual Support Center**

Route calls based on language preference.

::

    Setup:
    +--------------------------------------------+
    | Tags: english, spanish, french, german    |
    |                                            |
    | Queues:                                    |
    | - English Support: [english, support]     |
    | - Spanish Support: [spanish, support]     |
    | - French Support:  [french, support]      |
    |                                            |
    | Agents assigned appropriate language tags |
    +--------------------------------------------+

    Flow:
    IVR: "Press 1 for English, 2 for Spanish..."
         |
         v
    Route to appropriate language queue

**Scenario 2: Tiered Support System**

Escalate based on skill level.

::

    Tier Structure:
    +--------------------------------------------+
    | Tier 1: Basic issues                       |
    |   Queue requires: [tier_1]                 |
    |                                            |
    | Tier 2: Complex issues                     |
    |   Queue requires: [tier_2]                 |
    |                                            |
    | Tier 3: Escalations                        |
    |   Queue requires: [tier_3, supervisor]     |
    +--------------------------------------------+

    Escalation:
    Tier 1 agent can't resolve
         |
         v
    Transfer to Tier 2 queue
         |
         v
    Agent with [tier_2] handles

**Scenario 3: VIP Customer Routing**

Priority routing for VIP customers.

::

    VIP Detection:
    +--------------------------------------------+
    | Customer identified as VIP                 |
    |   (via caller ID or account lookup)        |
    |                                            |
    | Route to VIP Queue                         |
    |   Requires: [vip, senior_agent]            |
    |                                            |
    | Only experienced agents handle VIP calls   |
    +--------------------------------------------+


Best Practices
--------------

**1. Naming Conventions**

- Use consistent naming (lowercase, underscores)
- Group by category (lang_, team_, skill_)
- Keep names short but descriptive
- Document what each tag means

**2. Tag Assignment**

- Assign only relevant tags to agents
- Review and update tags regularly
- Train agents before assigning skill tags
- Remove tags when no longer applicable

**3. Queue Configuration**

- Don't require too many tags (limits eligible agents)
- Use minimum necessary tags for routing
- Consider fallback queues with fewer requirements

**4. Organization**

- Create a tag taxonomy document
- Review tag usage periodically
- Consolidate similar tags
- Archive unused tags


Troubleshooting
---------------

**Routing Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| No agents matched         | Check agents have ALL required tags; reduce    |
|                           | queue tag requirements                         |
+---------------------------+------------------------------------------------+
| Wrong agents receiving    | Verify agent tag assignments; check queue      |
| calls                     | tag requirements                               |
+---------------------------+------------------------------------------------+
| Agent not getting calls   | Ensure agent has required tags; check agent    |
| from queue                | status is "available"                          |
+---------------------------+------------------------------------------------+

**Tag Management Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Tag assignment failed     | Verify tag IDs exist; check API permissions    |
+---------------------------+------------------------------------------------+
| Duplicate tag error       | Tag names must be unique per customer;         |
|                           | use existing tag instead                       |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Agent Overview <agent_overview>` - Agent tag assignment
- :ref:`Queue Overview <queue-overview>` - Queue tag requirements
- :ref:`Customer Overview <customer-overview>` - Resource organization

