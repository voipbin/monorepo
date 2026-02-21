.. _quickstart_call:

Call
----
Make an outbound voice call to the extension you created in :ref:`Extension & Softphone Setup <quickstart_extension>`.

Prerequisites
+++++++++++++

* A valid authentication token (String) or accesskey (String). See :ref:`Authentication <quickstart_authentication>`.
* A source phone number in E.164 format (e.g., ``+15551234567``). Must be a number owned by your VoIPBIN account. Obtain available numbers via ``GET /numbers``.
* A registered SIP extension and softphone. See :ref:`Extension & Softphone Setup <quickstart_extension>`.

.. note:: **AI Implementation Hint**

   Phone numbers must be in E.164 format: ``+`` followed by country code and number, no dashes or spaces (e.g., ``+15551234567``, ``+821012345678``). The ``source`` number must be a VoIPBIN-owned number — using an unowned number will result in a ``400 Bad Request``. The destination ``type`` is ``extension`` (not ``tel``), and ``target_name`` (String) is the extension's ``name`` field from the :ref:`Extension & Softphone Setup <quickstart_extension>`.

Make your first call
~~~~~~~~~~~~~~~~~~~~
This example calls your registered extension and plays a text-to-speech greeting. Make sure Linphone is registered and ready to receive the call.

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/calls?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "<your-source-number>"
            },
            "destinations": [
                {
                    "type": "extension",
                    "target_name": "quickstart-phone"
                }
            ],
            "actions": [
                {
                    "type": "talk",
                    "option": {
                        "text": "Hello. This is a VoIPBIN test call. Thank you, bye.",
                        "language": "en-US"
                    }
                }
            ]
        }'

When calling an extension, the response contains a ``groupcalls`` entry (the system creates a group call to ring the extension):

.. code::

    {
        "calls": [],
        "groupcalls": [
            {
                "id": "8dfb3692-9403-49d5-b2b2-031ef7f52d9d",
                "customer_id": "550e8400-e29b-41d4-a716-446655440000",
                "status": "progressing",
                "source": {
                    "type": "tel",
                    "target": "<your-source-number>"
                },
                "destinations": [
                    {
                        "type": "extension",
                        "target_name": "quickstart-phone"
                    }
                ],
                "ring_method": "ring_all",
                "answer_method": "hangup_others",
                "call_ids": [
                    "98146643-1b81-498c-a3e2-9dd480899945"
                ],
                "call_count": 1,
                ...
            }
        ]
    }

Linphone rings — **answer the call** to hear the TTS greeting.

.. note:: **AI Implementation Hint**

   When calling an extension, the response contains a ``groupcalls`` array (not ``calls``). The ``groupcalls[0].id`` (UUID) is the group call ID. The individual call IDs are in ``groupcalls[0].call_ids`` — use these with ``GET /calls/{id}`` to check call status or ``DELETE /calls/{id}`` to hang up. The ``ring_method`` (String) controls how multiple devices are rung (``ring_all`` rings all registered devices simultaneously). For the full call lifecycle and advanced call scenarios, see :ref:`Call overview <call-overview>` and :ref:`Call tutorial <call-tutorial>`.

Troubleshooting
+++++++++++++++

* **400 Bad Request:**
    * **Cause:** The ``source`` number is not owned by your VoIPBIN account, or the phone number is not in E.164 format.
    * **Fix:** Verify your numbers via ``GET /numbers``. Ensure all phone numbers start with ``+`` followed by digits only (e.g., ``+15551234567``).

* **Call created but Linphone does not ring:**
    * **Cause:** Linphone is not registered, or the ``target_name`` does not match the extension ``name``.
    * **Fix:** Verify Linphone shows "Registered" status. Verify the ``target_name`` in the call request matches the extension ``name`` from :ref:`Extension & Softphone Setup <quickstart_extension>` exactly (case-sensitive).

* **Call status immediately shows "hangup":**
    * **Cause:** The destination extension has no registered devices, or the source number has no telephony provider attached.
    * **Fix:** Verify Linphone is registered. Check extension status via ``GET /extensions``.
