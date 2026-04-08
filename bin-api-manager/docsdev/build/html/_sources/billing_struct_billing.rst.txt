.. _billing-struct-billing:

Billing
=======

.. _billing-struct-billing-billing:

Billing
-------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "account_id": "<string>",
        "transaction_type": "<string>",
        "status": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "cost_type": "<string>",
        "usage_duration": "<number>",
        "billable_units": "<number>",
        "rate_token_per_unit": "<number>",
        "rate_credit_per_unit": "<number>",
        "amount_token": "<number>",
        "amount_credit": "<number>",
        "balance_token_snapshot": "<number>",
        "balance_credit_snapshot": "<number>",
        "idempotency_key": "<string>",
        "tm_billing_start": "<string>",
        "tm_billing_end": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The billing record's unique identifier. Returned when listing via ``GET /billings``.
* ``customer_id`` (UUID): The customer who owns this billing record. Obtained from the ``id`` field of ``GET /customers``.
* ``account_id`` (UUID): The billing account charged for this transaction. Obtained from the ``id`` field of ``GET /billing-accounts``.
* ``transaction_type`` (enum string): The type of billing transaction. See :ref:`Transaction Type <billing-struct-billing-transaction-type>`.
* ``status`` (enum string): The billing record's current status. See :ref:`Status <billing-struct-billing-status>`.
* ``reference_type`` (enum string): The type of resource that triggered this billing event. See :ref:`Reference Type <billing-struct-billing-reference-type>`.
* ``reference_id`` (UUID): The ID of the resource that triggered this billing event. For example, a call ID or number ID.
* ``cost_type`` (enum string): The cost classification for this billing event. See :ref:`Cost Type <billing-struct-billing-cost-type>`.
* ``usage_duration`` (integer): The usage duration in seconds (for time-based billing such as calls).
* ``billable_units`` (integer): The number of billable units consumed.
* ``rate_token_per_unit`` (integer): The token rate charged per billing unit.
* ``rate_credit_per_unit`` (integer): The credit rate charged per billing unit.
* ``amount_token`` (integer): The total token amount charged for this billing event.
* ``amount_credit`` (integer): The total credit amount charged for this billing event.
* ``balance_token_snapshot`` (integer): The account's token balance at the time of this billing event.
* ``balance_credit_snapshot`` (integer): The account's credit balance at the time of this billing event.
* ``idempotency_key`` (UUID): A unique key ensuring this billing event is not duplicated.
* ``tm_billing_start`` (string, ISO 8601): Timestamp when the billable activity started.
* ``tm_billing_end`` (string, ISO 8601): Timestamp when the billable activity ended.
* ``tm_create`` (string, ISO 8601): Timestamp when this billing record was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to this billing record.
* ``tm_delete`` (string, ISO 8601): Timestamp when this billing record was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Timestamps set to ``9999-01-01 00:00:00.000000`` indicate the event has not yet occurred. For example, ``tm_delete`` with this value means the billing record has not been deleted.

.. _billing-struct-billing-transaction-type:

Transaction Type
----------------

All possible values for the ``transaction_type`` field:

============ ===========
Type         Description
============ ===========
usage        A charge for resource usage (calls, SMS, numbers, etc.)
top_up       A credit top-up to the billing account
adjustment   A manual balance adjustment
refund       A refund for a previous charge
============ ===========

.. _billing-struct-billing-status:

Status
------

All possible values for the ``status`` field:

============= ===========
Status        Description
============= ===========
progressing   The billing event is in progress (e.g., an active call)
end           The billing event has ended
pending       The billing event is pending processing
finished      The billing event has been fully processed
============= ===========

.. _billing-struct-billing-reference-type:

Reference Type
--------------

All possible values for the ``reference_type`` field:

========================= ===========
Type                      Description
========================= ===========
call                      A PSTN phone call
call_extension            A call to an extension
sms                       An SMS message
email                     An email message
number                    A phone number purchase
number_renew              A phone number renewal
speaking                  A speaking session
recording                 A recording session
credit_free_tier          A free-tier credit allocation
monthly_allowance         A monthly credit allowance
credit_adjustment         A manual credit balance adjustment
token_adjustment          A manual token balance adjustment
paddle_credit_purchase    A credit purchase via Paddle payment
paddle_subscription       A Paddle subscription charge
paddle_refund             A Paddle payment refund
========================= ===========

.. _billing-struct-billing-cost-type:

Cost Type
---------

All possible values for the ``cost_type`` field:

====================== ===========
Type                   Description
====================== ===========
call_pstn_outgoing     Outgoing PSTN call
call_pstn_incoming     Incoming PSTN call
call_vn                Call to a virtual number
call_extension         Call to an extension
call_direct_ext        Direct extension call
sms                    SMS message
email                  Email message
number                 Phone number purchase
number_renew           Phone number renewal
tts                    Text-to-speech usage
recording              Recording storage/processing
====================== ===========

Example
-------

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "account_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "transaction_type": "usage",
        "status": "finished",
        "reference_type": "call",
        "reference_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
        "cost_type": "call_pstn_outgoing",
        "usage_duration": 120,
        "billable_units": 2,
        "rate_token_per_unit": 0,
        "rate_credit_per_unit": 200,
        "amount_token": 0,
        "amount_credit": 400,
        "balance_token_snapshot": 10000,
        "balance_credit_snapshot": 9600,
        "idempotency_key": "d4e5f6a7-b8c9-0123-defa-234567890123",
        "tm_billing_start": "2024-03-01T10:00:00.000000Z",
        "tm_billing_end": "2024-03-01T10:02:00.000000Z",
        "tm_create": "2024-03-01T10:02:01.000000Z",
        "tm_update": "2024-03-01T10:02:01.000000Z",
        "tm_delete": "9999-01-01T00:00:00.000000Z"
    }
