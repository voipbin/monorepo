.. _provider-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free. Providers are configuration records. Costs are incurred when calls are routed through the configured provider, not when creating the provider entry.
   * **Async:** No. ``POST /providers`` returns immediately with the created provider.

VoIPBIN's Provider API enables management of telecommunication service providers that handle external call routing. Providers are SIP trunking services that connect VoIPBIN to the PSTN (Public Switched Telephone Network) and other external networks.

With the Provider API you can:

- Configure SIP trunk providers
- Set technical parameters for call routing
- Manage provider credentials
- Define routing preferences
- Monitor provider status


How Providers Work
------------------
Providers connect VoIPBIN to external telephone networks.

**Provider Architecture**

::

    +-----------------------------------------------------------------------+
    |                         Provider System                               |
    +-----------------------------------------------------------------------+

    VoIPBIN                         Provider                    External
       |                               |                           |
       | Outbound call                 |                           |
       +------------------------------>|                           |
       |                               | Route to PSTN             |
       |                               +-------------------------->|
       |                               |                           |
       |                               |        Call connects      |
       |<------------------------------|<--------------------------+
       |                               |                           |

    +-------------------+        +-------------------+        +-----------+
    |    VoIPBIN        |        |     Provider      |        |   PSTN    |
    |    Platform       |<------>|   (SIP Trunk)     |<------>|  Network  |
    +-------------------+        +-------------------+        +-----------+
                                        |
                              +---------+---------+
                              |                   |
                              v                   v
                        +---------+         +---------+
                        | Telnyx  |         | Twilio  |
                        +---------+         +---------+

**Key Components**

- **Provider**: A SIP trunk service for external call routing
- **Hostname**: The SIP server address
- **Tech Prefix/Postfix**: Number formatting rules
- **Tech Headers**: Custom SIP headers for authentication


Provider Configuration
----------------------
Configure providers with technical parameters.

**Provider Properties**

+-------------------+------------------------------------------------------------------+
| Property          | Description                                                      |
+===================+==================================================================+
| id                | Unique identifier for the provider                               |
+-------------------+------------------------------------------------------------------+
| name              | Display name of the provider                                     |
+-------------------+------------------------------------------------------------------+
| type              | Protocol type (e.g., "sip")                                      |
+-------------------+------------------------------------------------------------------+
| hostname          | SIP server address (e.g., sip.telnyx.com)                        |
+-------------------+------------------------------------------------------------------+
| tech_prefix       | Prefix to add to dialed numbers                                  |
+-------------------+------------------------------------------------------------------+
| tech_postfix      | Suffix to add to dialed numbers                                  |
+-------------------+------------------------------------------------------------------+
| tech_headers      | Custom SIP headers for the provider                              |
+-------------------+------------------------------------------------------------------+

**Example Provider Configuration**

.. code::

    {
        "id": "provider-uuid-123",
        "name": "Telnyx Production",
        "type": "sip",
        "hostname": "sip.telnyx.com",
        "tech_prefix": "",
        "tech_postfix": "",
        "tech_headers": {
            "X-Custom-Header": "value"
        },
        "tm_create": "2024-01-01T00:00:00Z"
    }


.. note:: **AI Implementation Hint**

   Providers are typically managed by platform administrators. The ``tech_prefix`` and ``tech_postfix`` fields modify the dialed number before sending to the provider. Most providers do not require these fields (leave as empty strings). The ``tech_headers`` field is an object of custom SIP headers sent with every call to this provider -- use it for provider-specific authentication tokens or routing hints.

Provider Types
--------------
VoIPBIN supports various provider configurations.

**SIP Provider**

::

    +--------------------------------------------+
    | SIP Provider                               |
    +--------------------------------------------+
    | Protocol: SIP (Session Initiation Protocol)|
    | Used for: Voice calls                      |
    | Example: Telnyx, Twilio, Bandwidth         |
    +--------------------------------------------+

    Outbound Call Flow:
    VoIPBIN -> SIP INVITE -> Provider -> PSTN


Number Formatting
-----------------
Providers may require specific number formats.

**Tech Prefix/Postfix Usage**

::

    Original Number: +15551234567

    With Prefix "1":
    +--------------------------------------------+
    | Dialed: 1+15551234567                      |
    +--------------------------------------------+

    With Postfix "@provider.com":
    +--------------------------------------------+
    | Dialed: +15551234567@provider.com          |
    +--------------------------------------------+

    With Both:
    +--------------------------------------------+
    | Dialed: 1+15551234567@provider.com         |
    +--------------------------------------------+


Common Scenarios
----------------

**Scenario 1: Primary Provider Setup**

Configure main outbound provider.

::

    Provider: "Main Carrier"
    +--------------------------------------------+
    | name: "Telnyx Production"                  |
    | hostname: "sip.telnyx.com"                 |
    | type: "sip"                                |
    |                                            |
    | Used for: All outbound calls               |
    +--------------------------------------------+

**Scenario 2: Multi-Provider Configuration**

Configure multiple providers for failover.

::

    Primary Provider: "Telnyx"
    +--------------------------------------------+
    | hostname: "sip.telnyx.com"                 |
    | Priority: 1 (first choice)                 |
    +--------------------------------------------+

    Secondary Provider: "Twilio"
    +--------------------------------------------+
    | hostname: "sip.twilio.com"                 |
    | Priority: 2 (failover)                     |
    +--------------------------------------------+

    Routing Logic:
    1. Try primary provider
    2. If failed -> try secondary

**Scenario 3: Regional Providers**

Use different providers for different regions.

::

    US Provider: "Telnyx US"
    +--------------------------------------------+
    | For numbers: +1xxx                         |
    | Optimal routing for US calls               |
    +--------------------------------------------+

    EU Provider: "Telnyx EU"
    +--------------------------------------------+
    | For numbers: +44, +49, +33...             |
    | Optimal routing for EU calls               |
    +--------------------------------------------+


Best Practices
--------------

**1. Provider Selection**

- Choose providers with good coverage for your regions
- Consider pricing, quality, and reliability
- Test providers before production use

**2. Configuration**

- Keep provider credentials secure
- Document provider-specific requirements
- Test number formatting thoroughly

**3. Failover**

- Configure multiple providers
- Set up proper failover routing
- Monitor provider health

**4. Monitoring**

- Track call success rates per provider
- Monitor latency and quality
- Set up alerts for provider issues


Troubleshooting
---------------

**Connection Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Calls not connecting      | Verify hostname is correct; check provider     |
|                           | credentials; test network connectivity         |
+---------------------------+------------------------------------------------+
| Authentication failures   | Check tech_headers for auth info; verify       |
|                           | credentials with provider                      |
+---------------------------+------------------------------------------------+

**Routing Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Wrong number format       | Check tech_prefix and tech_postfix; verify     |
|                           | provider requirements                          |
+---------------------------+------------------------------------------------+
| Provider rejecting calls  | Check account status with provider; verify     |
|                           | number is allowed                              |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Route Overview <route-overview>` - Call routing configuration
- :ref:`Trunk Overview <trunk-overview>` - SIP trunking
- :ref:`Call Overview <call-overview>` - Making calls

