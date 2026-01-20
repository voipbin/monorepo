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
        "balance": 69.77263,
        "payment_type": "",
        "payment_method": "",
        "tm_create": "2013-06-17 00:00:00.000000",
        "tm_update": "2023-06-30 19:18:08.466742",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

The ``balance`` field shows the remaining balance in USD.

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

VoIPBIN uses a fixed-rate pricing model for transparent and predictable costs.

**Current Rates:**

=================== ======================
Service Type        Cost (USD)
=================== ======================
Number buying       $5.00
Calling per second  $0.020
SMS per message     $0.008
=================== ======================

**Calculate Call Cost:**

.. code::

    # Example: 5 minute call
    Duration: 300 seconds
    Rate: $0.020 per second
    Total Cost: 300 × $0.020 = $6.00

**Calculate SMS Cost:**

.. code::

    # Example: 10 messages
    Messages: 10
    Rate: $0.008 per message
    Total Cost: 10 × $0.008 = $0.08

**Calculate Total Monthly Cost:**

.. code::

    Phone Numbers: 3 numbers × $5.00 = $15.00
    Calls: 500 seconds × $0.020 = $10.00
    SMS: 100 messages × $0.008 = $0.80
    Total: $25.80

Check Balance Before Call
--------------------------

Programmatically verify balance before initiating calls to ensure successful completion.

**Python Example:**

.. code::

    import requests

    def check_balance_and_call(billing_account_id, call_duration_seconds):
        # Get billing account
        url = f"https://api.voipbin.net/v1.0/billing_accounts/{billing_account_id}"
        params = {"token": "<YOUR_AUTH_TOKEN>"}

        response = requests.get(url, params=params)
        account = response.json()

        # Check if balance is sufficient
        estimated_cost = call_duration_seconds * 0.020
        current_balance = account['balance']

        if current_balance < estimated_cost:
            print(f"Insufficient balance: ${current_balance:.2f}")
            print(f"Required: ${estimated_cost:.2f}")
            return False

        # Balance is sufficient, proceed with call
        print(f"Balance OK: ${current_balance:.2f}")
        print(f"Estimated cost: ${estimated_cost:.2f}")

        # Create call
        call_data = {
            "source": {
                "type": "tel",
                "target": "+15551234567"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "+15559876543"
                }
            ]
        }

        call_response = requests.post(
            "https://api.voipbin.net/v1.0/calls",
            params=params,
            json=call_data
        )

        return call_response.json()

    # Check balance for 10 minute call (600 seconds)
    result = check_balance_and_call(
        "62918cd8-0cd7-11ee-8571-b738bed3a5c4",
        600
    )

**Node.js Example:**

.. code::

    const axios = require('axios');

    async function checkBalanceAndCall(billingAccountId, callDurationSeconds) {
        try {
            // Get billing account
            const accountResponse = await axios.get(
                `https://api.voipbin.net/v1.0/billing_accounts/${billingAccountId}`,
                { params: { token: '<YOUR_AUTH_TOKEN>' } }
            );

            const account = accountResponse.data;

            // Check if balance is sufficient
            const estimatedCost = callDurationSeconds * 0.020;
            const currentBalance = account.balance;

            if (currentBalance < estimatedCost) {
                console.log(`Insufficient balance: $${currentBalance.toFixed(2)}`);
                console.log(`Required: $${estimatedCost.toFixed(2)}`);
                return null;
            }

            // Balance is sufficient, proceed with call
            console.log(`Balance OK: $${currentBalance.toFixed(2)}`);
            console.log(`Estimated cost: $${estimatedCost.toFixed(2)}`);

            // Create call
            const callResponse = await axios.post(
                'https://api.voipbin.net/v1.0/calls',
                {
                    source: {
                        type: 'tel',
                        target: '+15551234567'
                    },
                    destinations: [
                        {
                            type: 'tel',
                            target: '+15559876543'
                        }
                    ]
                },
                { params: { token: '<YOUR_AUTH_TOKEN>' } }
            );

            return callResponse.data;

        } catch (error) {
            console.error('Error:', error.message);
            return null;
        }
    }

    // Check balance for 10 minute call (600 seconds)
    checkBalanceAndCall('62918cd8-0cd7-11ee-8571-b738bed3a5c4', 600);

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
    import smtplib

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
                # Send alert email
                send_low_balance_alert(account_id, balance)

                # Store alert in database
                store_balance_alert(account_id, balance)

                # Optionally pause campaigns or services
                pause_services_if_needed(account_id, balance)

        return jsonify({'status': 'received'}), 200

    def send_low_balance_alert(account_id, balance):
        subject = f"Low Balance Alert: ${balance:.2f}"
        body = f"""
        Your billing account balance is low.

        Account ID: {account_id}
        Current Balance: ${balance:.2f}
        Threshold: ${LOW_BALANCE_THRESHOLD:.2f}

        Please add funds to continue using VoIPBIN services.
        """
        # Send email implementation
        print(f"Sending low balance alert: {subject}")

    def store_balance_alert(account_id, balance):
        # Store alert in database for tracking
        pass

    def pause_services_if_needed(account_id, balance):
        # Optionally pause campaigns if balance is critically low
        CRITICAL_THRESHOLD = 5.00
        if balance < CRITICAL_THRESHOLD:
            # Pause campaigns or services
            print(f"Pausing services for account {account_id}")

Common Use Cases
----------------

**1. Pre-Call Balance Verification:**

.. code::

    # Verify balance before campaign
    def verify_campaign_balance(billing_account_id, estimated_calls, avg_duration):
        account = get_billing_account(billing_account_id)

        # Calculate estimated cost
        total_seconds = estimated_calls * avg_duration
        estimated_cost = total_seconds * 0.020

        # Add 20% buffer for safety
        required_balance = estimated_cost * 1.20

        if account['balance'] < required_balance:
            return {
                'can_proceed': False,
                'current_balance': account['balance'],
                'required_balance': required_balance,
                'shortfall': required_balance - account['balance']
            }

        return {'can_proceed': True}

**2. Real-Time Balance Tracking:**

.. code::

    # Track balance changes during campaign
    def monitor_campaign_balance(billing_account_id, campaign_id):
        account = get_billing_account(billing_account_id)
        initial_balance = account['balance']

        # Store initial balance
        campaign_data = {
            'campaign_id': campaign_id,
            'initial_balance': initial_balance,
            'start_time': datetime.now()
        }

        # Monitor balance every minute
        while campaign_is_running(campaign_id):
            current_account = get_billing_account(billing_account_id)
            current_balance = current_account['balance']

            # Calculate burn rate
            elapsed_time = (datetime.now() - campaign_data['start_time']).seconds
            spent = initial_balance - current_balance
            burn_rate = spent / (elapsed_time / 60)  # USD per minute

            # Estimate remaining runtime
            remaining_minutes = current_balance / burn_rate if burn_rate > 0 else 0

            print(f"Balance: ${current_balance:.2f}")
            print(f"Burn rate: ${burn_rate:.2f}/min")
            print(f"Est. remaining: {remaining_minutes:.0f} minutes")

            # Check if balance is too low
            if remaining_minutes < 10:
                alert_low_balance(billing_account_id, current_balance)

            time.sleep(60)  # Check every minute

**3. Multi-Service Cost Tracking:**

.. code::

    # Calculate total cost for multiple services
    def calculate_total_cost(phone_numbers, call_seconds, sms_count):
        costs = {
            'phone_numbers': phone_numbers * 5.00,
            'calls': call_seconds * 0.020,
            'sms': sms_count * 0.008
        }

        costs['total'] = sum(costs.values())

        return costs

    # Example usage
    monthly_costs = calculate_total_cost(
        phone_numbers=3,      # 3 phone numbers
        call_seconds=15000,   # 250 minutes of calls
        sms_count=500         # 500 SMS messages
    )

    print(f"Phone Numbers: ${monthly_costs['phone_numbers']:.2f}")
    print(f"Calls: ${monthly_costs['calls']:.2f}")
    print(f"SMS: ${monthly_costs['sms']:.2f}")
    print(f"Total: ${monthly_costs['total']:.2f}")

**4. Automated Balance Top-Up:**

.. code::

    # Automatically add balance when threshold is reached
    def auto_topup_balance(billing_account_id, threshold=50.00, topup_amount=100.00):
        account = get_billing_account(billing_account_id)

        if account['balance'] < threshold:
            # Add balance (requires admin permissions)
            response = requests.post(
                f"https://api.voipbin.net/v1.0/billing_accounts/{billing_account_id}/balance",
                params={'token': '<YOUR_ADMIN_TOKEN>'},
                json={'amount': topup_amount}
            )

            if response.status_code == 200:
                new_account = response.json()
                print(f"Balance topped up: ${new_account['balance']:.2f}")

                # Send notification
                send_topup_notification(billing_account_id, topup_amount)

                return new_account
            else:
                print(f"Top-up failed: {response.text}")
                return None

        return account

**5. Cost Analysis and Reporting:**

.. code::

    # Generate cost report for billing period
    def generate_cost_report(billing_account_id, start_date, end_date):
        # Fetch billing history (if available)
        billings = get_billings_for_period(billing_account_id, start_date, end_date)

        report = {
            'account_id': billing_account_id,
            'period': {
                'start': start_date,
                'end': end_date
            },
            'costs': {
                'calls': 0.00,
                'sms': 0.00,
                'phone_numbers': 0.00
            },
            'total_spent': 0.00
        }

        # Aggregate costs by type
        for billing in billings:
            if billing['type'] == 'call':
                report['costs']['calls'] += billing['amount']
            elif billing['type'] == 'sms':
                report['costs']['sms'] += billing['amount']
            elif billing['type'] == 'number':
                report['costs']['phone_numbers'] += billing['amount']

        report['total_spent'] = sum(report['costs'].values())

        return report

Best Practices
--------------

**1. Balance Verification:**

- Always check balance before initiating high-cost operations
- Add a buffer (10-20%) to estimated costs for safety
- Implement automatic balance checks in your workflow

**2. Monitoring:**

- Set up webhooks for real-time balance updates
- Monitor balance during long-running campaigns
- Track burn rate to predict when balance will run out

**3. Alerts:**

- Configure low balance alerts (e.g., below $20)
- Set up critical balance alerts (e.g., below $5)
- Send notifications via email, SMS, or dashboard

**4. Cost Management:**

- Calculate estimated costs before starting campaigns
- Track actual costs vs. estimated costs
- Generate regular cost reports for analysis

**5. Security:**

- Protect admin tokens used for balance operations
- Implement role-based access for balance management
- Audit balance changes regularly

**6. Automation:**

- Consider automated top-up for uninterrupted service
- Pause services automatically if balance is critically low
- Schedule regular balance checks

Balance Management Workflow
----------------------------

**1. Initial Setup:**

.. code::

    # Check current balance
    GET /v1.0/billing_accounts/<account-id>

    # Set up webhook for balance monitoring
    POST /v1.0/webhooks
    → Configure billing_account.updated events

**2. Before Operations:**

.. code::

    # Calculate estimated cost
    estimate = calculate_cost(operation_params)

    # Verify sufficient balance
    if current_balance < estimate:
        alert_and_prevent_operation()

**3. During Operations:**

.. code::

    # Monitor balance via webhooks
    → Receive balance update events

    # Check burn rate
    if burn_rate_too_high():
        adjust_operations()

**4. After Operations:**

.. code::

    # Review actual costs
    actual_cost = get_operation_cost()

    # Compare with estimate
    variance = actual_cost - estimate

    # Adjust future estimates
    update_cost_model(variance)

Troubleshooting
---------------

**Common Issues:**

**Insufficient balance error:**

- Check current balance: ``GET /v1.0/billing_accounts/<account-id>``
- Verify service rates are up to date
- Add balance if needed (admin only)

**Balance not updating:**

- Check webhook configuration
- Verify webhook endpoint is reachable
- Review webhook logs for errors

**Unexpected costs:**

- Review billing history
- Verify rate calculations
- Check for failed calls that still incur costs

**Permission denied when adding balance:**

- Ensure user has admin permissions
- Verify authentication token is valid
- Check user role in account settings

For more information about billing account management, see :ref:`Billing Account Overview <billing_account_overview>`.
