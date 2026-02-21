.. _billing_account-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before managing billing accounts, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* The billing account ID (UUID). Obtain it via ``GET /billing_accounts`` which lists all billing accounts for the authenticated customer.
* (For adding balance) Admin-level permissions on the account.

.. note:: **AI Implementation Hint**

   The ``balance_credit`` field is in micros (int64), not dollars. To convert: divide by 1,000,000 for USD (e.g., ``69772630`` micros = $69.77). The ``balance_token`` field is a plain integer representing remaining tokens. When estimating costs, multiply the per-unit rate in micros by billable units. Call durations are ceiling-rounded to the next whole minute for billing purposes (e.g., a 2 minute 15 second call is billed as 3 minutes).

Check Account Balance
----------------------

Check your billing account balance before initiating calls or sending messages to ensure sufficient funds.

**Get Billing Account:**

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/billing_accounts/<billing-account-id>?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "62918cd8-0cd7-11ee-8571-b738bed3a5c4",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Primary Account",
        "detail": "Main billing account",
        "plan_type": "free",
        "balance_credit": 69772630,
        "balance_token": 70,
        "payment_type": "",
        "payment_method": "",
        "tm_last_topup": "2024-01-01T00:00:00Z",
        "tm_next_topup": "2024-02-01T00:00:00Z",
        "tm_create": "2024-01-01T00:00:00Z",
        "tm_update": "2024-01-15T10:30:00Z"
    }

The ``balance_credit`` field is in micros (69772630 = $69.77). The ``balance_token`` field is the current token count.

**List All Billing Accounts:**

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/billing_accounts?token=<YOUR_AUTH_TOKEN>'

    {
        "result": [
            {
                "id": "62918cd8-0cd7-11ee-8571-b738bed3a5c4",
                "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                "name": "Primary Account",
                "detail": "Main billing account",
                "plan_type": "free",
                "balance_credit": 69772630,
                "balance_token": 70,
                "payment_type": "",
                "payment_method": "",
                "tm_last_topup": "2024-01-01T00:00:00Z",
                "tm_next_topup": "2024-02-01T00:00:00Z",
                "tm_create": "2024-01-01T00:00:00Z",
                "tm_update": "2024-01-15T10:30:00Z"
            }
        ]
    }

Add Balance (Admin Only)
-------------------------

Only users with admin permissions can add balance to accounts. This ensures account security and prevents unauthorized access.

**Add Balance:**

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/billing_accounts/<billing-account-id>/balance?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "amount": 100.00
        }'

    {
        "id": "62918cd8-0cd7-11ee-8571-b738bed3a5c4",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Primary Account",
        "detail": "Main billing account",
        "plan_type": "free",
        "balance_credit": 169772630,
        "balance_token": 70,
        "payment_type": "",
        "payment_method": "",
        "tm_last_topup": "2024-01-01T00:00:00Z",
        "tm_next_topup": "2024-02-01T00:00:00Z",
        "tm_create": "2024-01-01T00:00:00Z",
        "tm_update": "2024-01-15T10:30:00Z"
    }

**Important:** This operation requires admin permissions. Regular users will receive a permission error.

Understanding Service Rates
----------------------------

VoIPBIN uses a hybrid billing model: token-eligible services consume tokens first, then overflow to credits. Credit-only services always charge the credit balance directly. All calls are billed per minute with ceiling rounding.

For the complete rate table, see :ref:`Rate Structure <billing_account_rate_structure>`.

**Calculate VN Call Cost (with tokens):**

.. code::

    # Example: 5 minute VN call with tokens available
    Duration: 5 minutes
    Token cost: 5 x 1 = 5 tokens
    Credit cost: $0.00 (covered by tokens)

**Calculate VN Call Cost (tokens exhausted):**

.. code::

    # Example: 5 minute VN call with no tokens remaining
    Duration: 5 minutes
    Credit cost: 5 x $0.001 = $0.005

**Calculate PSTN Call Cost:**

.. code::

    # Example: 2 minute 30 second PSTN outgoing call
    Duration: 2 min 30 sec -> 3 minutes (ceiling-rounded)
    Credit cost: 3 x $0.0060 = $0.018

**Calculate SMS Cost (with tokens):**

.. code::

    # Example: 10 messages with tokens available
    Token cost: 10 x 10 = 100 tokens
    Credit cost: $0.00 (covered by tokens)

**Calculate SMS Cost (tokens exhausted):**

.. code::

    # Example: 10 messages with no tokens remaining
    Credit cost: 10 x $0.008 = $0.08

Check Balance Before Call
--------------------------

Programmatically verify balance before initiating calls to ensure successful completion.

**Python Example:**

.. code::

    import requests
    import math

    def check_balance_and_call(billing_account_id, call_duration_minutes, call_type="vn"):
        base_url = "https://api.voipbin.net/v1.0"
        params = {"token": "<YOUR_AUTH_TOKEN>"}

        # Get billing account
        account = requests.get(
            f"{base_url}/billing_accounts/{billing_account_id}",
            params=params
        ).json()

        balance_credit = account['balance_credit']  # int64 micros
        balance_token = account['balance_token']     # int64 token count
        duration = math.ceil(call_duration_minutes)  # ceiling-rounded

        if call_type == "vn":
            # VN call: check tokens first, then credit for overflow
            tokens_needed = duration * 1  # 1 token per minute
            if balance_token >= tokens_needed:
                print(f"VN call covered by tokens: {tokens_needed} tokens")
                return True
            # Tokens insufficient — estimate credit overflow
            overflow_minutes = duration - balance_token
            estimated_cost_micros = overflow_minutes * 1000  # 1,000 micros/min
            print(f"VN call overflow: {overflow_minutes} min x 1,000 = {estimated_cost_micros} micros")
            can_proceed = balance_credit >= estimated_cost_micros

        elif call_type == "pstn":
            # PSTN call: always credit
            estimated_cost_micros = duration * 6000  # 6,000 micros/min
            print(f"PSTN call cost: {duration} min x 6,000 = {estimated_cost_micros} micros")
            can_proceed = balance_credit >= estimated_cost_micros

        if not can_proceed:
            print(f"Insufficient credit: {balance_credit} micros (${balance_credit / 1_000_000:.2f})")
            return False

        print(f"Balance OK: {balance_credit} micros (${balance_credit / 1_000_000:.2f})")
        return True

    # Check for a 10 minute VN call
    check_balance_and_call("62918cd8-0cd7-11ee-8571-b738bed3a5c4", 10, "vn")

    # Check for a 5 minute PSTN call
    check_balance_and_call("62918cd8-0cd7-11ee-8571-b738bed3a5c4", 5, "pstn")

**Node.js Example:**

.. code::

    const axios = require('axios');

    async function checkBalanceAndCall(billingAccountId, callDurationMinutes, callType = 'vn') {
        try {
            const baseUrl = 'https://api.voipbin.net/v1.0';
            const params = { token: '<YOUR_AUTH_TOKEN>' };

            // Get billing account
            const accountResponse = await axios.get(
                `${baseUrl}/billing_accounts/${billingAccountId}`,
                { params }
            );
            const account = accountResponse.data;

            const balanceCredit = account.balance_credit;  // int64 micros
            const balanceToken = account.balance_token;     // int64 token count
            const duration = Math.ceil(callDurationMinutes);  // ceiling-rounded

            let canProceed = false;

            if (callType === 'vn') {
                // VN call: check tokens first, then credit for overflow
                const tokensNeeded = duration * 1;  // 1 token per minute
                if (balanceToken >= tokensNeeded) {
                    console.log(`VN call covered by tokens: ${tokensNeeded} tokens`);
                    return true;
                }
                const overflowMinutes = duration - balanceToken;
                const estimatedCostMicros = overflowMinutes * 1000;  // 1,000 micros/min
                console.log(`VN call overflow: ${overflowMinutes} min x 1,000 = ${estimatedCostMicros} micros`);
                canProceed = balanceCredit >= estimatedCostMicros;
            } else if (callType === 'pstn') {
                // PSTN call: always credit
                const estimatedCostMicros = duration * 6000;  // 6,000 micros/min
                console.log(`PSTN call cost: ${duration} min x 6,000 = ${estimatedCostMicros} micros`);
                canProceed = balanceCredit >= estimatedCostMicros;
            }

            if (!canProceed) {
                console.log(`Insufficient credit: ${balanceCredit} micros ($${(balanceCredit / 1000000).toFixed(2)})`);
                return null;
            }

            console.log(`Balance OK: ${balanceCredit} micros ($${(balanceCredit / 1000000).toFixed(2)})`);
            return true;

        } catch (error) {
            console.error('Error:', error.message);
            return null;
        }
    }

    // Check for a 10 minute VN call
    checkBalanceAndCall('62918cd8-0cd7-11ee-8571-b738bed3a5c4', 10, 'vn');

Monitor Balance with Webhooks
------------------------------

Set up webhooks to receive notifications when billing account state changes. You can implement client-side low balance alerts by checking the balance in the webhook payload.

**Create Webhook for Billing Events:**

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/webhooks?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "Balance Monitoring Webhook",
            "uri": "https://your-server.com/webhook/billing",
            "method": "POST",
            "event_types": [
                "account_updated"
            ]
        }'

**Webhook Payload Example:**

.. code::

    POST https://your-server.com/webhook/billing

    {
        "event_type": "account_updated",
        "timestamp": "2024-01-15T10:30:00Z",
        "data": {
            "id": "62918cd8-0cd7-11ee-8571-b738bed3a5c4",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "name": "Primary Account",
            "plan_type": "free",
            "balance_credit": 15500000,
            "balance_token": 20
        }
    }

**Process Webhook with Low Balance Alert:**

.. code::

    # Python Flask example
    from flask import Flask, request, jsonify

    app = Flask(__name__)

    LOW_BALANCE_THRESHOLD_MICROS = 20_000_000  # $20.00 in micros

    @app.route('/webhook/billing', methods=['POST'])
    def billing_webhook():
        payload = request.get_json()
        event_type = payload.get('event_type')

        if event_type == 'account_updated':
            data = payload['data']
            balance_credit = data['balance_credit']  # int64 micros
            balance_token = data['balance_token']     # int64 token count
            account_id = data['id']

            # Check if credit balance is low
            if balance_credit < LOW_BALANCE_THRESHOLD_MICROS:
                send_low_balance_alert(account_id, balance_credit, balance_token)

        return jsonify({'status': 'received'}), 200

    def send_low_balance_alert(account_id, balance_credit, balance_token):
        balance_usd = balance_credit / 1_000_000
        threshold_usd = LOW_BALANCE_THRESHOLD_MICROS / 1_000_000
        subject = f"Low Balance Alert: ${balance_usd:.2f}"
        body = f"""
        Your billing account credit balance is low.

        Account ID: {account_id}
        Current Credit Balance: ${balance_usd:.2f} ({balance_credit} micros)
        Current Token Balance: {balance_token}
        Threshold: ${threshold_usd:.2f}

        Note: Token-eligible services (VN calls, SMS) will continue
        working as long as monthly tokens are available. Credit balance
        is needed for PSTN calls, number purchases, and token overflow.
        """
        print(f"Sending low balance alert: {subject}")

Common Use Cases
----------------

**1. Pre-Campaign Cost Estimation:**

.. code::

    def estimate_campaign_cost(pstn_calls, pstn_avg_minutes, sms_count):
        """Estimate campaign credit cost in micros (1 USD = 1,000,000 micros)."""
        import math

        # PSTN calls: always credit (6,000 micros/min outgoing)
        pstn_total_minutes = pstn_calls * math.ceil(pstn_avg_minutes)
        pstn_credit_micros = pstn_total_minutes * 6000

        # SMS: credit cost when tokens exhausted (8,000 micros/msg)
        sms_credit_micros = sms_count * 8000

        total_micros = pstn_credit_micros + sms_credit_micros

        return {
            'pstn_credit_micros': pstn_credit_micros,
            'sms_credit_micros': sms_credit_micros,
            'total_credit_micros': total_micros
        }

    # Example: mixed campaign
    costs = estimate_campaign_cost(
        pstn_calls=50, pstn_avg_minutes=2,
        sms_count=200
    )
    print(f"PSTN credit: {costs['pstn_credit_micros']} micros (${costs['pstn_credit_micros'] / 1_000_000:.2f})")
    print(f"SMS credit: {costs['sms_credit_micros']} micros (${costs['sms_credit_micros'] / 1_000_000:.2f})")
    print(f"Total credit needed: {costs['total_credit_micros']} micros (${costs['total_credit_micros'] / 1_000_000:.2f})")

**2. Plan Tier Comparison:**

.. code::

    def recommend_plan(monthly_vn_minutes, monthly_sms):
        """Recommend the plan tier with the least overflow credit cost."""
        plans = {
            'free':         {'tokens': 100},
            'basic':        {'tokens': 1000},
            'professional': {'tokens': 10000},
        }

        for plan_name, plan in plans.items():
            vn_tokens = monthly_vn_minutes * 1   # 1 token per minute
            sms_tokens = monthly_sms * 10         # 10 tokens per message
            total_tokens = vn_tokens + sms_tokens

            if total_tokens <= plan['tokens']:
                overflow_micros = 0
            else:
                # When tokens run out, remaining usage overflows to credit.
                # Calculate worst-case: all overflow as VN minutes (1,000 micros/min)
                # and SMS (8,000 micros/msg). Split proportionally.
                overflow_tokens = total_tokens - plan['tokens']
                vn_ratio = vn_tokens / total_tokens if total_tokens > 0 else 0
                overflow_vn_micros = int(overflow_tokens * vn_ratio) * 1000
                overflow_sms_micros = int(overflow_tokens * (1 - vn_ratio) / 10) * 8000
                overflow_micros = overflow_vn_micros + overflow_sms_micros

            print(f"{plan_name}: {plan['tokens']} tokens, "
                  f"need {total_tokens}, overflow: {overflow_micros} micros "
                  f"(${overflow_micros / 1_000_000:.2f})")

    recommend_plan(monthly_vn_minutes=500, monthly_sms=100)

Best Practices
--------------

**1. Balance Verification:**

- Always check credit balance before high-cost operations
- For PSTN calls and number purchases: check credit balance directly
- Add a buffer (10-20%) to estimated credit costs for safety

**2. Monitoring:**

- Set up webhooks for real-time balance updates
- Monitor credit balance during campaigns
- Generate regular cost reports for analysis

**3. Cost Management:**

- Separate estimates into token-eligible and credit-only services
- Calculate worst-case costs assuming full token overflow
- Generate regular cost reports for analysis

**4. Security:**

- Protect admin tokens used for balance operations
- Implement role-based access for balance management
- Audit balance changes regularly

Balance Management Workflow
-----------------------------

**1. Initial Setup:**

.. code::

    # Check current balance
    GET /v1.0/billing_accounts/<account-id>

    # Set up webhook for balance monitoring
    POST /v1.0/webhooks
    -> Configure account_updated events

**2. Before Operations:**

.. code::

    # Determine service type
    if service_type in ['pstn_call', 'number']:
        # Always check credit balance
        check_credit_balance()

**3. During Operations:**

.. code::

    # Monitor via webhooks
    -> Receive balance update events

**4. After Operations:**

.. code::

    # Review credit charges (in micros)
    actual_credit_micros = initial_balance_credit - current_balance_credit

Troubleshooting
---------------

**Common Issues:**

**Insufficient balance error:**

- Check credit balance: ``GET /v1.0/billing_accounts/<account-id>``
- PSTN calls and number purchases require credit balance

**Unexpected credit charges:**

- Verify call durations are ceiling-rounded to the next whole minute
- Review PSTN call history (always charged to credit)

**Permission denied when adding balance:**

- Ensure user has admin permissions
- Verify authentication token is valid
- Check user role in account settings

For more information about billing account management, see :ref:`Billing Account Overview <billing_account_overview>`.
