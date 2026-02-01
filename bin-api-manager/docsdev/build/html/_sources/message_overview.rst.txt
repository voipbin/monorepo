.. _message-overview:

Overview
========
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

+--------+------------------------------------------------------------------+
| Type   | Description                                                      |
+========+==================================================================+
| SMS    | Text-only messages up to 160 characters (or 70 for Unicode).    |
|        | Longer messages are split and reassembled by the recipient.     |
+--------+------------------------------------------------------------------+
| MMS    | Multimedia messages supporting images, videos, audio, and text. |
|        | Subject line and multiple media attachments supported.          |
+--------+------------------------------------------------------------------+


Message Lifecycle
-----------------
Every message moves through a predictable set of states from sending to delivery.

**Outbound Message States**

::

    POST /messages
           |
           v
    +------------+
    |  sending   |
    +-----+------+
          |
          v
    +------------+     delivery failed     +------------+
    |   sent     |------------------------>|   failed   |
    +-----+------+                         +------------+
          |
          | carrier accepted
          v
    +------------+     delivery failed     +------------+
    | delivered  |------------------------>|   failed   |
    +------------+                         +------------+

**State Descriptions**

+-------------+------------------------------------------------------------------+
| State       | What's happening                                                 |
+=============+==================================================================+
| sending     | Message is being processed and routed to carrier                 |
+-------------+------------------------------------------------------------------+
| sent        | Message has been sent to carrier network                         |
+-------------+------------------------------------------------------------------+
| delivered   | Carrier confirmed delivery to recipient device                   |
+-------------+------------------------------------------------------------------+
| failed      | Message could not be delivered (invalid number, carrier error)   |
+-------------+------------------------------------------------------------------+

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
       |  status: "sending"        |                           |
       |<--------------------------+                           |
       |                           |                           |
       | Webhook: status update    |   SMS delivered           |
       |<--------------------------+-------------------------->|
       |                           |                           |

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

+-------------------+----------------------------------------------------------------+
| Method            | Best for                                                       |
+===================+================================================================+
| API               | Direct integration, transactional messages, custom logic       |
+-------------------+----------------------------------------------------------------+
| Flow Action       | Automated responses, call-triggered SMS, workflow integration  |
+-------------------+----------------------------------------------------------------+


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

+-------------------+------------------+----------------------------------------+
| Encoding          | Single Message   | Concatenated (per segment)             |
+===================+==================+========================================+
| GSM-7 (standard)  | 160 characters   | 153 characters                         |
+-------------------+------------------+----------------------------------------+
| Unicode (emoji,   | 70 characters    | 67 characters                          |
| non-Latin)        |                  |                                        |
+-------------------+------------------+----------------------------------------+

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

+-------------------+----------------------------------+
| Media Type        | Supported Formats                |
+===================+==================================+
| Images            | JPEG, PNG, GIF                   |
+-------------------+----------------------------------+
| Video             | MP4, 3GP                         |
+-------------------+----------------------------------+
| Audio             | MP3, WAV                         |
+-------------------+----------------------------------+
| Documents         | PDF, vCard                       |
+-------------------+----------------------------------+


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

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Status stays "sending"    | Check carrier connectivity; verify destination |
|                           | number format (+E.164)                         |
+---------------------------+------------------------------------------------+
| Status "failed"           | Check error code; common issues: invalid       |
|                           | number, carrier rejection, insufficient credit |
+---------------------------+------------------------------------------------+
| Delivered but not         | Recipient's phone may be off or out of range;  |
| received                  | carrier may delay delivery                     |
+---------------------------+------------------------------------------------+

**Inbound Messages Not Received**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| No webhook calls          | Verify webhook URL is configured and publicly  |
|                           | accessible                                     |
+---------------------------+------------------------------------------------+
| Webhook returns error     | Ensure endpoint returns 200 OK within 5        |
|                           | seconds                                        |
+---------------------------+------------------------------------------------+
| Missing messages          | Check webhook logs; implement idempotency      |
|                           | using message ID                               |
+---------------------------+------------------------------------------------+

**Character Encoding Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Message truncated         | Check for Unicode characters; they reduce      |
|                           | character limit to 70                          |
+---------------------------+------------------------------------------------+
| Strange characters        | Ensure UTF-8 encoding in API requests          |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Conversation Overview <conversation-overview>` - Unified multi-channel messaging
- :ref:`Email Overview <email-overview>` - Email integration
- :ref:`Flow Actions <flow-struct-action-message_send>` - Message flow actions
- :ref:`Webhook Overview <webhook-overview>` - Webhook configuration
