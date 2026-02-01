.. _campaign-overview:

Overview
========
VoIPBIN's Campaign API provides a comprehensive platform for managing large-scale outbound communication campaigns. Whether you need to make thousands of calls, send bulk SMS, or deliver email notifications, the Campaign API orchestrates the entire process with intelligent dialing strategies and automatic retry handling.

With the Campaign API you can:

- Execute mass outbound calling campaigns
- Send bulk SMS and email notifications
- Configure intelligent retry strategies
- Monitor campaign progress in real-time
- Integrate with agent queues for live connections


How Campaigns Work
------------------
A campaign coordinates multiple resources to execute outbound communications efficiently.

**Campaign Architecture**

::

    +-----------------------------------------------------------------------+
    |                         Campaign System                               |
    +-----------------------------------------------------------------------+

    +-------------------+
    |     Campaign      |
    |  (orchestrator)   |
    +--------+----------+
             |
             | controls
             v
    +--------+----------+--------+----------+--------+----------+
    |                   |                   |                   |
    v                   v                   v                   v
    +----------+   +----------+   +----------+   +----------+
    |  Outdial |   |  Outplan |   |   Queue  |   |   Flow   |
    |  (who)   |   |  (how)   |   |  (agents)|   |  (what)  |
    +----------+   +----------+   +----------+   +----------+
         |              |              |              |
         v              v              v              v
    +---------+    +---------+    +---------+    +---------+
    | Targets |    | Retry   |    | Agent   |    | Actions |
    | to dial |    | rules   |    | pool    |    | to run  |
    +---------+    +---------+    +---------+    +---------+

**Key Components**

- **Campaign**: The orchestrator that manages the entire outbound operation
- **Outdial**: Contains the list of target destinations (phone numbers, emails)
- **Outplan**: Defines the dialing strategy (timing, retries, intervals)
- **Queue**: Groups agents who handle answered calls
- **Flow**: Specifies actions to take when calls connect


Campaign Lifecycle
------------------
Campaigns progress through predictable states.

**Campaign States**

::

    POST /campaigns (status: stop)
           |
           v
    +------------+
    |    stop    |<-----------------+
    +-----+------+                  |
          |                         |
          | start                   | stop
          v                         |
    +------------+                  |
    |   running  |------------------+
    +-----+------+
          |
          | all targets completed
          v
    +------------+
    |  finished  |
    +------------+

**State Descriptions**

+-------------+------------------------------------------------------------------+
| State       | What's happening                                                 |
+=============+==================================================================+
| stop        | Campaign created but not active; no dialing occurring            |
+-------------+------------------------------------------------------------------+
| running     | Campaign actively dialing targets based on outplan               |
+-------------+------------------------------------------------------------------+
| finished    | All targets attempted; campaign completed                        |
+-------------+------------------------------------------------------------------+


Dialing Process
---------------
The campaign dialer processes targets according to the outplan strategy.

**Dialing Flow**

::

    Campaign Running
          |
          v
    +-------------------+
    | Get next target   |
    | from outdial      |
    +--------+----------+
             |
             v
    +-------------------+
    | Apply outplan     |
    | (timing, rules)   |
    +--------+----------+
             |
             v
    +-------------------+     Success     +-------------------+
    | Dial target       |---------------->| Execute flow      |
    |                   |                 | (connect to agent)|
    +--------+----------+                 +-------------------+
             |
             | Failed (busy, no answer, etc.)
             v
    +-------------------+     Yes         +-------------------+
    | Retry available?  |---------------->| Schedule retry    |
    |                   |                 | per outplan       |
    +--------+----------+                 +-------------------+
             |
             | No retries left
             v
    +-------------------+
    | Mark as failed    |
    | Move to next      |
    +-------------------+


5W1H Framework
--------------
The campaign system follows the 5W1H principle for clarity.

::

    +-----------------------------------------------------------------------+
    |                        5W1H Campaign Framework                        |
    +-----------------------------------------------------------------------+

    +-----------+------------------+----------------------------------------+
    | Question  | Component        | Purpose                                |
    +===========+==================+========================================+
    | WHY/WHAT  | Campaign + Flow  | Purpose of calling and actions         |
    |           |                  | to take after connection               |
    +-----------+------------------+----------------------------------------+
    | WHO       | Queue + Agents   | People who handle answered calls       |
    +-----------+------------------+----------------------------------------+
    | WHERE     | Outdial          | Target destinations to reach           |
    +-----------+------------------+----------------------------------------+
    | HOW/WHEN  | Outplan          | Dialing strategy and timing            |
    +-----------+------------------+----------------------------------------+


Creating a Campaign
-------------------
Create a campaign by defining its components.

**Create Campaign Example**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/campaigns?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "Customer Survey Q1",
            "detail": "Quarterly satisfaction survey",
            "type": "call",
            "service_level": 80,
            "end_handle": "stop",
            "outdial_id": "<outdial-id>",
            "outplan_id": "<outplan-id>",
            "queue_id": "<queue-id>",
            "flow_id": "<flow-id>",
            "source": "+15551234567"
        }'

**Start Campaign**

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/campaigns/<campaign-id>?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "status": "running"
        }'


Campaign Types
--------------
VoIPBIN supports different campaign types for various communication needs.

+------------+------------------------------------------------------------------+
| Type       | Description                                                      |
+============+==================================================================+
| call       | Voice calls to phone numbers, with optional agent connection     |
+------------+------------------------------------------------------------------+
| message    | SMS/MMS messages to mobile numbers                               |
+------------+------------------------------------------------------------------+
| email      | Email campaigns to addresses                                     |
+------------+------------------------------------------------------------------+

**Call Campaign Flow**

::

    Campaign dials target
           |
           v
    +-------------------+
    | Target answers    |
    +--------+----------+
             |
             v
    +-------------------+
    | Execute flow      |
    | (IVR, AI, etc.)   |
    +--------+----------+
             |
             v
    +-------------------+
    | Connect to agent  |
    | from queue        |
    +-------------------+


Common Scenarios
----------------

**Scenario 1: Telemarketing Campaign**

Outbound sales calls with agent connection.

::

    Setup:
    1. Outdial: 10,000 customer phone numbers
    2. Outplan: Business hours, max 3 retries
    3. Queue: Sales team (20 agents)
    4. Flow: Play intro -> Connect to agent

    Execution:
    +--------------------------------------------+
    | Campaign dials customer                    |
    | -> Customer answers                        |
    | -> Flow plays: "Special offer from..."    |
    | -> Press 1 to speak with representative   |
    | -> Connected to sales agent               |
    +--------------------------------------------+

**Scenario 2: Appointment Reminders**

Automated reminder calls with AI.

::

    Setup:
    1. Outdial: Tomorrow's appointments
    2. Outplan: Call 24 hours before, retry once
    3. Flow: AI confirms appointment details

    Execution:
    +--------------------------------------------+
    | Campaign dials patient                     |
    | -> AI: "This is a reminder for your       |
    |    appointment tomorrow at 2pm."          |
    | -> AI: "Press 1 to confirm, 2 to cancel"  |
    | -> Response saved via set_variables       |
    +--------------------------------------------+

**Scenario 3: Emergency Notifications**

High-priority mass notifications.

::

    Setup:
    1. Outdial: All affected customers
    2. Outplan: Immediate, aggressive retry (5x)
    3. Flow: Play recorded message

    Execution:
    +--------------------------------------------+
    | Campaign dials all targets simultaneously  |
    | -> Play: "Important service notification" |
    | -> Retry unanswered every 15 minutes      |
    | -> Mark as reached when answered          |
    +--------------------------------------------+


Best Practices
--------------

**1. Target List Management**

- Validate phone numbers before adding to outdial
- Remove duplicates and invalid entries
- Segment targets for better campaign control
- Update lists based on previous campaign results

**2. Dialing Strategy**

- Match outplan to campaign type (sales vs. notification)
- Respect time zones and business hours
- Set appropriate retry counts (3-5 for sales, 1-2 for reminders)
- Use progressive intervals between retries

**3. Agent Coordination**

- Ensure enough agents before starting campaign
- Monitor queue wait times during campaign
- Plan for peak call volume periods
- Have backup agents available

**4. Compliance**

- Honor do-not-call lists
- Follow local regulations (TCPA, GDPR)
- Provide opt-out options
- Keep accurate calling records


Troubleshooting
---------------

**Campaign Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Campaign not dialing      | Check status is "running"; verify outdial has  |
|                           | targets; check outplan timing settings         |
+---------------------------+------------------------------------------------+
| Low answer rate           | Review calling times; check source number      |
|                           | reputation; verify target number quality       |
+---------------------------+------------------------------------------------+
| High abandon rate         | Add more agents to queue; reduce dial rate;    |
|                           | check flow execution time                      |
+---------------------------+------------------------------------------------+

**Agent Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Agents not receiving      | Check agent status is available; verify queue  |
| calls                     | assignment; check queue routing                |
+---------------------------+------------------------------------------------+
| Long wait times           | Add more agents; reduce concurrent dial rate;  |
|                           | optimize flow execution                        |
+---------------------------+------------------------------------------------+

**Target Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Many failed calls         | Validate phone numbers; check carrier routing; |
|                           | review dial timeout settings                   |
+---------------------------+------------------------------------------------+
| Retries not happening     | Check outplan max_try_count; verify retry      |
|                           | interval settings                              |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Outdial Overview <outdial_overview>` - Managing dial targets
- :ref:`Outplan Overview <outplan-overview>` - Dialing strategy configuration
- :ref:`Queue Overview <queue-overview>` - Agent queue management
- :ref:`Flow Overview <flow-overview>` - Call flow configuration

