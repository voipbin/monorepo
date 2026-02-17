.. _variable-variable:

Variable
========

.. note:: **AI Implementation Hint**

   All system variables use the ``voipbin.`` prefix. Variables are read-only unless explicitly set via the ``variable_set`` flow action. Use the ``${voipbin.<category>.<field>}`` syntax to reference them in flow action text fields, webhook URLs, and conditional expressions.

Activeflow
----------
* ``voipbin.activeflow.id`` (UUID): The activeflow's unique identifier. Obtained from the current flow execution context.
* ``voipbin.activeflow.reference_type`` (String): The type of resource that triggered this flow (e.g., ``"call"``, ``"message"``).
* ``voipbin.activeflow.reference_id`` (UUID): The ID of the resource that triggered this flow (e.g., the call ID or message ID).
* ``voipbin.activeflow.reference_activeflow_id`` (UUID): The parent activeflow's ID when this flow was triggered from another flow.
* ``voipbin.activeflow.flow_id`` (UUID): The flow template ID used for this execution. Obtained from ``GET /flows``.

Call
----

Source address
++++++++++++++
* ``voipbin.call.source.name`` (String): Source address's display name.
* ``voipbin.call.source.detail`` (String): Source address's detail information.
* ``voipbin.call.source.target`` (String): Source address's target (e.g., phone number in E.164 format like ``+15551234567``).
* ``voipbin.call.source.target_name`` (String): Source address's target name.
* ``voipbin.call.source.type`` (String): Source address's type (e.g., ``"tel"``, ``"sip"``).

Destination address
+++++++++++++++++++
* ``voipbin.call.destination.name`` (String): Destination address's display name.
* ``voipbin.call.destination.detail`` (String): Destination address's detail information.
* ``voipbin.call.destination.target`` (String): Destination address's target (e.g., phone number in E.164 format).
* ``voipbin.call.destination.target_name`` (String): Destination address's target name.
* ``voipbin.call.destination.type`` (String): Destination address's type (e.g., ``"tel"``, ``"sip"``).

Others
++++++
* ``voipbin.call.direction`` (enum string): Call's direction (``"incoming"`` or ``"outgoing"``).
* ``voipbin.call.master_call_id`` (UUID): The master call's ID in a call chain.
* ``voipbin.call.digits`` (String): DTMF digits received during the call (e.g., from a ``digits_receive`` action).

Message
-------

Source address
++++++++++++++
* ``voipbin.message.source.name`` (String): Source address's display name.
* ``voipbin.message.source.detail`` (String): Source address's detail information.
* ``voipbin.message.source.target`` (String): Source address's target (e.g., phone number in E.164 format).
* ``voipbin.message.source.target_name`` (String): Source address's target name.
* ``voipbin.message.source.type`` (String): Source address's type (e.g., ``"tel"``).

Target destination address
++++++++++++++++++++++++++
* ``voipbin.message.target.destination.name`` (String): Destination address's display name.
* ``voipbin.message.target.destination.detail`` (String): Destination address's detail information.
* ``voipbin.message.target.destination.target`` (String): Destination address's target (e.g., phone number in E.164 format).
* ``voipbin.message.target.destination.target_name`` (String): Destination address's target name.
* ``voipbin.message.target.destination.type`` (String): Destination address's type (e.g., ``"tel"``).

Message
+++++++
* ``voipbin.message.id`` (UUID): The message's unique identifier.
* ``voipbin.message.text`` (String): The message's text content.
* ``voipbin.message.direction`` (enum string): The message's direction (``"incoming"`` or ``"outgoing"``).

Queue
-----

Queue info
++++++++++
* ``voipbin.queue.id`` (UUID): The entered queue's unique identifier. Obtained from ``GET /queues``.
* ``voipbin.queue.name`` (String): The entered queue's name.
* ``voipbin.queue.detail`` (String): The entered queue's detail description.

Queuecall info
++++++++++++++
* ``voipbin.queuecall.id`` (UUID): The created queuecall's unique identifier.
* ``voipbin.queuecall.timeout_wait`` (Integer): The queuecall's wait timeout in seconds.
* ``voipbin.queuecall.timeout_service`` (Integer): The queuecall's service timeout in seconds.

AI Call
-------
* ``voipbin.aicall.id`` (UUID): The created AI call's unique identifier.
* ``voipbin.aicall.ai_id`` (UUID): The AI configuration ID used. Obtained from ``GET /ais``.
* ``voipbin.aicall.ai_engine_model`` (String): The AI engine model name (e.g., ``"gpt-4"``).
* ``voipbin.aicall.confbridge_id`` (UUID): The conference bridge ID hosting the AI call.
* ``voipbin.aicall.gender`` (enum string): The AI voice gender (e.g., ``"male"``, ``"female"``).
* ``voipbin.aicall.language`` (String): The AI voice language (e.g., ``"en-US"``).

AI Summary
----------
* ``voipbin.ai_summary.id`` (UUID): The created AI summary's unique identifier.
* ``voipbin.ai_summary.reference_type`` (String): The type of resource summarized (e.g., ``"call"``).
* ``voipbin.ai_summary.reference_id`` (UUID): The ID of the resource that was summarized.
* ``voipbin.ai_summary.language`` (String): The language of the summary (e.g., ``"en-US"``).
* ``voipbin.ai_summary.content`` (String): The generated summary text content.

Recording
---------
* ``voipbin.recording.id`` (UUID): The created recording's unique identifier. Obtained from ``GET /recordings``.
* ``voipbin.recording.reference_type`` (String): The type of resource being recorded (e.g., ``"call"``).
* ``voipbin.recording.reference_id`` (UUID): The ID of the resource being recorded (e.g., the call ID).
* ``voipbin.recording.format`` (String): The recording format (e.g., ``"wav"``, ``"mp3"``).
* ``voipbin.recording.recording_name`` (String): The recording's name.
* ``voipbin.recording.filenames`` (String): The recording's output filenames.

Transcribe
----------
* ``voipbin.transcribe.id`` (UUID): The created transcribe's unique identifier. Obtained from ``GET /transcribes``.
* ``voipbin.transcribe.language`` (String): The transcription language (e.g., ``"en-US"``).
* ``voipbin.transcribe.direction`` (enum string): The transcription direction (``"in"``, ``"out"``, or ``"both"``).

Transcript
----------
* ``voipbin.transcript.id`` (UUID): The created transcript's unique identifier.
* ``voipbin.transcript.transcribe_id`` (UUID): The parent transcribe's unique identifier.
* ``voipbin.transcript.direction`` (enum string): The transcript's direction (``"in"`` for caller, ``"out"`` for callee).
* ``voipbin.transcript.message`` (String): The transcript's text content.

Conference
----------
* ``voipbin.conference.id`` (UUID): The created conference's unique identifier. Obtained from ``GET /conferences``.
* ``voipbin.conference.name`` (String): The conference's name.
* ``voipbin.conference.type`` (enum string): The conference's type (``"connect"`` or ``"confbridge"``).
* ``voipbin.conference.status`` (enum string): The conference's current status.

Confbridge
++++++++++
* ``voipbin.confbridge.id`` (UUID): The created confbridge's unique identifier.
* ``voipbin.confbridge.type`` (String): The confbridge's type.
* ``voipbin.confbridge.status`` (enum string): The confbridge's current status.

Agent
-----
* ``voipbin.agent.id`` (UUID): The agent's unique identifier. Obtained from ``GET /agents``.
* ``voipbin.agent.name`` (String): The agent's display name.
* ``voipbin.agent.detail`` (String): The agent's detail description.
* ``voipbin.agent.status`` (enum string): The agent's current status (e.g., ``"available"``, ``"busy"``).

Webhook Response
----------------
Variables available after a ``webhook_send`` action with ``sync=true``.

* ``voipbin.webhook.status_code`` (Integer): The HTTP response status code (e.g., ``200``, ``404``).
* ``voipbin.webhook.body`` (String): The HTTP response body as a string.

Email
-----
* ``voipbin.email.id`` (UUID): The created email's unique identifier.
* ``voipbin.email.status`` (enum string): The email's delivery status.
* ``voipbin.email.subject`` (String): The email's subject line.

Outdial
-------
* ``voipbin.outdial.id`` (UUID): The created outdial's unique identifier. Obtained from ``GET /outdials``.
* ``voipbin.outdial.status`` (enum string): The outdial's current status.

Outdialtarget
+++++++++++++
* ``voipbin.outdialtarget.id`` (UUID): The created outdialtarget's unique identifier.
* ``voipbin.outdialtarget.status`` (enum string): The outdialtarget's current status.
* ``voipbin.outdialtarget.try_count`` (Integer): The current try count for this target.

Custom Variables
----------------
Custom variables can be set using the ``variable_set`` action. These variables are scoped to the current activeflow and persist until the flow ends.

Example of setting a custom variable:

.. code::

    {
        "type": "variable_set",
        "option": {
            "key": "user.selected_option",
            "value": "premium"
        }
    }

The variable can then be referenced as ``${user.selected_option}`` in subsequent actions.