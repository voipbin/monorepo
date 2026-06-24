.. _timeline-analysis-struct-timeline-analysis:

Timeline analysis
=================

.. _timeline-analysis-struct-timeline-analysis-timeline-analysis:

Timeline analysis
-----------------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "activeflow_id": "<string>",
        "status": "<string>",
        "result": { ... },
        "error": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>"
    }

* ``id`` (UUID): The analysis's unique identifier. Returned when creating via ``POST /timeline-analyses`` or listing via ``GET /timeline-analyses``.
* ``customer_id`` (UUID): The customer who owns this analysis. Obtained from the ``id`` field of ``GET /customers``.
* ``activeflow_id`` (UUID): The ended activeflow this analysis was produced for. Obtained from the ``id`` field of ``GET /activeflows``.
* ``status`` (enum string): The analysis lifecycle state. See :ref:`Status <timeline-analysis-struct-timeline-analysis-status>`.
* ``result`` (object): The structured verdict. Present only when ``status`` is ``completed``. See :ref:`Result <timeline-analysis-struct-timeline-analysis-result>`.
* ``error`` (string): A sanitized, operator-safe failure reason. Present only when ``status`` is ``failed``.
* ``tm_create`` (string, ISO 8601): Timestamp when this analysis was created.
* ``tm_update`` (string, ISO 8601): Timestamp when this analysis was last updated.

.. _timeline-analysis-struct-timeline-analysis-status:

Status
------

All possible values for the ``status`` field:

=========== ===========
Status      Description
=========== ===========
progressing The analysis chain is running. No ``result`` yet.
completed   The structured verdict has been produced and stored in ``result``.
failed      The analysis chain failed. ``error`` carries a sanitized reason.
=========== ===========

.. _timeline-analysis-struct-timeline-analysis-result:

Result
------

The structured verdict produced for a ``completed`` analysis.

.. code::

    {
        "version": 3,
        "overall_status": "<string>",
        "input_reduced": false,
        "session_context": {
            "reference_type": "<string>",
            "channel": "<string>",
            "direction": "<string>",
            "direction_raw": "<string>",
            "participants": [
                {"role": "<string>", "address": "<string>"}
            ],
            "flow_name": "<string>",
            "started_at": "<string>",
            "origin_kind": "<string>",
            "origin_type": "<string>",
            "multi_leg": false,
            "ai_handled": false,
            "human_involved": false
        },
        "outcome": {
            "result": "<string>",
            "ended_by": "<string>",
            "reason": "<string>",
            "detail": {"<string>": "<string>"}
        },
        "metrics": {
            "turns_user": 0,
            "turns_bot": 0,
            "first_response_ms": 0,
            "avg_response_ms": 0,
            "max_response_ms": 0,
            "max_gap_ms": 0
        },
        "resources_used": [
            {"type": "<string>", "count": 0}
        ],
        "interactions": [
            {"resource_type": "<string>", "summary": "<string>"}
        ],
        "narrative": "<string>",
        "issues": [
            {
                "severity": "<string>",
                "area": "<string>",
                "summary": "<string>",
                "evidence": [
                    {
                        "evidence_index": 0,
                        "event_type": "<string>",
                        "timestamp": "<string>",
                        "resource_id": "<string>"
                    }
                ]
            }
        ]
    }

* ``version`` (integer): The verdict schema version. ``3`` adds the ``session_context``, ``outcome``, and ``metrics`` blocks; ``2`` adds ``interactions``; ``1`` records have neither.
* ``overall_status`` (enum string): The holistic verdict. One of ``ok``, ``warning``, ``error``.
* ``input_reduced`` (boolean): True when the analyzed input was reduced (very large flow) before the verdict was produced.
* ``session_context`` (object): The channel-neutral 5W1H header describing what the session was (who, when, how, which channel). Present on ``version`` ``3`` and later when the activeflow's reference resolves; the key is OMITTED (not ``null``) when nothing resolves. See :ref:`Session context <timeline-analysis-struct-timeline-analysis-session-context>`.
* ``outcome`` (object): How the session ended and, for a call, who ended it. Present on ``version`` ``3`` and later for ``call``/``conversation``/``ai`` references (best-effort for ``ai``); the key is OMITTED when not applicable. See :ref:`Outcome <timeline-analysis-struct-timeline-analysis-outcome>`.
* ``metrics`` (object): Deterministic voice/AI interaction metrics (turn counts and latencies). Present on ``version`` ``3`` and later for voice/AI references only; the key is OMITTED for chat/API references and when there are no interaction turns. See :ref:`Metrics <timeline-analysis-struct-timeline-analysis-metrics>`.
* ``resources_used`` (array): The resources involved in the flow, with per-type counts.
* ``interactions`` (array): The per-resource content summary of what was communicated and the intent/outcome. Present on ``version`` ``2`` and later (always an array, ``[]`` when there is nothing to summarize).

  * ``resource_type`` (string): The resource/channel the summary describes (e.g. ``call``, ``transcribe``).
  * ``summary`` (string): What was communicated and the intent/outcome for that resource.

* ``narrative`` (string): A human-readable summary of what happened in the flow.
* ``issues`` (array): Detected problems. Empty when ``overall_status`` is ``ok``.

  * ``severity`` (enum string): One of ``info``, ``warning``, ``error``.
  * ``area`` (string): The functional area the issue relates to (e.g. ``media``, ``flow``).
  * ``summary`` (string): A short description of the issue.
  * ``evidence`` (array): The events that support this issue. Each entry points to a specific event in the flow's timeline.

.. note::

   ``session_context``, ``outcome``, and ``metrics`` are computed deterministically by the platform (never by the AI). Consumers should branch on key-presence: an absent block is an omitted key, not ``null`` or an empty object. ``interactions`` and ``issues`` keep their existing always-array contract.

.. _timeline-analysis-struct-timeline-analysis-session-context:

Session context
---------------

The channel-neutral header. Fields marked optional are omitted when not resolvable.

* ``reference_type`` (string): The activeflow's reference type, verbatim. One of ``call``, ``conversation``, ``ai``, ``api``, ``transcribe``, ``recording``, ``campaign``, or empty.
* ``channel`` (string): The normalized channel. One of ``voice``, ``chat``, ``ai``, ``api``, or empty. There is no ``sms``/``email`` channel (SMS/email are actions inside a call or conversation, not standalone sessions).
* ``direction`` (string, optional): The normalized direction. One of ``inbound``, ``outbound``. For a ``conversation`` this is empty (a thread is bidirectional; per-message direction lives in ``outcome.detail``).
* ``direction_raw`` (string, optional): The source-channel direction verbatim (e.g. a call's ``incoming``/``outgoing``).
* ``participants`` (array, optional): The session participants. Omitted (not ``[]``) when none resolve. Each entry has ``role`` (``source``/``destination`` for a call, ``self``/``peer`` for a conversation) and ``address``.
* ``flow_name`` (string, optional): The name of the executed flow.
* ``started_at`` (string, ISO 8601, optional): The channel-appropriate session start time.
* ``origin_kind`` (string, optional): For a ``transcribe``/``recording`` reference, marks that this card shows the underlying source it was made from. One of ``transcription``, ``recording``.
* ``origin_type`` (string, optional): For a ``transcribe``/``recording`` reference, the type of the chased origin (``call``, ``conversation``, or ``confbridge``). The body (participants/direction/outcome) is the origin's, while ``reference_type`` stays ``transcribe``/``recording``.
* ``multi_leg`` (boolean): True when the reference expands to more than one leg (e.g. a group call).
* ``ai_handled`` (boolean): True when an AI (pipecat) session was present. Suppressed (false) on a chased ``transcribe``/``recording`` card.
* ``human_involved`` (boolean): True when a human agent leg connected. Suppressed (false) on a chased ``transcribe``/``recording`` card.

.. _timeline-analysis-struct-timeline-analysis-outcome:

Outcome
-------

How the session ended. Its meaning depends on ``reference_type``.

* ``result`` (enum string): The normalized result. One of ``completed``, ``failed``, ``no_answer``, ``busy``, ``in_progress``, ``unknown``.
* ``ended_by`` (string, optional): For a ``call`` only, who ended it, relative to the platform: ``remote`` or ``local``. Omitted for non-call references. The UI derives a human label from ``(reference_type, direction, ended_by)``:

  ===================== ============ =========== =======================
  reference_type        direction    ended_by    label
  ===================== ============ =========== =======================
  call                  inbound      remote      Customer ended
  call                  inbound      local       System ended
  call                  outbound     remote      Callee ended
  call                  outbound     local       System ended
  call                  (any)        (empty)     No answer / N/A
  ===================== ============ =========== =======================

  There is no ``abandoned`` result: a customer hanging up early is ``result`` ``completed`` with ``ended_by`` ``remote`` and a short ``duration_sec``; the interpretation is left to the consumer.

* ``reason`` (string, optional): The channel-raw reason (a call's hangup reason, a conversation's last-message status).
* ``detail`` (object, optional): Channel-raw extras. For a ``call``: ``duration_sec``, ``hangup_reason``. For a ``conversation``: ``chat_platform`` (``message``/``line``/``whatsapp``), ``last_activity_by`` (``self``/``peer``), ``turns_self``, ``turns_peer``, ``unanswered`` (``true`` when the end-user spoke last and the business did not reply), ``delivery_failures`` (a count, not a thread-wide failure verdict), ``thread_span_sec``, ``truncated``.

.. _timeline-analysis-struct-timeline-analysis-metrics:

Metrics
-------

Deterministic voice/AI interaction metrics. Present for ``call``/``ai`` references only; omitted otherwise and when there are no interaction turns.

* ``turns_user`` (integer): Number of end-user transcription turns.
* ``turns_bot`` (integer): Number of bot transcription turns (intermediate LLM ticks are excluded).
* ``first_response_ms`` (integer, optional): Milliseconds from session answer to the first bot turn. Omitted when the inputs are not available.
* ``avg_response_ms`` (integer, optional): Average bot response latency over user-then-bot turn pairs.
* ``max_response_ms`` (integer, optional): Maximum bot response latency.
* ``max_gap_ms`` (integer, optional): The largest gap between adjacent interaction events.

Example
-------

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "activeflow_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "status": "completed",
        "result": {
            "version": 3,
            "overall_status": "warning",
            "input_reduced": false,
            "session_context": {
                "reference_type": "call",
                "channel": "voice",
                "direction": "inbound",
                "direction_raw": "incoming",
                "participants": [
                    {"role": "source", "address": "+14155550100"},
                    {"role": "destination", "address": "+14155550199"}
                ],
                "flow_name": "support-ivr",
                "started_at": "2024-03-01T10:00:05Z",
                "multi_leg": true,
                "ai_handled": true,
                "human_involved": true
            },
            "outcome": {
                "result": "completed",
                "ended_by": "remote",
                "reason": "normal",
                "detail": {"duration_sec": "312", "hangup_reason": "normal"}
            },
            "metrics": {
                "turns_user": 6,
                "turns_bot": 6,
                "first_response_ms": 1200,
                "avg_response_ms": 1450,
                "max_response_ms": 2600,
                "max_gap_ms": 4100
            },
            "resources_used": [
                {"type": "call", "count": 2},
                {"type": "transcribe", "count": 1}
            ],
            "interactions": [
                {"resource_type": "call", "summary": "Two inbound calls were answered; the caller asked about a billing charge and was transferred to an agent."},
                {"resource_type": "transcribe", "summary": "The conversation was transcribed; the caller confirmed their account number and the agent explained the charge."}
            ],
            "narrative": "Two inbound calls were answered and transcribed; one call's media quality degraded mid-conversation.",
            "issues": [
                {
                    "severity": "warning",
                    "area": "media",
                    "summary": "Call media quality (MOS) degraded to 2.8.",
                    "evidence": [
                        {
                            "evidence_index": 42,
                            "event_type": "call_hangup",
                            "timestamp": "2024-03-01T10:05:12.000000Z",
                            "resource_id": "c3d4e5f6-a7b8-9012-cdef-123456789012"
                        }
                    ]
                }
            ]
        },
        "error": "",
        "tm_create": "2024-03-01T10:06:00.000000Z",
        "tm_update": "2024-03-01T10:06:20.000000Z"
    }
