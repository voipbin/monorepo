.. _queue-overview:

Overview
========
Call queueing allows calls to be placed on hold without handling the actual inquiries or transferring callers to the desired party. While in the call queue, the caller is played pre-recorded music or messages. Call queues are often used in call centers when there are not enough staff to handle a large number of calls. Call center operators generally receive information about the number of callers in the call queue and the duration of the waiting time. This allows them to respond flexibly to peak demand by deploying extra call center staff.

With the VoIPBIN's queueing feature, businesses and call centers can effectively manage inbound calls, provide a smooth waiting experience for callers, and ensure that calls are efficiently distributed to available agents, improving overall customer service and call center performance.


The purpose of call queueing
----------------------------
Call queueing is intended to prevent callers from being turned away in the case of insufficient staff capacity. The purpose of the pre-recorded music or messages is to shorten the subjective waiting time. At the same time, call queues can be used for advertising products or services. As soon as the call can be dealt with, the caller is automatically transferred from the call queue to the member of staff responsible. If customer or contract data has to be requested in several stages, multiple downstream call queues can be used.


How Queue Routing Works
-----------------------
When a call joins a queue, a coordinated process begins to match the caller with an available agent.

**The Queue Journey**

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                       Call's Journey Through Queue                       │
    └─────────────────────────────────────────────────────────────────────────┘

    1. Call Joins Queue          2. Wait Flow Plays         3. Agent Search
    ┌─────────────────┐         ┌─────────────────┐        ┌─────────────────┐
    │                 │         │  ♪ Hold music   │        │  Check for      │
    │  queue_join     │────────▶│  "Please wait"  │───────▶│  available      │
    │  action         │         │  announcements  │        │  agents         │
    └─────────────────┘         └─────────────────┘        └────────┬────────┘
                                        ▲                          │
                                        │                    ┌─────┴─────┐
                                        │                    │           │
                                        │               No agents    Agent found!
                                        │                    │           │
                                        └────────────────────┘           │
                                          Retry every 1 second           │
                                                                         ▼
                                                              ┌─────────────────┐
                                                              │  4. Connect     │
                                                              │  agent to call  │
                                                              └────────┬────────┘
                                                                       │
                                                                       ▼
                                                              ┌─────────────────┐
                                                              │  5. Caller and  │
                                                              │  agent talking  │
                                                              └─────────────────┘

**What Happens Step by Step**

1. **Call Joins Queue**: When your flow executes a ``queue_join`` action, a queuecall is created
2. **Conference Bridge Created**: A private conference room is set up for this call
3. **Wait Flow Starts**: The caller hears music and announcements from your configured wait flow
4. **Agent Search Begins**: The system immediately starts looking for available agents
5. **Continuous Checking**: Every second, the system checks if any matching agents are now available
6. **Agent Found**: When an agent matches, they're invited to join the conference
7. **Service Begins**: Both parties are connected and can talk


Queuecall Lifecycle
-------------------
Every call in a queue moves through a series of states:

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                      Queuecall State Machine                             │
    └─────────────────────────────────────────────────────────────────────────┘

                                ┌────────────┐
                                │ initiating │
                                │  (brief)   │
                                └─────┬──────┘
                                      │
                                      ▼
                           ┌──────────────────┐
           wait_timeout───▶│     waiting      │
           exceeded        │ (in wait flow)   │
               │           └────────┬─────────┘
               │                    │ agent found
               │                    ▼
               │           ┌──────────────────┐
               │           │    connecting    │
               │           │  (dialing agent) │
               │           └────────┬─────────┘
               │                    │ agent joins
               │                    ▼
               │           ┌──────────────────┐         service_timeout
               │           │     service      │◀────────── exceeded
               │           │ (agent on call)  │                │
               │           └────────┬─────────┘                │
               │                    │                          │
               │           ┌────────┴────────┐                 │
               │           ▼                 ▼                 ▼
               │    ┌────────────┐    ┌─────────────┐
               └───▶│  abandoned │    │    done     │
                    │  (failed)  │    │ (success)   │
                    └────────────┘    └─────────────┘

**State Descriptions**

+-------------+--------------------------------------------------------------+
| State       | What's happening                                             |
+=============+==============================================================+
| initiating  | Call just entered the queue. Conference bridge being set up. |
+-------------+--------------------------------------------------------------+
| waiting     | Caller is hearing wait flow. System is searching for agents. |
+-------------+--------------------------------------------------------------+
| connecting  | Agent found and being called. Caller still in wait flow.    |
+-------------+--------------------------------------------------------------+
| service     | Agent and caller are connected. Conversation in progress.   |
+-------------+--------------------------------------------------------------+
| done        | Call completed successfully. Agent finished helping caller.  |
+-------------+--------------------------------------------------------------+
| abandoned   | Call ended before completion - caller hung up, timeout, etc. |
+-------------+--------------------------------------------------------------+


Agent Searching
---------------
While the call is in the queue, the queue continuously searches for available agents to handle the call. Each queue has tags associated with it that determine which agents can receive calls from that queue.

**How Agent Matching Works**

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                      Agent Matching Process                              │
    └─────────────────────────────────────────────────────────────────────────┘

    Queue Configuration:                    Agent Pool:
    ┌────────────────────────┐             ┌────────────────────────────────┐
    │ Tag IDs:               │             │ Agent A                        │
    │  • "english"           │             │  Tags: english, billing, vip   │
    │  • "billing"           │             │  Status: available             │
    │                        │             │  ──────────────────────────    │
    └────────────────────────┘             │ Agent B                        │
              │                            │  Tags: english                 │
              │                            │  Status: available             │
              │                            │  ──────────────────────────    │
              ▼                            │ Agent C                        │
    ┌────────────────────────┐             │  Tags: english, billing        │
    │ Search for agents with │             │  Status: busy                  │
    │ ALL of these tags:     │             │  ──────────────────────────    │
    │  • english ✓           │             │ Agent D                        │
    │  • billing ✓           │             │  Tags: spanish, billing        │
    └────────────────────────┘             │  Status: available             │
              │                            └────────────────────────────────┘
              │
              ▼
    ┌────────────────────────────────────────────────────────────────────────┐
    │ Results:                                                                │
    │  ✓ Agent A - Has english + billing + vip, is available                 │
    │  ✗ Agent B - Missing "billing" tag                                     │
    │  ✗ Agent C - Has tags but is busy                                      │
    │  ✗ Agent D - Missing "english" tag                                     │
    │                                                                         │
    │  → Agent A is selected!                                                 │
    └────────────────────────────────────────────────────────────────────────┘

.. image:: _static/images/queue_overview_agent.png

**Tag Requirements**

Tags work as a skill-based filter:

- Agents must have **ALL** tags the queue requires (AND logic)
- Having extra tags is fine (Agent A has "vip" but queue doesn't require it)
- If an agent is missing even one required tag, they're excluded
- Tags define skills, languages, departments, or any grouping you need

**Agent Status**

The agent's status must be "available" to receive queue calls:

::

    Agent Statuses and Queue Eligibility:
    ┌────────────────┬─────────────────────────────────────────────┐
    │ Status         │ Can receive queue calls?                    │
    ├────────────────┼─────────────────────────────────────────────┤
    │ available      │ ✓ Yes - Agent is ready to take calls        │
    │ busy           │ ✗ No - Agent is handling another call       │
    │ wrap-up        │ ✗ No - Agent is finishing previous call     │
    │ away           │ ✗ No - Agent is temporarily away            │
    │ offline        │ ✗ No - Agent is not logged in               │
    └────────────────┴─────────────────────────────────────────────┘

**Selection Method**

When multiple agents match, the system uses random selection:

::

    Multiple agents available:
    ┌────────────┐   ┌────────────┐   ┌────────────┐
    │  Agent A   │   │  Agent C   │   │  Agent E   │
    │ (matches)  │   │ (matches)  │   │ (matches)  │
    └────────────┘   └────────────┘   └────────────┘
          │               │               │
          └───────────────┼───────────────┘
                          │
                          ▼
                    Random Selection
                          │
                          ▼
                   One agent picked


Flow Execution
---------------
A call placed in the queue will progress through the queue's waiting actions, continuing through pre-defined steps until an available agent is located. These waiting actions may involve playing pre-recorded music, messages, or custom actions, enhancing the caller's experience while awaiting assistance in the queue.

**Wait Flow Structure**

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                          Wait Flow Execution                             │
    └─────────────────────────────────────────────────────────────────────────┘

    Caller enters queue
           │
           ▼
    ┌────────────────────────────────────────────────────────────────────────┐
    │                            Wait Flow Loop                               │
    │                                                                         │
    │    ┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐    │
    │    │ "Please  │────▶│  ♪ Hold  │────▶│ "Your    │────▶│  ♪ Hold  │    │
    │    │  hold"   │     │  music   │     │ position │     │  music   │    │
    │    │  (talk)  │     │  (play)  │     │  is 3"   │     │  (play)  │    │
    │    └──────────┘     └──────────┘     └──────────┘     └──────────┘    │
    │         ▲                                                    │         │
    │         │                                                    │         │
    │         └────────────────────────────────────────────────────┘         │
    │                        (loops until agent found)                        │
    └────────────────────────────────────────────────────────────────────────┘
           │
           │ Agent found!
           ▼
    ┌────────────────┐
    │ Wait flow ends │
    │ Service begins │
    └────────────────┘

.. image:: _static/images/queue_overview_flow.png

**Wait Flow Features**

- **Looping**: The wait flow automatically repeats until the call is answered or times out
- **Custom Actions**: You can use any flow action that makes sense while waiting
- **Announcements**: Play position updates, estimated wait time, or promotional messages
- **Music**: Play hold music to make the wait feel shorter


Timeout Handling
----------------
Queues have two distinct timeout mechanisms to prevent calls from being stuck indefinitely:

**Wait Timeout**

Controls how long a caller can wait before being removed from the queue:

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                          Wait Timeout                                    │
    └─────────────────────────────────────────────────────────────────────────┘

    Call joins queue                                     Wait timeout reached
         │                                                     │
         ▼                                                     ▼
    ─────●─────────────────────────────────────────────────────●─────────────▶
         │◀─────────────── wait_timeout (ms) ────────────────▶│    Time
         │                                                     │
         │ Caller hears wait flow                              │ Call is kicked
         │ System searches for agents                          │ Status: abandoned
         │                                                     │

- Set via queue's ``wait_timeout`` field (milliseconds)
- 0 means no timeout (wait forever)
- When exceeded, the call is removed with status "abandoned"

**Service Timeout**

Controls how long a conversation can last once connected:

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                         Service Timeout                                  │
    └─────────────────────────────────────────────────────────────────────────┘

    Agent connects                                    Service timeout reached
         │                                                     │
         ▼                                                     ▼
    ─────●─────────────────────────────────────────────────────●─────────────▶
         │◀───────────── service_timeout (ms) ───────────────▶│    Time
         │                                                     │
         │ Agent and caller talking                            │ Call is ended
         │ (status: service)                                   │ Status: done
         │                                                     │

- Set via queue's ``service_timeout`` field (milliseconds)
- 0 means no timeout (talk forever)
- When exceeded, the call is ended gracefully

**Timeout Scenarios**

::

    Scenario 1: Caller waits too long
    ┌──────────┐                                        ┌───────────┐
    │ waiting  │───── wait_timeout exceeded ──────────▶│ abandoned │
    └──────────┘                                        └───────────┘

    Scenario 2: Caller hangs up while waiting
    ┌──────────┐                                        ┌───────────┐
    │ waiting  │───── caller hangup ──────────────────▶│ abandoned │
    └──────────┘                                        └───────────┘

    Scenario 3: Successful call, normal end
    ┌──────────┐   ┌───────────┐   ┌─────────┐         ┌──────┐
    │ waiting  │──▶│ connecting│──▶│ service │──hangup▶│ done │
    └──────────┘   └───────────┘   └─────────┘         └──────┘


Queue Metrics and Tracking
--------------------------
Queues track detailed metrics for reporting and analysis:

**Per-Queue Statistics**

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                        Queue Statistics                                  │
    └─────────────────────────────────────────────────────────────────────────┘

    Queue: "Customer Support"
    ┌─────────────────────────────────┬───────────────────────────────────────┐
    │ total_incoming_count            │ 1,234 calls entered this queue        │
    ├─────────────────────────────────┼───────────────────────────────────────┤
    │ total_serviced_count            │ 1,100 calls reached an agent          │
    ├─────────────────────────────────┼───────────────────────────────────────┤
    │ total_abandoned_count           │ 134 calls abandoned (11% abandon rate)│
    └─────────────────────────────────┴───────────────────────────────────────┘

**Per-Queuecall Metrics**

Each call in the queue tracks:

::

    Queuecall: "abc-123-def"
    ┌─────────────────────────────────┬───────────────────────────────────────┐
    │ duration_waiting                │ 45,000 ms (45 seconds in wait flow)   │
    ├─────────────────────────────────┼───────────────────────────────────────┤
    │ duration_service                │ 180,000 ms (3 minutes with agent)     │
    ├─────────────────────────────────┼───────────────────────────────────────┤
    │ tm_create                       │ 2024-01-15 10:30:00 (entered queue)   │
    ├─────────────────────────────────┼───────────────────────────────────────┤
    │ tm_service                      │ 2024-01-15 10:30:45 (agent connected) │
    ├─────────────────────────────────┼───────────────────────────────────────┤
    │ tm_end                          │ 2024-01-15 10:33:45 (call ended)      │
    └─────────────────────────────────┴───────────────────────────────────────┘

**Calculating Service Levels**

You can calculate key performance indicators from these metrics:

::

    Service Level     = (total_serviced_count / total_incoming_count) × 100
                      = (1100 / 1234) × 100 = 89.1%

    Abandonment Rate  = (total_abandoned_count / total_incoming_count) × 100
                      = (134 / 1234) × 100 = 10.9%

    Average Wait Time = Sum of all duration_waiting / total_incoming_count


When No Agents Are Available
----------------------------
If no agents match or all matching agents are busy, the caller waits:

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                    No Agents Available Scenario                          │
    └─────────────────────────────────────────────────────────────────────────┘

    10:00:00 - Call joins queue, no agents available
         │
         ▼
    ┌─────────────────────────────────────────────────────────────────────────┐
    │ System checks for agents every 1 second:                                 │
    │                                                                          │
    │ 10:00:01 - Check agents... none available                               │
    │ 10:00:02 - Check agents... none available                               │
    │ 10:00:03 - Check agents... none available                               │
    │ 10:00:04 - Check agents... Agent A just became available!               │
    │                                                                          │
    │ Caller hears wait flow the entire time                                  │
    └─────────────────────────────────────────────────────────────────────────┘
         │
         ▼
    10:00:04 - Agent A is connected

**Retry Behavior**

- System retries every 1 second
- Caller continuously hears the wait flow
- As soon as any agent becomes available and matches, they're connected
- This continues until wait_timeout (if configured) or an agent answers


Conference Bridge Architecture
------------------------------
Each queuecall gets its own private conference bridge:

::

    ┌─────────────────────────────────────────────────────────────────────────┐
    │                      Conference Bridge Model                             │
    └─────────────────────────────────────────────────────────────────────────┘

    Queuecall for Caller A                    Queuecall for Caller B
    ┌─────────────────────────┐              ┌─────────────────────────┐
    │ Conference Bridge #1    │              │ Conference Bridge #2    │
    │ ┌─────────┐ ┌─────────┐ │              │ ┌─────────┐ ┌─────────┐ │
    │ │ Caller  │ │  Agent  │ │              │ │ Caller  │ │  Agent  │ │
    │ │   A     │ │   1     │ │              │ │   B     │ │   2     │ │
    │ └─────────┘ └─────────┘ │              │ └─────────┘ └─────────┘ │
    └─────────────────────────┘              └─────────────────────────┘

**Why This Design?**

- Each call is isolated in its own bridge
- Caller joins immediately, hears wait flow
- Agent joins when available, both can talk
- Clean separation - no cross-talk between calls


Common Queue Scenarios
----------------------

**Scenario 1: Quick Answer**

::

    Caller dials → Enters queue → Agent available → Connected in 2 seconds
                                                    │
                                                    ▼
                                            duration_waiting: 2000ms
                                            status: done

**Scenario 2: Wait Then Connect**

::

    Caller dials → Enters queue → No agents → Waits 45 seconds → Agent available
                                     │                                  │
                                     ▼                                  ▼
                              ♪ Hold music plays              Connected!
                              Position announcements          duration_waiting: 45000ms

**Scenario 3: Caller Abandons**

::

    Caller dials → Enters queue → Waits 2 minutes → Caller hangs up
                                     │                      │
                                     ▼                      ▼
                              ♪ Hold music plays      status: abandoned
                                                      duration_waiting: 120000ms

**Scenario 4: Wait Timeout**

::

    Caller dials → Enters queue → Waits 5 minutes → wait_timeout exceeded
                                     │                      │
                                     ▼                      ▼
                              ♪ Hold music plays      Kicked from queue
                                                      status: abandoned


Best Practices
--------------

**1. Configure Appropriate Timeouts**

::

    wait_timeout: 300000    (5 minutes - reasonable for customer service)
    service_timeout: 0      (no limit for service - let conversations finish)

**2. Design Engaging Wait Flows**

- Mix music with periodic announcements
- Provide position in queue updates
- Offer callback options for long waits
- Keep announcements concise

**3. Tag Agents Thoughtfully**

::

    Good tag structure:
    ┌─────────────────────────────────────────┐
    │ Language tags: english, spanish, french │
    │ Skill tags: billing, tech_support, sales│
    │ Tier tags: tier1, tier2, supervisor     │
    └─────────────────────────────────────────┘

**4. Monitor Key Metrics**

- Track abandonment rates - high rates indicate understaffing or long waits
- Monitor average wait times - aim for your service level target
- Review service durations - identify training opportunities
