.. _billing_account-struct:

Struct
======

.. _billing_account-struct-billing_account:

Billing account
---------------

.. code::

    {
        "id": "62918cd8-0cd7-11ee-8571-b738bed3a5c4",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "",
        "detail": "",
        "plan_type": "free",
        "balance_credit": 69772630,
        "balance_token": 650,
        "payment_type": "",
        "payment_method": "",
        "tm_last_topup": "2024-01-01T00:00:00Z",
        "tm_next_topup": "2024-02-01T00:00:00Z",
        "tm_create": "2013-06-17 00:00:00.000000",
        "tm_update": "2023-06-30 19:18:08.466742",
        "tm_delete": null
    }

* ``id`` (UUID): The billing account's unique identifier. Returned when listing billing accounts via ``GET /billing_accounts`` or retrieving a specific account via ``GET /billing_accounts/{id}``.
* ``customer_id`` (UUID): The customer that owns this billing account. Obtained from the ``id`` field of ``GET /customers``.
* ``name`` (String): The billing account's display name. Optional.
* ``detail`` (String): An optional description of the billing account.
* ``plan_type`` (enum string): The plan tier of the account. Determines resource creation limits and monthly token allocations. Values: ``free``, ``basic``, ``professional``, ``unlimited``.
* ``balance_credit`` (Integer, int64 micros): Credit balance in micros. 1 USD = 1,000,000 micros. Example: ``69772630`` = $69.77. Used for PSTN calls, number purchases, and token overflow charges.
* ``balance_token`` (Integer, int64): Current token balance. Tokens are consumed by VN calls (1 token/minute) and SMS (10 tokens/message). Replenished monthly via automated top-up.
* ``payment_type`` (String): Payment type. Reserved for future use.
* ``payment_method`` (String): Payment method. Reserved for future use.
* ``tm_last_topup`` (string, ISO 8601): Timestamp of the last token top-up.
* ``tm_next_topup`` (string, ISO 8601): Timestamp of the next scheduled token top-up.
* ``tm_create`` (string, ISO 8601): Timestamp when the billing account was created.
* ``tm_update`` (string, ISO 8601): Timestamp when the billing account was last updated.
* ``tm_delete`` (string, ISO 8601 or null): Timestamp when the billing account was deleted. ``null`` indicates the account is active.

.. note:: **AI Implementation Hint**

   Unlike other VoIPBIN resources that use ``9999-01-01 00:00:00.000000`` as the sentinel for "not deleted," the billing account's ``tm_delete`` field uses ``null`` to indicate an active account. Always check for ``null`` rather than the sentinel timestamp when determining if a billing account is active. The ``balance_credit`` field is in micros (int64) -- divide by 1,000,000 to get USD.

.. _billing_account-struct-billing:

Billing (Ledger Entry)
----------------------

Each billing record is an immutable ledger entry recording a single transaction. The ledger tracks both the delta (change in balance) and a post-transaction snapshot.

.. code::

    {
        "id": "69cacd9e-f542-11ee-ab6d-afb3c2c93e56",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "account_id": "62918cd8-0cd7-11ee-8571-b738bed3a5c4",
        "transaction_type": "usage",
        "status": "end",
        "reference_type": "call",
        "reference_id": "a1b2c3d4-5678-abcd-ef12-345678901234",
        "cost_type": "call_pstn_outgoing",
        "usage_duration": 135,
        "billable_units": 3,
        "rate_token_per_unit": 0,
        "rate_credit_per_unit": 6000,
        "amount_token": 0,
        "amount_credit": -18000,
        "balance_token_snapshot": 650,
        "balance_credit_snapshot": 69754630,
        "idempotency_key": "b2c3d4e5-6789-bcde-f123-456789012345",
        "tm_billing_start": "2024-01-15T10:30:00Z",
        "tm_billing_end": "2024-01-15T10:32:15Z",
        "tm_create": "2024-01-15T10:32:15Z",
        "tm_update": "2024-01-15T10:32:15Z",
        "tm_delete": null
    }

* ``id`` (UUID): The unique identifier of this ledger entry. Returned when listing billings via ``GET /billings``.
* ``customer_id`` (UUID): The customer that owns this billing account. Obtained from the ``id`` field of ``GET /customers``.
* ``account_id`` (UUID): The billing account this entry belongs to. Obtained from the ``id`` field of ``GET /billing_accounts``.
* ``transaction_type`` (enum string): The nature of the transaction. Values: ``usage`` (service consumption), ``top_up`` (token replenishment), ``adjustment`` (manual correction), ``refund`` (credit return).
* ``status`` (enum string): The billing entry status. Values: ``progressing`` (in progress), ``end`` (completed), ``pending`` (awaiting processing), ``finished`` (finalized).
* ``reference_type`` (enum string): The source of the transaction. Values: ``call``, ``call_extension``, ``sms``, ``number``, ``number_renew``, ``credit_free_tier``, ``monthly_allowance``.
* ``reference_id`` (UUID): The ID of the originating resource (e.g., call ID, number ID). Obtained from the ``id`` field of the corresponding resource endpoint (e.g., ``GET /calls/{id}``).
* ``cost_type`` (enum string): Classification of the billing cost. Values include: ``call_pstn_outgoing``, ``call_pstn_incoming``, ``call_vn``, ``call_extension``, ``call_direct_ext``, ``sms``, ``number``, ``number_renew``.
* ``usage_duration`` (Integer): Actual usage duration in seconds (for calls). Not applicable for non-call services.
* ``billable_units`` (Integer): Number of billable units after ceiling rounding (e.g., 135 seconds becomes 3 minutes for call billing).
* ``rate_token_per_unit`` (Integer, int64): Token rate per billable unit. Set to ``0`` for credit-only services.
* ``rate_credit_per_unit`` (Integer, int64 micros): Credit rate per billable unit in micros. Example: ``6000`` = $0.006.
* ``amount_token`` (Integer, int64): Token delta for this transaction. Negative for usage, positive for top-up.
* ``amount_credit`` (Integer, int64 micros): Credit delta in micros for this transaction. Negative for usage, positive for top-up/refund.
* ``balance_token_snapshot`` (Integer, int64): The token balance immediately after this transaction was applied.
* ``balance_credit_snapshot`` (Integer, int64 micros): The credit balance in micros immediately after this transaction was applied.
* ``idempotency_key`` (UUID): Unique key to prevent duplicate billing entries for the same event.
* ``tm_billing_start`` (string, ISO 8601): Timestamp marking the start of the billing period for this transaction.
* ``tm_billing_end`` (string, ISO 8601): Timestamp marking the end of the billing period for this transaction.
* ``tm_create`` (string, ISO 8601): Timestamp when this ledger entry was created.
* ``tm_update`` (string, ISO 8601): Timestamp when this ledger entry was last updated.
* ``tm_delete`` (string, ISO 8601 or null): Timestamp when this ledger entry was deleted. ``null`` indicates the entry is active.

.. note:: **AI Implementation Hint**

   All monetary fields (``rate_credit_per_unit``, ``amount_credit``, ``balance_credit_snapshot``) are in micros (int64). Divide by 1,000,000 to convert to USD. The ``amount_token`` and ``amount_credit`` fields are signed: negative values represent charges/consumption, positive values represent top-ups or refunds. Call durations are ceiling-rounded to the next whole minute for billing (e.g., 135 seconds = 3 billable minutes).
