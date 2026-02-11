.. _extension-overview:

Overview
========
VoIPBIN's Extension API enables management of SIP endpoints that can register with VoIPBIN to receive inbound calls. Extensions provide the bridge between VoIPBIN's cloud infrastructure and your SIP devices, softphones, or PBX systems.

With the Extension API you can:

- Create extensions for SIP device registration
- Configure authentication credentials
- Manage multiple endpoints per customer
- Route inbound calls to registered devices
- Enable direct extension access via public SIP URI
- Monitor registration status


How Extensions Work
-------------------
Extensions provide address endpoints for SIP device registration.

**Extension Architecture**

::

    +-----------------------------------------------------------------------+
    |                        Extension System                               |
    +-----------------------------------------------------------------------+

    +-------------------+
    |    Extension      |
    | (address of record)|
    +--------+----------+
             |
             | registers
             v
    +--------+----------+
    |   SIP Devices     |
    +--------+----------+
             |
             +--------> Softphone (computer/mobile)
             |
             +--------> IP Phone (hardware)
             |
             +--------> PBX System (Asterisk, FreePBX)
             |
             +--------> SIP Gateway

    Registration Address Format:
    +-----------------------------------------------------------------------+
    | {extension}@{customer-id}.registrar.voipbin.net                       |
    +-----------------------------------------------------------------------+

**Key Components**

- **Extension**: A SIP address of record (AOR) for device registration
- **Username**: Authentication identity for the extension
- **Password**: Secret credential for authentication
- **Registrar**: VoIPBIN's SIP registration server


.. _extension-overview-registration:

Registration Process
--------------------
SIP devices must register with VoIPBIN to receive inbound calls.

**Registration Flow**

::

    SIP Device                                VoIPBIN Registrar

    |                                              |
    | 1. REGISTER (no credentials)                 |
    +--------------------------------------------->|
    |                                              |
    |      2. 407 Proxy Authentication Required    |
    |         (includes nonce challenge)           |
    |<---------------------------------------------+
    |                                              |
    | 3. ACK                                       |
    +--------------------------------------------->|
    |                                              |
    | 4. REGISTER (with Authorization header)      |
    |    (username + password + nonce response)    |
    +--------------------------------------------->|
    |                                              |
    |      5. 200 OK (registration accepted)       |
    |<---------------------------------------------+
    |                                              |
    | 6. ACK                                       |
    +--------------------------------------------->|
    |                                              |
    |  Device is now registered and can            |
    |  receive inbound calls                       |

**Registration Lifecycle**

::

    +-------------------+
    |   Unregistered    |
    +--------+----------+
             |
             | Send REGISTER
             v
    +-------------------+     407 Challenge
    |   Authenticating  |<-------------------+
    +--------+----------+                    |
             |                               |
             | Send credentials              | Credentials invalid
             v                               |
    +-------------------+     200 OK         |
    |    Registered     |--------------------+
    +--------+----------+
             |
             | Expiration or REGISTER (expires=0)
             v
    +-------------------+
    |   Unregistered    |
    +-------------------+


407 Proxy Authentication Required
---------------------------------
VoIPBIN uses digest authentication for secure registration.

**Authentication Challenge Process**

::

    Challenge Response:
    +-----------------------------------------------------------------------+
    | 407 Proxy Authentication Required                                     |
    +-----------------------------------------------------------------------+
    | WWW-Authenticate: Digest                                              |
    |   realm="voipbin.net",                                               |
    |   nonce="unique-random-string",                                       |
    |   algorithm=MD5                                                       |
    +-----------------------------------------------------------------------+

    Client Response:
    +-----------------------------------------------------------------------+
    | REGISTER sip:registrar.voipbin.net                                    |
    +-----------------------------------------------------------------------+
    | Authorization: Digest                                                 |
    |   username="extension-name",                                         |
    |   realm="voipbin.net",                                               |
    |   nonce="unique-random-string",                                       |
    |   uri="sip:registrar.voipbin.net",                                   |
    |   response="calculated-hash"                                          |
    +-----------------------------------------------------------------------+

**Nonce Purpose**

The nonce value prevents replay attacks by ensuring each authentication attempt is unique.


Extension Configuration
-----------------------
Create and manage extensions for your SIP endpoints.

**Create an Extension**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/extensions?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "office-phone-1",
            "detail": "Main office IP phone",
            "username": "office1",
            "password": "secure-password-123"
        }'

**Response:**

.. code::

    {
        "id": "extension-uuid-123",
        "customer_id": "customer-uuid-456",
        "name": "office-phone-1",
        "detail": "Main office IP phone",
        "username": "office1",
        "tm_create": "2024-01-15T10:30:00Z"
    }

**List Extensions**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/extensions?token=<token>'

**Get Extension Details**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/extensions/<extension-id>?token=<token>'

**Delete Extension**

.. code::

    $ curl -X DELETE 'https://api.voipbin.net/v1.0/extensions/<extension-id>?token=<token>'


Calling Registered Extensions
-----------------------------
Inbound calls reach registered devices via the extension address.

**Address Format**

::

    Full SIP URI:
    +-----------------------------------------------------------------------+
    | sip:{extension}@{customer-id}.registrar.voipbin.net                   |
    +-----------------------------------------------------------------------+

    Example:
    +-----------------------------------------------------------------------+
    | sip:office1@abc123-def456.registrar.voipbin.net                       |
    +-----------------------------------------------------------------------+

**Inbound Call Flow**

::

    Incoming Call                     VoIPBIN                    SIP Device
         |                               |                           |
         | Call to extension             |                           |
         +------------------------------>|                           |
         |                               |                           |
         |                               | Lookup registration       |
         |                               | Find device IP            |
         |                               |                           |
         |                               | INVITE                    |
         |                               +-------------------------->|
         |                               |                           |
         |                               |        180 Ringing        |
         |                               |<--------------------------+
         |                               |                           |
         |      Ringback tone            |        200 OK             |
         |<------------------------------|<--------------------------+
         |                               |                           |
         |      Call connected           |        Media flow         |
         |<------------------------------|<------------------------->|


.. _extension-overview-direct:

Direct Extension
----------------
Direct extensions provide a public SIP URI that allows external callers to reach a registered extension without needing to know the customer's registrar domain. When direct access is enabled for an extension, VoIPBIN generates a unique hash and exposes a simplified SIP address.

**Direct SIP URI Format**

::

    Standard extension address (requires customer domain knowledge):
    +-----------------------------------------------------------------------+
    | sip:{extension}@{customer-id}.registrar.voipbin.net                   |
    +-----------------------------------------------------------------------+

    Direct extension address (public, simplified):
    +-----------------------------------------------------------------------+
    | sip:direct.{hash}@sip.voipbin.net                                     |
    +-----------------------------------------------------------------------+

    Example:
    +-----------------------------------------------------------------------+
    | sip:direct.a1b2c3d4e5f6@sip.voipbin.net                              |
    +-----------------------------------------------------------------------+

**How Direct Extensions Work**

::

    External Caller                    VoIPBIN                     SIP Device

         |                                |                            |
         | INVITE                         |                            |
         | sip:direct.<hash>@sip.voipbin.net                           |
         +------------------------------->|                            |
         |                                |                            |
         |                                | 1. Lookup hash             |
         |                                | 2. Find extension          |
         |                                | 3. Lookup registration     |
         |                                |                            |
         |                                | INVITE                     |
         |                                +--------------------------->|
         |                                |                            |
         |                                |        180 Ringing         |
         |                                |<---------------------------+
         |                                |                            |
         |      Ringback tone             |        200 OK              |
         |<-------------------------------|<---------------------------+
         |                                |                            |
         |      Call connected            |        Media flow          |
         |<-------------------------------|<-------------------------->|

**Managing Direct Extensions**

- **Enable**: Update the extension with ``"direct": true`` to generate a hash
- **Disable**: Update the extension with ``"direct": false`` to remove the hash
- **Regenerate**: Update the extension with ``"direct_regenerate": true`` to create a new hash (invalidates the old one)

The ``direct_hash`` field in the extension response contains the current hash. An empty string indicates direct access is disabled.

**Use Cases**

- Share a simple SIP address with external partners or customers
- Allow inbound calls from SIP trunks that cannot be configured with customer-specific domains
- Provide a stable public contact point that can be regenerated if compromised


Common Scenarios
----------------

**Scenario 1: IP Phone Registration**

Configure a hardware IP phone to register with VoIPBIN.

::

    IP Phone Configuration:
    +--------------------------------------------+
    | SIP Server: registrar.voipbin.net          |
    | Username: office-phone-1                   |
    | Password: ********                         |
    | Domain: {customer-id}.registrar.voipbin.net|
    +--------------------------------------------+

    Registration Result:
    +--------------------------------------------+
    | Status: Registered                         |
    | Expires: 3600 seconds                      |
    | Contact: sip:office-phone-1@192.168.1.100  |
    +--------------------------------------------+

    The phone can now receive inbound calls
    at: sip:office-phone-1@{customer-id}.registrar.voipbin.net

**Scenario 2: Softphone on Mobile**

Register a mobile softphone for remote workers.

::

    Mobile Softphone Setup:
    +--------------------------------------------+
    | App: Any SIP-compatible softphone          |
    | Account Name: Work Mobile                  |
    |                                            |
    | Server: registrar.voipbin.net              |
    | User: mobile-user-john                     |
    | Password: ********                         |
    | Domain: {customer-id}.registrar.voipbin.net|
    +--------------------------------------------+

    Use Case:
    +--------------------------------------------+
    | 1. Employee travels with mobile phone      |
    | 2. Softphone registers over 4G/WiFi        |
    | 3. Office calls reach employee anywhere    |
    | 4. Same extension, any location            |
    +--------------------------------------------+

**Scenario 3: PBX System Integration**

Connect an on-premise PBX to VoIPBIN for inbound calls.

::

    PBX Configuration:
    +--------------------------------------------+
    | PBX Type: Asterisk / FreePBX               |
    |                                            |
    | SIP Trunk to VoIPBIN:                      |
    | - Register: Yes                            |
    | - Host: registrar.voipbin.net              |
    | - Username: pbx-main                       |
    | - Password: ********                       |
    | - From Domain: {customer-id}.registrar...  |
    +--------------------------------------------+

    Inbound Call Flow:
    +--------------------------------------------+
    | 1. Call arrives at VoIPBIN number          |
    | 2. Flow routes to extension: pbx-main      |
    | 3. VoIPBIN sends INVITE to registered PBX  |
    | 4. PBX IVR answers and routes internally   |
    +--------------------------------------------+


Best Practices
--------------

**1. Security**

- Use strong, unique passwords for each extension
- Rotate credentials periodically
- Use TLS for SIP registration when available
- Monitor for unauthorized registration attempts

**2. Registration Management**

- Set appropriate registration expiry times
- Handle re-registration before expiry
- Implement registration failure handling
- Use keep-alive mechanisms for NAT traversal

**3. Extension Naming**

- Use descriptive, meaningful names
- Follow a consistent naming convention
- Include location or purpose in name
- Avoid special characters in usernames

**4. Monitoring**

- Track registration status
- Alert on registration failures
- Monitor for duplicate registrations
- Log authentication attempts


Troubleshooting
---------------

**Registration Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| 401/407 auth failure      | Verify username and password; check realm;     |
|                           | ensure credentials match exactly               |
+---------------------------+------------------------------------------------+
| Registration timeout      | Check network connectivity; verify firewall    |
|                           | allows SIP (UDP 5060); check NAT settings      |
+---------------------------+------------------------------------------------+
| Registration expires      | Increase expiry time; enable keep-alives;      |
| frequently                | check for NAT timeout issues                   |
+---------------------------+------------------------------------------------+

**Call Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Calls not reaching device | Verify registration is active; check extension |
|                           | address in flow; confirm device is online      |
+---------------------------+------------------------------------------------+
| One-way audio             | Check NAT configuration; verify RTP ports;     |
|                           | enable STUN/TURN if behind NAT                 |
+---------------------------+------------------------------------------------+
| Call drops after seconds  | Check session timers; verify re-INVITE         |
|                           | handling; review NAT keep-alive settings       |
+---------------------------+------------------------------------------------+

**Configuration Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Extension not found       | Verify extension ID; check customer ID in      |
|                           | domain; ensure extension exists                |
+---------------------------+------------------------------------------------+
| Duplicate registration    | Only one device per extension; use unique      |
| error                     | extensions for each device                     |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Trunk Overview <trunk-overview>` - SIP trunking for outbound calls
- :ref:`Flow Overview <flow-overview>` - Routing calls to extensions
- :ref:`Call Overview <call-overview>` - Making and receiving calls
- :ref:`Number Overview <number-overview>` - Associating numbers with extensions

