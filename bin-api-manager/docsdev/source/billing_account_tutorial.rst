.. _billing_account-tutorial:

Tutorial
========

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

Check Token Allowance
----------------------

Each billing account has a monthly token allowance that covers VN calls and SMS messages. The allowance is represented as a **cycle** — a record that tracks your token allocation and consumption for the current month. A new cycle is created automatically on the 1st of each month with a fresh allocation based on your plan tier.

Use the ``/allowance`` endpoint (singular) to get the current active cycle, or ``/allowances`` (plural) to list all cycles including past months.

**Get Current Active Allowance Cycle:**

Returns the single active cycle whose ``cycle_start <= now < cycle_end``. This is the most common call for checking how many tokens remain.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/billing_accounts/<billing-account-id>/allowance?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "a1b2c3d4-1234-5678-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "account_id": "62918cd8-0cd7-11ee-8571-b738bed3a5c4",
        "cycle_start": "2024-01-01T00:00:00Z",
        "cycle_end": "2024-02-01T00:00:00Z",
        "tokens_total": 1000,
        "tokens_used": 350,
        "tm_create": "2024-01-01T00:00:00Z",
        "tm_update": "2024-01-15T10:30:00Z"
    }

**List All Allowance Cycles:**

Returns a paginated list of all cycles (current and past) for the account, ordered by creation time descending. Use ``page_size`` and ``page_token`` query parameters for pagination.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/billing_accounts/<billing-account-id>/allowances?token=<YOUR_AUTH_TOKEN>'

    [
        {
            "id": "a1b2c3d4-1234-5678-abcd-ef1234567890",
            "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
            "account_id": "62918cd8-0cd7-11ee-8571-b738bed3a5c4",
            "cycle_start": "2024-01-01T00:00:00Z",
            "cycle_end": "2024-02-01T00:00:00Z",
            "tokens_total": 1000,
            "tokens_used": 350,
            "tm_create": "2024-01-01T00:00:00Z",
            "tm_update": "2024-01-15T10:30:00Z"
        }
    ]

**Understanding the Response Fields:**

- ``tokens_total``: Total tokens allocated for this cycle. Set by the account's plan tier when the cycle is created. Platform admins can adjust this value.
- ``tokens_used``: Tokens consumed so far this cycle. Incremented each time a VN call or SMS uses tokens.
- **Remaining tokens**: ``tokens_total - tokens_used`` = 650 in this example. When this reaches 0, further VN call and SMS usage overflows to the credit balance.
- ``cycle_start`` / ``cycle_end``: The billing cycle period. Always runs from the 1st of the month to the 1st of the next month. A new cycle with fresh tokens is created when the previous one ends.

**Token Allowances by Plan:**

=================== ======================
Plan Tier           Monthly Tokens
=================== ======================
Free                1,000
Basic               10,000
Professional        100,000
Unlimited           Unlimited (no limit)
=================== ======================

Unused tokens do not carry over. Each cycle starts with the full allocation for the plan tier.

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

**Token Rates:**

=================== ====================== ======================
Service Type        Token Cost             Unit
=================== ====================== ======================
VN Calls            1 token                Per minute (ceiling)
SMS Messages        10 tokens              Per message
=================== ====================== ======================

**Credit Rates (Overflow and Credit-Only):**

========================= ====================== ======================
Service Type              Cost (USD)             Unit
========================= ====================== ======================
VN Calls (overflow)       $0.0045                Per minute (ceiling)
PSTN Outgoing Calls       $0.0060                Per minute (ceiling)
PSTN Incoming Calls       $0.0045                Per minute (ceiling)
SMS (overflow)            $0.008                 Per message
Number Purchase           $5.00                  Per number
Number Renewal            $5.00                  Per number
Extension Calls           Free                   No charge
========================= ====================== ======================

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
    Credit cost: 5 x $0.0045 = $0.0225

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

**Calculate Total Monthly Cost:**

.. code::

    Token-eligible (with 1000 tokens available):
      VN Calls: 200 min x 1 token = 200 tokens consumed
      SMS: 50 messages x 10 tokens = 500 tokens consumed
      Total tokens used: 700 / 1000 (within allowance)
      Credit from overflow: $0.00

    Credit-only:
      PSTN Calls: 100 min x $0.006 = $0.60
      Phone Numbers: 3 x $5.00 = $15.00

    Total credit cost: $15.60

Check Balance Before Call
--------------------------

Programmatically verify balance and token availability before initiating calls to ensure successful completion.

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

        # Get current allowance cycle
        current_allowance = requests.get(
            f"{base_url}/billing_accounts/{billing_account_id}/allowance",
            params=params
        ).json()

        current_balance = account['balance']
        tokens_remaining = 0
        if current_allowance:
            tokens_remaining = current_allowance['tokens_total'] - current_allowance['tokens_used']

        duration = math.ceil(call_duration_minutes)  # ceiling-rounded

        if call_type == "vn":
            # VN call: check tokens first, then credit overflow
            tokens_needed = duration * 1  # 1 token per minute
            if tokens_remaining >= tokens_needed:
                print(f"Covered by tokens: {tokens_needed} tokens")
                print(f"Tokens remaining after call: {tokens_remaining - tokens_needed}")
                can_proceed = True
            else:
                # Partial or full overflow to credit
                overflow_minutes = duration - tokens_remaining
                overflow_cost = overflow_minutes * 0.0045
                print(f"Tokens available: {tokens_remaining}")
                print(f"Overflow to credit: {overflow_minutes} min x $0.0045 = ${overflow_cost:.4f}")
                can_proceed = current_balance >= overflow_cost

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

            // Get current allowance cycle
            const allowanceResponse = await axios.get(
                `${baseUrl}/billing_accounts/${billingAccountId}/allowance`,
                { params }
            );
            const currentAllowance = allowanceResponse.data;

            const currentBalance = account.balance;
            let tokensRemaining = 0;
            if (currentAllowance) {
                tokensRemaining = currentAllowance.tokens_total - currentAllowance.tokens_used;
            }

            const duration = Math.ceil(callDurationMinutes);  // ceiling-rounded

            let canProceed = false;

            if (callType === 'vn') {
                // VN call: check tokens first
                const tokensNeeded = duration * 1;
                if (tokensRemaining >= tokensNeeded) {
                    console.log(`Covered by tokens: ${tokensNeeded} tokens`);
                    canProceed = true;
                } else {
                    const overflowMinutes = duration - tokensRemaining;
                    const overflowCost = overflowMinutes * 0.0045;
                    console.log(`Tokens available: ${tokensRemaining}`);
                    console.log(`Overflow cost: $${overflowCost.toFixed(4)}`);
                    canProceed = currentBalance >= overflowCost;
                }
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

Monitor Token Usage
--------------------

Track token consumption during the billing cycle to plan usage and avoid unexpected overflow charges.

**Python Example:**

.. code::

    import requests

    def monitor_token_usage(billing_account_id):
        base_url = "https://api.voipbin.net/v1.0"
        params = {"token": "<YOUR_AUTH_TOKEN>"}

        # Get current allowance cycle
        current = requests.get(
            f"{base_url}/billing_accounts/{billing_account_id}/allowance",
            params=params
        ).json()

        if not current:
            print("No active allowance cycle found.")
            return

        total = current['tokens_total']
        used = current['tokens_used']
        remaining = total - used
        usage_pct = (used / total * 100) if total > 0 else 0

        print(f"Billing Cycle: {current['cycle_start']} to {current['cycle_end']}")
        print(f"Tokens: {used} / {total} used ({usage_pct:.1f}%)")
        print(f"Remaining: {remaining} tokens")

        # Estimate remaining capacity
        vn_call_minutes = remaining  # 1 token per minute
        sms_messages = remaining // 10  # 10 tokens per message
        print(f"Remaining capacity:")
        print(f"  - VN calls: ~{vn_call_minutes} minutes")
        print(f"  - SMS: ~{sms_messages} messages")

        # Warn if running low
        if usage_pct > 80:
            print("WARNING: Token usage above 80%. Consider upgrading plan tier.")

    monitor_token_usage("62918cd8-0cd7-11ee-8571-b738bed3a5c4")

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

    def estimate_campaign_cost(billing_account_id, vn_calls, vn_avg_minutes,
                               pstn_calls, pstn_avg_minutes, sms_count):
        """Estimate campaign cost considering tokens and credits."""
        import math

        # Get current token availability
        allowances = get_allowances(billing_account_id)
        tokens_remaining = 0
        if allowances:
            current = allowances[0]
            tokens_remaining = current['tokens_total'] - current['tokens_used']

        # VN calls: tokens first, then overflow
        vn_total_minutes = vn_calls * math.ceil(vn_avg_minutes)
        vn_tokens_needed = vn_total_minutes  # 1 token per minute
        vn_tokens_consumed = min(vn_tokens_needed, tokens_remaining)
        vn_overflow_minutes = vn_total_minutes - vn_tokens_consumed
        vn_credit = vn_overflow_minutes * 0.0045
        tokens_remaining -= vn_tokens_consumed

        # SMS: tokens first, then overflow
        sms_tokens_needed = sms_count * 10  # 10 tokens per message
        sms_tokens_consumed = min(sms_tokens_needed, tokens_remaining)
        sms_overflow_count = (sms_tokens_needed - sms_tokens_consumed) // 10
        sms_credit = sms_overflow_count * 0.008
        tokens_remaining -= sms_tokens_consumed

        # PSTN calls: always credit
        pstn_total_minutes = pstn_calls * math.ceil(pstn_avg_minutes)
        pstn_credit = pstn_total_minutes * 0.006

        total_credit = vn_credit + sms_credit + pstn_credit

        return {
            'tokens_consumed': vn_tokens_consumed + sms_tokens_consumed,
            'vn_overflow_credit': vn_credit,
            'sms_overflow_credit': sms_credit,
            'pstn_credit': pstn_credit,
            'total_credit_needed': total_credit
        }

    # Example: mixed campaign
    costs = estimate_campaign_cost(
        "62918cd8-0cd7-11ee-8571-b738bed3a5c4",
        vn_calls=100, vn_avg_minutes=3,
        pstn_calls=50, pstn_avg_minutes=2,
        sms_count=200
    )
    print(f"Tokens consumed: {costs['tokens_consumed']}")
    print(f"VN overflow credit: ${costs['vn_overflow_credit']:.2f}")
    print(f"SMS overflow credit: ${costs['sms_overflow_credit']:.2f}")
    print(f"PSTN credit: ${costs['pstn_credit']:.2f}")
    print(f"Total credit needed: ${costs['total_credit_needed']:.2f}")

**2. Monthly Cost Report:**

.. code::

    def generate_monthly_report(billing_account_id, start_date, end_date):
        """Generate a cost breakdown for the billing period."""
        account = get_billing_account(billing_account_id)
        allowances = get_allowances(billing_account_id)

        report = {
            'account_id': billing_account_id,
            'plan_type': account['plan_type'],
            'period': {'start': start_date, 'end': end_date},
            'token_usage': {},
            'credit_usage': {
                'vn_overflow': 0.00,
                'sms_overflow': 0.00,
                'pstn_calls': 0.00,
                'numbers': 0.00
            },
            'total_credit_spent': 0.00
        }

        if allowances:
            current = allowances[0]
            report['token_usage'] = {
                'total': current['tokens_total'],
                'used': current['tokens_used'],
                'remaining': current['tokens_total'] - current['tokens_used']
            }

        report['total_credit_spent'] = sum(report['credit_usage'].values())
        return report

**3. Plan Tier Comparison:**

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
                overflow_credit = overflow * 0.0045  # approximate

            print(f"{plan_name}: {plan['tokens']} tokens, "
                  f"need {total_tokens}, overflow credit: ${overflow_credit:.2f}")

    recommend_plan(monthly_vn_minutes=500, monthly_sms=100)

Best Practices
--------------

**1. Balance and Token Verification:**

- Always check both credit balance and token allowance before high-cost operations
- For VN calls and SMS: check tokens first; if exhausted, ensure credit balance covers overflow
- For PSTN calls and numbers: check credit balance directly
- Add a buffer (10-20%) to estimated credit costs for safety

**2. Token Management:**

- Monitor token consumption weekly to predict month-end usage
- Upgrade plan tier before tokens are consistently exhausted early
- Track which services consume the most tokens (VN calls vs SMS)
- Remember: unused tokens do not carry over to the next cycle

**3. Monitoring:**

- Set up webhooks for real-time balance updates
- Monitor both credit balance and token usage during campaigns
- Track overflow charges to determine if a plan upgrade is worthwhile

**4. Cost Management:**

- Separate estimates into token-eligible and credit-only services
- Calculate worst-case costs assuming full token overflow
- Generate regular cost reports for analysis

**5. Security:**

- Protect admin tokens used for balance operations
- Implement role-based access for balance management
- Audit balance changes regularly

Balance and Token Management Workflow
---------------------------------------

**1. Initial Setup:**

.. code::

    # Check current balance
    GET /v1.0/billing_accounts/<account-id>

    # Check current token allowance
    GET /v1.0/billing_accounts/<account-id>/allowance

    # Set up webhook for balance monitoring
    POST /v1.0/webhooks
    -> Configure billing_account.updated events

**2. Before Operations:**

.. code::

    # Determine service type
    if service_type in ['vn_call', 'sms']:
        # Check token allowance first
        tokens_remaining = get_tokens_remaining()
        if tokens_remaining > 0:
            proceed()  # tokens will cover it
        else:
            check_credit_balance()  # need credit for overflow

    elif service_type in ['pstn_call', 'number']:
        # Always check credit balance
        check_credit_balance()

**3. During Operations:**

.. code::

    # Monitor via webhooks
    -> Receive balance update events

    # Check token burn rate
    if tokens_depleted_faster_than_expected():
        alert_and_check_credit()

**4. After Operations:**

.. code::

    # Review token usage
    GET /v1.0/billing_accounts/<account-id>/allowance

    # Review credit charges
    actual_credit = initial_balance - current_balance

    # Assess plan adequacy
    if overflow_charges > plan_upgrade_cost:
        consider_plan_upgrade()

Troubleshooting
---------------

**Common Issues:**

**Insufficient balance error:**

- Check credit balance: ``GET /v1.0/billing_accounts/<account-id>``
- Check token allowance: ``GET /v1.0/billing_accounts/<account-id>/allowance``
- VN calls and SMS may still work if tokens are available, even with low credit balance
- PSTN calls and number purchases require credit balance

**Tokens exhausted mid-month:**

- Review token consumption patterns via the allowances endpoint
- Consider upgrading to a higher plan tier for more monthly tokens
- Budget for credit overflow charges until the next billing cycle

**Unexpected credit charges:**

- Check if tokens were exhausted, causing VN calls or SMS to overflow to credits
- Verify call durations are ceiling-rounded to the next whole minute
- Review PSTN call history (always charged to credit)

**Permission denied when adding balance:**

- Ensure user has admin permissions
- Verify authentication token is valid
- Check user role in account settings

For more information about billing account management, see :ref:`Billing Account Overview <billing_account_overview>`.
