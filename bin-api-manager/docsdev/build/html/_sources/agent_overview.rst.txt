.. _agent_overview:

Overview
========
The agent, also known as the call center agent or phone agent, plays a crucial role as a representative of a company, handling calls with private or business customers on behalf of the organization. Typically, agents work in a call center environment, where multiple agents are employed to efficiently manage incoming and outgoing calls. The call center may be operated by the company itself or outsourced to an external service provider. In the case of external service providers, a single site may serve various clients from different businesses.

In VoIPBIN, agents are the people (or endpoints) that receive calls from queues. They have statuses, skills (tags), and contact addresses that determine when and how they can receive calls.


Agent Status
------------
Every agent has a status that reflects their current availability. The status determines whether an agent can receive calls from queues.

**Status Overview**

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                         Agent Status States                              │
    └─────────────────────────────────────────────────────────────────────────┘

                      ┌───────────────────────────────────────────┐
                      │              User Actions                  │
                      │   (Agent manually changes their status)    │
                      └───────────────────┬───────────────────────┘
                                          │
              ┌───────────────────────────┼───────────────────────────┐
              │                           │                           │
              ▼                           ▼                           ▼
        ┌───────────┐              ┌───────────┐              ┌───────────┐
        │ available │              │   away    │              │  offline  │
        │ (ready)   │              │ (break)   │              │(logged out)│
        └─────┬─────┘              └───────────┘              └───────────┘
              │
              │ Call routed to agent
              ▼
        ┌───────────┐
        │  ringing  │
        │ (incoming)│
        └─────┬─────┘
              │
              │ Agent answers
              ▼
        ┌───────────┐
        │   busy    │
        │ (on call) │
        └─────┬─────┘
              │
              │ Call ends
              ▼
        (returns to previous status)

**Status Descriptions**

+-----------+-----------------------------------------------------------------+------------------+
| Status    | What it means                                                   | Can receive      |
|           |                                                                 | queue calls?     |
+===========+=================================================================+==================+
| available | Agent is logged in and ready to take calls                      | Yes              |
+-----------+-----------------------------------------------------------------+------------------+
| away      | Agent is temporarily unavailable (break, meeting, etc.)         | No               |
+-----------+-----------------------------------------------------------------+------------------+
| busy      | Agent is currently handling a call                              | No               |
+-----------+-----------------------------------------------------------------+------------------+
| offline   | Agent is logged out of the system                               | No               |
+-----------+-----------------------------------------------------------------+------------------+
| ringing   | System is attempting to deliver a call to the agent             | No               |
+-----------+-----------------------------------------------------------------+------------------+

**Status Transitions**

::

    Manual transitions (agent controls):
    ┌───────────┐ ◀────────────────────────────────▶ ┌───────────┐
    │ available │          User changes status       │   away    │
    └───────────┘ ◀────────────────────────────────▶ └───────────┘
          ▲                                                ▲
          │              User logs in/out                  │
          ▼                                                ▼
    ┌───────────┐ ◀────────────────────────────────▶ ┌───────────┐
    │  offline  │                                    │  offline  │
    └───────────┘                                    └───────────┘

    Automatic transitions (system controls):
    ┌───────────┐        Call routed         ┌───────────┐
    │ available │───────────────────────────▶│  ringing  │
    └───────────┘                            └─────┬─────┘
          ▲                                        │
          │ Call ends                              │ Agent answers
          │ (or timeout)                           ▼
          │                                  ┌───────────┐
          └──────────────────────────────────│   busy    │
                                             └───────────┘


Agent Tags (Skills)
-------------------
Tags define what skills or groups an agent belongs to. They're used for skill-based routing from queues.

**How Tags Work**

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                         Skill-Based Routing                              │
    └─────────────────────────────────────────────────────────────────────────┘

    Agent Tags:                              Queue Requirements:
    ┌─────────────────────────┐             ┌─────────────────────────┐
    │ Agent Smith             │             │ Support Queue           │
    │ Tags:                   │             │ Required Tags:          │
    │  • english              │             │  • english              │
    │  • billing              │             │  • billing              │
    │  • vip_support          │             └─────────────────────────┘
    └─────────────────────────┘                        │
              │                                        │
              └────────────────────┬───────────────────┘
                                   │
                                   ▼
                          ┌───────────────────┐
                          │ Agent Smith has   │
                          │ ALL required tags │
                          │ → Eligible!       │
                          └───────────────────┘

**Tag Matching Rules**

- Agent must have **ALL** tags the queue requires
- Having extra tags is fine (agent has "vip_support" but queue doesn't require it)
- Tags can represent anything: languages, skills, departments, locations

::

    Example Tag Structure:
    ┌─────────────────────────────────────────────────────────────────────────┐
    │                                                                          │
    │  Language Tags:     english, spanish, french, german                    │
    │                                                                          │
    │  Skill Tags:        billing, tech_support, sales, returns               │
    │                                                                          │
    │  Tier Tags:         tier1, tier2, supervisor, manager                   │
    │                                                                          │
    │  Department Tags:   customer_service, operations, hr                    │
    │                                                                          │
    └─────────────────────────────────────────────────────────────────────────┘

**Managing Agent Tags**

Update an agent's tags via the API:

::

    PUT /v1/agents/{agent-id}/tag_ids

    {
        "tag_ids": [
            "uuid-for-english-tag",
            "uuid-for-billing-tag",
            "uuid-for-tier2-tag"
        ]
    }


Contact Addresses
-----------------
Each agent can have multiple contact addresses - these are the endpoints where calls are delivered when the agent is selected.

**Address Types**

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                        Agent Contact Addresses                           │
    └─────────────────────────────────────────────────────────────────────────┘

    Agent: John Smith
    ┌─────────────────────────────────────────────────────────────────────────┐
    │ Addresses:                                                               │
    │                                                                          │
    │  1. Extension: 1001                                                     │
    │     → Rings desk phone registered to extension 1001                     │
    │                                                                          │
    │  2. Tel: +14155551234                                                   │
    │     → Rings mobile phone at this number                                 │
    │                                                                          │
    │  3. SIP: john@company.com                                               │
    │     → Rings SIP softphone client                                        │
    │                                                                          │
    └─────────────────────────────────────────────────────────────────────────┘

+------------+----------------------------------------------------------------+
| Type       | Description                                                    |
+============+================================================================+
| extension  | Internal extension number (must be registered with VoIPBIN)    |
+------------+----------------------------------------------------------------+
| tel        | External phone number in E.164 format (+15551234567)           |
+------------+----------------------------------------------------------------+
| sip        | Direct SIP address (user@domain)                               |
+------------+----------------------------------------------------------------+

**Address Uniqueness**

Each address can only belong to one agent per customer:

::

    ✓ Agent A has extension 1001 in Customer X
    ✓ Agent B has extension 1001 in Customer Y  (different customer, OK)
    ✗ Agent B has extension 1001 in Customer X  (same customer, CONFLICT)


Call to agent
-------------
To reach an agent, VoIPBIN employs a system that allows the agent to have multiple addresses. When a call is initiated to agents, VoIPBIN generates calls to every agent's address simultaneously. If an agent answers one of the calls, VoIPBIN automatically terminates the other calls, streamlining the communication process and ensuring that only one connection is established with the available agent.

**Simultaneous Ring**

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                      Agent with Multiple Addresses                       │
    └─────────────────────────────────────────────────────────────────────────┘

    Queue finds Agent Smith (available)
                   │
                   ▼
    ┌──────────────────────────────┐
    │  Ring ALL addresses at once  │
    └──────────────────────────────┘
                   │
         ┌─────────┼─────────┐
         ▼         ▼         ▼
    ┌─────────┐ ┌─────────┐ ┌─────────┐
    │ Ext 1001│ │ Mobile  │ │ SIP App │
    │  RING!  │ │  RING!  │ │  RING!  │
    └────┬────┘ └────┬────┘ └────┬────┘
         │          │          │
         │          │          │ Agent answers on mobile!
         │          │          ▼
         │          │    ┌───────────┐
         │          │    │ CONNECTED │
         │          │    └───────────┘
         │          │
         ▼          ▼
    ┌─────────┐ ┌─────────┐
    │ CANCEL  │ │ CANCEL  │  ← Other calls automatically cancelled
    └─────────┘ └─────────┘

.. image:: _static/images/agent_call.png

This approach enables efficient call handling, minimizing the time customers spend waiting for an available agent. The call distribution mechanism ensures that agents are optimally utilized, enhancing customer service and overall call center productivity.


Agent Lifecycle
---------------
Understanding the agent lifecycle helps you manage your call center effectively.

**Creating an Agent**

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                         Agent Creation                                   │
    └─────────────────────────────────────────────────────────────────────────┘

    POST /v1/agents
    ┌───────────────────────────────────────┐
    │ {                                     │
    │   "username": "john@company.com",     │
    │   "password": "secure_password",      │
    │   "name": "John Smith",               │
    │   "tag_ids": [...],                   │
    │   "addresses": [...]                  │
    │ }                                     │
    └───────────────────────────────────────┘
                     │
                     ▼
    ┌───────────────────────────────────────┐
    │ Agent Created                         │
    │ • Status: offline (default)           │
    │ • Permission: user (default)          │
    │ • Ready to be configured              │
    └───────────────────────────────────────┘

**Agent Login Flow**

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                          Agent Login                                     │
    └─────────────────────────────────────────────────────────────────────────┘

    1. Agent calls login API
       POST /v1/login
       { "username": "...", "password": "..." }
              │
              ▼
    2. System validates credentials
       ┌────────────────┐
       │ Check password │
       │ hash in DB     │
       └────────────────┘
              │
              ▼
    3. Return agent info
       ┌────────────────────────────────────┐
       │ {                                  │
       │   "id": "agent-uuid",              │
       │   "name": "John Smith",            │
       │   "status": "offline",             │
       │   "permission": "user",            │
       │   ...                              │
       │ }                                  │
       └────────────────────────────────────┘
              │
              ▼
    4. Agent sets status to "available"
       PUT /v1/agents/{id}/status
       { "status": "available" }
              │
              ▼
    5. Agent is now ready to receive calls!

**Guest Agent**

Every customer account automatically has a guest agent created:

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                          Guest Agent                                     │
    └─────────────────────────────────────────────────────────────────────────┘

    • Created automatically when customer account is created
    • Has admin permissions
    • Cannot be deleted
    • Cannot have password changed
    • Ensures account always has at least one admin


How Agents Receive Queue Calls
------------------------------
The complete flow of how a call is routed from a queue to an agent:

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                    Queue to Agent Call Flow                              │
    └─────────────────────────────────────────────────────────────────────────┘

    Step 1: Call waiting in queue
    ┌──────────────────────┐
    │ Queuecall: waiting   │
    │ Queue: Support       │
    │ Required tags:       │
    │  • english           │
    │  • billing           │
    └──────────┬───────────┘
               │
               ▼
    Step 2: Queue searches for agents
    ┌──────────────────────────────────────────────────────────────────────┐
    │ SELECT agents WHERE:                                                  │
    │   • customer_id = queue's customer_id                                │
    │   • status = 'available'                                             │
    │   • has ALL required tags (english AND billing)                      │
    └──────────────────────────────────────────────────────────────────────┘
               │
               ▼
    Step 3: Select one agent (random)
    ┌──────────────────────────────────────────────────────────────────────┐
    │ Found: Agent A, Agent C, Agent E                                      │
    │ Selected: Agent C (random)                                            │
    └──────────────────────────────────────────────────────────────────────┘
               │
               ▼
    Step 4: Route call to agent
    ┌──────────────────────┐
    │ Agent C status:      │
    │ available → ringing  │
    └──────────┬───────────┘
               │
               │ Agent's phones ring
               ▼
    Step 5: Agent answers
    ┌──────────────────────┐
    │ Agent C status:      │
    │ ringing → busy       │
    │                      │
    │ Queuecall status:    │
    │ waiting → service    │
    └──────────────────────┘


Status Events and Webhooks
--------------------------
Agent status changes trigger events that you can subscribe to:

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                      Agent Status Events                                 │
    └─────────────────────────────────────────────────────────────────────────┘

    Agent changes status
           │
           ▼
    ┌──────────────────────┐
    │ agent_status_updated │
    │ event generated      │
    └──────────┬───────────┘
               │
               ├─────────────────────────────────────────┐
               │                                         │
               ▼                                         ▼
    ┌──────────────────────┐              ┌──────────────────────┐
    │ Webhook notification │              │ Internal event       │
    │ to your endpoint     │              │ (other services)     │
    └──────────────────────┘              └──────────────────────┘

**Example Webhook Payload**

::

    {
        "event_type": "agent_status_updated",
        "data": {
            "id": "agent-uuid",
            "name": "John Smith",
            "status": "available",
            "previous_status": "offline",
            "tm_update": "2024-01-15T10:30:00Z"
        }
    }


Permission
----------
In the VoIPBIN ecosystem, permissions play a crucial role in governing the actions that can be performed by the system's agents. Each API within VoIPBIN is subject to specific permission limitations, ensuring a secure and controlled environment.

**Permission Levels**

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                       Permission Hierarchy                               │
    └─────────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                           ADMIN                                          │
    │  • Full access to all APIs                                              │
    │  • Can manage other agents                                              │
    │  • Can view billing and account settings                                │
    │  • Can create/delete resources                                          │
    └─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ More restricted
                                    ▼
    ┌─────────────────────────────────────────────────────────────────────────┐
    │                            USER                                          │
    │  • Can view and use assigned resources                                  │
    │  • Can update own status                                                │
    │  • Cannot manage other agents                                           │
    │  • Limited administrative access                                        │
    └─────────────────────────────────────────────────────────────────────────┘

VoIPBIN employs a robust permission framework to regulate access to its APIs, enhancing security and preventing unauthorized actions. Agents, representing entities interacting with the system, are assigned permissions that align with their intended functionalities.

Every API in VoIPBIN is associated with granular permission limitations. These limitations are designed to:

* Restrict Access: Ensure that only authorized agents can invoke specific APIs.

For clarity, consider an example where an agent is granted permission to access the "activeflows" API but is restricted from invoking certain actions within it. This granular control ensures that agents operate within defined boundaries.


Best Practices
--------------

**1. Set Up Appropriate Tags**

::

    Good tag design:
    ┌─────────────────────────────────────────┐
    │ • Use consistent naming                 │
    │ • Group by category (skill_, lang_)     │
    │ • Keep tags meaningful and specific     │
    │ • Document what each tag means          │
    └─────────────────────────────────────────┘

**2. Configure Multiple Addresses**

::

    For reliability:
    ┌─────────────────────────────────────────┐
    │ Agent Smith:                            │
    │  • Primary: Extension 1001 (desk)       │
    │  • Backup: +15551234567 (mobile)        │
    │                                         │
    │ → If desk phone is busy/down,           │
    │   mobile still rings                    │
    └─────────────────────────────────────────┘

**3. Monitor Agent Status**

- Subscribe to status webhooks to track agent availability
- Build dashboards showing real-time agent status
- Set up alerts for agents stuck in unusual states

**4. Handle Status Transitions Properly**

::

    Common flow:
    1. Agent logs in → Status: offline
    2. Agent sets available → Status: available
    3. Call comes in → Status: ringing (automatic)
    4. Agent answers → Status: busy (automatic)
    5. Call ends → Status: available (if was available before)
    6. Agent takes break → Status: away (manual)
    7. Agent logs out → Status: offline (manual)


Common Scenarios
----------------

**Scenario 1: Agent Day Start**

::

    08:55 - Agent logs in
           Status: offline

    09:00 - Agent sets available
           Status: available
           → Now eligible for queue calls

**Scenario 2: Taking a Break**

::

    Agent is available, receives call
           Status: available → ringing → busy

    Call ends
           Status: busy → available

    Agent takes lunch break
           Status: available → away
           → Not eligible for calls

    Agent returns
           Status: away → available
           → Eligible again

**Scenario 3: End of Day**

::

    Agent finishes last call
           Status: busy → available

    Agent logs out for the day
           Status: available → offline
           → Not eligible for calls
