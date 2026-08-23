.. _extension-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Medium
   * **Cost:** Free. Extensions are internal routing endpoints with no per-unit charges.
   * **Async:** No. ``POST https://api.voipbin.net/v1.0/extensions`` returns immediately with the created extension. SIP device registration is a separate process handled by the SIP device itself.

VoIPBIN's Extension API enables management of SIP endpoints that can register with VoIPBIN to receive inbound calls. Extensions provide the bridge between VoIPBIN's cloud infrastructure and your SIP devices, softphones, or PBX systems.

With the Extension API you can:

- Create extensions for SIP device registration
- Configure authentication credentials
- Manage multiple endpoints per customer
- Route inbound calls to registered devices
- Enable direct extension access via public SIP URI
- Provision the Linphone softphone via QR code
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
    | {extension}@{domain_name}                                             |
    |                                                                       |
    | domain_name is returned by the extension API,                         |
    | e.g. ab12.reg.voipbin.net                                             |
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
    | REGISTER sip:ab12.reg.voipbin.net                                     |
    +-----------------------------------------------------------------------+
    | Authorization: Digest                                                 |
    |   username="extension-name",                                         |
    |   realm="voipbin.net",                                               |
    |   nonce="unique-random-string",                                       |
    |   uri="sip:ab12.reg.voipbin.net",                                    |
    |   response="calculated-hash"                                          |
    +-----------------------------------------------------------------------+

**Nonce Purpose**

The nonce value prevents replay attacks by ensuring each authentication attempt is unique.


.. note:: **AI Implementation Hint**

   The SIP registration domain is customer-specific and follows the pattern ``{label}.reg.voipbin.net``, where ``{label}`` is a short identifier (4 characters) assigned to your account. Do not construct the domain yourself. Always use the ``domain_name`` field returned by the extension API verbatim. When configuring SIP devices, use the ``username`` and ``password`` from the extension, and the ``domain_name`` from the API response. The extension API always returns the current domain for your account.

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
            "extension": "office1",
            "password": "secure-password-123"
        }'

**Response:**

.. code::

    {
        "id": "extension-uuid-123",
        "customer_id": "customer-uuid-456",
        "name": "office-phone-1",
        "detail": "Main office IP phone",
        "extension": "office1",
        "domain_name": "ab12.reg.voipbin.net",
        "username": "office1",
        "password": "secure-password-123",
        "direct_hash": "a8f3b2c1d4e5",
        "tm_create": "2024-01-15T10:30:00Z",
        "tm_update": "",
        "tm_delete": ""
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
    | sip:{extension}@{domain_name}                                         |
    +-----------------------------------------------------------------------+

    Example:
    +-----------------------------------------------------------------------+
    | sip:office1@ab12.reg.voipbin.net                                      |
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
    | sip:{extension}@{domain_name}                                         |
    | e.g. sip:office1@ab12.reg.voipbin.net                                 |
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

- A direct hash is automatically created when the extension is created
- **Regenerate**: Call ``POST /extensions/{id}/direct-hash-regenerate`` to create a new hash (invalidates the old one)

The ``direct_hash`` field in the extension response contains the current hash.

**Use Cases**

- Share a simple SIP address with external partners or customers
- Allow inbound calls from SIP trunks that cannot be configured with customer-specific domains
- Provide a stable public contact point that can be regenerated if compromised


.. _extension-overview-provisioning:

Softphone QR Provisioning
-------------------------
Softphone QR provisioning lets a user configure the Linphone mobile app for an extension without typing any credentials. An administrator requests a short-lived provisioning token for the extension, renders the returned URL as a QR code, and the user scans the code with Linphone. Linphone fetches the provisioning URL and applies the SIP account settings (domain, username, password, transport) automatically. This uses Linphone's standard remote provisioning mechanism (lpconfig XML).

**Provisioning Flow**

::

    Admin (API)                    VoIPBIN                     Linphone App

    |                                 |                             |
    | 1. POST /extensions/{id}/       |                             |
    |    provisioning-token           |                             |
    +-------------------------------->|                             |
    |                                 |                             |
    |    2. { token, url, expire }    |                             |
    |<--------------------------------+                             |
    |                                 |                             |
    | 3. Render url as QR code        |                             |
    |    (user scans it with          |                             |
    |     the Linphone app)           |                             |
    |                                 |                             |
    |                                 |   4. GET /provisioning/     |
    |                                 |      extension?token=...    |
    |                                 |<----------------------------+
    |                                 |                             |
    |                                 |   5. lpconfig XML           |
    |                                 |      (SIP account settings) |
    |                                 +---------------------------->|
    |                                 |                             |
    |                                 |   6. REGISTER               |
    |                                 |<----------------------------+
    |                                 |                             |
    |                                 |   Account configured and    |
    |                                 |   registered                |

**Endpoints**

- **Issue a token** (authenticated, admin or manager permission): ``POST /v1.0/extensions/{id}/provisioning-token`` returns a ``token``, a ready-to-use ``url``, and an ``expire`` timestamp. See :ref:`Provisioning Token <extension-struct-extension-provisioning-token>` for the response structure.
- **Fetch the configuration** (public, no authentication): ``GET https://api.voipbin.net/provisioning/extension?token=<token>`` returns the Linphone configuration XML (``application/xml``). This is the URL encoded in the QR code. Note that this public endpoint has no ``/v1.0`` prefix. Any invalid, expired, or unknown token receives a bare ``400`` response.

**Token Lifetime and Security**

- The provisioning token is a random 64-character hex string and expires **10 minutes** after issuance.
- The token can be used multiple times within its lifetime. Linphone may re-fetch the URL (e.g. on a retry or after an app restart), so the token is not consumed on first use.
- The provisioning URL serves the extension's SIP password in plain text to whoever holds the token. Treat the QR code and the URL like a credential: show the QR code only to the intended user, and do not send the URL over untrusted channels.
- After the 10-minute window, the URL stops working. Issue a new token to generate a fresh QR code.
- If the extension's password changes while a token is still valid, re-issue the token; a previously issued URL may serve the old credentials until it expires.

**Behavior Notes**

- **Linphone only.** The served XML is Linphone's lpconfig format. Other softphones (e.g. Zoiper, Grandstream) use proprietary QR provisioning formats and cannot consume this URL.
- **Account 0 replacement.** Scanning on a Linphone app that already has a SIP account configured replaces account 0 (the first account) with the provisioned one.
- **Older Linphone versions.** The configuration marks itself as transient so recent Linphone versions do not re-fetch the URL after it expires. Older versions may show a one-time provisioning warning after an app restart once the token has expired. The registered account keeps working; the warning can be dismissed.
- **Transport.** The provisioned account uses SIP over UDP.


Common Scenarios
----------------

**Scenario 1: IP Phone Registration**

Configure a hardware IP phone to register with VoIPBIN.

::

    IP Phone Configuration:
    +--------------------------------------------+
    | SIP Server: ab12.reg.voipbin.net           |
    | Username: office-phone-1                   |
    | Password: ********                         |
    | Domain: ab12.reg.voipbin.net               |
    | (use the extension's domain_name)          |
    +--------------------------------------------+

    Registration Result:
    +--------------------------------------------+
    | Status: Registered                         |
    | Expires: 3600 seconds                      |
    | Contact: sip:office-phone-1@192.168.1.100  |
    +--------------------------------------------+

    The phone can now receive inbound calls
    at: sip:office-phone-1@ab12.reg.voipbin.net

**Scenario 2: Softphone on Mobile**

Register a mobile softphone for remote workers.

::

    Mobile Softphone Setup:
    +--------------------------------------------+
    | App: Any SIP-compatible softphone          |
    | Account Name: Work Mobile                  |
    |                                            |
    | Server: ab12.reg.voipbin.net               |
    | User: mobile-user-john                     |
    | Password: ********                         |
    | Domain: ab12.reg.voipbin.net               |
    | (use the extension's domain_name)          |
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
    | - Host: ab12.reg.voipbin.net               |
    | - Username: pbx-main                       |
    | - Password: ********                       |
    | - From Domain: ab12.reg.voipbin.net        |
    |   (use the extension's domain_name)        |
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

**Transport note:** SIP over UDP is the default. If a SIP message grows too large for a single UDP datagram (e.g. many headers or a large SDP body), switch the device transport to TCP or TLS. Both are fully supported and remain the recommended fallback for oversized SIP messages.

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
| Extension not found       | Verify extension ID; check the domain matches  |
|                           | the extension's domain_name; ensure extension  |
|                           | exists                                         |
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

