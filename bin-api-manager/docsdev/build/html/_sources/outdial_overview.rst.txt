.. _outdial_overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low-Medium
   * **Cost:** Free. Creating outdials and targets is free. Costs are incurred when a campaign dials the targets.
   * **Async:** No. ``POST /outdials`` and ``POST /outdials/{id}/targets`` return immediately with the created resource.

VoIPBIN's Outdial API provides a scalable solution for managing outbound call destinations. An outdial is a collection of targets (phone numbers, SIP URIs, email addresses) that campaigns or flows dial sequentially. With built-in retry tracking and status management, the Outdial API handles large-scale outbound operations efficiently.

With the Outdial API you can:

- Create and manage dial target lists
- Track dial attempts and outcomes
- Configure per-target retry counts
- Monitor target status in real-time
- Import targets in bulk


How Outdials Work
-----------------
Outdials serve as the "who to contact" component of campaigns and outbound flows.

**Outdial Architecture**

::

    +-----------------------------------------------------------------------+
    |                         Outdial System                                |
    +-----------------------------------------------------------------------+

    +-------------------+
    |      Outdial      |
    |  (target list)    |
    +--------+----------+
             |
             | contains
             v
    +--------+----------+--------+----------+--------+----------+
    |                   |                   |                   |
    v                   v                   v                   v
    +----------+   +----------+   +----------+   +----------+
    | Target 1 |   | Target 2 |   | Target 3 |   | Target N |
    | +1555... |   | +1666... |   | +1777... |   |   ...    |
    +----+-----+   +----+-----+   +----+-----+   +----+-----+
         |              |              |              |
         v              v              v              v
    +---------+    +---------+    +---------+    +---------+
    | Status  |    | Status  |    | Status  |    | Status  |
    | Retries |    | Retries |    | Retries |    | Retries |
    +---------+    +---------+    +---------+    +---------+

**Key Components**

- **Outdial**: A container that groups multiple dial targets together
- **Outdial Target**: An individual destination with its address and retry count
- **Address**: The destination (phone number, SIP URI, email)
- **Try Count**: Number of dial attempts remaining for this target

.. note:: **AI Implementation Hint**

   Outdialtargets support up to 5 destination addresses (``destination_0`` through ``destination_4``). Each destination has its own independent try count. The campaign dials ``destination_0`` first, and if all retries are exhausted, moves to ``destination_1``, and so on. Do not modify targets while the parent campaign is in ``running`` status.


Target Lifecycle
----------------
Each outdial target progresses through states as dial attempts occur.

**Target States**

::

    POST /outdials/{id}/targets
           |
           v
    +------------+
    |   idle     |<-----------------+
    +-----+------+                  |
          |                         |
          | dial attempt            | retry scheduled
          v                         |
    +------------+                  |
    |  dialing   |------------------+
    +-----+------+
          |
          +---------+---------+
          |         |         |
          v         v         v
    +--------+ +--------+ +--------+
    |answered| |  busy  | |no answer|
    +--------+ +--------+ +--------+
          |         |         |
          v         v         v
    +------------+ +-------------------+
    |  finished  | | retry or failed   |
    +------------+ +-------------------+

**State Descriptions**

+-------------+------------------------------------------------------------------+
| State       | What's happening                                                 |
+=============+==================================================================+
| idle        | Target created, waiting to be dialed                            |
+-------------+------------------------------------------------------------------+
| dialing     | Call in progress to this target                                 |
+-------------+------------------------------------------------------------------+
| answered    | Target answered the call                                        |
+-------------+------------------------------------------------------------------+
| busy        | Target was busy, may retry                                      |
+-------------+------------------------------------------------------------------+
| no_answer   | No answer within timeout, may retry                             |
+-------------+------------------------------------------------------------------+
| finished    | Target successfully contacted, no more attempts needed          |
+-------------+------------------------------------------------------------------+
| failed      | All retry attempts exhausted                                     |
+-------------+------------------------------------------------------------------+


Managing Outdials
-----------------
Create outdials and add targets via the API.

**Create an Outdial**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/outdials?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "January Leads",
            "detail": "Sales leads for Q1 campaign"
        }'

**Add Targets to Outdial**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/outdials/<outdial-id>/targets?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "targets": [
                {
                    "destination": {
                        "type": "tel",
                        "target": "+15551234567"
                    },
                    "try_count_max": 3
                },
                {
                    "destination": {
                        "type": "tel",
                        "target": "+15559876543"
                    },
                    "try_count_max": 3
                }
            ]
        }'

**Get Outdial Targets**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/outdials/<outdial-id>/targets?token=<token>'


Retry Mechanism
---------------
Outdial targets track retry attempts automatically.

**Retry Flow**

::

    Target dialed
         |
         v
    +-------------------+     Answered     +-------------------+
    | Dial attempt      |----------------->| Mark as finished  |
    |                   |                  |                   |
    +--------+----------+                  +-------------------+
             |
             | Not answered (busy, no answer, failed)
             v
    +-------------------+
    | try_count_current |
    | += 1              |
    +--------+----------+
             |
             v
    +-------------------+     Yes          +-------------------+
    | try_count_current |----------------->| Schedule retry    |
    | < try_count_max?  |                  | (per outplan)     |
    +--------+----------+                  +-------------------+
             |
             | No (max reached)
             v
    +-------------------+
    | Mark as failed    |
    +-------------------+

**Retry Configuration**

+---------------------+------------------------------------------------------------------+
| Field               | Description                                                      |
+=====================+==================================================================+
| try_count_max       | Maximum number of dial attempts allowed                          |
+---------------------+------------------------------------------------------------------+
| try_count_current   | Current number of attempts made (read-only)                      |
+---------------------+------------------------------------------------------------------+

The retry timing and intervals are controlled by the :ref:`Outplan <outplan-overview>`.


Target Types
------------
Outdial targets support various destination types.

+------------+------------------------------------------------------------------+
| Type       | Description                                                      |
+============+==================================================================+
| tel        | Phone number in E.164 format (+15551234567)                      |
+------------+------------------------------------------------------------------+
| sip        | SIP URI (sip:user@domain.com)                                    |
+------------+------------------------------------------------------------------+
| email      | Email address (for email campaigns)                              |
+------------+------------------------------------------------------------------+

**Tel Target Example**

.. code::

    {
        "destination": {
            "type": "tel",
            "target": "+15551234567"
        },
        "try_count_max": 3
    }

**SIP Target Example**

.. code::

    {
        "destination": {
            "type": "sip",
            "target": "sip:user@pbx.company.com"
        },
        "try_count_max": 2
    }


Common Scenarios
----------------

**Scenario 1: Sales Lead List**

Import and manage a sales prospect list.

::

    1. Create outdial: "Q1 Sales Leads"

    2. Bulk import targets:
       +--------------------------------------------+
       | +15551234567 | John Smith   | 3 retries   |
       | +15552345678 | Jane Doe     | 3 retries   |
       | +15553456789 | Bob Johnson  | 3 retries   |
       | ... (1000 more targets)                   |
       +--------------------------------------------+

    3. Attach to campaign with sales outplan

    4. Monitor progress:
       - 800 answered (finished)
       - 150 failed (max retries)
       - 50 pending (scheduled retry)

**Scenario 2: Appointment Confirmations**

Daily appointment reminder targets.

::

    Daily Process:
    +--------------------------------------------+
    | 1. Create new outdial for tomorrow's date  |
    |    "Appointments 2024-01-16"               |
    |                                            |
    | 2. Add patients with appointments:         |
    |    - Each target gets 2 retry attempts     |
    |    - Include appointment time in metadata  |
    |                                            |
    | 3. Attach to reminder campaign             |
    |                                            |
    | 4. Campaign runs, marks confirmed targets  |
    +--------------------------------------------+

**Scenario 3: Emergency Contact List**

Critical notifications requiring high delivery.

::

    Setup:
    +--------------------------------------------+
    | Outdial: "Emergency Contacts"              |
    | - All employees/customers                  |
    | - High retry count (5)                     |
    | - Multiple contact methods per person      |
    +--------------------------------------------+

    Target Priority:
    1. Primary mobile: +15551234567 (5 retries)
    2. Secondary mobile: +15559876543 (5 retries)
    3. Office phone: +15551111111 (3 retries)


Best Practices
--------------

**1. Target Quality**

- Validate phone numbers before import
- Use E.164 format for all phone numbers
- Remove duplicates within the same outdial
- Keep target lists focused and segmented

**2. Retry Configuration**

- Set try_count_max based on campaign urgency
- Sales: 3-5 retries over multiple days
- Reminders: 1-2 retries within hours
- Emergencies: 5+ retries with short intervals

**3. List Management**

- Archive completed outdials for reporting
- Don't modify active outdials during campaign
- Create new outdials for recurring campaigns
- Use descriptive names with dates

**4. Performance**

- Break large lists into multiple outdials
- Monitor target status during campaigns
- Remove invalid targets proactively
- Track delivery rates for optimization


Troubleshooting
---------------

**Target Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Target not being dialed   | Check target status is "idle"; verify outdial  |
|                           | is attached to running campaign                |
+---------------------------+------------------------------------------------+
| All targets failing       | Validate phone number format; check carrier    |
|                           | routing; verify source number                  |
+---------------------------+------------------------------------------------+
| Retries not happening     | Check try_count_max > try_count_current;       |
|                           | verify outplan retry settings                  |
+---------------------------+------------------------------------------------+

**Outdial Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Can't add targets         | Check outdial exists; verify API permissions   |
+---------------------------+------------------------------------------------+
| Targets missing           | Check pagination in GET request; verify        |
|                           | targets were created successfully              |
+---------------------------+------------------------------------------------+
| Wrong target count        | Duplicates may have been filtered; check       |
|                           | for import errors                              |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Campaign Overview <campaign-overview>` - Using outdials in campaigns
- :ref:`Outplan Overview <outplan-overview>` - Retry timing and strategy
- :ref:`Queue Overview <queue-overview>` - Agent routing for answered calls
- :ref:`Call Overview <call-overview>` - Call handling details

