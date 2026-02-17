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

+-------------------+------------------+----------------------------------------+
| Campaign Type     | Recommended      | Reasoning                              |
+===================+==================+========================================+
| Sales calls       | 25-30 seconds    | Allow time to reach phone              |
+-------------------+------------------+----------------------------------------+
| Reminders         | 20-25 seconds    | Quick notification, don't wait long   |
+-------------------+------------------+----------------------------------------+
| Emergency         | 30-45 seconds    | Maximum opportunity to answer          |
+-------------------+------------------+----------------------------------------+

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

+-------------------+------------------+----------------------------------------+
| Campaign Type     | Recommended      | Reasoning                              |
+===================+==================+========================================+
| Sales calls       | 2-4 hours        | Try different times of day             |
+-------------------+------------------+----------------------------------------+
| Reminders         | 30-60 minutes    | Moderate urgency                       |
+-------------------+------------------+----------------------------------------+
| Emergency         | 5-15 minutes     | High urgency, frequent retries         |
+-------------------+------------------+----------------------------------------+

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

+-------------------+------------------------------------------------------------------+
| Field             | Description                                                      |
+===================+==================================================================+
| dial_timeout      | Ring timeout in milliseconds (30000 = 30 seconds)                |
+-------------------+------------------------------------------------------------------+
| try_interval      | Wait between retries in milliseconds (7200000 = 2 hours)         |
+-------------------+------------------------------------------------------------------+
| max_try_count_0   | Max retries for result type 0 (machine/voicemail)                |
+-------------------+------------------------------------------------------------------+
| max_try_count_1   | Max retries for result type 1 (busy)                             |
+-------------------+------------------------------------------------------------------+
| max_try_count_2   | Max retries for result type 2 (no answer)                        |
+-------------------+------------------------------------------------------------------+
| max_try_count_3   | Max retries for result type 3 (failed/error)                     |
+-------------------+------------------------------------------------------------------+
| max_try_count_4   | Max retries for result type 4 (other)                            |
+-------------------+------------------------------------------------------------------+


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
    | max_try_count:   5 (all types)             |
    +--------------------------------------------+

    Timeline:
    |--0m--|--10m--|--20m--|--30m--|--40m--|
       1      2       3       4       5
    (attempts)

**Standard Strategy (Sales)**

::

    +--------------------------------------------+
    | Sales Outplan                              |
    +--------------------------------------------+
    | dial_timeout:    30 seconds                |
    | try_interval:    2 hours                   |
    | max_try_count:   3 (no answer/busy)        |
    |                  1 (machine)               |
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
    | max_try_count:   2 (all types)             |
    +--------------------------------------------+

    Timeline:
    |--0h--|--1h--|
       1      2
    (minimal disruption)


Result-Based Retry Counts
-------------------------
Configure different retry counts based on the outcome of each attempt.

**Result Types**

+-------+---------------+----------------------------------------------------------+
| Type  | Result        | Typical Retry Strategy                                   |
+=======+===============+==========================================================+
| 0     | Machine/VM    | Low retries (1-2); person may not check voicemail        |
+-------+---------------+----------------------------------------------------------+
| 1     | Busy          | Medium retries (2-3); likely to be free later            |
+-------+---------------+----------------------------------------------------------+
| 2     | No answer     | High retries (3-5); try different times                  |
+-------+---------------+----------------------------------------------------------+
| 3     | Failed/Error  | Low retries (0-1); may be invalid number                 |
+-------+---------------+----------------------------------------------------------+
| 4     | Other         | Configurable based on use case                           |
+-------+---------------+----------------------------------------------------------+

**Example Configuration**

.. code::

    {
        "max_try_count_0": 1,  // Machine: try once more
        "max_try_count_1": 3,  // Busy: retry 3 times
        "max_try_count_2": 4,  // No answer: retry 4 times
        "max_try_count_3": 0,  // Failed: don't retry
        "max_try_count_4": 1   // Other: try once more
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
    | max_try_count_1: 3 (busy)                  |
    | max_try_count_2: 4 (no answer)             |
    +--------------------------------------------+

    Strategy:
    - Long intervals to try morning, midday, afternoon
    - More retries for no answer (might be in meetings)
    - Fewer retries for busy (clearly occupied)

**Scenario 2: Appointment Confirmation**

Confirming next-day appointments.

::

    Outplan: "Appointment Reminder"
    +--------------------------------------------+
    | dial_timeout:    25 seconds                |
    | try_interval:    2 hours                   |
    | max_try_count_1: 2 (busy)                  |
    | max_try_count_2: 2 (no answer)             |
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
    | max_try_count_0: 5 (all types)             |
    | max_try_count_1: 5                         |
    | max_try_count_2: 5                         |
    | max_try_count_3: 3                         |
    +--------------------------------------------+

    Strategy:
    - Long timeout (maximize answer chance)
    - Frequent retries (urgent delivery)
    - Retry even failures (might be temporary issue)


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
- Use result-based retry counts for efficiency
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

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Calls timing out too      | Increase dial_timeout; check network latency;  |
| quickly                   | verify carrier routing                         |
+---------------------------+------------------------------------------------+
| Retries happening too     | Increase try_interval; verify millisecond      |
| fast                      | values are correct                             |
+---------------------------+------------------------------------------------+
| Not enough retries        | Increase max_try_count values; check result    |
|                           | type configuration                             |
+---------------------------+------------------------------------------------+

**Configuration Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Outplan not applied       | Verify outplan_id in campaign; check outplan   |
|                           | exists and is active                           |
+---------------------------+------------------------------------------------+
| Wrong retry behavior      | Review result-type specific max_try_count      |
|                           | values; check which result type is being set   |
+---------------------------+------------------------------------------------+
| Values seem wrong         | Confirm millisecond format; 30 seconds =       |
|                           | 30000ms, 1 hour = 3600000ms                    |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Campaign Overview <campaign-overview>` - Using outplans in campaigns
- :ref:`Outdial Overview <outdial_overview>` - Target management
- :ref:`Queue Overview <queue-overview>` - Agent routing
- :ref:`Call Overview <call-overview>` - Call handling details

