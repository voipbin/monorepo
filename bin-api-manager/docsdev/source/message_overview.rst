.. _message-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Chargeable (per message segment sent)
   * **Async:** Yes. ``POST https://api.voipbin.net/v1.0/messages`` returns immediately with target status ``queued``. Poll ``GET https://api.voipbin.net/v1.0/messages/{id}`` or use webhooks to track delivery status changes.

VoIPBIN's Message API enables you to send and receive SMS (Short Message Service) and MMS (Multimedia Messaging Service) globally. Whether you need to send notifications, alerts, verification codes, or marketing messages, the Message API provides a reliable solution for text-based communication.

With the Message API you can:

- Send SMS messages to phone numbers worldwide
- Send MMS messages with images, videos, and other media
- Receive inbound messages via webhooks
- Track message delivery status
- Integrate messaging into automated workflows


How Messaging Works
-------------------
When you send a message, VoIPBIN routes it through carrier networks to reach the recipient's mobile device.

**Message Architecture**

::

    +----------+        +----------------+        +-----------+
    | Your App |--API-->|    VoIPBIN     |--SMPP->|  Carrier  |
    +----------+        |  Message Hub   |        |  Network  |
                        +----------------+        +-----+-----+
                               |                        |
                               |                        v
                        +------+------+           +-----------+
                        |   Webhook   |           | Recipient |
                        |  (status)   |           |  Device   |
                        +-------------+           +-----------+

**Key Components**

- **Message Hub**: Routes messages to appropriate carriers based on destination
- **Carrier Network**: Delivers messages to recipient devices (SMS/MMS)
- **Webhooks**: Notify your application of delivery status and inbound messages

**Message Types**

.. list-table::
   :header-rows: 1

   * - Type
     - Description
   * - SMS
     - Text-only messages up to 160 characters (or 70 for Unicode). Longer messages are split and reassembled by the recipient.
   * - MMS
     - Multimedia messages supporting images, videos, audio, and text. Subject line and multiple media attachments supported.



Message Lifecycle
-----------------
Every message moves through a predictable set of states from sending to delivery.

**Outbound Message Target States**

::

    POST https://api.voipbin.net/v1.0/messages
           |
           v
    +------------+
    |  queued    |
    +-----+------+
          |
          v
    +------------+     gateway timeout     +-------------+
    |   sent     |------------------------>| gw_timeout  |
    +-----+------+                         +-------------+
          |
          | carrier accepted
          v
    +------------+     delivery failed     +------------+
    | delivered  |------------------------>|   failed   |
    +------------+                         +------------+

**Inbound Message Target States**

::

    Carrier delivers message
           |
           v
    +------------+
    | received   |
    +-----+------+
          |
          v
    +------------+
    | delivered  |
    +------------+

**Target Status Descriptions**

.. list-table::
   :header-rows: 1

   * - Status
     - What's happening
   * - queued
     - Message is queued and submitted to the gateway
   * - sent
     - Gateway confirmed the message has been sent downstream
   * - delivered
     - Message delivered to recipient (outbound) or transmitted to you (inbound)
   * - gw_timeout
     - No delivery receipt received from gateway
   * - dlr_timeout
     - No delivery receipt received from downstream carrier
   * - failed
     - Delivery failure reported by gateway or downstream carrier
   * - received
     - Inbound message received by VoIPBIN messaging services


**Inbound Message Flow**

::

    Sender Device        Carrier Network           VoIPBIN              Your App
         |                     |                      |                    |
         | SMS/MMS             |                      |                    |
         +------------------->|                      |                    |
         |                     | Forward message      |                    |
         |                     +-------------------->|                    |
         |                     |                      |                    |
         |                     |                      | message_received   |
         |                     |                      | webhook            |
         |                     |                      +------------------->|
         |                     |                      |                    |


Sending Messages
----------------
VoIPBIN provides multiple ways to send messages based on your use case.

**Method 1: Via API**

Send messages directly using the REST API.

::

    Your App                    VoIPBIN                    Recipient
       |                           |                           |
       | POST /messages            |                           |
       +-------------------------->|                           |
       |                           | Route to carrier          |
       |                           +-------------------------->|
       |  message_id               |                           |
       |  target status: "queued"  |                           |
       |<--------------------------+                           |
       |                           |                           |
       | Webhook: status update    |   SMS delivered           |
       |<--------------------------+-------------------------->|
       |                           |                           |

.. note:: **AI Implementation Hint**

   All phone numbers must be in E.164 format: start with ``+``, followed by country code and number, no dashes or spaces. For example, ``+15551234567`` (US) or ``+821012345678`` (Korea). The ``source`` must be a number you own, obtainable via ``GET https://api.voipbin.net/v1.0/numbers``. Unicode characters (emoji, non-Latin scripts) reduce the per-segment character limit from 160 to 70.

**Send SMS Example:**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/messages?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "source": "+15551234567",
            "destinations": [
                {
                    "type": "tel",
                    "target": "+15559876543"
                }
            ],
            "text": "Your verification code is 123456"
        }'

**Send MMS Example:**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/messages?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "source": "+15551234567",
            "destinations": [
                {
                    "type": "tel",
                    "target": "+15559876543"
                }
            ],
            "text": "Check out this image!",
            "medias": [
                {
                    "type": "image/jpeg",
                    "url": "https://example.com/image.jpg"
                }
            ]
        }'

**Method 2: Via Flow Action**

Send messages as part of an automated flow.

.. code::

    {
        "type": "message_send",
        "option": {
            "source": "+15551234567",
            "text": "Your appointment is confirmed for tomorrow at 2pm."
        }
    }

**When to Use Each Method**

.. list-table::
   :header-rows: 1

   * - Method
     - Best for
   * - API
     - Direct integration, transactional messages, custom logic
   * - Flow Action
     - Automated responses, call-triggered SMS, workflow integration



Receiving Messages
------------------
VoIPBIN delivers inbound messages to your application via webhooks.

**Webhook Delivery**

::

    VoIPBIN                           Your App
        |                                 |
        | POST /your-webhook-endpoint     |
        | {message_received event}        |
        +-------------------------------->|
        |                                 |
        |            200 OK               |
        |<--------------------------------+
        |                                 |

**Inbound Message Webhook Payload:**

.. code::

    {
        "type": "message_received",
        "data": {
            "id": "msg-abc-123",
            "source": {
                "type": "tel",
                "target": "+15559876543"
            },
            "destination": {
                "type": "tel",
                "target": "+15551234567"
            },
            "text": "Hello, I need help with my order",
            "direction": "inbound",
            "tm_create": "2024-01-15T10:30:00Z"
        }
    }

**Status Update Webhook:**

.. code::

    {
        "type": "message_updated",
        "data": {
            "id": "msg-abc-123",
            "status": "delivered",
            "tm_update": "2024-01-15T10:30:05Z"
        }
    }


Message Formatting
------------------
Understanding message limits and encoding helps optimize delivery.

**SMS Character Limits**

.. list-table::
   :header-rows: 1

   * - Encoding
     - Single Message
     - Concatenated (per segment)
   * - GSM-7 (standard)
     - 160 characters
     - 153 characters
   * - Unicode (emoji, non-Latin)
     - 70 characters
     - 67 characters


**Long Message Handling**

::

    Your message: "This is a longer message that exceeds 160 characters..."
                                        |
                                        v
    +-------------------------------------------------------------------+
    | VoIPBIN automatically splits into segments:                       |
    |                                                                   |
    | Segment 1: "This is a longer message that exceeds 160 char..."   |
    | Segment 2: "...acters and continues here with more content..."   |
    | Segment 3: "...and finally ends here."                           |
    |                                                                   |
    | Recipient's phone reassembles into single message                 |
    +-------------------------------------------------------------------+

**MMS Media Types**

.. list-table::
   :header-rows: 1

   * - Media Type
     - Supported Formats
   * - Images
     - JPEG, PNG, GIF
   * - Video
     - MP4, 3GP
   * - Audio
     - MP3, WAV
   * - Documents
     - PDF, vCard



Common Scenarios
----------------

**Scenario 1: Verification Code**

Send a one-time password for user verification.

::

    User requests login
         |
         v
    +--------------------+
    | Generate OTP: 1234 |
    +--------+-----------+
             |
             v
    POST /messages
    "Your code is 1234"
             |
             v
    User receives SMS
    Enters code to verify

**Scenario 2: Appointment Reminder**

Send automated reminders before appointments.

::

    24 hours before appointment
              |
              v
    +--------------------------+
    | Trigger: scheduled job   |
    +------------+-------------+
                 |
                 v
    POST /messages
    "Reminder: Appointment tomorrow at 2pm"
                 |
                 v
    Customer receives SMS

**Scenario 3: Two-Way Conversation**

Enable customers to reply to messages.

::

    Your App                VoIPBIN                  Customer
        |                      |                        |
        | "Order shipped!"     |                        |
        +--------------------->+----------------------->|
        |                      |                        |
        |                      |    "When arrives?"     |
        |<---------------------+<-----------------------+
        |  message_received    |                        |
        |                      |                        |
        | "Expected Friday"    |                        |
        +--------------------->+----------------------->|
        |                      |                        |

**Scenario 4: MMS Marketing**

Send promotional messages with images.

::

    +------------------------------------------+
    | POST /messages                           |
    | {                                        |
    |   "text": "Summer sale! 50% off!",       |
    |   "medias": [                            |
    |     {"url": "https://.../promo.jpg"}     |
    |   ]                                      |
    | }                                        |
    +------------------------------------------+
                      |
                      v
    Customer receives image + text message


Best Practices
--------------

**1. Sender ID Selection**

- Use a phone number your customers recognize
- For transactional messages, use a consistent sender
- Check country-specific sender ID regulations

**2. Message Content**

- Keep messages concise and actionable
- Include opt-out instructions for marketing messages
- Avoid URL shorteners that may trigger spam filters

**3. Rate Limiting**

- Respect carrier rate limits to avoid throttling
- Spread bulk messages over time
- Monitor for carrier feedback signals

**4. Compliance**

- Obtain consent before sending marketing messages
- Honor opt-out requests promptly
- Follow local regulations (TCPA, GDPR, etc.)


Troubleshooting
---------------

**Message Not Delivered**

.. list-table::
   :header-rows: 1

   * - Symptom
     - Solution
   * - Status stays "sending"
     - Check carrier connectivity; verify destination number format (+E.164)
   * - Status "failed"
     - Check error code; common issues: invalid number, carrier rejection, insufficient credit
   * - Delivered but not received
     - Recipient's phone may be off or out of range; carrier may delay delivery


**Inbound Messages Not Received**

.. list-table::
   :header-rows: 1

   * - Symptom
     - Solution
   * - No webhook calls
     - Verify webhook URL is configured and publicly accessible
   * - Webhook returns error
     - Ensure endpoint returns 200 OK within 5 seconds
   * - Missing messages
     - Check webhook logs; implement idempotency using message ID


**Character Encoding Issues**

.. list-table::
   :header-rows: 1

   * - Symptom
     - Solution
   * - Message truncated
     - Check for Unicode characters; they reduce character limit to 70
   * - Strange characters
     - Ensure UTF-8 encoding in API requests



Related Documentation
---------------------

- :ref:`Conversation Overview <conversation-overview>` - Unified multi-channel messaging
- :ref:`Email Overview <email-overview>` - Email integration
- :ref:`Flow Actions <flow-struct-action-message_send>` - Message flow actions
- :ref:`Webhook Overview <webhook-overview>` - Webhook configuration
