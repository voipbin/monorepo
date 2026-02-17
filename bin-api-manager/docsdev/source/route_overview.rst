.. _route-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free. Routes are configuration rules with no per-unit charges.
   * **Async:** No. ``POST /routes`` returns immediately with the created route.

VoIPBIN's Route API provides intelligent SIP routing with automatic failover capabilities. Routes determine which provider handles outbound calls and define fallback strategies when primary routes fail.

With the Route API you can:

- Define routing rules for outbound calls
- Configure provider failover sequences
- Set conditions for route selection
- Customize routing per customer
- Optimize call delivery success rates


How Routes Work
---------------
Routes determine which provider handles each outbound call.

**Route Architecture**

::

    +-----------------------------------------------------------------------+
    |                         Routing System                                |
    +-----------------------------------------------------------------------+

    Outbound Call Request
          |
          v
    +-------------------+
    |   Route Engine    |
    +--------+----------+
             |
             | Evaluate routes
             v
    +--------+----------+--------+----------+
    |                   |                   |
    v                   v                   v
    +----------+   +----------+   +----------+
    | Route 1  |   | Route 2  |   | Route 3  |
    | Priority |   | Priority |   | Priority |
    |    1     |   |    2     |   |    3     |
    +----+-----+   +----+-----+   +----+-----+
         |              |              |
         v              v              v
    +---------+    +---------+    +---------+
    |Provider |    |Provider |    |Provider |
    |    A    |    |    B    |    |    C    |
    +---------+    +---------+    +---------+

**Key Components**

- **Route**: A rule that maps calls to providers
- **Priority**: Order in which routes are attempted
- **Failover**: Automatic retry with different provider on failure
- **Conditions**: Criteria for route selection


.. _route-overview-route_failover:

Route Failover
--------------
Routes support automatic failover when providers fail.

**Failover Flow**

::

    Outbound Call
          |
          v
    +-------------------+
    | Try Route 1       |
    | (Provider A)      |
    +--------+----------+
             |
             +--------> Success? --> Call connected
             |
             v Failure (4xx, 5xx, 6xx)
    +-------------------+
    | Try Route 2       |
    | (Provider B)      |
    +--------+----------+
             |
             +--------> Success? --> Call connected
             |
             v Failure
    +-------------------+
    | Try Route 3       |
    | (Provider C)      |
    +--------+----------+
             |
             +--------> Success? --> Call connected
             |
             v All failed
    +-------------------+
    | Return error      |
    | to caller         |
    +-------------------+

**Failover Triggers**

+-------------------+------------------------------------------------------------------+
| SIP Response      | Description                                                      |
+===================+==================================================================+
| 404               | Number not found - try alternate provider                        |
+-------------------+------------------------------------------------------------------+
| 5xx               | Server error - provider issue, try backup                        |
+-------------------+------------------------------------------------------------------+
| 6xx               | Global failure - try different route                             |
+-------------------+------------------------------------------------------------------+


.. _route-overview-dynamic_routing_decisions:

Dynamic Routing Decisions
-------------------------
Routes can be selected based on various criteria.

**Routing Criteria**

::

    +-------------------+
    | Incoming Call     |
    | Destination: +1...|
    +--------+----------+
             |
             v
    +-------------------+
    | Check criteria:   |
    | o Destination     |
    | o Customer        |
    | o Time of day     |
    | o Cost            |
    +--------+----------+
             |
             v
    +-------------------+
    | Select best route |
    +-------------------+

**Routing Options**

+---------------------+----------------------------------------------------------------+
| Criteria            | Description                                                    |
+=====================+================================================================+
| Default Route       | Applied to all calls unless overridden                         |
+---------------------+----------------------------------------------------------------+
| Customer Route      | Specific routes for individual customers                       |
+---------------------+----------------------------------------------------------------+
| Destination Route   | Routes based on called number prefix                           |
+---------------------+----------------------------------------------------------------+


.. note:: **AI Implementation Hint**

   A ``customer_id`` of ``00000000-0000-0000-0000-000000000001`` is the system-wide default. Routes with this customer ID apply to all customers unless a customer-specific route exists. A ``target`` of ``"all"`` matches every destination. For country-specific routing, use the E.164 country code prefix (e.g., ``+82`` for South Korea, ``+1`` for US/Canada).

Route Configuration
-------------------
Configure routes with priorities and providers.

**Create a Route**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/routes?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "US Primary Route",
            "provider_id": "provider-uuid-123",
            "priority": 1,
            "detail": "Primary route for US outbound calls"
        }'

**Route Configuration Example**

::

    Route: "US Outbound"
    +--------------------------------------------+
    | name: "US Primary Route"                   |
    | provider_id: provider-uuid-telnyx          |
    | priority: 1                                |
    +--------------------------------------------+

    Route: "US Backup"
    +--------------------------------------------+
    | name: "US Backup Route"                    |
    | provider_id: provider-uuid-twilio          |
    | priority: 2                                |
    +--------------------------------------------+


Common Scenarios
----------------

**Scenario 1: Simple Failover**

Two providers with automatic failover.

::

    Route Configuration:
    +--------------------------------------------+
    | Route 1: Telnyx (priority 1)               |
    |   -> Primary for all calls                 |
    |                                            |
    | Route 2: Twilio (priority 2)               |
    |   -> Used if Telnyx fails                  |
    +--------------------------------------------+

    Call Flow:
    1. Call attempt via Telnyx
    2. If Telnyx returns 503 -> try Twilio
    3. If Twilio succeeds -> call connected

**Scenario 2: Cost-Based Routing**

Route based on destination for cost optimization.

::

    Route Configuration:
    +--------------------------------------------+
    | US Calls (+1):                             |
    |   Provider A (lowest US rates)             |
    |                                            |
    | EU Calls (+44, +49, +33):                  |
    |   Provider B (best EU coverage)            |
    |                                            |
    | Default:                                   |
    |   Provider C (global coverage)             |
    +--------------------------------------------+

**Scenario 3: Customer-Specific Routes**

Different routes for different customers.

::

    Customer A: Enterprise (priority routing)
    +--------------------------------------------+
    | Primary: Premium Provider (quality focus)  |
    | Backup: Standard Provider                  |
    +--------------------------------------------+

    Customer B: Startup (cost-focused)
    +--------------------------------------------+
    | Primary: Budget Provider (cost focus)      |
    | Backup: Standard Provider                  |
    +--------------------------------------------+


Best Practices
--------------

**1. Failover Configuration**

- Always configure at least 2 routes
- Order routes by reliability, then cost
- Test failover regularly

**2. Provider Selection**

- Choose providers with complementary coverage
- Consider quality vs. cost tradeoffs
- Monitor success rates per provider

**3. Route Organization**

- Use descriptive route names
- Document routing logic
- Review and optimize periodically

**4. Monitoring**

- Track route usage and success rates
- Set up alerts for high failure rates
- Analyze failover patterns


Troubleshooting
---------------

**Routing Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Calls not routing         | Check route exists; verify provider is         |
|                           | configured; check priority order               |
+---------------------------+------------------------------------------------+
| Wrong provider selected   | Review route priorities; check for customer-   |
|                           | specific overrides                             |
+---------------------------+------------------------------------------------+
| Failover not working      | Verify backup routes exist; check failover     |
|                           | conditions are met                             |
+---------------------------+------------------------------------------------+

**Provider Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| All routes failing        | Check provider status; verify credentials;     |
|                           | test network connectivity                      |
+---------------------------+------------------------------------------------+
| High failover rate        | Check primary provider health; consider        |
|                           | changing route priorities                      |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Provider Overview <provider-overview>` - Provider configuration
- :ref:`Call Overview <call-overview>` - Making outbound calls
- :ref:`Trunk Overview <trunk-overview>` - SIP trunking setup

