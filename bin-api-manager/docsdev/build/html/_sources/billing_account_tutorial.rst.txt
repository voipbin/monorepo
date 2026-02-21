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

   The ``balance`` field in billing account responses is a float representing USD (e.g., ``69.77263`` = $69.77). When estimating costs, multiply the per-unit rate by billable units. Call durations are ceiling-rounded to the next whole minute for billing purposes (e.g., a 2 minute 15 second call is billed as 3 minutes).

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
        "balance": 69.77263,
        "payment_type": "",
        "payment_method": "",
        "tm_create": "2013-06-17 00:00:00.000000",
        "tm_update": "2023-06-30 19:18:08.466742",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

The ``balance`` field shows the remaining credit balance in USD.

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
                "balance": 69.77263,
                "payment_type": "",
                "payment_method": "",
                "tm_create": "2013-06-17 00:00:00.000000",
                "tm_update": "2023-06-30 19:18:08.466742",
                "tm_delete": "9999-01-01 00:00:00.000000"
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
        "balance": 169.77263,
        "payment_type": "",
        "payment_method": "",
        "tm_create": "2013-06-17 00:00:00.000000",
        "tm_update": "2023-06-30 19:20:15.123456",
        "tm_delete": "9999-01-01 00:00:00.000000"
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

        current_balance = account['balance']
        duration = math.ceil(call_duration_minutes)  # ceiling-rounded

        if call_type == "vn":
            # VN call: estimate credit cost for overflow scenario
            estimated_cost = duration * 0.001
            print(f"VN call cost (if tokens exhausted): {duration} min x $0.001 = ${estimated_cost:.4f}")
            can_proceed = current_balance >= estimated_cost

        elif call_type == "pstn":
            # PSTN call: always credit
            estimated_cost = duration * 0.006
            print(f"PSTN call cost: {duration} min x $0.006 = ${estimated_cost:.4f}")
            can_proceed = current_balance >= estimated_cost

        if not can_proceed:
            print(f"Insufficient balance: ${current_balance:.2f}")
            return False

        print(f"Balance OK: ${current_balance:.2f}")
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

            const currentBalance = account.balance;
            const duration = Math.ceil(callDurationMinutes);  // ceiling-rounded

            let canProceed = false;

            if (callType === 'vn') {
                // VN call: estimate credit cost for overflow scenario
                const estimatedCost = duration * 0.001;
                console.log(`VN call cost (if tokens exhausted): $${estimatedCost.toFixed(4)}`);
                canProceed = currentBalance >= estimatedCost;
            } else if (callType === 'pstn') {
                // PSTN call: always credit
                const estimatedCost = duration * 0.006;
                console.log(`PSTN call cost: $${estimatedCost.toFixed(4)}`);
                canProceed = currentBalance >= estimatedCost;
            }

            if (!canProceed) {
                console.log(`Insufficient balance: $${currentBalance.toFixed(2)}`);
                return null;
            }

            console.log(`Balance OK: $${currentBalance.toFixed(2)}`);
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

Set up webhooks to receive notifications when balance changes or falls below a threshold.

**Create Webhook for Billing Events:**

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/webhooks?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "Balance Monitoring Webhook",
            "uri": "https://your-server.com/webhook/billing",
            "method": "POST",
            "event_types": [
                "billing_account.updated",
                "billing_account.low_balance"
            ]
        }'

**Webhook Payload Example:**

.. code::

    POST https://your-server.com/webhook/billing

    {
        "event_type": "billing_account.updated",
        "timestamp": "2023-06-30T19:20:15.000000Z",
        "data": {
            "id": "62918cd8-0cd7-11ee-8571-b738bed3a5c4",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "name": "Primary Account",
            "balance": 15.50,
            "previous_balance": 25.50,
            "change": -10.00
        }
    }

**Process Webhook with Low Balance Alert:**

.. code::

    # Python Flask example
    from flask import Flask, request, jsonify

    app = Flask(__name__)

    LOW_BALANCE_THRESHOLD = 20.00

    @app.route('/webhook/billing', methods=['POST'])
    def billing_webhook():
        payload = request.get_json()
        event_type = payload.get('event_type')

        if event_type == 'billing_account.updated':
            data = payload['data']
            balance = data['balance']
            account_id = data['id']

            # Check if balance is low
            if balance < LOW_BALANCE_THRESHOLD:
                send_low_balance_alert(account_id, balance)

        return jsonify({'status': 'received'}), 200

    def send_low_balance_alert(account_id, balance):
        subject = f"Low Balance Alert: ${balance:.2f}"
        body = f"""
        Your billing account credit balance is low.

        Account ID: {account_id}
        Current Balance: ${balance:.2f}
        Threshold: ${LOW_BALANCE_THRESHOLD:.2f}

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
        """Estimate campaign cost considering credits."""
        import math

        # PSTN calls: always credit
        pstn_total_minutes = pstn_calls * math.ceil(pstn_avg_minutes)
        pstn_credit = pstn_total_minutes * 0.006

        # SMS: credit cost when tokens exhausted
        sms_credit = sms_count * 0.008

        total_credit = pstn_credit + sms_credit

        return {
            'pstn_credit': pstn_credit,
            'sms_credit': sms_credit,
            'total_credit_needed': total_credit
        }

    # Example: mixed campaign
    costs = estimate_campaign_cost(
        pstn_calls=50, pstn_avg_minutes=2,
        sms_count=200
    )
    print(f"PSTN credit: ${costs['pstn_credit']:.2f}")
    print(f"SMS credit: ${costs['sms_credit']:.2f}")
    print(f"Total credit needed: ${costs['total_credit_needed']:.2f}")

**2. Plan Tier Comparison:**

.. code::

    def recommend_plan(monthly_vn_minutes, monthly_sms):
        """Recommend the most cost-effective plan tier."""
        plans = {
            'free':         {'tokens': 1000,   'cost': 0},
            'basic':        {'tokens': 10000,  'cost': 0},   # plan cost TBD
            'professional': {'tokens': 100000, 'cost': 0},   # plan cost TBD
        }

        for plan_name, plan in plans.items():
            vn_tokens = monthly_vn_minutes * 1
            sms_tokens = monthly_sms * 10
            total_tokens = vn_tokens + sms_tokens

            if total_tokens <= plan['tokens']:
                overflow_credit = 0.00
            else:
                overflow = total_tokens - plan['tokens']
                # Simplified: assume overflow split proportionally
                overflow_credit = overflow * 0.001  # approximate

            print(f"{plan_name}: {plan['tokens']} tokens, "
                  f"need {total_tokens}, overflow credit: ${overflow_credit:.2f}")

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
    -> Configure billing_account.updated events

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

    # Review credit charges
    actual_credit = initial_balance - current_balance

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
