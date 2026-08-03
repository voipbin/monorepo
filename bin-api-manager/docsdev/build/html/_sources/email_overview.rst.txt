.. _email-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Chargeable (per email sent)
   * **Async:** Yes. ``POST https://api.voipbin.net/v1.0/emails`` returns immediately with status ``initiated``. Poll ``GET https://api.voipbin.net/v1.0/emails/{id}`` or use webhooks to track delivery status changes.

VoIPBIN's Email API provides a reliable and scalable email delivery service for your applications. Whether you need to send transactional emails, notifications, or marketing communications, the Email API handles delivery while you focus on your content.

With the Email API you can:

- Send plain-text emails to one or more recipients
- Attach existing VoIPBIN resources (e.g. call recordings) to your emails
- Track email delivery status
- Integrate email into automated workflows


How Email Works
---------------
When you send an email, VoIPBIN processes and delivers it through email infrastructure optimized for deliverability.

**Email Architecture**

::

    +----------+        +----------------+        +-------------+
    | Your App |--API-->|    VoIPBIN     |--SMTP->|   Email     |
    +----------+        |   Email Hub    |        |   Provider  |
                        +----------------+        +------+------+
                               |                         |
                               |                         v
                        +------+------+           +-------------+
                        |   Webhook   |           |  Recipient  |
                        |  (status)   |           |   Inbox     |
                        +-------------+           +-------------+

**Key Components**

- **Email Hub**: Processes emails, manages delivery queue, handles retries
- **Email Provider**: Routes emails through established email infrastructure
- **Webhooks**: Notify your application of delivery events


Email Lifecycle
---------------
Every email moves through states from composition to delivery.

**Email States**

::

    POST /emails
          |
          v
    +-------------+
    | initiated   |
    +------+------+
           |
           v
    +-------------+     delivery issue     +-------------+
    | processed   |----------------------->|   bounce    |
    +------+------+                        +-------------+
           |
           | accepted by server            +-------------+
           v                          +--->|   dropped   |
    +-------------+     engagement    |    +-------------+
    | delivered   |---+    events     |
    +------+------+   |               |    +-------------+
           |          +-------------->+--->|  deferred   |
           v                               +-------------+
    +-------------+
    |    open     |
    +------+------+
           |
           v
    +-------------+
    |   click     |
    +-------------+

**State Descriptions**

.. list-table::
   :header-rows: 1

   * - State
     - What's happening
   * - (empty)
     - Email has no status yet
   * - initiated
     - Email has been initiated and submitted for processing
   * - processed
     - Email has been received and is being processed by the provider
   * - delivered
     - Email was accepted by the recipient's mail server
   * - open
     - Recipient opened the email
   * - click
     - Recipient clicked a link in the email
   * - bounce
     - Email bounced (permanent or temporary delivery failure)
   * - dropped
     - Provider dropped the email (invalid recipient, spam report, or blocked IP)
   * - deferred
     - Provider has temporarily deferred delivery; will retry later
   * - unsubscribe
     - Recipient unsubscribed from the email list
   * - spamreport
     - Recipient marked the email as spam



Sending Emails
--------------
Send emails through the VoIPBIN API with full control over content and formatting.

**Send Email via API**

::

    Your App                    VoIPBIN                    Recipient
       |                           |                           |
       | POST /emails              |                           |
       +-------------------------->|                           |
       |                           | Process and send          |
       |                           +-------------------------->|
       |  email_id                 |                           |
       |  status: "initiated"      |                           |
       |<--------------------------+                           |
       |                           |                           |
       | Webhook: delivered        |   Email in inbox          |
       |<--------------------------+-------------------------->|
       |                           |                           |

.. note:: **AI Implementation Hint**

   The sender (``source``) is always VoIPBIN's own platform address (``service@voipbin.net``) -- it is not configurable per request and there is no sender-domain verification step. Sending is gated instead on your customer account's identity verification status; unverified customer accounts cannot send email. The ``destinations`` field accepts an array of :ref:`Address <common-struct-address-address>` objects with ``type`` set to ``email``, not plain email strings. ``content`` is plain text only -- there is no separate HTML body.

**Basic Email Example:**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/emails?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "destinations": [
                {
                    "type": "email",
                    "target": "customer@example.com"
                }
            ],
            "subject": "Your Order Confirmation",
            "content": "Thank you for your order #12345."
        }'

**Email with Attachment:**

Attachments reference an existing VoIPBIN resource by ``reference_type`` and ``reference_id`` (for example a call recording) -- you cannot upload arbitrary file content.

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/emails?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "destinations": [
                {
                    "type": "email",
                    "target": "customer@example.com"
                }
            ],
            "subject": "Your Call Recording",
            "content": "Please find your call recording attached.",
            "attachments": [
                {
                    "reference_type": "recording",
                    "reference_id": "1f25e6c9-6709-44d1-b93e-a5f1c5f80411"
                }
            ]
        }'


Email Components
----------------
Understanding email structure helps you create effective messages.

**Email Object Structure**

::

    +---------------------------------------------------------------+
    |                         Email                                 |
    +---------------------------------------------------------------+
    | Source (fixed): service@voipbin.net (voipbin service)         |
    +---------------------------------------------------------------+
    | Destinations: recipient@example.com, ...                      |
    +---------------------------------------------------------------+
    | Subject: Your Order Confirmation                               |
    +---------------------------------------------------------------+
    | Content: plain text body                                      |
    +---------------------------------------------------------------+
    | Attachments:                                                  |
    |   - reference_type: recording, reference_id: <uuid>           |
    +---------------------------------------------------------------+

**Email Fields**

.. list-table::
   :header-rows: 1

   * - Field
     - Description
   * - source
     - Fixed VoIPBIN platform address (``service@voipbin.net``). Not set by the caller.
   * - destinations
     - List of recipient email addresses (:ref:`Address <common-struct-address-address>` objects, ``type`` ``email``). No ``cc``/``bcc`` support.
   * - subject
     - Email subject line
   * - content
     - Plain text body of the email. There is no separate HTML body.
   * - attachments
     - List of attachments referencing existing VoIPBIN resources. See :ref:`Attachment <email-struct-attachment>`.



Content Best Practices
----------------------
The ``content`` field is sent as plain text (there is no HTML body option), so write copy that reads clearly without formatting.

**Content Guidelines**

.. list-table::
   :header-rows: 1

   * - Do
     - Don't
   * - Keep lines short and scannable
     - Rely on HTML markup or inline styling (it is not rendered)
   * - Spell out links in full (``https://...``)
     - Assume clickable rich-text links
   * - Keep messages concise and actionable
     - Send long, densely formatted content meant for HTML rendering



Common Scenarios
----------------

**Scenario 1: Order Confirmation**

Send transactional emails for e-commerce.

::

    Order placed
         |
         v
    +--------------------------+
    | Generate confirmation    |
    | email with order details |
    +------------+-------------+
                 |
                 v
    POST /emails
    Subject: "Order #12345 Confirmed"
                 |
                 v
    Customer receives confirmation

**Scenario 2: Password Reset**

Send security-related emails.

::

    User requests reset
         |
         v
    +------------------------+
    | Generate reset token   |
    | Create reset link      |
    +------------+-----------+
                 |
                 v
    POST /emails
    Subject: "Reset Your Password"
    Content: Link with token
                 |
                 v
    User receives reset email

**Scenario 3: Call Recording Delivery**

Send a customer their call recording by referencing it as an attachment.

::

    Recording completed
         |
         v
    +--------------------------+
    | Look up recording ID     |
    | via GET /recordings      |
    +------------+-------------+
                 |
                 v
    POST /emails
    attachments: [{"reference_type": "recording", "reference_id": "<uuid>"}]
                 |
                 v
    Customer receives email referencing the recording

**Scenario 4: Marketing Newsletter**

Send bulk marketing emails.

::

    +------------------------------------------+
    | For each subscriber:                     |
    |                                          |
    | POST /emails                             |
    | {                                        |
    |   "destinations": [                      |
    |     {"type": "email", "target": sub}     |
    |   ],                                     |
    |   "subject": "Weekly Newsletter",        |
    |   "content": "..."                       |
    | }                                        |
    +------------------------------------------+
                      |
                      v
    Monitor delivery status via webhooks


Best Practices
--------------

**1. Sender Reputation**

- The sending address is fixed to VoIPBIN's platform address; you cannot set a custom "from" or authenticate your own domain
- Keep your customer account's identity verification current -- sending is rejected for unverified accounts
- Maintain low bounce and complaint rates

**2. Content Quality**

- Remember content is plain text only -- no HTML rendering
- Keep subject lines concise and relevant
- Avoid spam trigger words and excessive punctuation

**3. List Management**

- Honor unsubscribe requests immediately
- Remove bounced addresses from your lists
- Segment your audience for relevant content

**4. Deliverability**

- Monitor delivery metrics and adjust content/timing as needed
- Keep recipient lists clean and remove bounced addresses promptly


Troubleshooting
---------------

**Delivery Issues**

.. list-table::
   :header-rows: 1

   * - Symptom
     - Solution
   * - Email bounced
     - Check recipient address validity; verify mailbox exists
   * - Marked as spam
     - Review content for spam triggers; check sender reputation
   * - Delayed delivery
     - Check sending rate; verify no throttling


**Attachment Issues**

.. list-table::
   :header-rows: 1

   * - Symptom
     - Solution
   * - Attachment missing from email
     - Verify ``reference_type``/``reference_id`` point to an existing, accessible resource (e.g. a recording owned by the same customer)
   * - Request rejected
     - Attachments only support referencing existing VoIPBIN resources; arbitrary file uploads are not supported



Related Documentation
---------------------

- :ref:`Message Overview <message-overview>` - SMS/MMS messaging
- :ref:`Conversation Overview <conversation-overview>` - Unified multi-channel messaging
- :ref:`Webhook Overview <webhook-overview>` - Webhook configuration
