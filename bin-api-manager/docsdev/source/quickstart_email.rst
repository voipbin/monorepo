.. _quickstart_email:

Send an Email
-------------
Send an email using the VoIPBIN API.

Prerequisites
+++++++++++++

* A valid authentication token (String) or accesskey (String). See :ref:`Authentication <quickstart_authentication>`.
* A source email address with a verified domain in VoIPBIN (e.g., ``service@yourdomain.com``).
* A destination email address.

.. note:: **AI Implementation Hint**

   The ``source`` email address must use a domain that has been verified with VoIPBIN. Using an unverified domain will cause delivery failures. Both ``source`` and ``destinations`` use the Address format with ``type`` set to ``email``. Sending emails incurs charges per email sent.

Send an email
~~~~~~~~~~~~~
Send an email by providing a source address, destination, subject, and content:

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/emails?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "email",
                "target": "service@yourdomain.com",
                "target_name": "Your Service"
            },
            "destinations": [
                {
                    "type": "email",
                    "target": "recipient@example.com"
                }
            ],
            "subject": "Hello from VoIPBIN",
            "content": "This is a test email sent via the VoIPBIN API."
        }'

Response:

.. code::

    {
        "id": "1f25e6c9-6709-44d1-b93e-a5f1c5f80411",
        "customer_id": "550e8400-e29b-41d4-a716-446655440000",
        "source": {
            "type": "email",
            "target": "service@yourdomain.com",
            "target_name": "Your Service"
        },
        "destinations": [
            {
                "type": "email",
                "target": "recipient@example.com"
            }
        ],
        "status": "initiated",
        "subject": "Hello from VoIPBIN",
        "content": "This is a test email sent via the VoIPBIN API.",
        "attachments": [],
        ...
    }

.. note:: **AI Implementation Hint**

   The email ``id`` (UUID) can be used to check delivery status via ``GET /emails/{id}``. The ``status`` starts as ``initiated`` and transitions to ``processed`` → ``delivered``. The ``target_name`` (String, Optional) on the source address sets the sender display name. For the full email lifecycle and advanced scenarios (attachments, HTML content), see :ref:`Email overview <email-overview>`.

Troubleshooting
+++++++++++++++

* **400 Bad Request:**
    * **Cause:** Missing required fields (``source``, ``destinations``, ``subject``, ``content``), or the source address type is not ``email``.
    * **Fix:** Ensure all required fields are present and ``source.type`` is ``"email"``.

* **Email status shows "bounced":**
    * **Cause:** The destination email address is invalid, the mailbox does not exist, or the mailbox is full.
    * **Fix:** Verify the destination email address. Check the email details via ``GET /emails/{id}`` for bounce details.

* **Email not arriving (status shows "delivered"):**
    * **Cause:** The email may be in the recipient's spam folder, or the sender domain lacks proper DNS records (SPF, DKIM).
    * **Fix:** Ask the recipient to check spam. Verify your sender domain has SPF and DKIM configured correctly.
