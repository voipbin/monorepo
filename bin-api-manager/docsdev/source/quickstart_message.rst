.. _quickstart_message:

Send an SMS
-----------
Send an outbound SMS message using the VoIPBIN API.

Prerequisites
+++++++++++++

* A valid authentication token (String) or accesskey (String). See :ref:`Authentication <quickstart_authentication>`.
* A source phone number in E.164 format (e.g., ``+15551234567``). Must be a number owned by your VoIPBIN account. Obtain available numbers via ``GET /numbers``.
* A destination phone number in E.164 format (e.g., ``+15559876543``).

.. note:: **AI Implementation Hint**

   Phone numbers must be in E.164 format: ``+`` followed by country code and number, no dashes or spaces (e.g., ``+15551234567``, ``+821012345678``). The ``source`` number must be a VoIPBIN-owned number. Sending messages incurs charges per message segment. Unicode characters (emoji, non-Latin scripts) reduce the per-segment limit from 160 to 70 characters.

Send a message
~~~~~~~~~~~~~~
Send an SMS by providing a source number, destination, and message text:

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/messages?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "<your-source-number>"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "<destination-number>"
                }
            ],
            "text": "Hello from VoIPBIN! This is a test message."
        }'

Response:

.. code::

    {
        "id": "a5d2114a-8e84-48cd-8bb2-c406eeb08cd1",
        "type": "sms",
        "source": {
            "type": "tel",
            "target": "<your-source-number>"
        },
        "targets": [
            {
                "destination": {
                    "type": "tel",
                    "target": "<destination-number>"
                },
                "status": "sending",
                "parts": 1
            }
        ],
        "text": "Hello from VoIPBIN! This is a test message.",
        "direction": "outbound",
        ...
    }

.. note:: **AI Implementation Hint**

   The message ``id`` (UUID) can be used to check delivery status via ``GET /messages/{id}``. The ``status`` starts as ``sending`` and transitions to ``sent`` → ``delivered`` or ``failed``. The ``parts`` field indicates how many SMS segments the message was split into. For the full message lifecycle and advanced scenarios, see :ref:`Message overview <message-overview>` and :ref:`Message tutorial <message-tutorial>`.

Troubleshooting
+++++++++++++++

* **400 Bad Request:**
    * **Cause:** The ``source`` number is not owned by your VoIPBIN account, or phone numbers are not in E.164 format.
    * **Fix:** Verify your numbers via ``GET /numbers``. Ensure all phone numbers start with ``+`` followed by digits only.

* **Message status shows "failed":**
    * **Cause:** The destination number is invalid, unreachable, or the carrier rejected the message.
    * **Fix:** Verify the destination number is a valid mobile number. Check the message details via ``GET /messages/{id}`` for error details.
