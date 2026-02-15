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

* id: Billing account's id.
* customer_id: Customer's id.
* name: Billing account's name.
* detail: Billing account's detail.
* plan_type: Plan tier of the account. Determines resource creation limits. Available values: ``free``, ``basic``, ``professional``, ``unlimited``.
* balance_credit: Credit balance in micros (int64). 1 USD = 1,000,000 micros. Example: 69772630 = $69.77.
* balance_token: Token balance (int64). Tokens are replenished monthly via top-up.
* payment_type: Payment type. Reserved.
* payment_method: Payment method. Reserved.
* tm_last_topup: Timestamp of the last token top-up.
* tm_next_topup: Timestamp of the next scheduled token top-up.

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

* id: Unique identifier of the ledger entry.
* customer_id: Customer that owns this account.
* account_id: Billing account this entry belongs to.
* transaction_type: Nature of the transaction. Values: ``usage``, ``top_up``, ``adjustment``, ``refund``.
* status: Billing status. Values: ``progressing``, ``end``, ``pending``, ``finished``.
* reference_type: Source of the transaction. Values: ``call``, ``call_extension``, ``sms``, ``number``, ``number_renew``, ``credit_free_tier``, ``monthly_allowance``.
* reference_id: ID of the originating resource (call ID, number ID, etc.).
* cost_type: Classification of the billing cost (e.g. ``call_pstn_outgoing``, ``call_vn``, ``sms``, ``number``).
* usage_duration: Actual usage duration in seconds (for calls).
* billable_units: Number of billable units after ceiling rounding (e.g. 135 seconds becomes 3 minutes).
* rate_token_per_unit: Token rate per billable unit (int64).
* rate_credit_per_unit: Credit rate per billable unit in micros (int64). Example: 6000 = $0.006.
* amount_token: Token delta. Negative for usage, positive for top-up.
* amount_credit: Credit delta in micros. Negative for usage, positive for top-up.
* balance_token_snapshot: Token balance after this transaction.
* balance_credit_snapshot: Credit balance in micros after this transaction.
* idempotency_key: Unique key to prevent duplicate billing.
* tm_billing_start: Start of the billing period.
* tm_billing_end: End of the billing period.
