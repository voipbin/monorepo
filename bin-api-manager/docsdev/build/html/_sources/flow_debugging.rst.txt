.. _flow-debugging:

Flow Debugging and Troubleshooting
==================================

This section provides tools and techniques for debugging flow execution issues.

Prerequisites
+++++++++++++

Before debugging, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* The call ID (UUID) or activeflow ID (UUID) of the session you want to debug. Obtain from ``GET /calls`` or ``GET /activeflows``.
* (Optional) A webhook endpoint (e.g., https://webhook.site) to receive real-time flow events for monitoring.

.. note:: **AI Implementation Hint**

   The most common debugging path is: (1) Find the call via ``GET /calls`` using the phone number or time range. (2) Get the activeflow via ``GET /activeflows?reference_id={call-id}``. (3) Inspect variables via ``GET /activeflows/{id}/variables`` to see what values were set. (4) Check ``current_action_id`` and ``execute_count`` to understand where execution stopped and why.

Debugging Tools
---------------

VoIPBIN provides several tools for debugging flows:

.. code::

    Available Debugging Resources:

    +------------------------------------------------------------------+
    |                     API Endpoints                                |
    +------------------------------------------------------------------+
    | GET /v1/activeflows                                              |
    |   List all activeflow instances for your account                 |
    |                                                                  |
    | GET /v1/activeflows/{id}                                         |
    |   Get detailed state of a specific activeflow                    |
    |                                                                  |
    | GET /v1/activeflows/{id}/variables                               |
    |   Get current variables for an activeflow                        |
    |                                                                  |
    | GET /v1/calls/{id}                                               |
    |   Get call details including flow execution status               |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    |                     Webhooks                                     |
    +------------------------------------------------------------------+
    | activeflow.created     | When a new activeflow starts            |
    | activeflow.updated     | When activeflow state changes           |
    | activeflow.deleted     | When activeflow ends                    |
    | call.progressing       | Call state updates during flow          |
    +------------------------------------------------------------------+


Examining Activeflow State
--------------------------

Use the API to inspect activeflow execution:

.. code::

    Get Activeflow Details:

    Request:
    GET /v1/activeflows/abc-123-def

    Response:
    {
      "id": "abc-123-def",
      "customer_id": "cust-456",
      "flow_id": "flow-789",
      "reference_type": "call",
      "reference_id": "call-xyz",
      "status": "executing",
      "current_action_id": "action-456",
      "execute_count": 5,
      "variables": {
        "voipbin.call.digits": "2",
        "voipbin.call.source.target": "+15551234567",
        "customer.name": "John"
      },
      "tm_create": "2024-01-15T10:30:00Z",
      "tm_update": "2024-01-15T10:30:45Z"
    }

    Key fields to examine:
    +------------------------------------------------------------------+
    | status             | Is the flow executing, waiting, or ended?  |
    | current_action_id  | Which action is the cursor pointing to?    |
    | execute_count      | How many times has the flow executed?      |
    | variables          | What variables have been set?              |
    +------------------------------------------------------------------+


Common Issues and Solutions
---------------------------

Flow Never Starts
+++++++++++++++++

.. code::

    Problem: Call connects but no flow actions execute

    Diagnostic Steps:
    +------------------------------------------------------------------+
    | 1. Check if flow is attached to the number                       |
    |    GET /v1/numbers/{number-id}                                   |
    |    Look for: flow_id field                                       |
    +------------------------------------------------------------------+
    | 2. Check if activeflow was created                               |
    |    GET /v1/activeflows?reference_id={call-id}                    |
    |    If empty: flow failed to start                                |
    +------------------------------------------------------------------+
    | 3. Check flow definition exists                                  |
    |    GET /v1/flows/{flow-id}                                       |
    |    If 404: flow was deleted                                      |
    +------------------------------------------------------------------+

    Common Causes:
    +------------------------------------------------------------------+
    | - Number not linked to a flow                                    |
    | - Flow ID is invalid or deleted                                  |
    | - Flow actions array is empty                                    |
    | - Customer billing issue blocking execution                      |
    +------------------------------------------------------------------+


Flow Stops Unexpectedly
+++++++++++++++++++++++

.. code::

    Problem: Flow stops in the middle without completing

    Diagnostic Steps:
    +------------------------------------------------------------------+
    | 1. Check activeflow status                                       |
    |    GET /v1/activeflows/{id}                                      |
    |    Look at: status and current_action_id                         |
    +------------------------------------------------------------------+
    | 2. Check call status                                             |
    |    GET /v1/calls/{call-id}                                       |
    |    If hangup_reason present: call ended                          |
    +------------------------------------------------------------------+
    | 3. Check execute_count                                           |
    |    If near 100: hit execution limit                              |
    +------------------------------------------------------------------+

    Common Causes:
    +------------------------------------------------------------------+
    | - Caller hung up (check call.hangup_reason)                      |
    | - Hit execution limit (check execute_count)                      |
    | - Invalid action in flow (skipped, flow ended)                   |
    | - Infinite loop triggered safety stop                            |
    +------------------------------------------------------------------+


Branch Not Working
++++++++++++++++++

.. note:: **AI Implementation Hint**

   Branch matching is **exact string comparison** and case-sensitive. The most common cause of branch failures is that ``digits_receive`` with a ``key`` terminator (e.g., ``#``) includes the terminator in the ``voipbin.call.digits`` variable. For example, if the user presses ``1#``, the variable value is ``1#``, which does not match a ``target_ids`` key of ``"1"``. Either remove the ``key`` parameter or account for the terminator in your target keys.

.. code::

    Problem: Branch always goes to default or wrong target

    Diagnostic Steps:
    +------------------------------------------------------------------+
    | 1. Check the variable value                                      |
    |    GET /v1/activeflows/{id}/variables                            |
    |    Look at: the variable used in branch                          |
    +------------------------------------------------------------------+
    | 2. Compare with branch targets                                   |
    |    The value must EXACTLY match a target_ids key                 |
    +------------------------------------------------------------------+

    Example Debug:
    +------------------------------------------------------------------+
    | Branch action:                                                   |
    | {                                                                |
    |   "type": "branch",                                              |
    |   "option": {                                                    |
    |     "variable": "voipbin.call.digits",                           |
    |     "target_ids": {                                              |
    |       "1": "action-a",                                           |
    |       "2": "action-b"                                            |
    |     }                                                            |
    |   }                                                              |
    | }                                                                |
    |                                                                  |
    | Variable value: "1#"                                             |
    |                                                                  |
    | Problem: "1#" != "1"                                             |
    | The # terminator was included in the digit string                |
    +------------------------------------------------------------------+

    Common Causes:
    +------------------------------------------------------------------+
    | - Variable contains extra characters (whitespace, terminators)   |
    | - Variable name is misspelled                                    |
    | - Variable was never set (action didn't execute)                 |
    | - Case sensitivity (variable values are case-sensitive)          |
    +------------------------------------------------------------------+


Talk Action Not Playing
+++++++++++++++++++++++

.. code::

    Problem: Talk action executes but no audio heard

    Diagnostic Steps:
    +------------------------------------------------------------------+
    | 1. Check call state when talk executes                           |
    |    Call must be "progressing" (answered)                         |
    +------------------------------------------------------------------+
    | 2. Check if answer action came before talk                       |
    |    For inbound calls, must answer first                          |
    +------------------------------------------------------------------+
    | 3. Check language parameter                                      |
    |    Must be valid BCP47 code (en-US, not english)                 |
    +------------------------------------------------------------------+

    Common Causes:
    +------------------------------------------------------------------+
    | - Missing "answer" action before "talk"                          |
    | - Invalid language code                                          |
    | - Call not in answered state                                     |
    | - Empty text string                                              |
    | - Invalid SSML syntax                                            |
    +------------------------------------------------------------------+


Digits Not Received
+++++++++++++++++++

.. code::

    Problem: digits_receive action never captures input

    Diagnostic Steps:
    +------------------------------------------------------------------+
    | 1. Check if any digits were captured                             |
    |    GET /v1/activeflows/{id}/variables                            |
    |    Look at: voipbin.call.digits                                  |
    +------------------------------------------------------------------+
    | 2. Check duration parameter                                      |
    |    Is it long enough for user to respond?                        |
    +------------------------------------------------------------------+
    | 3. Check if previous audio completed                             |
    |    digits_receive starts after previous action                   |
    +------------------------------------------------------------------+

    Configuration Check:
    +------------------------------------------------------------------+
    | {                                                                |
    |   "type": "digits_receive",                                      |
    |   "option": {                                                    |
    |     "duration": 5000,    <- 5 seconds, is this enough?           |
    |     "length": 1,         <- Expecting 1 digit                    |
    |     "key": "#"           <- Terminate on # key                   |
    |   }                                                              |
    | }                                                                |
    +------------------------------------------------------------------+

    Common Causes:
    +------------------------------------------------------------------+
    | - Duration too short for user to respond                         |
    | - Caller pressed wrong key (DTMF vs voice input)                 |
    | - Phone doesn't support DTMF tones                               |
    | - Audio codec incompatibility stripping DTMF                     |
    +------------------------------------------------------------------+


Connect Action Failing
++++++++++++++++++++++

.. code::

    Problem: Connect action doesn't reach destination

    Diagnostic Steps:
    +------------------------------------------------------------------+
    | 1. Check the outbound call status                                |
    |    GET /v1/calls?reference_id={activeflow-id}                    |
    |    Look for child calls created by connect                       |
    +------------------------------------------------------------------+
    | 2. Check destination format                                      |
    |    Must be E.164 format (+15551234567)                           |
    +------------------------------------------------------------------+
    | 3. Check source number                                           |
    |    Must be a number you own or have permissions to use           |
    +------------------------------------------------------------------+

    Common Causes:
    +------------------------------------------------------------------+
    | - Invalid destination phone number format                        |
    | - Source number not in your account                              |
    | - Destination is blocking calls                                  |
    | - Carrier routing issue                                          |
    | - Account doesn't have outbound calling enabled                  |
    +------------------------------------------------------------------+


Variables Not Substituting
++++++++++++++++++++++++++

.. code::

    Problem: ${variable} appears literally in output instead of value

    Diagnostic Steps:
    +------------------------------------------------------------------+
    | 1. Check variable exists                                         |
    |    GET /v1/activeflows/{id}/variables                            |
    +------------------------------------------------------------------+
    | 2. Check variable name spelling                                  |
    |    Variable names are case-sensitive                             |
    +------------------------------------------------------------------+
    | 3. Check syntax                                                  |
    |    Must be ${name} not $name or {name}                           |
    +------------------------------------------------------------------+

    Example:
    +------------------------------------------------------------------+
    | Correct:   "${customer.name}"     -> "John Smith"                |
    | Wrong:     "$customer.name"       -> "$customer.name"            |
    | Wrong:     "{customer.name}"      -> "{customer.name}"           |
    | Wrong:     "${Customer.Name}"     -> "" (wrong case)             |
    +------------------------------------------------------------------+


Queue Join Not Working
++++++++++++++++++++++

.. code::

    Problem: Caller joins queue but never gets connected to agent

    Diagnostic Steps:
    +------------------------------------------------------------------+
    | 1. Check queue status                                            |
    |    GET /v1/queues/{queue-id}                                     |
    |    Look at: agent count, online agents                           |
    +------------------------------------------------------------------+
    | 2. Check agent status                                            |
    |    GET /v1/agents?queue_id={queue-id}                            |
    |    Are agents online and available?                              |
    +------------------------------------------------------------------+
    | 3. Check queue configuration                                     |
    |    Is timeout configured? Ring strategy?                         |
    +------------------------------------------------------------------+

    Common Causes:
    +------------------------------------------------------------------+
    | - No agents logged into queue                                    |
    | - All agents are busy                                            |
    | - Queue timeout expired                                          |
    | - Agent ring timeout too short                                   |
    | - Queue strategy misconfigured                                   |
    +------------------------------------------------------------------+


Using Webhooks for Debugging
----------------------------

Set up webhooks to monitor flow execution in real-time:

.. code::

    Webhook Event Flow:

    +------------------------------------------------------------------+
    |                     Debug Webhook Setup                          |
    +------------------------------------------------------------------+

    1. Create a webhook endpoint to receive events
    POST /v1/webhooks
    {
      "name": "Debug Webhook",
      "url": "https://your-server.com/debug",
      "events": [
        "activeflow.created",
        "activeflow.updated",
        "activeflow.deleted",
        "call.progressing"
      ]
    }

    2. Events you'll receive during flow execution:

    Flow starts:
    {
      "type": "activeflow.created",
      "data": {
        "activeflow_id": "abc-123",
        "flow_id": "flow-456",
        "reference_id": "call-789"
      }
    }

    Each action execution:
    {
      "type": "activeflow.updated",
      "data": {
        "activeflow_id": "abc-123",
        "current_action_id": "action-xyz",
        "status": "executing"
      }
    }

    Flow ends:
    {
      "type": "activeflow.deleted",
      "data": {
        "activeflow_id": "abc-123",
        "reason": "completed"
      }
    }


Testing Flows
-------------

Best practices for testing flows before production:

.. code::

    Testing Strategy:

    +------------------------------------------------------------------+
    |                     Development Testing                          |
    +------------------------------------------------------------------+

    1. API-Triggered Testing
    +------------------------------------------------------------------+
    | Create activeflow directly via API without a call:               |
    |                                                                  |
    | POST /v1/activeflows                                             |
    | {                                                                |
    |   "flow_id": "your-flow-id",                                     |
    |   "reference_type": "api"                                        |
    | }                                                                |
    |                                                                  |
    | This runs the flow without media actions (talk, play ignored)    |
    | Useful for testing branching logic and webhooks                  |
    +------------------------------------------------------------------+

    2. Test Phone Number
    +------------------------------------------------------------------+
    | Reserve a dedicated test number for development                  |
    | Attach test flows to this number                                 |
    | Call it manually to verify behavior                              |
    +------------------------------------------------------------------+

    3. Webhook Logging
    +------------------------------------------------------------------+
    | Use webhook.site or similar service to capture webhooks          |
    | Verify all expected events are fired                             |
    | Check variable values in webhook payloads                        |
    +------------------------------------------------------------------+


.. code::

    Test Checklist:

    +------------------------------------------------------------------+
    |                     Flow Test Checklist                          |
    +------------------------------------------------------------------+

    Basic Functionality:
    [ ] Flow starts when call arrives
    [ ] Answer action picks up call
    [ ] Talk actions play audio
    [ ] Flow completes without errors

    Branching:
    [ ] Each branch option routes correctly
    [ ] Default branch catches invalid input
    [ ] Retry loop works (goto with loop_count)

    Variables:
    [ ] Variables are set correctly
    [ ] Variables substitute in talk text
    [ ] Variables pass to webhooks

    Error Cases:
    [ ] Caller hangup is handled gracefully
    [ ] Timeout on digits_receive works
    [ ] Invalid input routes to default

    Integrations:
    [ ] Webhooks receive expected data
    [ ] External APIs respond correctly
    [ ] Queue routing works
    [ ] Recording starts/stops properly


Debugging Webhook Integration
-----------------------------

Common issues with webhook_send:

.. code::

    Webhook Send Debugging:

    +------------------------------------------------------------------+
    | Problem: Webhook not received                                    |
    +------------------------------------------------------------------+

    Check in your flow:
    {
      "type": "webhook_send",
      "option": {
        "sync": false,           <- Async won't block flow
        "uri": "https://...",    <- Is URL accessible?
        "method": "POST",
        "data_type": "application/json",
        "data": "{...}"          <- Valid JSON?
      }
    }

    Common Issues:
    +------------------------------------------------------------------+
    | - URL not reachable from VoIPBIN servers                         |
    | - Firewall blocking incoming requests                            |
    | - SSL certificate issues (use valid cert)                        |
    | - Invalid JSON in data field                                     |
    | - Server returning error status code                             |
    +------------------------------------------------------------------+

    Debug Tips:
    +------------------------------------------------------------------+
    | 1. Test URL accessibility:                                       |
    |    Can you curl the URL from a public server?                    |
    |                                                                  |
    | 2. Check SSL certificate:                                        |
    |    Use valid, non-self-signed certificate                        |
    |                                                                  |
    | 3. Verify JSON syntax:                                           |
    |    Run data through JSON validator                               |
    |                                                                  |
    | 4. Check server logs:                                            |
    |    Is request arriving? What response code?                      |
    +------------------------------------------------------------------+


Flow Execution Limits
---------------------

Understanding and avoiding execution limits:

.. code::

    Execution Limits:

    +------------------------------------------------------------------+
    |                     Per-Cycle Limit: 1000                        |
    +------------------------------------------------------------------+
    | What counts: Each action executed in one cycle                   |
    | Reset: When flow waits for async event (talk, connect, etc.)     |
    |                                                                  |
    | Trigger scenario:                                                |
    | goto -> goto -> goto -> ... (1000 times) -> STOPPED              |
    |                                                                  |
    | Prevention:                                                      |
    | - Always use loop_count on goto actions                          |
    | - Add async actions (talk, sleep) in loops                       |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    |                     Total Execution Limit: 100                   |
    +------------------------------------------------------------------+
    | What counts: Each time flow resumes from async event             |
    | Reset: Never (lifetime of activeflow)                            |
    |                                                                  |
    | Trigger scenario:                                                |
    | A very long call with many interactions                          |
    |                                                                  |
    | Prevention:                                                      |
    | - Design efficient flows                                         |
    | - Avoid unnecessary action loops                                 |
    | - Consider breaking into sub-flows                               |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    |                     On-Complete Chain Limit: 5                   |
    +------------------------------------------------------------------+
    | What counts: on_complete_flow_id triggers                        |
    |                                                                  |
    | Flow A -> Flow B -> Flow C -> Flow D -> Flow E -> STOPPED        |
    |   0         1         2         3         4        5(blocked)    |
    |                                                                  |
    | Prevention:                                                      |
    | - Limit on-complete chains                                       |
    | - Use webhooks for post-call work instead                        |
    +------------------------------------------------------------------+


Error Messages Reference
------------------------

Common error messages and their meanings:

.. code::

    Error Reference:

    +------------------------------------------------------------------+
    | Error: "activeflow not found"                                    |
    +------------------------------------------------------------------+
    | Cause: Activeflow ID doesn't exist or was deleted                |
    | Solution: Check ID, verify flow started successfully             |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    | Error: "flow not found"                                          |
    +------------------------------------------------------------------+
    | Cause: Flow definition ID doesn't exist                          |
    | Solution: Verify flow_id, check if flow was deleted              |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    | Error: "action not found in flow"                                |
    +------------------------------------------------------------------+
    | Cause: goto/branch target_id doesn't exist in actions array      |
    | Solution: Verify action IDs match between targets and actions    |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    | Error: "execution limit exceeded"                                |
    +------------------------------------------------------------------+
    | Cause: Hit 1000 per-cycle or 100 total execution limit           |
    | Solution: Add loop_count to gotos, break up complex flows        |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    | Error: "invalid action type"                                     |
    +------------------------------------------------------------------+
    | Cause: Action type field has unknown value                       |
    | Solution: Check spelling, use documented action types            |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    | Error: "variable not found"                                      |
    +------------------------------------------------------------------+
    | Cause: Referenced variable doesn't exist                         |
    | Solution: Ensure variable is set before use, check spelling      |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    | Error: "call not in valid state for action"                      |
    +------------------------------------------------------------------+
    | Cause: Trying media action on non-answered call                  |
    | Solution: Add answer action, verify call state                   |
    +------------------------------------------------------------------+


Logging Best Practices
----------------------

Add strategic webhooks for debugging:

.. code::

    Strategic Logging Points:

    {
      "name": "Flow with Debug Logging",
      "actions": [
        {
          "type": "webhook_send",
          "option": {
            "uri": "https://your-api.com/debug",
            "method": "POST",
            "data_type": "application/json",
            "data": "{\"event\": \"flow_started\", \"call_id\": \"${voipbin.call.id}\"}"
          }
        },
        {
          "type": "answer"
        },
        {
          "type": "talk",
          "option": {
            "text": "Welcome. Press 1 or 2.",
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
          "type": "webhook_send",
          "option": {
            "uri": "https://your-api.com/debug",
            "method": "POST",
            "data_type": "application/json",
            "data": "{\"event\": \"digits_received\", \"digits\": \"${voipbin.call.digits}\"}"
          }
        },
        {
          "type": "branch",
          "option": {
            "variable": "voipbin.call.digits",
            "target_ids": {
              "1": "option-1",
              "2": "option-2"
            }
          }
        },
        {
          "id": "option-1",
          "type": "webhook_send",
          "option": {
            "uri": "https://your-api.com/debug",
            "method": "POST",
            "data_type": "application/json",
            "data": "{\"event\": \"branch_selected\", \"branch\": \"option-1\"}"
          }
        }
      ]
    }


Quick Diagnostic Commands
-------------------------

Common API calls for troubleshooting:

.. code::

    Diagnostic API Calls:

    # List recent activeflows
    curl -H "Authorization: Bearer $TOKEN" \
      "https://api.voipbin.net/v1/activeflows?limit=10"

    # Get specific activeflow
    curl -H "Authorization: Bearer $TOKEN" \
      "https://api.voipbin.net/v1/activeflows/{id}"

    # Get activeflow variables
    curl -H "Authorization: Bearer $TOKEN" \
      "https://api.voipbin.net/v1/activeflows/{id}/variables"

    # Find activeflow by call
    curl -H "Authorization: Bearer $TOKEN" \
      "https://api.voipbin.net/v1/activeflows?reference_id={call-id}"

    # Get flow definition
    curl -H "Authorization: Bearer $TOKEN" \
      "https://api.voipbin.net/v1/flows/{flow-id}"

    # Get call details
    curl -H "Authorization: Bearer $TOKEN" \
      "https://api.voipbin.net/v1/calls/{call-id}"

    # List calls for a flow
    curl -H "Authorization: Bearer $TOKEN" \
      "https://api.voipbin.net/v1/calls?flow_id={flow-id}"

