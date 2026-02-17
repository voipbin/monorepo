.. _email-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Chargeable (per email sent)
   * **Async:** Yes. ``POST /emails`` returns immediately with status ``queued``. Poll ``GET /emails/{id}`` or use webhooks to track delivery status changes.

VoIPBIN's Email API provides a reliable and scalable email delivery service for your applications. Whether you need to send transactional emails, notifications, or marketing communications, the Email API handles delivery while you focus on your content.

With the Email API you can:

- Send emails with HTML and plain text content
- Attach files to your emails
- Track email delivery status
- Manage mailboxes for receiving emails
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
    +------------+
    |  queued    |
    +-----+------+
          |
          v
    +------------+     delivery issue      +------------+
    |  sending   |------------------------>|  bounced   |
    +-----+------+                         +------------+
          |
          | accepted by server
          v
    +------------+     recipient issue     +------------+
    | delivered  |------------------------>|  bounced   |
    +------------+                         +------------+

**State Descriptions**

+-------------+------------------------------------------------------------------+
| State       | What's happening                                                 |
+=============+==================================================================+
| queued      | Email is in the delivery queue, waiting to be sent               |
+-------------+------------------------------------------------------------------+
| sending     | Email is being transmitted to the recipient's mail server        |
+-------------+------------------------------------------------------------------+
| delivered   | Email was accepted by the recipient's mail server                |
+-------------+------------------------------------------------------------------+
| bounced     | Email could not be delivered (invalid address, mailbox full)     |
+-------------+------------------------------------------------------------------+


Sending Emails
--------------
Send emails through the VoIPBIN API with full control over content and formatting.

**Send Email via API**

::

    Your App                    VoIPBIN                    Recipient
       |                           |                           |
       | POST /emails              |                           |
       +-------------------------->|                           |
       |                           | Queue and send            |
       |                           +-------------------------->|
       |  email_id                 |                           |
       |  status: "queued"         |                           |
       |<--------------------------+                           |
       |                           |                           |
       | Webhook: delivered        |   Email in inbox          |
       |<--------------------------+-------------------------->|
       |                           |                           |

.. note:: **AI Implementation Hint**

   The ``source`` email address must be from a domain you have verified with VoIPBIN. Using an unverified domain will cause delivery failures. The ``destinations`` field accepts an array of address objects, not plain email strings.

**Basic Email Example:**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/emails?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "from": {
                "email": "noreply@yourcompany.com",
                "name": "Your Company"
            },
            "to": [
                {
                    "email": "customer@example.com",
                    "name": "John Doe"
                }
            ],
            "subject": "Your Order Confirmation",
            "content": {
                "text": "Thank you for your order #12345.",
                "html": "<h1>Thank you!</h1><p>Your order #12345 has been confirmed.</p>"
            }
        }'

**Email with Attachment:**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/emails?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "from": {
                "email": "billing@yourcompany.com",
                "name": "Billing Department"
            },
            "to": [
                {
                    "email": "customer@example.com"
                }
            ],
            "subject": "Your Invoice",
            "content": {
                "text": "Please find your invoice attached.",
                "html": "<p>Please find your invoice attached.</p>"
            },
            "attachments": [
                {
                    "filename": "invoice-12345.pdf",
                    "content": "<base64-encoded-content>",
                    "type": "application/pdf"
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
    | From: sender@company.com (Sender Name)                        |
    +---------------------------------------------------------------+
    | To: recipient@example.com                                     |
    | CC: copy@example.com                                          |
    | BCC: hidden@example.com                                       |
    +---------------------------------------------------------------+
    | Subject: Your Order Confirmation                              |
    +---------------------------------------------------------------+
    | Content:                                                      |
    |   - Plain text version (for simple clients)                   |
    |   - HTML version (for rich formatting)                        |
    +---------------------------------------------------------------+
    | Attachments:                                                  |
    |   - invoice.pdf                                               |
    |   - receipt.png                                               |
    +---------------------------------------------------------------+

**Email Fields**

+-------------------+------------------------------------------------------------------+
| Field             | Description                                                      |
+===================+==================================================================+
| from              | Sender email address and optional display name                   |
+-------------------+------------------------------------------------------------------+
| to                | List of recipient email addresses                                |
+-------------------+------------------------------------------------------------------+
| cc                | Carbon copy recipients (visible to all)                          |
+-------------------+------------------------------------------------------------------+
| bcc               | Blind carbon copy recipients (hidden from others)                |
+-------------------+------------------------------------------------------------------+
| subject           | Email subject line                                               |
+-------------------+------------------------------------------------------------------+
| content.text      | Plain text version of the email body                             |
+-------------------+------------------------------------------------------------------+
| content.html      | HTML version of the email body                                   |
+-------------------+------------------------------------------------------------------+
| attachments       | List of file attachments                                         |
+-------------------+------------------------------------------------------------------+


Content Best Practices
----------------------
Create emails that render well and avoid spam filters.

**HTML Email Structure**

.. code::

    <!DOCTYPE html>
    <html>
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
    </head>
    <body style="font-family: Arial, sans-serif; margin: 0; padding: 20px;">
        <table width="100%" cellpadding="0" cellspacing="0">
            <tr>
                <td align="center">
                    <h1 style="color: #333;">Welcome!</h1>
                    <p>Your email content here.</p>
                </td>
            </tr>
        </table>
    </body>
    </html>

**Content Guidelines**

+---------------------------+------------------------------------------------+
| Do                        | Don't                                          |
+===========================+================================================+
| Use inline CSS styles    | Use external stylesheets                       |
+---------------------------+------------------------------------------------+
| Use table-based layouts  | Rely on CSS floats or flexbox                  |
+---------------------------+------------------------------------------------+
| Include plain text       | Send HTML-only emails                          |
| version                  |                                                |
+---------------------------+------------------------------------------------+
| Test across email        | Assume all clients render the same             |
| clients                  |                                                |
+---------------------------+------------------------------------------------+


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

**Scenario 3: Invoice Delivery**

Send emails with attachments.

::

    Invoice generated
         |
         v
    +------------------------+
    | Create PDF invoice     |
    | Base64 encode content  |
    +------------+-----------+
                 |
                 v
    POST /emails
    Attachment: invoice.pdf
                 |
                 v
    Customer receives invoice

**Scenario 4: Marketing Newsletter**

Send bulk marketing emails.

::

    +------------------------------------------+
    | For each subscriber:                     |
    |                                          |
    | POST /emails                             |
    | {                                        |
    |   "to": [{"email": subscriber}],         |
    |   "subject": "Weekly Newsletter",        |
    |   "content": {...}                       |
    | }                                        |
    +------------------------------------------+
                      |
                      v
    Monitor delivery status via webhooks


Best Practices
--------------

**1. Sender Reputation**

- Use a consistent "from" address for each email type
- Authenticate your domain (SPF, DKIM, DMARC)
- Maintain low bounce and complaint rates

**2. Content Quality**

- Always include both HTML and plain text versions
- Keep subject lines concise and relevant
- Avoid spam trigger words and excessive punctuation

**3. List Management**

- Honor unsubscribe requests immediately
- Remove bounced addresses from your lists
- Segment your audience for relevant content

**4. Deliverability**

- Warm up new sending domains gradually
- Monitor delivery metrics and adjust as needed
- Use dedicated IP addresses for high-volume sending


Troubleshooting
---------------

**Delivery Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Email bounced             | Check recipient address validity; verify       |
|                           | mailbox exists                                 |
+---------------------------+------------------------------------------------+
| Marked as spam            | Review content for spam triggers; check        |
|                           | sender reputation                              |
+---------------------------+------------------------------------------------+
| Delayed delivery          | Check sending rate; verify no throttling       |
+---------------------------+------------------------------------------------+

**Rendering Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| HTML not displaying       | Use inline CSS; avoid external resources       |
+---------------------------+------------------------------------------------+
| Images not showing        | Use absolute URLs; include alt text            |
+---------------------------+------------------------------------------------+
| Layout broken             | Use table-based layouts; test across clients   |
+---------------------------+------------------------------------------------+

**Attachment Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Attachment blocked        | Avoid executable files; use common formats     |
+---------------------------+------------------------------------------------+
| File too large            | Compress files; use file hosting links         |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Message Overview <message-overview>` - SMS/MMS messaging
- :ref:`Conversation Overview <conversation-overview>` - Unified multi-channel messaging
- :ref:`Webhook Overview <webhook-overview>` - Webhook configuration
