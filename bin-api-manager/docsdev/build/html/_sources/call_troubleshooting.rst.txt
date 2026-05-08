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

    Get activeflow status:
    GET /v1/activeflows/{activeflow-id}

    Get recordings (use recording_ids from the call object):
    GET /v1/recordings/{recording-id}

    List all recordings:
    GET /v1/recordings

.. note:: **AI Implementation Hint**

   The ``call-id`` is a UUID returned when creating a call via ``POST /v1/calls`` or from webhook events. The ``activeflow-id`` is obtained from the call's ``activeflow_id`` field. The ``recording-id`` is obtained from the call's ``recording_ids`` array. All debugging endpoints require a valid authentication token (JWT or access key).

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

**Cause 2: Number Not Provisioned or Invalid Type**

.. code::

    Problem:
    Source number not in your account, or is a virtual number.

    Diagnostic:
    GET /v1/numbers
    Verify the source number appears in the response with:
    - type: "normal" (not "virtual")
    - status: "active"

    Fix:
    Purchase a number via POST /v1/numbers or use a normal,
    active number you already own. Virtual numbers cannot be
    used as the source for outgoing PSTN calls.

    If the source fails validation, VoIPBIN falls back to the
    OutboundConfig's ``default_outgoing_source_number_id``. If
    that field is not set (uuid.Nil), the call is rejected.

    To configure the default outgoing source number, update
    your OutboundConfig:
    PUT https://api.voipbin.net/v1.0/outbound_config
    { "default_outgoing_source_number_id": "<number-uuid>" }

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

Source Number / Caller ID Issues
---------------------------------

**Symptom:** Outgoing PSTN call shows "Anonymous" caller ID, or shows a different number than the one provided in ``source.target``.

**Diagnostic API call:**

.. code::

    GET /v1/calls/{call-id}

    Check the source field in the response:
    {
      "source": {
        "type": "tel",
        "target": "anonymous",
        "target_name": "Anonymous"
      }
    }

    If target is "anonymous", the call leg used anonymous caller ID
    (e.g., a non-PSTN leg that skipped source validation, or a PSTN
    call from before the OutboundConfig migration). For new PSTN calls
    with no valid source and no OutboundConfig default, the call is
    now rejected instead.

**Cause 1: Source Number Not in E.164 Format**

.. code::

    Problem:
    Source does not start with "+" (e.g., "15551234567").

    Fix:
    Always use E.164 format: "+15551234567".

**Cause 2: Source is a Virtual Number**

.. code::

    Problem:
    Source number exists in your account but has type "virtual".
    Only "normal" type numbers can be used as source for PSTN calls.

    Diagnostic:
    GET /v1/numbers
    Find the number and check its "type" field.

    Fix:
    Use a number with type: "normal" and status: "active".

**Cause 3: Source Number Not Owned by Customer**

.. code::

    Problem:
    The E.164 number is not in the customer's account or
    has been deleted.

    Diagnostic:
    GET /v1/numbers
    Verify the source number appears with status: "active".

    Fix:
    Purchase the number via POST /v1/numbers or use one
    you already own.

**Cause 4: Default Number Fallback**

.. code::

    If the source fails validation, VoIPBIN checks the customer's
    OutboundConfig for ``default_outgoing_source_number_id``.

    If set: The call uses that number as caller ID (after re-validation
            against ``GET /v1/numbers`` filters: customer-owned, normal,
            active, not soft-deleted).
    If unset (uuid.Nil) or the validated number is no longer valid:
            The call is rejected with no fallback.

    To configure the default:
    PUT https://api.voipbin.net/v1.0/outbound_config
    {
      "default_outgoing_source_number_id": "<number-uuid>"
    }

    The number-uuid must be from GET /v1/numbers (an active
    normal number you own). The default is re-validated at
    call time, so a number that was valid when configured but
    later released or deactivated will fail.

.. note:: **AI Implementation Hint**

   When a user reports unexpected caller ID behavior, check three things: (1) the source number format (must be E.164 with ``+``), (2) the number type via ``GET https://api.voipbin.net/v1.0/numbers`` (must be ``normal``, not ``virtual``), and (3) whether the customer's OutboundConfig has ``default_outgoing_source_number_id`` set, via ``GET https://api.voipbin.net/v1.0/outbound_config`` (which returns the customer's OutboundConfig). Non-PSTN calls (SIP, extension) skip source validation entirely and always use the provided source.

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
      "mute_direction": "both"
    }

    Fix (unhold the call):
    DELETE /v1/calls/{call-id}/hold

    Fix (unmute the call):
    DELETE /v1/calls/{call-id}/mute

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

   If flow actions are not executing at all, verify that the call has a ``flow_id`` set (not ``00000000-0000-0000-0000-000000000000``). For outbound calls, you must either provide ``actions`` inline in ``POST /v1/calls`` or reference an existing flow via ``flow_id``. For inbound calls, the flow is determined by the phone number configuration -- check ``GET /v1/numbers/{number-id}`` for the ``flow_id`` assignment.

Webhooks Not Received
---------------------

**Symptom:** No webhooks arrive at your endpoint, or only some webhooks arrive.

**Diagnostic steps:**

.. code::

    1. Verify webhook configuration on your customer profile:
    GET /v1/customer

    Check the webhook_method and webhook_uri fields:
    {
      "webhook_method": "post",
      "webhook_uri": "https://your-server.com/webhook"
    }

    If webhook_uri is empty, webhooks are not configured.
    Update via PUT /v1/customer with webhook_method and webhook_uri.

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

**Cause 3: Webhook not configured on customer profile**

.. code::

    Symptoms:
    - No webhooks received at all

    Diagnostic:
    GET /v1/customer
    Check that webhook_method is "post" and webhook_uri
    is set to your endpoint URL.

    Fix:
    PUT /v1/customer
    {
      "webhook_method": "post",
      "webhook_uri": "https://your-server.com/webhook"
    }

.. note:: **AI Implementation Hint**

   Webhook delivery is retried up to 3 times with exponential backoff. If all retries fail, the event is dropped. Always return HTTP 200 immediately and process asynchronously. Webhook configuration is managed via the customer profile: use ``GET /v1/customer`` to check ``webhook_method`` and ``webhook_uri``, and ``PUT /v1/customer`` to update them. VoIPBIN sends all event types to your configured endpoint -- there is no per-event subscription.

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

**Diagnostic steps:**

.. code::

    Use the transfer_id from the POST /transfers response
    to look up the transfer object and inspect its state.

    List transfers:
    GET /v1/transfers

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
    Check the transfer via GET /v1/transfers to find
    the transfer and its associated call IDs.

    Fix:
    Verify the transferee address is correct and reachable.
    Create a new transfer with a different destination.

**Cause 3: Caller Hears Dead Air During Transfer**

.. code::

    Symptoms:
    - Hold music not playing during transfer

    Possible reasons:
    - MOH not configured
    - Mute applied instead of hold

    Fix:
    Put the caller on hold before initiating the transfer:
    POST /v1/calls/{call-id}/hold
    POST /v1/calls/{call-id}/moh

.. note:: **AI Implementation Hint**

   Transfers are initiated via ``POST /v1/transfers`` with ``transferer_call_id`` (UUID, obtained from ``GET /v1/calls``) and ``transferee_addresses`` in the request body. The ``transfer_type`` field specifies ``attended`` or ``blind``. The response includes a ``transfer_id`` (UUID) and associated call/groupcall IDs. If a blind transfer fails, the caller may be disconnected with no way to recover. For critical calls, always prefer attended transfer.

Queue Problems
--------------

**Symptom:** Calls not distributed to agents, long wait times, or agents not receiving calls.

**Diagnostic API calls:**

.. code::

    Check queue status:
    GET /v1/queues/{queue-id}
    {
      "wait_queuecall_ids": ["uuid-1", "uuid-2"],
      "service_queuecall_ids": []
    }

    Check agent status:
    GET /v1/agents
    Verify agents have status: "available"

**Cause 1: No Available Agents**

.. code::

    Symptoms:
    - service_queuecall_ids is empty
    - wait_queuecall_ids is growing

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
      "wait_timeout": 30000,
      "service_timeout": 60000
    }

    Fix:
    Increase wait_timeout to give agents more time to answer:
    PUT /v1/queues/{queue-id}
    {
      "wait_timeout": 300000,
      "wait_flow_id": "voicemail-flow-uuid"
    }

.. note:: **AI Implementation Hint**

   The ``queue-id`` is a UUID obtained from ``GET /v1/queues``. The ``wait_flow_id`` must reference a valid flow (obtained from ``GET /v1/flows``) that handles the fallback (e.g., play a message and take a voicemail). Agent status is managed via ``PUT /v1/agents/{agent-id}`` with the ``status`` field. Common agent statuses are ``available``, ``busy``, ``away``, and ``offline``.

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

   For HTTP 400 errors, the response body contains a detailed error message describing which field is invalid. For HTTP 404, always verify the resource ID by calling the corresponding list endpoint first (e.g., ``GET /v1/calls`` to verify a call-id exists before operating on it). For HTTP 429, implement exponential backoff starting at 1 second. For HTTP 500, include the ``x-request-id`` response header when contacting support.

Getting Help
------------

If issues persist after troubleshooting, see :ref:`Support <support>` for contact information and additional resources.
