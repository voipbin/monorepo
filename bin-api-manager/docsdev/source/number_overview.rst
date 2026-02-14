.. _number-overview:

Overview
========
VoIPBIN's Number API enables you to provision, manage, and configure phone numbers for your communication applications. Numbers serve as the entry points for inbound calls and messages, and can be configured with custom flows for automated handling.

With the Number API you can:

- Search and provision phone numbers from available inventory
- Configure call and message handling flows
- Port existing numbers from other providers
- Manage number settings and metadata
- Release numbers when no longer needed


How Numbers Work
----------------
Numbers connect external callers to your VoIPBIN applications through configurable flows.

**Number Architecture**

::

    +-----------------------------------------------------------------------+
    |                         Number System                                 |
    +-----------------------------------------------------------------------+

    External World                    VoIPBIN                    Your Application
         |                               |                              |
         | Inbound call/SMS              |                              |
         | to +15551234567               |                              |
         +------------------------------>|                              |
         |                               |                              |
         |                        +------+------+                       |
         |                        |   Number    |                       |
         |                        | +1555123... |                       |
         |                        +------+------+                       |
         |                               |                              |
         |                   +-----------+-----------+                  |
         |                   |                       |                  |
         |                   v                       v                  |
         |            +------------+          +------------+            |
         |            | call_flow  |          | msg_flow   |            |
         |            +------+-----+          +------+-----+            |
         |                   |                       |                  |
         |                   v                       v                  |
         |            Execute flow            Execute flow              |
         |            (IVR, AI,               (auto-reply,              |
         |             queue, etc.)            forward, etc.)           |
         |                   |                       |                  |
         |                   +--------->+<-----------+                  |
         |                              |                               |
         |                              v                               |
         |                       +-----------+                          |
         |                       |  Webhook  |------------------------->|
         |                       +-----------+                          |

**Key Components**

- **Number**: A phone number provisioned in VoIPBIN
- **Call Flow**: Actions to execute when a call arrives
- **Message Flow**: Actions to execute when an SMS/MMS arrives
- **Webhook**: Notifications sent to your application


Number Lifecycle
----------------
Numbers progress through predictable states from provisioning to release.

**Number States**

::

    Search available numbers
           |
           v
    +------------+
    | available  | (in VoIPBIN inventory)
    +-----+------+
          |
          | POST /numbers (provision)
          v
    +------------+
    |   active   |<-----------------+
    +-----+------+                  |
          |                         |
          | suspend                 | reactivate
          v                         |
    +------------+                  |
    | suspended  |------------------+
    +-----+------+
          |
          | DELETE /numbers/{id}
          v
    +------------+
    |  released  |
    +------------+

**State Descriptions**

+-------------+------------------------------------------------------------------+
| State       | What's happening                                                 |
+=============+==================================================================+
| available   | Number is in inventory, ready to be provisioned                  |
+-------------+------------------------------------------------------------------+
| active      | Number is provisioned and ready to receive calls/messages        |
+-------------+------------------------------------------------------------------+
| suspended   | Number is temporarily disabled                                   |
+-------------+------------------------------------------------------------------+
| released    | Number returned to inventory or carrier                          |
+-------------+------------------------------------------------------------------+


Provisioning Numbers
--------------------
Provision numbers through a two-step process: search, then provision.

**Step 1: Search Available Numbers**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/number_availables?token=<token>&country=US&type=local'

**Response:**

.. code::

    {
        "result": [
            {
                "number": "+15551234567",
                "country": "US",
                "type": "local",
                "region": "California",
                "city": "San Francisco"
            },
            {
                "number": "+15551234568",
                "country": "US",
                "type": "local",
                "region": "California",
                "city": "Los Angeles"
            }
        ]
    }

**Step 2: Provision the Number**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/numbers?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "number": "+15551234567"
        }'


.. _number-overview-flow_execution:

Flow Execution
--------------
VoIPBIN's Number resource allows you to associate multiple flows with a single number for handling different types of communications.

**Flow Configuration**

::

    +-----------------------------------------------------------------------+
    |                    Number Flow Configuration                          |
    +-----------------------------------------------------------------------+

    Number: +15551234567
    +-----------------------------------------------------------------------+
    |                                                                       |
    |  call_flow_id: "flow-abc-123"                                        |
    |  +---------------------------+                                        |
    |  | Inbound Call Flow         |                                        |
    |  | 1. Play greeting          |                                        |
    |  | 2. Collect DTMF           |                                        |
    |  | 3. Route to queue         |                                        |
    |  +---------------------------+                                        |
    |                                                                       |
    |  message_flow_id: "flow-xyz-789"                                     |
    |  +---------------------------+                                        |
    |  | Inbound Message Flow      |                                        |
    |  | 1. Parse keywords         |                                        |
    |  | 2. Auto-reply             |                                        |
    |  | 3. Forward to agent       |                                        |
    |  +---------------------------+                                        |
    |                                                                       |
    +-----------------------------------------------------------------------+

.. image:: _static/images/number-flow_execution.png

**Configure Flows**

.. code::

    $ curl -X PUT 'https://api.voipbin.net/v1.0/numbers/<number-id>?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "call_flow_id": "flow-abc-123",
            "message_flow_id": "flow-xyz-789"
        }'


Number Types
------------
VoIPBIN supports various number types for different use cases.

+------------+------------------------------------------------------------------+
| Type       | Description                                                      |
+============+==================================================================+
| local      | Geographic numbers tied to a specific city or region             |
+------------+------------------------------------------------------------------+
| toll-free  | Numbers that are free to call (e.g., 1-800)                      |
+------------+------------------------------------------------------------------+
| mobile     | Mobile phone numbers (where available)                           |
+------------+------------------------------------------------------------------+
| virtual    | Virtual numbers with +899 prefix. No provider purchase required. |
|            | Designed for non-PSTN callers such as AI calls, WebRTC calls,   |
|            | and internal routing.                                            |
+------------+------------------------------------------------------------------+

**Normal vs Virtual Number Routing**

Normal numbers are routed through an external provider (Telnyx/Twilio) from the PSTN, while virtual numbers are routed internally from non-PSTN callers (AI calls, WebRTC, SIP clients) without any provider involvement.

::

    Normal Number (e.g. +15551234567)
    ==================================

      PSTN/Mobile                                                   Your Application
           |                                                                |
           | Outbound call from PSTN                                        |
           | to +15551234567                                                |
           +------------>  Telnyx/Twilio (provider)                         |
                                |                                           |
                                v                                           |
                          +-----------+                                     |
                          | Kamailio  |                                     |
                          +-----+-----+                                     |
                                |                                           |
                                v                                           |
                          +-----------+     +----------+     +-----------+  |
                          | Asterisk  |---->| Number   |---->| Call Flow |--+-->
                          +-----------+     | Lookup   |     | Execution |  |
                                            +----------+     +-----------+  |
                                            type: normal                    |
                                            provider: telnyx                |


      Virtual Number (e.g. +899123456789)
      ====================================

      Non-PSTN Caller                                                Your Application
      (AI, WebRTC, SIP)                                                     |
           |                                                                |
           | Call from non-PSTN caller                                      |
           | to +899123456789                                               |
           +--------------- No Provider needed                              |
                                |                                           |
                                v                                           |
                          +-----------+                                     |
                          | Kamailio  |                                     |
                          +-----+-----+                                     |
                                |                                           |
                                v                                           |
                          +-----------+     +----------+     +-----------+  |
                          | Asterisk  |---->| Number   |---->| Call Flow |--+-->
                          +-----------+     | Lookup   |     | Execution |  |
                                            +----------+     +-----------+  |
                                            type: virtual                   |
                                            provider: none                  |

**Comparison**

+-------------------+-------------------------------+-------------------------------+
| Aspect            | Normal Number                 | Virtual Number                |
+===================+===============================+===============================+
| Format            | E.164 (e.g. +15551234567)     | +899 prefix (+899XXXXXXXXX)   |
+-------------------+-------------------------------+-------------------------------+
| Provider          | Telnyx or Twilio              | None (internal only)          |
+-------------------+-------------------------------+-------------------------------+
| Inbound routing   | PSTN -> Provider -> VoIPBIN   | Non-PSTN caller -> VoIPBIN    |
+-------------------+-------------------------------+-------------------------------+
| Flow execution    | Same (call_flow/message_flow) | Same (call_flow/message_flow) |
+-------------------+-------------------------------+-------------------------------+
| Best for          | Production, external callers  | AI, WebRTC, internal routing  |
+-------------------+-------------------------------+-------------------------------+

For billing and cost details, see :ref:`Billing Account <billing_account_overview>`.

**Virtual Number Tier Limits**

Virtual numbers are free to create but subject to tier-based limits. When the limit is reached, further virtual number creation is denied. See :ref:`Plan Tiers <billing_account_overview>` for tier limits and billing details.


Common Scenarios
----------------

**Scenario 1: Customer Service Hotline**

Set up a number for inbound customer calls.

::

    1. Search for toll-free number
       GET /number_availables?country=US&type=toll_free

    2. Provision the number
       POST /numbers { "number": "+18005551234" }

    3. Create IVR flow
       - Greeting message
       - Menu options (1=Sales, 2=Support, 3=Billing)
       - Route to appropriate queue

    4. Assign flow to number
       PUT /numbers/{id} { "call_flow_id": "..." }

    Result:
    +--------------------------------------------+
    | Caller dials 1-800-555-1234               |
    | -> Hears: "Welcome to Company X..."        |
    | -> Press 1 for Sales                       |
    | -> Connected to sales queue                |
    +--------------------------------------------+

**Scenario 2: SMS Auto-Responder**

Configure automatic SMS responses.

::

    1. Provision local number for SMS
       POST /numbers { "number": "+15551234567" }

    2. Create message flow
       - Check for keywords (HOURS, HELP, STOP)
       - Send appropriate auto-reply
       - Forward unknown messages to agent

    3. Assign message flow
       PUT /numbers/{id} { "message_flow_id": "..." }

    Result:
    +--------------------------------------------+
    | Customer texts "HOURS"                     |
    | -> Auto-reply: "We're open Mon-Fri 9-5"    |
    +--------------------------------------------+

**Scenario 3: Virtual Number for Testing**

Create a virtual number for development and testing without purchasing from a provider.

::

    1. Search for available virtual numbers
       GET /available_numbers?type=virtual&page_size=5

    2. Provision the virtual number
       POST /numbers { "number": "+899100000001" }

    3. Assign a call flow for testing
       PUT /numbers/{id} { "call_flow_id": "..." }

    Result:
    +--------------------------------------------+
    | Virtual number +899100000001 is active    |
    | -> No provider charges                     |
    | -> Ready for development/testing           |
    +--------------------------------------------+

**Scenario 4: Multi-Purpose Number**

Use one number for both calls and messages.

::

    Number: +15551234567
    +--------------------------------------------+
    | call_flow_id: IVR with queue routing       |
    | message_flow_id: SMS keyword handler       |
    +--------------------------------------------+

    Inbound call -> Execute call flow
    Inbound SMS  -> Execute message flow


Best Practices
--------------

**1. Number Selection**

- Choose local numbers for regional presence
- Use toll-free for national customer service
- Consider number memorability for marketing

**2. Flow Configuration**

- Always configure both call and message flows
- Test flows before going live
- Use variables to personalize responses

**3. Number Management**

- Document what each number is used for
- Set up monitoring for call/message volumes
- Review and update flows regularly

**4. Compliance**

- Follow local regulations for number usage
- Include opt-out options for SMS
- Maintain records of number provisioning


Troubleshooting
---------------

**Provisioning Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Number not available      | Try different region or type; number may have  |
|                           | been provisioned by another customer           |
+---------------------------+------------------------------------------------+
| Provisioning failed       | Check account balance; verify number format;   |
|                           | contact support if issue persists              |
+---------------------------+------------------------------------------------+

**Call/Message Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Calls not connecting      | Check flow is assigned; verify flow is valid;  |
|                           | test flow in isolation                         |
+---------------------------+------------------------------------------------+
| Messages not received     | Verify message_flow_id is set; check carrier   |
|                           | routing; review webhook logs                   |
+---------------------------+------------------------------------------------+
| Wrong flow executing      | Verify correct flow ID is assigned to number;  |
|                           | check for flow execution errors                |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Flow Overview <flow-overview>` - Creating call and message flows
- :ref:`Queue Overview <queue-overview>` - Routing calls to agents
- :ref:`Message Overview <message-overview>` - SMS/MMS messaging
- :ref:`Webhook Overview <webhook-overview>` - Event notifications

