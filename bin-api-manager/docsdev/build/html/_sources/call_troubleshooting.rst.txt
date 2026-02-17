.. _call-troubleshooting:

Call Troubleshooting
====================

This guide helps you diagnose and resolve common call issues. Each problem includes symptoms, causes, and solutions.

.. note:: **AI Implementation Hint**

   When troubleshooting, always start by fetching the call details: ``GET /v1/calls/{call-id}``. The ``status``, ``hangup_by``, ``hangup_reason``, and timestamp fields provide the primary diagnostic information. If the call has an associated ``flow_id``, also check ``GET /v1/activeflows/{activeflow-id}`` for flow execution errors.

Debugging Tools
---------------

Before troubleshooting, understand the tools available:

**API Endpoints for Debugging:**

.. code::

    Get call details:
    GET /v1/calls/{call-id}

    Get call events (webhooks sent):
    GET /v1/calls/{call-id}/events

    Get activeflow status:
    GET /v1/activeflows/{activeflow-id}

    Get recordings:
    GET /v1/calls/{call-id}/recordings

    Get transcripts:
    GET /v1/calls/{call-id}/transcripts

.. note:: **AI Implementation Hint**

   The ``call-id`` is a UUID returned when creating a call via ``POST /calls`` or from webhook events. The ``activeflow-id`` is obtained from the call's associated flow execution. All debugging endpoints require a valid authentication token (JWT or access key).

**WebSocket for Real-time Monitoring:**

.. code::

    Connect:
    wss://api.voipbin.net/v1.0/ws?token=<token>

    Subscribe to all call events:
    {
        "type": "subscribe",
        "topics": ["customer_id:<your-id>:call:*"]
    }

    Events you'll receive:
    - call_created
    - call_ringing
    - call_answered (progressing)
    - call_hungup
    - call_recording_started
    - call_transcribing

Call Never Connects
-------------------

**Symptom:** Call status goes directly from ``dialing`` to ``hangup``. The ``hangup_reason`` is ``failed``. No ringing ever occurred (``tm_ringing`` is ``9999-01-01 00:00:00.000000``).

**Diagnostic API call:**

.. code::

    GET /v1/calls/{call-id}

    Look for:
    {
      "status": "hangup",
      "hangup_reason": "failed",
      "hangup_by": "remote",
      "tm_ringing": "9999-01-01 00:00:00.000000"
    }

**Cause 1: Invalid Phone Number Format**

.. code::

    Problem:
    "+1 555-123-4567" (has spaces/dashes)

    Fix:
    "+15551234567" (E.164 format, digits only after +)

**Cause 2: Number Not Provisioned**

.. code::

    Problem:
    Source number not in your account.

    Diagnostic:
    GET /v1/numbers
    Verify the source number appears in the response.

    Fix:
    Purchase a number via POST /v1/numbers or use a number
    you already own.

**Cause 3: Insufficient Balance**

.. code::

    Problem:
    Account balance too low for call.

    Diagnostic:
    GET /v1/billing-accounts
    Check that balance > 0 and status = "active".

    Fix:
    Add funds to your account.

**Cause 4: Carrier Rejection**

.. code::

    Problem:
    Destination carrier rejected the call.

    Symptoms:
    - Works to some numbers, not others
    - Specific area codes fail

    Fix:
    Contact support with the call-id for investigation.
    Try alternate routes if available.

Call Rings But No Answer
------------------------

**Symptom:** Call reaches ``ringing`` status, then hangs up. The ``hangup_reason`` is ``noanswer`` or ``dialout``.

**Understanding the difference:**

.. code::

    "noanswer":
    +------------------------------------------+
    | The destination phone rang until the     |
    | destination's voicemail or timeout       |
    | kicked in.                               |
    |                                          |
    | Duration: Typically 30-60 seconds        |
    | Cause: Nobody picked up                  |
    +------------------------------------------+

    "dialout":
    +------------------------------------------+
    | VoIPBIN's dial timeout expired before    |
    | the call was answered.                   |
    |                                          |
    | Duration: Your configured timeout        |
    | Cause: Your timeout is shorter than      |
    |        typical ring time                 |
    +------------------------------------------+

**Fix for "noanswer":**

.. code::

    This is expected behavior when nobody answers.
    - Consider using AMD to detect voicemail and leave a message
    - Implement retry logic in your application
    - Use groupcall with multiple destinations for fallback

**Fix for "dialout":**

.. code::

    Increase dial_timeout in your call request:

    POST /v1/calls
    {
      "dial_timeout": 45000,
      "destinations": [...]
    }

    Default timeout is 30000 ms (30 seconds).
    Recommended: 45000-60000 ms for PSTN calls.

Call Answers But No Audio
-------------------------

**Symptom:** Call reaches ``progressing`` status. One or both parties cannot hear each other. Call may disconnect after silence.

**Diagnostic API call:**

.. code::

    GET /v1/calls/{call-id}

    Check these fields:
    - status: should be "progressing"
    - hold: should be false
    - mute_direction: should be "" (empty = unmuted)

**Cause 1: NAT/Firewall Issues (WebRTC)**

.. code::

    Symptoms:
    - WebRTC call connects (signaling OK)
    - No audio in either direction

    Fix:
    - Ensure TURN server is configured
    - Check client firewall allows UDP
    - Verify ICE gathering completes
    - Test with a different network

**Cause 2: Codec Mismatch**

.. code::

    Symptoms:
    - SIP call connects
    - RTP flows but audio is garbled or silent

    Fix:
    VoIPBIN auto-transcodes between codecs, but check
    endpoint codec configuration if using SIP trunking.

**Cause 3: Hold State Stuck**

.. code::

    Symptoms:
    - Call was working, then went silent
    - One party can hear, other cannot

    Diagnostic:
    GET /v1/calls/{call-id}
    {
      "mute_direction": "both",
      "hold": true
    }

    Fix:
    POST /v1/calls/{call-id}/resume
    POST /v1/calls/{call-id}/unmute

.. note:: **AI Implementation Hint**

   For WebRTC no-audio issues, the problem is almost always network-related (firewall, NAT, TURN). Check the browser's developer console for ICE connection state errors. For SIP calls with one-way audio, the issue is typically NAT -- the RTP packets are being sent to the wrong IP address. VoIPBIN's RTPEngine handles most NAT traversal automatically.

Flow Actions Not Executing
--------------------------

**Symptom:** Call answers but expected TTS/media does not play. Actions seem to be skipped.

**Diagnostic API call:**

.. code::

    GET /v1/activeflows/{activeflow-id}

    Look for error in current_action:
    {
      "current_action": {
        "type": "talk",
        "error": "TTS service unavailable"
      }
    }

**Cause 1: early_execution Timing**

.. code::

    Problem:
    Actions execute before call answers.

    With early_execution: true, actions start on INVITE
    (before 200 OK). The call may not be ready for audio.

    Fix:
    Set early_execution: false (default).
    Actions start after the call is answered (200 OK).

**Cause 2: Action Errors**

.. code::

    Problem:
    An action fails and flow stops.

    Diagnostic:
    GET /v1/activeflows/{activeflow-id}

    Fix:
    Check the error message in current_action.
    Verify action configuration (correct fields, valid IDs).

**Cause 3: Missing Action IDs for Branching**

.. code::

    Problem:
    Branch targets an action ID that does not exist.

    Example:
    {
      "type": "branch",
      "option": {
        "target_ids": {
          "1": "nonexistent-id"
        }
      }
    }

    Fix:
    Verify all target_ids match an action "id" in the same flow.
    Use GET /v1/flows/{flow-id} to inspect the flow definition.

.. note:: **AI Implementation Hint**

   If flow actions are not executing at all, verify that the call has a ``flow_id`` set (not ``00000000-0000-0000-0000-000000000000``). For outbound calls, you must either provide ``actions`` inline in ``POST /calls`` or reference an existing flow via ``flow_id``. For inbound calls, the flow is determined by the phone number configuration -- check ``GET /numbers/{number-id}`` for the ``flow_id`` assignment.

Webhooks Not Received
---------------------

**Symptom:** No webhooks arrive at your endpoint, or only some webhooks arrive.

**Diagnostic API calls:**

.. code::

    1. Verify webhook configuration:
    GET /v1/webhooks
    {
      "url": "https://your-server.com/webhook",
      "events": ["call_hungup", "call_answered"],
      "status": "active"
    }

    2. Check webhook delivery history:
    GET /v1/webhooks/{webhook-id}/deliveries
    {
      "deliveries": [
        {
          "id": "delivery-uuid",
          "event_type": "call_hungup",
          "status": "failed",
          "http_code": 500,
          "attempts": 3,
          "last_attempt": "2026-01-20T12:00:00Z",
          "error": "Connection timeout"
        }
      ]
    }

**Cause 1: Endpoint not accessible**

.. code::

    Symptoms:
    - Delivery status is "failed"
    - Error indicates connection refused or timeout

    Fix:
    - Whitelist VoIPBIN IP ranges in your firewall
    - Use valid SSL certificate (self-signed certs are rejected)
    - Ensure your endpoint returns HTTP 200 OK

**Cause 2: Endpoint too slow**

.. code::

    Symptoms:
    - Webhook times out (> 5 seconds)
    - VoIPBIN retries, causing duplicate deliveries

    Fix:
    - Return HTTP 200 immediately
    - Process the webhook payload asynchronously
    - Use a message queue (e.g., SQS, RabbitMQ) for processing

**Cause 3: Wrong event subscription**

.. code::

    Symptoms:
    - Subscribed to "call_created" but expecting "call_hungup"

    Fix:
    Update webhook events:
    PUT /v1/webhooks/{webhook-id}
    {"events": ["call_hungup", "call_answered", "call_created"]}

.. note:: **AI Implementation Hint**

   Webhook delivery is retried up to 3 times with exponential backoff. If all retries fail, the event is dropped. Always return HTTP 200 immediately and process asynchronously. Check ``GET /v1/webhooks/{webhook-id}/deliveries`` to see delivery attempts and failure reasons. If you are not receiving any webhooks, verify your webhook ``status`` is ``active`` and the ``events`` array includes the event types you need.

Recording Issues
----------------

**Symptom:** Recording not found, recording is empty or truncated, or recording URL does not work.

**Diagnostic API call:**

.. code::

    GET /v1/calls/{call-id}
    Check the recording_ids array:
    {
      "recording_ids": []
    }

**Cause 1: Recording Not Created**

.. code::

    Symptoms:
    - recording_ids array is empty

    Possible reasons:
    - record_start action not in flow
    - Call hung up before recording started
    - Error in recording action

    Fix:
    Verify flow has record_start action.
    Check GET /v1/activeflows/{activeflow-id} for errors.

**Cause 2: Recording Empty (duration 0)**

.. code::

    Symptoms:
    - Recording exists but duration is 0

    Possible reasons:
    - Recording started after call ended
    - Audio not flowing during recording (call on mute)

    Fix:
    Place record_start early in the flow action list.
    Verify call had audio (not on mute/hold).

**Cause 3: Recording URL Returns 403**

.. code::

    Symptoms:
    - GET /v1/recordings/{id} returns a URL
    - Downloading the URL fails with HTTP 403

    Cause:
    Signed URLs expire after 1 hour.

    Fix:
    Fetch a fresh URL from the API:
    GET /v1/recordings/{recording-id}
    Download immediately after getting the URL.

.. note:: **AI Implementation Hint**

   Recording upload to cloud storage is asynchronous. After a call ends, the recording may take a few seconds to become available. If ``GET /v1/recordings/{recording-id}`` returns a recording without a ``url``, the upload is still in progress. Poll the endpoint until ``url`` is populated. Cloud storage retention is 90 days by default -- download recordings before they expire.

Transfer Problems
-----------------

**Symptom:** Transfer fails, caller dropped during transfer, or consult call does not connect.

**Diagnostic API call:**

.. code::

    GET /v1/transfers/{transfer-id}
    {
      "status": "consulting",
      "consult_call": {
        "status": "hangup",
        "hangup_reason": "noanswer"
      }
    }

**Cause 1: Blind Transfer Fails**

.. code::

    Symptoms:
    - Caller disconnected during transfer

    Possible reasons:
    - Transfer destination busy or unavailable
    - No failover configured

    Fix:
    Use attended transfer for important calls.
    Configure fallback action on failure.

**Cause 2: Attended Transfer - Consult Fails**

.. code::

    Symptoms:
    - Agent A cannot reach Agent B

    Diagnostic:
    GET /v1/transfers/{transfer-id}

    Fix:
    Cancel transfer and try a different agent:
    POST /v1/transfers/{transfer-id}/cancel

**Cause 3: Caller Hears Dead Air During Transfer**

.. code::

    Symptoms:
    - Hold music not playing during transfer

    Possible reasons:
    - MOH not configured
    - Mute applied instead of hold

    Fix:
    Ensure transfer uses proper hold:
    POST /v1/calls/{call-id}/transfer
    {
      "hold_caller": true,
      "play_moh": true
    }

.. note:: **AI Implementation Hint**

   The ``transfer-id`` is returned in the response to ``POST /v1/calls/{call-id}/transfer``. Use it to check transfer status (``GET /v1/transfers/{transfer-id}``), complete the transfer (``POST /v1/transfers/{transfer-id}/complete``), or cancel it (``POST /v1/transfers/{transfer-id}/cancel``). If a blind transfer fails, the caller may be disconnected with no way to recover. For critical calls, always prefer attended transfer.

Queue Problems
--------------

**Symptom:** Calls not distributed to agents, long wait times, or agents not receiving calls.

**Diagnostic API calls:**

.. code::

    Check queue status:
    GET /v1/queues/{queue-id}
    {
      "available_agents": 0,
      "waiting_calls": 5
    }

    Check agent status:
    GET /v1/agents?queue_id={queue-id}
    Verify agents have status: "available"

**Cause 1: No Available Agents**

.. code::

    Symptoms:
    - available_agents is 0
    - waiting_calls is increasing

    Possible reasons:
    - All agents in "busy" or "offline" status
    - Agents not logged into queue
    - Agent status not updated after previous call

    Fix:
    Verify agent status via GET /v1/agents.
    Ensure agents set their status to "available" after calls.

**Cause 2: Calls Timing Out in Queue**

.. code::

    Symptoms:
    - Calls hang up after short wait

    Diagnostic:
    GET /v1/queues/{queue-id}
    {
      "timeout": 30000,
      "timeout_action": "hangup"
    }

    Fix:
    Increase timeout and add a fallback action:
    PUT /v1/queues/{queue-id}
    {
      "timeout": 300000,
      "timeout_flow_id": "voicemail-flow-uuid"
    }

.. note:: **AI Implementation Hint**

   The ``queue-id`` is a UUID obtained from ``GET /queues``. The ``timeout_flow_id`` must reference a valid flow (obtained from ``GET /flows``) that handles the fallback (e.g., play a message and take a voicemail). Agent status is managed via ``PUT /v1/agents/{agent-id}`` with the ``status`` field. Common agent statuses are ``available``, ``busy``, ``away``, and ``offline``.

Error Reference
---------------

**Hangup Reason Quick Reference:**

.. code::

    +----------------+----------------------------------+------------------------+
    | Reason         | Meaning                          | Action                 |
    +----------------+----------------------------------+------------------------+
    | normal         | Call completed successfully      | No action needed       |
    | failed         | Network/routing failure          | Check number, routes   |
    | busy           | Destination busy                 | Retry later            |
    | noanswer       | No answer before timeout         | Leave voicemail        |
    | cancel         | Caller cancelled                 | No action needed       |
    | dialout        | VoIPBIN timeout                  | Increase dial_timeout  |
    | timeout        | Max call duration exceeded       | Check timeout settings |
    | amd            | Answering machine detected       | Expected behavior      |
    +----------------+----------------------------------+------------------------+

.. note:: **AI Implementation Hint**

   The ``hangup_reason`` field is the most important diagnostic field. ``failed`` means the call never reached the phone network (check number format, provisioning, balance). ``noanswer`` and ``dialout`` both mean nobody picked up, but ``dialout`` is your timeout while ``noanswer`` is the destination's timeout. ``cancel`` means the originator hung up before the call was answered. ``amd`` means voicemail was detected and your AMD settings triggered a hangup.

**HTTP Error Codes:**

.. code::

    +------+----------------------------------+-------------------------------+
    | Code | Meaning                          | Fix                           |
    +------+----------------------------------+-------------------------------+
    | 400  | Invalid request format           | Check request body JSON       |
    |      |                                  | and required fields           |
    +------+----------------------------------+-------------------------------+
    | 401  | Authentication failed            | Check JWT token or access key |
    |      |                                  | is valid and not expired      |
    +------+----------------------------------+-------------------------------+
    | 403  | Permission denied                | Check account permissions     |
    |      |                                  | and resource ownership        |
    +------+----------------------------------+-------------------------------+
    | 404  | Resource not found               | Verify the UUID exists via    |
    |      |                                  | the corresponding GET endpoint|
    +------+----------------------------------+-------------------------------+
    | 409  | Conflict (e.g., call ended)      | Resource state changed;       |
    |      |                                  | re-fetch and retry            |
    +------+----------------------------------+-------------------------------+
    | 429  | Rate limit exceeded              | Slow down requests; implement |
    |      |                                  | exponential backoff           |
    +------+----------------------------------+-------------------------------+
    | 500  | Server error                     | Contact support with the      |
    |      |                                  | request ID from the response  |
    +------+----------------------------------+-------------------------------+

.. note:: **AI Implementation Hint**

   For HTTP 400 errors, the response body contains a detailed error message describing which field is invalid. For HTTP 404, always verify the resource ID by calling the corresponding list endpoint first (e.g., ``GET /calls`` to verify a call-id exists before operating on it). For HTTP 429, implement exponential backoff starting at 1 second. For HTTP 500, include the ``x-request-id`` response header when contacting support.

Getting Help
------------

If issues persist after troubleshooting, see :ref:`Support <support>` for contact information and additional resources.
