:orphan:

.. _quickstart_email:

Send an Email
-------------
Send an email using the VoIPBIN API.

Prerequisites
+++++++++++++

* A valid authentication token (String) or accesskey (String). See :ref:`Authentication <quickstart-authentication>`.
* A verified customer identity. Sending is gated on your customer account's identity verification status, not on a sender domain; unverified accounts cannot send.
* A destination email address.

.. note:: **AI Implementation Hint**

   The sender address is always VoIPBIN's own platform address (``service@voipbin.net``) -- it is fixed and not configurable per request. The request body has no ``source`` field. The ``destinations`` field uses the Address format with ``type`` set to ``email``. The ``attachments`` field is required (send an empty array ``[]`` when there is nothing to attach). ``content`` is plain text only. Sending emails incurs charges per email sent.

Send an email
~~~~~~~~~~~~~
Send an email by providing the destination, subject, content, and attachments:

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/emails?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "destinations": [
                {
                    "type": "email",
                    "target": "recipient@example.com"
                }
            ],
            "subject": "Hello from VoIPBIN",
            "content": "This is a test email sent via the VoIPBIN API.",
            "attachments": []
        }'

To attach a call recording, replace the empty ``attachments`` array above with an entry like the following. The only supported attachment ``reference_type`` is ``recording``; ``reference_id`` is a recording ID from ``GET /recordings``.

.. code::

    "attachments": [
        {
            "reference_type": "recording",
            "reference_id": "e5f6a7b8-c9d0-1234-5678-90abcdef0123"
        }
    ]

Response:

.. code::

    {
        "id": "1f25e6c9-6709-44d1-b93e-a5f1c5f80411",
        "customer_id": "550e8400-e29b-41d4-a716-446655440000",
        "source": {
            "type": "email",
            "target": "service@voipbin.net",
            "target_name": "voipbin service"
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

   The email ``id`` (UUID) can be used to check delivery status via ``GET /emails/{id}``. The ``status`` starts as ``initiated`` and transitions to ``processed`` then ``delivered``. The response's ``source`` is always ``service@voipbin.net``. For the full email lifecycle and advanced scenarios, see :ref:`Email overview <email-overview>`.

Troubleshooting
+++++++++++++++

* **400 Bad Request:**
    * **Cause:** Missing required fields (``destinations``, ``subject``, ``content``, ``attachments``), or a destination address type is not ``email``.
    * **Fix:** Ensure all required fields are present (send ``attachments`` as an empty array ``[]`` if there is nothing to attach) and each ``destinations[].type`` is ``"email"``.

* **Email status stuck at "initiated":**
    * **Cause:** Your customer account's identity is not verified.
    * **Fix:** Verify your customer account's identity before sending.

* **Email status shows "bounce":**
    * **Cause:** The destination email address is invalid, the mailbox does not exist, or the mailbox is full.
    * **Fix:** Verify the destination email address. Check the email details via ``GET /emails/{id}`` for bounce details.

* **Email not arriving (status shows "delivered"):**
    * **Cause:** The email may be in the recipient's spam folder.
    * **Fix:** Ask the recipient to check spam.
