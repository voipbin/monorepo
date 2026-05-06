.. _customer-struct-customer:

Customer
========

.. _customer-struct-customer-customer:

Customer
--------

.. code::

    {
        "id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "email": "<string>",
        "phone_number": "<string>",
        "address": "<string>",
        "webhook_method": "<string>",
        "webhook_uri": "<string>",
        "billing_account_id": "<string>",
        "default_outgoing_source_number_id": "<string>",
        "metadata": {
            "rtp_debug": <boolean>,
            "outbound_codecs": "<string>"
        },
        "email_verified": <boolean>,
        "status": "<string>",
        "identity_verification_status": "<string>",
        "tm_deletion_scheduled": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The customer's unique identifier. Returned when retrieving the customer via ``GET https://api.voipbin.net/v1.0/customer``.
* ``name`` (String, Optional): The display name of the customer organization.
* ``detail`` (String, Optional): An optional description or notes about the customer account.
* ``email`` (String, Optional): The email address associated with the customer account.
* ``phone_number`` (String, Optional): The phone number associated with the customer account.
* ``address`` (String, Optional): The physical or mailing address of the customer.
* ``webhook_method`` (enum string, Optional): The HTTP method used for webhook notifications. One of: ``POST``, ``GET``, ``PUT``, ``DELETE``.
* ``webhook_uri`` (String, Optional): The URI where webhook event notifications are sent for this customer.
* ``billing_account_id`` (UUID, Optional): The default billing account ID for this customer. Obtained from ``GET https://api.voipbin.net/v1.0/billing_accounts``.
* ``default_outgoing_source_number_id`` (UUID, Optional): The default source phone number used as the caller ID for outgoing PSTN calls when no explicit source number is specified. Obtained from the ``id`` field of ``GET https://api.voipbin.net/v1.0/numbers``. Set to ``00000000-0000-0000-0000-000000000000`` if no default is configured. Updatable via ``PUT https://api.voipbin.net/v1.0/customer/default_outgoing_source_number_id``.
* ``metadata`` (Object): Configuration flags for the customer account. Contains:

  - ``rtp_debug`` (Boolean): When ``true``, RTPEngine captures RTP traffic as PCAP files for this customer's calls. Use this to debug audio quality issues (one-way audio, codec problems, jitter). Default is ``false``. Updatable by CustomerAdmin via ``PUT https://api.voipbin.net/v1.0/customer/metadata``.
  - ``outbound_codecs`` (String, Optional): Comma-separated, preference-ordered list of codec names for outbound PSTN calls (e.g. ``PCMU,PCMA,G729``). When set, the platform restricts the outgoing INVITE's SDP to only these codecs. Empty string (default) means use the server default (pass-through with PCMU/PCMA transcoding). Codec names match standard SDP ``a=rtpmap`` names. The per-call metadata key ``codecs`` overrides this customer-level default when present. Updatable by CustomerAdmin via ``PUT https://api.voipbin.net/v1.0/customer/metadata``.

* ``email_verified`` (Boolean): Whether the customer's email address has been verified. ``true`` if verified, ``false`` otherwise.
* ``identity_verification_status`` (enum string): The customer's identity verification status. Determines access to PSTN number purchases and outbound PSTN calls. One of:

  - ``none``: Not yet verified. Default for new customers.
  - ``pending``: Verification in progress.
  - ``verified``: Identity verified. Customer can purchase PSTN numbers and make outbound PSTN calls.
  - ``rejected``: Verification failed.

* ``status`` (enum string): The current account status. One of:

  - ``initial``: Account created, pending email verification.
  - ``active``: Normal operation, fully verified.
  - ``frozen``: Deletion scheduled, 30-day grace period (or immediate deletion in progress).
  - ``deleted``: Permanently deleted, PII anonymized.
  - ``expired``: Unverified signup expired.

* ``tm_deletion_scheduled`` (String, ISO 8601, nullable): Timestamp when the account is scheduled for permanent deletion. Set when the account transitions to ``frozen`` status. ``null`` if no deletion is scheduled.
* ``tm_create`` (String, ISO 8601): Timestamp when the customer was created.
* ``tm_update`` (String, ISO 8601): Timestamp when the customer was last updated.
* ``tm_delete`` (String, ISO 8601, nullable): Timestamp when the customer was deleted. ``null`` if the customer has not been deleted.

.. note:: **AI Implementation Hint**

   The ``status`` field determines what operations are allowed on the account. Only ``active`` accounts can create resources and make calls. A ``frozen`` account has deletion scheduled but can be recovered by cancelling unregistration before the grace period expires. A ``deleted`` account cannot be recovered -- all PII has been anonymized and all resources cascade-deleted. The ``tm_deletion_scheduled`` field is only set when ``status`` is ``frozen``.

   The ``identity_verification_status`` field gates PSTN operations. Only customers with ``verified`` status can purchase non-virtual phone numbers or make outbound PSTN calls. New customers default to ``none`` and must complete identity verification before using PSTN services. Existing customers at the time of this feature launch were grandfathered to ``verified``.

Example
+++++++

.. code::

    {
        "id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Acme Corporation",
        "detail": "Enterprise customer account",
        "email": "admin@acme-corp.com",
        "phone_number": "+15551234567",
        "address": "123 Main St, San Francisco, CA 94105",
        "webhook_method": "POST",
        "webhook_uri": "https://webhooks.acme-corp.com/voipbin",
        "billing_account_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
        "default_outgoing_source_number_id": "f1e2d3c4-b5a6-7890-fedc-ba9876543210",
        "metadata": {
            "rtp_debug": false,
            "outbound_codecs": "PCMU,PCMA,G729"
        },
        "email_verified": true,
        "status": "active",
        "identity_verification_status": "verified",
        "tm_deletion_scheduled": null,
        "tm_create": "2024-01-15T10:30:00Z",
        "tm_update": "2024-06-20T14:22:35Z",
        "tm_delete": null
    }
