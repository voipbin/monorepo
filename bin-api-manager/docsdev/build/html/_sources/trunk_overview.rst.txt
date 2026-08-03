.. _trunk-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low. A trunk is a single, standalone resource: create it with a domain name and authentication, then point your SIP device or PBX at it.
   * **Cost:** Free to create and manage. Outbound calls placed through the trunk incur per-call charges based on the destination.
   * **Async:** No. ``POST https://api.voipbin.net/v1.0/trunks`` returns immediately with the created trunk.

VoIPBIN's Trunk API lets you connect your own SIP device or PBX to VoIPBIN as a SIP trunk, so it can place outbound calls to the PSTN through VoIPBIN. Creating a trunk reserves a dedicated SIP domain address (``{domain_name}.trunk.voipbin.net``) and configures how VoIPBIN authenticates INVITE requests arriving at that address.

The Trunk API provides:

- A dedicated, auto-generated SIP domain address for outbound trunking
- Basic authentication (SIP username/password)
- IP-based authentication (allowed source IP list)
- Support for both authentication methods on the same trunk at once
- Standard CRUD management (create, list, retrieve, update, delete)

A trunk is a standalone resource. It does not require a :ref:`Provider <provider-overview>`, :ref:`Route <route-overview>`, or :ref:`Extension <extension-overview>` to be created first, and it is not used to configure any of those resources. See `Trunking vs Other Resources`_ below for how it relates to them.


How Trunks Work
---------------
A trunk lets an external SIP device authenticate against VoIPBIN and place outbound calls through it.

**Trunk Architecture**

::

    +-----------------------------------------------------------------------+
    |                          Trunk System                                 |
    +-----------------------------------------------------------------------+

    +-------------------+
    |   Your SIP Device |
    |   or PBX (UA)      |
    +--------+----------+
             |
             | INVITE to {domain_name}.trunk.voipbin.net
             | (authenticated via username/password or source IP)
             v
    +-------------------+
    |     VoIPBIN       |
    |   Trunk Endpoint  |
    +--------+----------+
             |
             v
    +-------------------+
    |       PSTN         |
    |  (Phone Network)   |
    +-------------------+

**Key Components**

- **Trunk**: The resource that defines the SIP domain and authentication for your device
- **Domain name**: The unique subdomain, reachable at ``{domain_name}.trunk.voipbin.net``
- **Auth types**: ``basic`` (username/password) and/or ``ip`` (allowed source IPs)
- **PSTN**: Public Switched Telephone Network, reached once the INVITE is authenticated


Trunking vs Other Resources
----------------------------
"SIP trunking" appears in several related but independent VoIPBIN resources. A trunk is not built from these other resources, and they are not built from a trunk.

+-----------------------+---------------------------------------------+------------------------------------------------+
| Resource              | Direction                                   | Purpose                                        |
+=======================+=============================================+================================================+
| Trunk (this API)      | Your device -> VoIPBIN -> PSTN              | Lets your own SIP device/PBX                   |
|                       | (outbound)                                  | authenticate into VoIPBIN and place            |
|                       |                                             | outbound calls                                 |
+-----------------------+---------------------------------------------+------------------------------------------------+
| Provider              | VoIPBIN -> Carrier -> PSTN                  | Configures the upstream carrier                |
|                       | (outbound)                                  | VoIPBIN itself uses to place                   |
|                       |                                             | outbound calls                                 |
+-----------------------+---------------------------------------------+------------------------------------------------+
| Route                 | VoIPBIN -> Provider selection               | Chooses which Provider handles an              |
|                       |                                             | outbound call, with failover                   |
+-----------------------+---------------------------------------------+------------------------------------------------+
| Extension             | PSTN -> VoIPBIN -> your device              | Registers your SIP device to receive           |
|                       | (inbound)                                   | inbound calls                                  |
+-----------------------+---------------------------------------------+------------------------------------------------+

See :ref:`Trunk Overview <trunk-overview-trunking>` for the authentication and call-handling details of this Trunk API.


.. note:: **AI Implementation Hint**

   A trunk is created with a single call to ``POST https://api.voipbin.net/v1.0/trunks``. There is no separate provider, route, or extension setup required. Choose ``auth_types: ["basic"]`` and supply ``username``/``password``, or ``auth_types: ["ip"]`` and supply ``allowed_ips``, or both.


Managing Trunks
----------------
Trunks are managed through the ``/v1.0/trunks`` endpoints.

**Get list of trunks**

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/trunks?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "name": "Primary Carrier",
                "detail": "Main PSTN trunk for outbound calls",
                "domain_name": "carrier.example.com",
                "auth_types": ["basic"],
                "username": "trunk_user",
                "password": "trunk_pass",
                "allowed_ips": [],
                "tm_create": "2024-03-01T10:00:00.000000Z",
                "tm_update": "2024-03-01T10:00:00.000000Z",
                "tm_delete": "9999-01-01T00:00:00.000000Z"
            }
        ],
        "next_page_token": "2024-03-01T10:00:00.000000Z"
    }

**Get trunk detail**

.. code::

    $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/trunks/a1b2c3d4-e5f6-7890-abcd-ef1234567890?token=<YOUR_AUTH_TOKEN>'

**Create a trunk**

``name``, ``detail``, ``domain_name``, ``auth_types``, ``username``, ``password``, and ``allowed_ips`` are all required.

.. code::

    $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/trunks?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "Primary Carrier",
            "detail": "Main PSTN trunk for outbound calls",
            "domain_name": "carrier.example.com",
            "auth_types": ["basic"],
            "username": "trunk_user",
            "password": "trunk_pass",
            "allowed_ips": []
        }'

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Primary Carrier",
        "detail": "Main PSTN trunk for outbound calls",
        "domain_name": "carrier.example.com",
        "auth_types": ["basic"],
        "username": "trunk_user",
        "password": "trunk_pass",
        "allowed_ips": [],
        "tm_create": "2024-03-01T10:00:00.000000Z",
        "tm_update": "",
        "tm_delete": ""
    }

**Update a trunk**

``name``, ``detail``, ``auth_types``, ``username``, ``password``, and ``allowed_ips`` are all required. The ``domain_name`` cannot be changed after creation.

.. code::

    $ curl -k --location --request PUT 'https://api.voipbin.net/v1.0/trunks/a1b2c3d4-e5f6-7890-abcd-ef1234567890?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "Primary Carrier (updated)",
            "detail": "Main PSTN trunk for outbound calls",
            "auth_types": ["basic", "ip"],
            "username": "trunk_user",
            "password": "trunk_pass",
            "allowed_ips": ["203.0.113.1", "203.0.113.2"]
        }'

**Delete a trunk**

.. code::

    $ curl -k --location --request DELETE 'https://api.voipbin.net/v1.0/trunks/a1b2c3d4-e5f6-7890-abcd-ef1234567890?token=<YOUR_AUTH_TOKEN>'


Common Scenarios
----------------

**Scenario 1: PBX with basic authentication**

Point an on-premise PBX at VoIPBIN using a username and password.

::

    Configuration:
    +--------------------------------------------+
    | Trunk: "Office PBX"                        |
    | domain_name: office-pbx.example.com        |
    | auth_types: ["basic"]                      |
    | username / password: PBX SIP credentials   |
    +--------------------------------------------+

    Call Flow:
    +--------------------------------------------+
    | PBX sends INVITE with credentials          |
    |   -> office-pbx.example.com.trunk.voipbin.net |
    |   -> VoIPBIN authenticates and connects    |
    |      the call to the PSTN                  |
    +--------------------------------------------+

**Scenario 2: Fixed-IP device with IP-based authentication**

Allow a device with a known static IP to place calls without credentials.

::

    Configuration:
    +--------------------------------------------+
    | Trunk: "Static Gateway"                    |
    | domain_name: gateway.example.com           |
    | auth_types: ["ip"]                         |
    | allowed_ips: ["203.0.113.10"]              |
    +--------------------------------------------+

**Scenario 3: Both authentication methods**

Enable basic and IP-based authentication on the same trunk at once, so either method authorizes the call.

::

    Configuration:
    +--------------------------------------------+
    | Trunk: "Hybrid Trunk"                      |
    | auth_types: ["basic", "ip"]                |
    | username / password: set                   |
    | allowed_ips: ["203.0.113.1"]               |
    +--------------------------------------------+


Best Practices
--------------

**1. Authentication**

- Use strong, unique passwords for basic authentication
- Prefer IP-based authentication for devices with a static, known IP
- Combine both methods only when needed; each accepted method widens the attack surface
- Rotate credentials periodically

**2. Domain Naming**

- Choose a ``domain_name`` that is easy to identify in logs (e.g. per office or per device)
- Remember ``domain_name`` cannot be changed after creation; delete and recreate the trunk if it must change

**3. Monitoring**

- Track call success/failure rates on calls placed through the trunk
- Review call logs regularly for unexpected source IPs or authentication failures


Troubleshooting
---------------

+---------------------------+--------------------------------------------------+
| Symptom                   | Solution                                         |
+===========================+==================================================+
| 407 loop / auth failure   | Confirm the SIP client sends credentials in      |
|                           | response to the 407 challenge; verify            |
|                           | username/password match the trunk                |
+---------------------------+--------------------------------------------------+
| INVITE rejected           | Confirm the source IP is in ``allowed_ips``      |
|                           | when using IP-based authentication               |
+---------------------------+--------------------------------------------------+
| Calls not connecting      | Confirm the ``domain_name`` in the INVITE        |
|                           | request-URI matches the trunk's domain           |
+---------------------------+--------------------------------------------------+
| No audio / one-way audio  | Check RTP ports, NAT traversal, and that both    |
|                           | endpoints can send/receive media                 |
+---------------------------+--------------------------------------------------+


Related Documentation
---------------------

- :ref:`Trunk Overview <trunk-overview-trunking>` - Authentication and call handling details
- :ref:`Domain name <trunk-overview-domain_name>` - How the SIP domain address works
- :ref:`Provider Overview <provider-overview>` - VoIPBIN's own outbound carrier configuration
- :ref:`Route Overview <route-overview>` - Call routing and failover between providers
- :ref:`Extension Overview <extension-overview>` - SIP endpoint registration for inbound calls
- :ref:`Call Overview <call-overview>` - Making and receiving calls
