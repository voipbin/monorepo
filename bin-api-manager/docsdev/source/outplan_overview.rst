.. _outplan-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free. Creating and managing outplans is free. Costs are incurred when the associated campaign dials targets.
   * **Async:** No. ``POST /outplans`` returns immediately with the created outplan.

VoIPBIN's Outplan API provides fine-grained control over dialing strategies for outbound campaigns. An outplan defines how the system should handle dial attempts, including timeouts, retry intervals, and maximum attempt counts. By configuring outplans, you can optimize contact rates while respecting recipient preferences and regulatory requirements.

With the Outplan API you can:

- Configure dial timeout durations
- Set retry intervals between attempts
- Define maximum retry counts
- Create different strategies for various campaign types
- Optimize contact rates and efficiency


How Outplans Work
-----------------
Outplans define the "when and how" of dialing in campaign operations.

**Outplan Architecture**

::

    +-----------------------------------------------------------------------+
    |                         Outplan System                                |
    +-----------------------------------------------------------------------+

    +-------------------+
    |      Outplan      |
    | (dialing strategy)|
    +--------+----------+
             |
             | defines
             v
    +--------+----------+--------+----------+--------+----------+
    |                   |                   |                   |
    v                   v                   v                   v
    +----------+   +----------+   +----------+   +----------+
    |  Dial    |   |  Retry   |   |  Max     |   |  Timing  |
    | Timeout  |   | Interval |   | Attempts |   | Windows  |
    +----------+   +----------+   +----------+   +----------+
         |              |              |              |
         v              v              v              v
    +---------+    +---------+    +---------+    +---------+
    | 30 sec  |    | 1 hour  |    | 3 tries |    | 9am-6pm |
    +---------+    +---------+    +---------+    +---------+

**Key Components**

- **Dial Timeout**: How long to wait for an answer before giving up
- **Try Interval**: Time between retry attempts
- **Max Try Count**: Maximum number of dial attempts per target
- **Timing Windows**: When dialing is permitted (optional)

.. note:: **AI Implementation Hint**

   The ``dial_timeout`` and ``try_interval`` fields are in **milliseconds**. A 30-second dial timeout is ``30000``, not ``30``. A 2-hour retry interval is ``7200000``. The ``max_try_count_0`` through ``max_try_count_4`` correspond to destination indices on the outdialtarget (``destination_0`` through ``destination_4``), not result types.


Outplan Configuration
---------------------
Configure outplan parameters based on your campaign needs.

**Dial Timeout**

The dial timeout specifies how long the system waits for the target to answer before marking the attempt as failed.

::

    Dial initiated
         |
         v
    +-------------------+
    | Ring...           |
    | (counting time)   |
    +--------+----------+
             |
             +-----> Answered within timeout? --> Success
             |
             +-----> Timeout reached? --> Mark as no_answer

**Recommended Timeouts**

.. list-table::
   :header-rows: 1

   * - Campaign Type
     - Recommended
     - Reasoning
   * - Sales calls
     - 25-30 seconds
     - Allow time to reach phone
   * - Reminders
     - 20-25 seconds
     - Quick notification, don't wait long
   * - Emergency
     - 30-45 seconds
     - Maximum opportunity to answer


**Try Interval**

The try interval defines the wait time between consecutive dial attempts to the same target.

::

    Attempt 1: No answer
         |
         v
    +-------------------+
    | Wait: try_interval|
    | (e.g., 1 hour)    |
    +--------+----------+
             |
             v
    Attempt 2: Try again
         |
         v
    +-------------------+
    | Wait: try_interval|
    +--------+----------+
             |
             v
    Attempt 3: Final attempt

**Recommended Intervals**

.. list-table::
   :header-rows: 1

   * - Campaign Type
     - Recommended
     - Reasoning
   * - Sales calls
     - 2-4 hours
     - Try different times of day
   * - Reminders
     - 30-60 minutes
     - Moderate urgency
   * - Emergency
     - 5-15 minutes
     - High urgency, frequent retries


**Max Try Count**

The maximum try count limits how many times the system will attempt to reach a target.

::

    +-------------------+
    | max_try_count: 3  |
    +-------------------+
             |
             v
    Attempt 1 --> No answer
    Attempt 2 --> Busy
    Attempt 3 --> No answer
    --> Mark as FAILED (max reached)


Creating an Outplan
-------------------
Create outplans via the API.

**Create Outplan Example**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/outplans?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "Sales Standard",
            "detail": "Standard sales campaign strategy",
            "dial_timeout": 30000,
            "try_interval": 7200000,
            "max_try_count_0": 3,
            "max_try_count_1": 2,
            "max_try_count_2": 1,
            "max_try_count_3": 0,
            "max_try_count_4": 0
        }'

**Field Descriptions**

.. list-table::
   :header-rows: 1

   * - Field
     - Description
   * - dial_timeout
     - Ring timeout in milliseconds (30000 = 30 seconds)
   * - try_interval
     - Wait between retries in milliseconds (7200000 = 2 hours)
   * - max_try_count_0
     - Max retries for destination_0 on the outdialtarget
   * - max_try_count_1
     - Max retries for destination_1 on the outdialtarget
   * - max_try_count_2
     - Max retries for destination_2 on the outdialtarget
   * - max_try_count_3
     - Max retries for destination_3 on the outdialtarget
   * - max_try_count_4
     - Max retries for destination_4 on the outdialtarget



Outplan Strategies
------------------
Different campaign types require different dialing strategies.

**Aggressive Strategy (Emergency)**

::

    +--------------------------------------------+
    | Emergency Outplan                          |
    +--------------------------------------------+
    | dial_timeout:    45 seconds                |
    | try_interval:    10 minutes                |
    | max_try_count_0: 5 (destination_0)         |
    | max_try_count_1: 5 (destination_1)         |
    +--------------------------------------------+

    Timeline:
    |--0m--|--10m--|--20m--|--30m--|--40m--|
       1      2       3       4       5
    (attempts per destination)

**Standard Strategy (Sales)**

::

    +--------------------------------------------+
    | Sales Outplan                              |
    +--------------------------------------------+
    | dial_timeout:    30 seconds                |
    | try_interval:    2 hours                   |
    | max_try_count_0: 3 (destination_0)         |
    | max_try_count_1: 2 (destination_1)         |
    +--------------------------------------------+

    Timeline:
    |--0h--|--2h--|--4h--|
       1      2      3
    (attempts spread across day)

**Conservative Strategy (Reminders)**

::

    +--------------------------------------------+
    | Reminder Outplan                           |
    +--------------------------------------------+
    | dial_timeout:    25 seconds                |
    | try_interval:    1 hour                    |
    | max_try_count_0: 2 (destination_0)         |
    | max_try_count_1: 0 (skip destination_1)    |
    +--------------------------------------------+

    Timeline:
    |--0h--|--1h--|
       1      2
    (minimal disruption)


Destination-Based Retry Counts
-------------------------------
Each outdialtarget supports up to 5 destination addresses (``destination_0`` through ``destination_4``). The ``max_try_count_0`` through ``max_try_count_4`` fields on the outplan control how many times each destination index is attempted before moving to the next destination.

**How Destination Fallback Works**

::

    destination_0 (+15551234567) -- max_try_count_0 attempts exhausted
         |
         v
    destination_1 (+15559876543) -- max_try_count_1 attempts exhausted
         |
         v
    destination_2 (sip:user@pbx)  -- max_try_count_2 attempts exhausted
         |
         v
    Target marked as done (all destinations exhausted)

**Example Configuration**

.. code::

    {
        "max_try_count_0": 3,  // Try destination_0 up to 3 times
        "max_try_count_1": 2,  // Try destination_1 up to 2 times
        "max_try_count_2": 1,  // Try destination_2 once
        "max_try_count_3": 0,  // Skip destination_3
        "max_try_count_4": 0   // Skip destination_4
    }


Common Scenarios
----------------

**Scenario 1: B2B Sales Campaign**

Reaching business contacts during work hours.

::

    Outplan: "B2B Sales"
    +--------------------------------------------+
    | dial_timeout:    30 seconds                |
    | try_interval:    4 hours                   |
    | max_try_count_0: 4 (primary number)        |
    | max_try_count_1: 3 (secondary number)      |
    +--------------------------------------------+

    Strategy:
    - Long intervals to try morning, midday, afternoon
    - More retries for primary number
    - Fewer retries for secondary number

**Scenario 2: Appointment Confirmation**

Confirming next-day appointments.

::

    Outplan: "Appointment Reminder"
    +--------------------------------------------+
    | dial_timeout:    25 seconds                |
    | try_interval:    2 hours                   |
    | max_try_count_0: 2 (primary number)        |
    | max_try_count_1: 1 (secondary number)      |
    +--------------------------------------------+

    Strategy:
    - Short timeout (quick notification)
    - Limited retries (not critical)
    - Spread across afternoon/evening

**Scenario 3: Critical Alert**

Emergency notifications requiring high delivery.

::

    Outplan: "Emergency Alert"
    +--------------------------------------------+
    | dial_timeout:    45 seconds                |
    | try_interval:    5 minutes                 |
    | max_try_count_0: 5 (primary number)        |
    | max_try_count_1: 5 (secondary number)      |
    | max_try_count_2: 5 (office phone)          |
    | max_try_count_3: 3 (alternate)             |
    +--------------------------------------------+

    Strategy:
    - Long timeout (maximize answer chance)
    - Frequent retries (urgent delivery)
    - Multiple destination fallback for high delivery


Best Practices
--------------

**1. Timeout Configuration**

- Set realistic timeouts (25-45 seconds)
- Account for call setup time
- Consider mobile vs. landline behavior
- Test across different carriers

**2. Retry Intervals**

- Space retries to try different times of day
- Avoid too-frequent retries (annoys recipients)
- Match urgency level of campaign
- Consider time zone differences

**3. Max Try Counts**

- Balance persistence with respect for recipients
- Use per-destination retry counts for efficiency
- Higher counts for critical campaigns
- Lower counts for promotional calls

**4. Compliance**

- Follow local regulations on retry limits
- Respect quiet hours (no early morning/late night)
- Document retry policies for auditing
- Honor do-not-call after failed attempts


Troubleshooting
---------------

**Timing Issues**

.. list-table::
   :header-rows: 1

   * - Symptom
     - Solution
   * - Calls timing out too quickly
     - Increase dial_timeout; check network latency; verify carrier routing
   * - Retries happening too fast
     - Increase try_interval; verify millisecond values are correct
   * - Not enough retries
     - Increase max_try_count values; check destination-specific configuration


**Configuration Issues**

.. list-table::
   :header-rows: 1

   * - Symptom
     - Solution
   * - Outplan not applied
     - Verify outplan_id in campaign; check outplan exists and is active
   * - Wrong retry behavior
     - Review per-destination max_try_count values; check which destination index is being dialed
   * - Values seem wrong
     - Confirm millisecond format; 30 seconds = 30000ms, 1 hour = 3600000ms



Related Documentation
---------------------

- :ref:`Campaign Overview <campaign-overview>` - Using outplans in campaigns
- :ref:`Outdial Overview <outdial-overview>` - Target management
- :ref:`Queue Overview <queue-overview>` - Agent routing
- :ref:`Call Overview <call-overview>` - Call handling details

