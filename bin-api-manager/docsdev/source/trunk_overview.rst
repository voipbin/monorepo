.. _trunk-overview:

Overview
========
VoIPBIN's Trunk system provides the infrastructure for SIP-based voice communication, supporting both outbound calls (trunking) and inbound calls (registration). Trunks connect VoIPBIN to the PSTN and enable SIP endpoints to communicate with external networks.

The Trunk system provides:

- SIP trunking for outbound call routing
- Registration services for inbound call delivery
- Connection between VoIPBIN and external providers
- Integration with on-premise PBX systems
- Flexible routing through providers


How Trunks Work
---------------
The trunk system handles bidirectional call flow between VoIPBIN and external networks.

**Trunk Architecture**

::

    +-----------------------------------------------------------------------+
    |                          Trunk System                                 |
    +-----------------------------------------------------------------------+

                         +-------------------+
                         |     VoIPBIN       |
                         |     Platform      |
                         +--------+----------+
                                  |
              +-------------------+-------------------+
              |                                       |
              v                                       v
    +-------------------+                   +-------------------+
    |    Trunking       |                   |   Registration    |
    | (Outbound Calls)  |                   | (Inbound Calls)   |
    +--------+----------+                   +--------+----------+
             |                                       |
             v                                       v
    +-------------------+                   +-------------------+
    |     Providers     |                   |   SIP Endpoints   |
    | (Telnyx, Twilio)  |                   | (Phones, PBX)     |
    +--------+----------+                   +--------+----------+
             |                                       |
             v                                       v
    +-------------------+                   +-------------------+
    |       PSTN        |                   |  Registered       |
    |   (Phone Network) |                   |  Devices          |
    +-------------------+                   +-------------------+

**Key Components**

- **Trunk**: Connection to external SIP provider for outbound calls
- **Registration**: Service for SIP endpoints to receive inbound calls
- **Provider**: External SIP service (Telnyx, Twilio, etc.)
- **PSTN**: Public Switched Telephone Network


Trunking vs Registration
------------------------
Understanding the difference between these two services is essential.

**Comparison**

::

    +-----------------------------------------------------------------------+
    |                    Trunking vs Registration                           |
    +-----------------------------------------------------------------------+

    TRUNKING (Outbound):
    +--------------------------------------------+
    | Purpose: Make calls to external numbers    |
    | Direction: VoIPBIN -> Provider -> PSTN     |
    | Uses: Provider configuration               |
    | Example: Call +1-555-123-4567              |
    +--------------------------------------------+

    REGISTRATION (Inbound):
    +--------------------------------------------+
    | Purpose: Receive calls on SIP devices      |
    | Direction: PSTN -> VoIPBIN -> Device       |
    | Uses: Extension configuration              |
    | Example: IP phone receives incoming call   |
    +--------------------------------------------+

**When to Use Each**

+----------------------+----------------------------------+----------------------------------+
| Capability           | Trunking                         | Registration                     |
+======================+==================================+==================================+
| Direction            | Outbound calls                   | Inbound calls                    |
+----------------------+----------------------------------+----------------------------------+
| Configuration        | Provider + Route                 | Extension                        |
+----------------------+----------------------------------+----------------------------------+
| Endpoint             | External phone number            | SIP device/softphone             |
+----------------------+----------------------------------+----------------------------------+
| Authentication       | Provider credentials             | Extension credentials            |
+----------------------+----------------------------------+----------------------------------+


Outbound Call Flow (Trunking)
-----------------------------
Outbound calls route through providers to reach external numbers.

**Trunking Flow**

::

    VoIPBIN                    Provider                      PSTN

    |                             |                             |
    | 1. Initiate call            |                             |
    |    to +1-555-1234           |                             |
    +---+                         |                             |
        |                         |                             |
        v                         |                             |
    +-------+                     |                             |
    | Route |                     |                             |
    | Engine|                     |                             |
    +---+---+                     |                             |
        |                         |                             |
        | 2. Select provider      |                             |
        |    based on route       |                             |
        |                         |                             |
        | 3. INVITE               |                             |
        +------------------------>|                             |
        |                         |                             |
        |                         | 4. Route to PSTN            |
        |                         +----------------------------->|
        |                         |                             |
        |                         |     5. 180 Ringing          |
        |                         |<----------------------------+
        |     6. Ringing          |                             |
        |<------------------------+                             |
        |                         |                             |
        |                         |     7. 200 OK (Answer)      |
        |                         |<----------------------------+
        |     8. Connected        |                             |
        |<------------------------+                             |
        |                         |                             |
        |         Media (RTP) <-------------------------------> |


Inbound Call Flow (Registration)
--------------------------------
Inbound calls reach registered SIP devices.

**Registration Flow**

::

    PSTN                       VoIPBIN                    SIP Device

    |                             |                             |
    | 1. Call to VoIPBIN number   |                             |
    +----------------------------->|                             |
    |                             |                             |
    |                             | 2. Flow execution           |
    |                             |    (dial extension)         |
    |                             +---+                         |
    |                             |   |                         |
    |                             |   v                         |
    |                             | +-------+                   |
    |                             | |Lookup |                   |
    |                             | |Reg.   |                   |
    |                             | +---+---+                   |
    |                             |     |                       |
    |                             | 3. INVITE to registered IP  |
    |                             +----------------------------->|
    |                             |                             |
    |                             |     4. 180 Ringing          |
    |                             |<----------------------------+
    |     5. Ringback             |                             |
    |<----------------------------+                             |
    |                             |                             |
    |                             |     6. 200 OK               |
    |                             |<----------------------------+
    |     7. Connected            |                             |
    |<----------------------------+                             |
    |                             |                             |
    |         Media (RTP) <-------------------------------------> |


Trunk Configuration
-------------------
Configure the trunk system through providers, routes, and extensions.

**Provider Configuration (for Trunking)**

See :ref:`Provider Overview <provider-overview>` for detailed configuration.

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/providers?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "Primary SIP Provider",
            "type": "sip",
            "hostname": "sip.provider.com"
        }'

**Route Configuration (for Trunking)**

See :ref:`Route Overview <route-overview>` for detailed configuration.

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/routes?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "Default Outbound Route",
            "provider_id": "provider-uuid-123",
            "priority": 1
        }'

**Extension Configuration (for Registration)**

See :ref:`Extension Overview <extension-overview>` for detailed configuration.

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/extensions?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "office-phone",
            "username": "office1",
            "password": "secure-password"
        }'


Complete Setup Example
----------------------
Set up a complete trunk system for bidirectional calling.

**Setup Steps**

::

    Step 1: Configure Provider (Outbound)
    +--------------------------------------------+
    | Provider: "Telnyx Production"              |
    | Hostname: sip.telnyx.com                   |
    | Type: sip                                  |
    +--------------------------------------------+
              |
              v
    Step 2: Configure Route (Outbound)
    +--------------------------------------------+
    | Route: "Default Route"                     |
    | Provider: Telnyx Production                |
    | Priority: 1                                |
    +--------------------------------------------+
              |
              v
    Step 3: Create Extensions (Inbound)
    +--------------------------------------------+
    | Extension: "office-main"                   |
    | Username: office-main                      |
    | Password: ********                         |
    +--------------------------------------------+
              |
              v
    Step 4: Register SIP Devices
    +--------------------------------------------+
    | Device registers to:                       |
    | office-main@{id}.registrar.voipbin.net    |
    +--------------------------------------------+
              |
              v
    Step 5: Configure Number Flow
    +--------------------------------------------+
    | Number: +1-555-123-4567                    |
    | Flow: Dial extension "office-main"         |
    +--------------------------------------------+

    Result:
    - Outbound: Make calls via Telnyx
    - Inbound: Calls to +1-555-123-4567 ring office phone


Common Scenarios
----------------

**Scenario 1: Basic Office Phone System**

Set up a simple office with outbound and inbound calling.

::

    Configuration:
    +--------------------------------------------+
    | Provider: Telnyx (for outbound)            |
    | Route: Default -> Telnyx                   |
    | Extensions: office-1, office-2, office-3   |
    | Number: +1-555-OFFICE                      |
    +--------------------------------------------+

    Call Flows:
    +--------------------------------------------+
    | Outbound: Any extension can dial out       |
    |   -> Routes through Telnyx to PSTN         |
    |                                            |
    | Inbound: Calls to +1-555-OFFICE            |
    |   -> IVR: "Press 1 for Sales..."          |
    |   -> Routes to appropriate extension       |
    +--------------------------------------------+

**Scenario 2: Multi-Provider Failover**

Configure redundant providers for reliability.

::

    Provider Configuration:
    +--------------------------------------------+
    | Primary: Telnyx                            |
    |   Route Priority: 1                        |
    |                                            |
    | Secondary: Twilio                          |
    |   Route Priority: 2                        |
    |                                            |
    | Tertiary: Bandwidth                        |
    |   Route Priority: 3                        |
    +--------------------------------------------+

    Failover Behavior:
    +--------------------------------------------+
    | 1. Try Telnyx first                        |
    | 2. If Telnyx fails (5xx) -> try Twilio    |
    | 3. If Twilio fails -> try Bandwidth        |
    | 4. All fail -> return error to caller      |
    +--------------------------------------------+

**Scenario 3: Remote Worker Setup**

Enable remote workers with softphones.

::

    Setup:
    +--------------------------------------------+
    | Extension per worker:                      |
    |   - remote-john                            |
    |   - remote-jane                            |
    |   - remote-bob                             |
    +--------------------------------------------+

    Worker Configuration:
    +--------------------------------------------+
    | Softphone App: Zoiper, Linphone, etc.      |
    | Server: registrar.voipbin.net              |
    | Username: remote-john                      |
    | Domain: {customer-id}.registrar.voipbin.net|
    +--------------------------------------------+

    Benefits:
    +--------------------------------------------+
    | - Work from anywhere with internet         |
    | - Same extension travels with worker       |
    | - Company number for outbound caller ID    |
    | - Inbound calls reach worker globally      |
    +--------------------------------------------+


Best Practices
--------------

**1. Provider Management**

- Configure at least two providers for failover
- Test provider connectivity regularly
- Monitor call success rates per provider
- Choose providers with coverage in your regions

**2. Route Configuration**

- Set up priority-based routing
- Configure failover for all routes
- Test failover scenarios periodically
- Document routing logic

**3. Extension Security**

- Use strong, unique passwords
- Rotate credentials periodically
- Monitor for unauthorized registrations
- Enable TLS when possible

**4. Monitoring**

- Track call success/failure rates
- Monitor provider latency and quality
- Set up alerts for registration failures
- Review call logs regularly


Troubleshooting
---------------

**Outbound (Trunking) Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Calls not connecting      | Check provider is configured; verify route     |
|                           | exists; confirm provider credentials           |
+---------------------------+------------------------------------------------+
| Wrong provider used       | Review route priorities; check for             |
|                           | destination-specific routes                    |
+---------------------------+------------------------------------------------+
| All providers failing     | Check provider status; verify network          |
|                           | connectivity; test each provider independently |
+---------------------------+------------------------------------------------+

**Inbound (Registration) Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Device not receiving      | Verify registration is active; check           |
| calls                     | extension in flow; confirm device is online    |
+---------------------------+------------------------------------------------+
| Registration failing      | Check credentials; verify registrar address;   |
|                           | confirm firewall allows SIP traffic            |
+---------------------------+------------------------------------------------+
| Intermittent registration | Check NAT settings; enable keep-alives;        |
|                           | increase registration expiry                   |
+---------------------------+------------------------------------------------+

**Audio Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| No audio                  | Check RTP ports; verify NAT traversal;         |
|                           | confirm media IP is reachable                  |
+---------------------------+------------------------------------------------+
| One-way audio             | Enable STUN/TURN; check symmetric RTP;         |
|                           | verify both endpoints can send/receive         |
+---------------------------+------------------------------------------------+
| Poor audio quality        | Check network bandwidth; reduce jitter;        |
|                           | consider QoS configuration                     |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Provider Overview <provider-overview>` - Provider configuration for trunking
- :ref:`Route Overview <route-overview>` - Call routing and failover
- :ref:`Extension Overview <extension-overview>` - SIP endpoint registration
- :ref:`Call Overview <call-overview>` - Making and receiving calls
- :ref:`Flow Overview <flow-overview>` - Call flow configuration

