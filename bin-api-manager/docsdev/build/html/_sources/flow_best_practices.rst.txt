.. _flow-best-practices:

Flow Best Practices
===================

This section covers design patterns, optimization techniques, and best practices for building maintainable and efficient flows.

.. note:: **AI Implementation Hint**

   When generating flows programmatically, follow three key rules: (1) Always set ``default_target_id`` on every ``branch`` action to handle unexpected input. (2) Always set ``loop_count`` on every ``goto`` action to prevent infinite loops. (3) Only set explicit ``id`` fields on actions that are referenced as targets by ``goto``, ``branch``, or ``condition_*`` actions.

Flow Design Principles
----------------------

Keep Flows Focused
++++++++++++++++++

.. code::

    Principle: Single Responsibility

    +------------------------------------------------------------------+
    |                         BAD: Monolithic Flow                     |
    +------------------------------------------------------------------+
    | One giant flow that handles:                                     |
    | - Language selection                                             |
    | - Main menu                                                      |
    | - Sales sub-menu                                                 |
    | - Support sub-menu                                               |
    | - Billing sub-menu                                               |
    | - Queue logic                                                    |
    | - Voicemail                                                      |
    | - Survey                                                         |
    |                                                                  |
    | Problems:                                                        |
    | - Hard to maintain                                               |
    | - Hard to test                                                   |
    | - Hard to reuse components                                       |
    | - Changes affect everything                                      |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    |                        GOOD: Modular Flows                       |
    +------------------------------------------------------------------+
    |                                                                  |
    | Main Router Flow                                                 |
    |   └── fetch_flow: Language Selection                             |
    |         └── fetch_flow: Main Menu                                |
    |               ├── fetch_flow: Sales Menu                         |
    |               ├── fetch_flow: Support Menu                       |
    |               └── fetch_flow: Billing Menu                       |
    |                                                                  |
    | Shared Flows:                                                    |
    |   - Queue Wait Flow (reused by all menus)                        |
    |   - Voicemail Flow (reused for after-hours)                      |
    |   - Survey Flow (attached via on_complete_flow_id)               |
    |                                                                  |
    | Benefits:                                                        |
    | - Each flow has one job                                          |
    | - Easy to test individually                                      |
    | - Reusable components                                            |
    | - Changes are localized                                          |
    +------------------------------------------------------------------+


Use Meaningful Action IDs
+++++++++++++++++++++++++

.. code::

    Action ID Best Practices:

    +------------------------------------------------------------------+
    |                            BAD                                   |
    +------------------------------------------------------------------+
    | {                                                                |
    |   "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",                  |
    |   "type": "talk",                                                |
    |   ...                                                            |
    | }                                                                |
    |                                                                  |
    | Debugging nightmare:                                             |
    | "Flow stopped at action a1b2c3d4-e5f6-7890-abcd-ef1234567890"    |
    | What does this action do?                                        |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    |                           GOOD                                   |
    +------------------------------------------------------------------+
    | {                                                                |
    |   "id": "welcome-greeting",                                      |
    |   "type": "talk",                                                |
    |   ...                                                            |
    | }                                                                |
    |                                                                  |
    | {                                                                |
    |   "id": "main-menu-branch",                                      |
    |   "type": "branch",                                              |
    |   ...                                                            |
    | }                                                                |
    |                                                                  |
    | {                                                                |
    |   "id": "invalid-input-retry",                                   |
    |   "type": "goto",                                                |
    |   ...                                                            |
    | }                                                                |
    |                                                                  |
    | Debugging is clear:                                              |
    | "Flow stopped at action main-menu-branch"                        |
    | Immediately know where and what                                  |
    +------------------------------------------------------------------+

    Naming Convention:
    +------------------------------------------------------------------+
    | Format: {context}-{purpose}                                      |
    |                                                                  |
    | Examples:                                                        |
    | - welcome-greeting                                               |
    | - menu-input-receive                                             |
    | - sales-branch                                                   |
    | - invalid-retry-loop                                             |
    | - after-hours-voicemail                                          |
    +------------------------------------------------------------------+


Always Include Default Branches
+++++++++++++++++++++++++++++++

.. code::

    Branch Best Practice:

    +------------------------------------------------------------------+
    |                            BAD                                   |
    +------------------------------------------------------------------+
    | {                                                                |
    |   "type": "branch",                                              |
    |   "option": {                                                    |
    |     "variable": "voipbin.call.digits",                           |
    |     "target_ids": {                                              |
    |       "1": "option-1",                                           |
    |       "2": "option-2"                                            |
    |     }                                                            |
    |   }                                                              |
    | }                                                                |
    |                                                                  |
    | Problem: What happens if user presses 3, 4, 5, etc.?             |
    | Flow falls through to next action unexpectedly                   |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    |                           GOOD                                   |
    +------------------------------------------------------------------+
    | {                                                                |
    |   "type": "branch",                                              |
    |   "option": {                                                    |
    |     "variable": "voipbin.call.digits",                           |
    |     "default_target_id": "invalid-input-handler",                |
    |     "target_ids": {                                              |
    |       "1": "option-1",                                           |
    |       "2": "option-2"                                            |
    |     }                                                            |
    |   }                                                              |
    | }                                                                |
    |                                                                  |
    | All unexpected inputs are caught and handled                     |
    +------------------------------------------------------------------+


Use Loop Counts
+++++++++++++++

.. code::

    Goto Best Practice:

    +------------------------------------------------------------------+
    |                            BAD                                   |
    +------------------------------------------------------------------+
    | {                                                                |
    |   "type": "goto",                                                |
    |   "option": {                                                    |
    |     "target_id": "menu-start"                                    |
    |   }                                                              |
    | }                                                                |
    |                                                                  |
    | Problem: Infinite loop possible                                  |
    | User keeps entering invalid input forever                        |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    |                           GOOD                                   |
    +------------------------------------------------------------------+
    | {                                                                |
    |   "type": "goto",                                                |
    |   "option": {                                                    |
    |     "target_id": "menu-start",                                   |
    |     "loop_count": 3                                              |
    |   }                                                              |
    | }                                                                |
    |                                                                  |
    | After 3 attempts, flow continues past goto                       |
    | Add a "too many attempts" handler after the goto                 |
    +------------------------------------------------------------------+


Performance Optimization
------------------------

Minimize Webhook Calls
++++++++++++++++++++++

.. code::

    Webhook Optimization:

    +------------------------------------------------------------------+
    |                            BAD                                   |
    +------------------------------------------------------------------+
    | Every action sends a webhook:                                    |
    |                                                                  |
    | answer -> webhook -> talk -> webhook -> digits -> webhook        |
    |                                                                  |
    | Problems:                                                        |
    | - Slows down flow execution                                      |
    | - High load on your server                                       |
    | - Increased latency for caller                                   |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    |                           GOOD                                   |
    +------------------------------------------------------------------+
    | Strategic webhooks at key points:                                |
    |                                                                  |
    | answer -> talk -> digits -> branch -> webhook (with all data)    |
    |                                                                  |
    | Batch data in a single webhook:                                  |
    | {                                                                |
    |   "call_id": "${voipbin.call.id}",                               |
    |   "caller": "${voipbin.call.source.target}",                     |
    |   "selection": "${voipbin.call.digits}",                         |
    |   "timestamp": "${voipbin.activeflow.tm_create}"                 |
    | }                                                                |
    +------------------------------------------------------------------+


Use Async Webhooks When Possible
++++++++++++++++++++++++++++++++

.. code::

    Sync vs Async Webhooks:

    +------------------------------------------------------------------+
    |                     Sync (sync: true)                            |
    +------------------------------------------------------------------+
    | Flow waits for response before continuing                        |
    |                                                                  |
    | Use when:                                                        |
    | - You need data from the response                                |
    | - Response sets variables for branching                          |
    | - Order of operations matters                                    |
    |                                                                  |
    | Example: Customer lookup before greeting                         |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    |                    Async (sync: false)                           |
    +------------------------------------------------------------------+
    | Flow continues immediately, doesn't wait                         |
    |                                                                  |
    | Use when:                                                        |
    | - Just logging/notification                                      |
    | - Response data not needed                                       |
    | - Background processing (analytics, etc.)                        |
    |                                                                  |
    | Example: Logging call events for analytics                       |
    +------------------------------------------------------------------+


Optimize Audio Prompts
++++++++++++++++++++++

.. code::

    Audio Optimization:

    +------------------------------------------------------------------+
    |                     TTS vs Pre-recorded                          |
    +------------------------------------------------------------------+

    Use TTS (talk action) when:
    +------------------------------------------------------------------+
    | - Content is dynamic (names, numbers, dates)                     |
    | - Frequent text changes                                          |
    | - Multiple languages needed                                      |
    | - Prototyping/development                                        |
    +------------------------------------------------------------------+

    Use Pre-recorded (play action) when:
    +------------------------------------------------------------------+
    | - Content is static                                              |
    | - Professional voice quality needed                              |
    | - Brand voice consistency important                              |
    | - High-volume production traffic                                 |
    +------------------------------------------------------------------+

    Performance Comparison:
    +------------------------------------------------------------------+
    | TTS: ~100-300ms generation time per phrase                       |
    | Pre-recorded: ~50ms to start playback                            |
    +------------------------------------------------------------------+


Keep Variable Names Short
+++++++++++++++++++++++++

.. code::

    Variable Naming:

    +------------------------------------------------------------------+
    |                            BAD                                   |
    +------------------------------------------------------------------+
    | ${customer_information.primary_contact.phone_number.country_code}|
    |                                                                  |
    | Problems:                                                        |
    | - Hard to read in JSON                                           |
    | - More data in database                                          |
    | - Prone to typos                                                 |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    |                           GOOD                                   |
    +------------------------------------------------------------------+
    | ${customer.phone}                                                |
    | ${customer.country}                                              |
    |                                                                  |
    | Or use namespaces:                                               |
    | ${cust.phone}                                                    |
    | ${cust.tier}                                                     |
    +------------------------------------------------------------------+


Error Handling Patterns
-----------------------

Graceful Degradation
++++++++++++++++++++

.. code::

    Fallback Pattern:

    +------------------------------------------------------------------+
    |                     Handle Service Failures                      |
    +------------------------------------------------------------------+

    {
      "name": "Resilient Flow",
      "actions": [
        {
          "type": "answer"
        },
        {
          "id": "try-personalization",
          "type": "webhook_send",
          "option": {
            "sync": true,
            "uri": "https://your-api.com/customer",
            "method": "POST",
            "data_type": "application/json",
            "data": "{\"phone\": \"${voipbin.call.source.target}\"}"
          }
        },
        {
          "type": "condition_variable",
          "option": {
            "condition": "!=",
            "variable": "customer.name",
            "value_type": "string",
            "value_string": "",
            "false_target_id": "generic-greeting"
          }
        },
        {
          "id": "personalized-greeting",
          "type": "talk",
          "option": {
            "text": "Hello ${customer.name}, welcome back.",
            "language": "en-US"
          }
        },
        {
          "type": "goto",
          "option": {
            "target_id": "main-menu"
          }
        },
        {
          "id": "generic-greeting",
          "type": "talk",
          "option": {
            "text": "Hello, welcome to our service.",
            "language": "en-US"
          }
        },
        {
          "id": "main-menu",
          "type": "talk",
          "option": {
            "text": "Press 1 for sales, 2 for support.",
            "language": "en-US"
          }
        }
      ]
    }


Timeout Handling
++++++++++++++++

.. code::

    Input Timeout Pattern:

    +------------------------------------------------------------------+
    |                     Handle No Response                           |
    +------------------------------------------------------------------+

    {
      "actions": [
        {
          "id": "prompt",
          "type": "talk",
          "option": {
            "text": "Press 1 or 2.",
            "language": "en-US"
          }
        },
        {
          "type": "digits_receive",
          "option": {
            "duration": 5000,
            "length": 1
          }
        },
        {
          "type": "condition_variable",
          "option": {
            "condition": "!=",
            "variable": "voipbin.call.digits",
            "value_type": "string",
            "value_string": "",
            "false_target_id": "no-input"
          }
        },
        {
          "type": "branch",
          "option": {
            "variable": "voipbin.call.digits",
            "default_target_id": "invalid-input",
            "target_ids": {
              "1": "option-1",
              "2": "option-2"
            }
          }
        },
        {
          "id": "no-input",
          "type": "talk",
          "option": {
            "text": "I didn't receive any input.",
            "language": "en-US"
          }
        },
        {
          "type": "goto",
          "option": {
            "target_id": "prompt",
            "loop_count": 2
          }
        },
        {
          "type": "talk",
          "option": {
            "text": "Goodbye.",
            "language": "en-US"
          }
        },
        {
          "type": "hangup"
        },
        {
          "id": "invalid-input",
          "type": "talk",
          "option": {
            "text": "Invalid selection.",
            "language": "en-US"
          }
        },
        {
          "type": "goto",
          "option": {
            "target_id": "prompt",
            "loop_count": 2
          }
        }
      ]
    }


Maintainability
---------------

Document Your Flows
+++++++++++++++++++

.. code::

    Flow Documentation:

    +------------------------------------------------------------------+
    |                     Use Name and Detail Fields                   |
    +------------------------------------------------------------------+

    {
      "name": "Main IVR Menu - English",
      "detail": "Primary entry point for English callers. Routes to Sales, Support, or Billing queues. Uses business hours check. Last updated: 2024-01-15",
      "actions": [...]
    }

    Naming Convention:
    +------------------------------------------------------------------+
    | Format: {Function} - {Context}                                   |
    |                                                                  |
    | Examples:                                                        |
    | - "Main IVR Menu - English"                                      |
    | - "Support Queue Wait Flow"                                      |
    | - "After Hours Voicemail"                                        |
    | - "Post-Call Survey - Premium Customers"                         |
    +------------------------------------------------------------------+


Version Your Flows
++++++++++++++++++

.. code::

    Versioning Strategy:

    +------------------------------------------------------------------+
    |                     Flow Versioning                              |
    +------------------------------------------------------------------+

    Option 1: Include version in name
    {
      "name": "Main IVR v2.1",
      "detail": "Version 2.1 - Added Spanish language option"
    }

    Option 2: Keep multiple flow versions
    +------------------------------------------------------------------+
    | Production: "main-ivr-prod"     <- Stable, live traffic          |
    | Staging:    "main-ivr-staging"  <- Testing new features          |
    | Dev:        "main-ivr-dev"      <- Development/experiments       |
    +------------------------------------------------------------------+

    Promotion workflow:
    1. Develop in dev flow
    2. Test in staging flow
    3. Copy to production flow when ready


Test in Isolation
+++++++++++++++++

.. code::

    Testing Approach:

    +------------------------------------------------------------------+
    |                     Modular Testing                              |
    +------------------------------------------------------------------+

    Each sub-flow can be tested independently:

    1. Language Selection Flow
       - Test: All language options work
       - Test: Default fallback works

    2. Main Menu Flow
       - Test: All branch options route correctly
       - Test: Invalid input handling
       - Test: Timeout handling

    3. Support Queue Flow
       - Test: Queue join works
       - Test: Wait music plays
       - Test: Timeout routes to voicemail

    4. Voicemail Flow
       - Test: Recording starts
       - Test: Recording stops on timeout
       - Test: Recording stops on key press


Security Best Practices
-----------------------

Validate External Data
++++++++++++++++++++++

.. code::

    Input Validation:

    +------------------------------------------------------------------+
    |                     Validate Webhook Responses                   |
    +------------------------------------------------------------------+

    When using data from webhook_send responses:

    BAD:
    {
      "type": "talk",
      "option": {
        "text": "Calling ${customer.phone}",   <- Could be anything!
        "language": "en-US"
      }
    }

    GOOD:
    {
      "type": "condition_variable",
      "option": {
        "condition": "!=",
        "variable": "customer.phone",
        "value_type": "string",
        "value_string": "",
        "false_target_id": "no-phone-error"
      }
    }
    ... then use customer.phone


Protect Sensitive Data
++++++++++++++++++++++

.. code::

    Data Protection:

    +------------------------------------------------------------------+
    |                     Sensitive Information                        |
    +------------------------------------------------------------------+

    Don't log sensitive data in webhooks:

    BAD:
    {
      "type": "webhook_send",
      "option": {
        "data": "{\"credit_card\": \"${customer.card}\"}"
      }
    }

    GOOD:
    {
      "type": "webhook_send",
      "option": {
        "data": "{\"customer_id\": \"${customer.id}\", \"action\": \"payment_attempt\"}"
      }
    }

    Look up sensitive data server-side using customer_id


Rate Limiting Awareness
+++++++++++++++++++++++

.. code::

    Rate Limiting:

    +------------------------------------------------------------------+
    |                     API Rate Limits                              |
    +------------------------------------------------------------------+

    Be aware of limits on:
    - Webhook requests to your server
    - VoIPBIN API calls
    - TTS generation requests

    Design flows to minimize API calls:
    +------------------------------------------------------------------+
    | - Batch data in single webhooks                                  |
    | - Cache customer data for session duration                       |
    | - Use variables instead of repeated lookups                      |
    +------------------------------------------------------------------+


Checklist: Flow Review
----------------------

Before deploying a new flow, verify:

.. code::

    Pre-Deployment Checklist:

    Structure:
    [ ] All action IDs are meaningful and unique
    [ ] All branch actions have default_target_id
    [ ] All goto actions have loop_count
    [ ] Flow name and detail are descriptive
    [ ] Modular design (sub-flows where appropriate)

    Error Handling:
    [ ] Invalid input is handled
    [ ] Timeouts are handled
    [ ] Service failures have fallbacks
    [ ] Caller hangup is considered

    Performance:
    [ ] Webhook calls are minimized
    [ ] Async webhooks used where possible
    [ ] No unnecessary loops
    [ ] Variables are concise

    Security:
    [ ] No sensitive data in webhooks
    [ ] External data is validated
    [ ] Recording consent is obtained (if required)

    Testing:
    [ ] All branches tested
    [ ] Timeout scenarios tested
    [ ] Error scenarios tested
    [ ] End-to-end call tested

    Documentation:
    [ ] Flow name describes purpose
    [ ] Flow detail includes version/date
    [ ] Complex logic is commented (in detail field)

