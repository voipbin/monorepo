.. _provider-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free. Providers are configuration records. Costs are incurred when calls are routed through the configured provider, not when creating the provider entry.
   * **Async:** No. ``POST https://api.voipbin.net/v1.0/providers`` returns immediately with the created provider.

VoIPBIN's Provider API enables management of telecommunication service providers that handle external call routing. Providers are SIP trunking services that connect VoIPBIN to the PSTN (Public Switched Telephone Network) and other external networks.

With the Provider API you can:

- Set up a carrier-backed SIP trunk in one step via ``POST /providers/setup``
- Configure SIP trunk providers manually
- Set technical parameters for call routing
- Manage provider credentials
- Define routing preferences
- Monitor provider status


Easy Provider Setup
-------------------

``POST https://api.voipbin.net/v1.0/providers/setup`` lets a ProjectSuperAdmin submit a carrier API key and have VoIPBIN automatically validate the key, create the carrier-side SIP credential connection, and create the VoIPBin provider record — all in a single request.

**Supported carriers:** ``telnyx``

**How it works:**

::

    +--------------------------------------------+
    |  POST /providers/setup                     |
    |                                            |
    |  1. Validate carrier API key               |
    |  2. Create carrier SIP credential conn.    |
    |  3. Create VoIPBIN provider record         |
    |  Returns: provider (id, hostname, …)       |
    +--------------------------------------------+

.. note:: **AI Implementation Hint**

   Use ``POST /providers/setup`` instead of ``POST /providers`` when you have a carrier API key and want the platform to handle the SIP trunk creation automatically. If the API key is invalid or lacks the required permissions, the endpoint returns ``422 Unprocessable Entity``. On success, the returned provider ``id`` can immediately be referenced in ``POST /routes`` to enable outbound call routing.


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
| codecs            | Comma-separated codec preference list for PSTN outbound calls    |
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


Codec Configuration
-------------------

The ``codecs`` field controls which audio codecs VoIPBIN offers during SDP negotiation for PSTN
outbound calls routed through this provider.

**How it works:**

When an outbound call is placed through a SIP provider, VoIPBIN sends an SDP offer listing the
allowed codecs in the order specified. The provider (carrier) selects from that list during SDP
negotiation. Setting ``codecs`` restricts VoIPBIN's offer to only the listed codecs.

**Codecs field rules:**

- **Format:** Comma-separated codec names (e.g., ``"PCMU,PCMA"``). No spaces around commas.
- **Empty string (``""``):** No restriction applied — VoIPBIN uses its system-default codec list during SDP negotiation.
- **PSTN outbound only:** This field only affects calls routed to the PSTN through this provider. SIP-to-SIP calls within VoIPBIN are not affected.
- **Valid only for type ``sip``:** Ignored for other provider types.

**Commonly used codecs:**

+----------+------------------------------------------+
| Codec    | Description                              |
+==========+==========================================+
| PCMU     | G.711 µ-law (North America, standard)    |
+----------+------------------------------------------+
| PCMA     | G.711 A-law (Europe/international)       |
+----------+------------------------------------------+
| G729     | G.729 (compressed, lower bandwidth)      |
+----------+------------------------------------------+

**Setting codecs:**

- **At creation:** Pass ``"codecs"`` in the request body of ``POST /providers``.
- **After creation:** Update via ``PUT /providers/{id}`` with ``{"codecs": "PCMU,PCMA"}``.
- **After setup:** ``POST /providers/setup`` does **not** accept a ``codecs`` parameter.
  Set codec restrictions afterward using ``PUT /providers/{id}``.

.. note:: **AI Implementation Hint**

   Use ``codecs`` to force a specific codec when a carrier has interoperability issues with
   VoIPBIN's default SDP offer. For example, if a carrier only supports G.711 variants, set
   ``"codecs": "PCMU,PCMA"`` to prevent codec mismatch errors. Leave ``codecs`` as an empty
   string (``""``) unless you have a specific reason to restrict the codec list — restricting
   unnecessarily can reduce audio quality options.

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

**Setup Endpoint Errors**

+---------------------------+------------------------------------------------+
| HTTP Status               | Cause and Fix                                  |
+===========================+================================================+
| 422 Unprocessable Entity  | The carrier API key is invalid or lacks        |
|                           | sufficient permissions. Verify the key in the  |
|                           | carrier's dashboard and try again.             |
+---------------------------+------------------------------------------------+
| 400 Bad Request           | Missing required fields (``carrier``,          |
|                           | ``name``, ``detail``, ``credentials.api_key``) |
|                           | or unsupported carrier name.                   |
+---------------------------+------------------------------------------------+

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

