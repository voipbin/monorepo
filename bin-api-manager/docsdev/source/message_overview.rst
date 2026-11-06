.. _message-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Chargeable (per message and recipient; a text longer than one segment is an MMS)
   * **Async:** Yes. ``POST https://api.voipbin.net/v1.0/messages`` returns immediately with target status ``queued``. Poll ``GET https://api.voipbin.net/v1.0/messages/{id}`` or use webhooks to track delivery status changes.

VoIPBin's Message API enables you to send and receive SMS (Short Message Service) messages globally. Whether you need to send notifications, alerts, verification codes, or marketing messages, the Message API provides a reliable solution for text-based communication.

With the Message API you can:

- Send SMS messages to phone numbers worldwide
- Receive inbound messages via webhooks
- Track message delivery status
- Integrate messaging into automated workflows


How Messaging Works
-------------------
When you send a message, VoIPBin routes it through carrier networks to reach the recipient's mobile device.

**Message Architecture**

::

    +----------+        +----------------+        +-----------+
    | Your App |--API-->|    VoIPBin     |--SMPP->|  Carrier  |
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
- **Carrier Network**: Delivers SMS messages to recipient devices
- **Webhooks**: Notify your application of delivery status and inbound messages

**Message Types**

.. list-table::
   :header-rows: 1

   * - Type
     - Description
   * - SMS
     - A text that fits one segment: up to 160 GSM-7 characters or up to 70 Unicode characters (see :ref:`Message Formatting <message-overview-formatting>`).
   * - MMS
     - A message with media attachments (images, audio, PDF), sent when ``POST /messages`` has a non-empty ``medias`` list. A text without media that is longer than one segment is also sent as an MMS (see :ref:`Long texts <message-overview-long-text>`).

.. note:: **AI Implementation Hint**

   ``POST /messages`` accepts ``source``, ``destinations``, ``text`` and the optional ``medias`` (a list of file ids). Without ``medias`` (missing, ``null`` or an empty list) a text of one segment or less is an SMS and a longer text is an MMS. With ``medias`` it is an MMS. Read ``type`` in the response.

.. _message-overview-long-text:

Long texts
----------
A text without media is counted in segments from the standard GSM-7 and UCS-2 rules, whatever the carrier reports. A text of one segment or less is sent as an SMS and charged at the SMS price. A text of more than one segment (more than 160 GSM-7 characters, or more than 70 characters when any character is outside GSM-7) is sent as an MMS without media. The response has ``type`` ``mms`` and each destination is charged one MMS message at the MMS price, whatever the number of segments. ``parts`` of each target holds the number of segments of the text.

A text of more than 10 segments is rejected with ``TEXT_TOO_LONG`` and nothing is charged. A promoted text is sent as an MMS where the destination supports it. Where an MMS cannot be sent, the carrier route sends the text as SMS segments, and the charge stays the MMS price. The price is fixed when the message is created, so it does not change with the route.

.. _message-overview-mms:

Sending an MMS
--------------
To send an MMS, upload each file first and then reference the file ids in ``medias``.

1. Upload each file with ``POST /storage_files`` using ``type=media``. Keep the returned ``id``.
2. Send ``POST /messages`` with the ``medias`` list of those ids. The ``text`` may be an empty string for a media-only message.

**Limits**

.. list-table::
   :header-rows: 1

   * - Item
     - Limit
   * - Number of files
     - At most 10 per message.
   * - Total size
     - At most 900 KiB for all files together.
   * - Image files (``jpg``, ``jpeg``, ``png``, ``gif``)
     - At most 900 KiB each.
   * - Audio and document files (``mp3``, ``wav``, ``pdf``)
     - At most 600 KiB each.
   * - Text
     - At most 2000 characters.
   * - Files
     - Must belong to your account and must have been uploaded with ``type=media``.

**Delivery**

An MMS is sent through the carrier network of the destination. Whether an MMS can be delivered depends on the destination number and its carrier. A target that could not be delivered gets the status ``failed``. This includes a target whose result could not be confirmed, which is not sent again through another route so that it is never delivered twice. An MMS is never sent as a text-only message. Each destination is charged as one MMS message when the message is created, so a target that ends as ``failed`` is charged as well. The price is listed in :ref:`Billing Account Overview <billing-account-overview>`.

**Errors**

A request that is rejected with one of the reasons below creates no message, charges nothing and sends nothing, so it can be retried safely after the cause is fixed.

.. list-table::
   :header-rows: 1

   * - Reason
     - Meaning
   * - ``MEDIA_NOT_FOUND``
     - A file id does not exist, is deleted, or belongs to another account.
   * - ``MEDIA_FILE_TYPE_INVALID``
     - A file was not uploaded with ``type=media``.
   * - ``MEDIA_TYPE_UNSUPPORTED``
     - The file extension is not one of the supported types.
   * - ``MEDIA_TOO_LARGE``
     - A file, or all the files together, exceed the size limit.
   * - ``MEDIA_TOO_MANY``, ``MEDIA_DUPLICATE``, ``INVALID_MEDIA_ID``
     - More than 10 files, the same id listed twice, or an id that is not a valid UUID.
   * - ``TEXT_TOO_LONG``
     - The text is longer than 2000 characters with media, or longer than 10 segments without media.
   * - ``INSUFFICIENT_BALANCE`` (HTTP 402)
     - The credit balance is not enough for the message.
   * - ``MEDIA_CHECK_TIMEOUT`` (HTTP 503)
     - The files could not be checked in time. Retry the request.



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
     - Inbound message received by VoIPBin messaging services


**Inbound Message Flow**

::

    Sender Device        Carrier Network           VoIPBin              Your App
         |                     |                      |                    |
         | SMS                 |                      |                    |
         +------------------->|                      |                    |
         |                     | Forward message      |                    |
         |                     +-------------------->|                    |
         |                     |                      |                    |
         |                     |                      | message_created    |
         |                     |                      | webhook            |
         |                     |                      +------------------->|
         |                     |                      |                    |


Sending Messages
----------------
VoIPBin provides multiple ways to send messages based on your use case.

**Method 1: Via API**

Send messages directly using the REST API.

::

    Your App                    VoIPBin                    Recipient
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

   All phone numbers must be in E.164 format: start with ``+``, followed by country code and number, no dashes or spaces. For example, ``+15551234567`` (US) or ``+821012345678`` (Korea). The ``source`` must be a number you own, obtainable via ``GET https://api.voipbin.net/v1.0/numbers``. Unicode characters (emoji, non-Latin scripts) reduce the single-segment limit from 160 to 70 and make a text over 70 characters an MMS.

**Send SMS Example:**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/messages?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "source": {
                "type": "tel",
                "target": "+15551234567"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "+15559876543"
                }
            ],
            "text": "Your verification code is 123456"
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
VoIPBin delivers inbound messages to your application via webhooks.

**Webhook Delivery**

::

    VoIPBin                           Your App
        |                                 |
        | POST /your-webhook-endpoint     |
        | {message_created event}         |
        +-------------------------------->|
        |                                 |
        |            200 OK               |
        |<--------------------------------+
        |                                 |

**Inbound Message Webhook Payload:**

.. code::

    {
        "type": "message_created",
        "data": {
            "id": "msg-abc-123",
            "type": "sms",
            "source": {
                "type": "tel",
                "target": "+15559876543"
            },
            "targets": [
                {
                    "destination": {
                        "type": "tel",
                        "target": "+15551234567"
                    },
                    "status": "received",
                    "parts": 1,
                    "tm_update": "2024-01-15T10:30:00Z"
                }
            ],
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

.. _message-overview-receiving-mms:

**Receiving an MMS**

An inbound message that carries media is delivered with ``type`` ``mms``. VoIPBin downloads each attachment from the carrier network and stores it as a file in your storage (see :ref:`Storage Overview <storage_overview>`), so the ``medias`` field holds file ids that you can read with ``GET /storage_files/{id}``.

The attachments are stored after the message has been delivered to you. The ``message_created`` event of an inbound MMS therefore has an empty ``medias`` field, and the final list arrives in a ``message_updated`` event a few seconds later. The text of the message is always kept.

.. code::

    {
        "type": "message_updated",
        "data": {
            "id": "msg-abc-124",
            "type": "mms",
            "text": "Here is the photo",
            "medias": ["<file-id-1>", "<file-id-2>"],
            "direction": "inbound"
        }
    }

An attachment is skipped, and is then missing from ``medias``, when it is not a supported type, is larger than the limits below, could not be downloaded, or could not be stored. Attachments are also skipped when the service is handling many received messages at the same time, and a skipped attachment is not retried. Check ``medias`` when you need every attachment.

.. list-table::
   :header-rows: 1

   * - Limit
     - Value
   * - Attachments per message
     - 10 (more are ignored)
   * - Size of one attachment
     - 2 MiB
   * - Total size of one message
     - 5 MiB
   * - Supported types
     - ``image/jpeg``, ``image/png``, ``image/gif``, ``audio/mpeg``, ``audio/wav``, ``application/pdf``, ``text/vcard``

The stored files use the file type ``media`` and count towards the storage quota of your account. These limits are larger than the limits for sending (see :ref:`Sending an MMS <message-overview-mms>`), so a received attachment may need to be reduced before it can be sent again. A received MMS is charged like a received SMS, at the MMS price listed in :ref:`Billing Account Overview <billing-account-overview>`, when the message is created.


Message Formatting
------------------
.. _message-overview-formatting:

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


Some characters of GSM-7 (for example ``{``, ``}``, ``[``, ``]``, ``~``, ``|``, ``^``, ``\`` and the euro sign) take two positions. One character outside GSM-7, such as an emoji or a non-Latin letter, makes the whole text Unicode. A character is never split between two segments.

**Long Message Handling**

A text of more than one segment is sent as an MMS, see :ref:`Long texts <message-overview-long-text>`. The ``parts`` field of the target is the number of segments of the text.


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

    Your App                VoIPBin                  Customer
        |                      |                        |
        | "Order shipped!"     |                        |
        +--------------------->+----------------------->|
        |                      |                        |
        |                      |    "When arrives?"     |
        |<---------------------+<-----------------------+
        |  message_created     |                        |
        |                      |                        |
        | "Expected Friday"    |                        |
        +--------------------->+----------------------->|
        |                      |                        |

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
   * - Status stays "queued"
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
