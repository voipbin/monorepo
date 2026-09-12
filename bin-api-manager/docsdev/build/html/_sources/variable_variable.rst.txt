.. _variable-variable:

Variable
========

.. note:: **AI Implementation Hint**

   All system variables use the ``voipbin.`` prefix. Variables are read-only unless explicitly set via the ``variable_set`` flow action. Use the ``${voipbin.<category>.<field>}`` syntax to reference them in flow action text fields, webhook URLs, and conditional expressions.

Activeflow
----------
* ``voipbin.activeflow.id`` (UUID): The activeflow's unique identifier. Obtained from the current flow execution context.
* ``voipbin.activeflow.reference_type`` (String): The type of resource that triggered this flow (e.g., ``"call"``, ``"conversation"``).
* ``voipbin.activeflow.reference_id`` (UUID): The ID of the resource that triggered this flow (e.g., the call ID or conversation ID).
* ``voipbin.activeflow.reference_activeflow_id`` (UUID): The parent activeflow's ID when this flow was triggered from another flow.
* ``voipbin.activeflow.flow_id`` (UUID): The flow template ID used for this execution. Obtained from ``GET /flows``.
* ``voipbin.activeflow.complete_count`` (Integer): How many times this activeflow chain has been re-created via ``on_complete_flow_id``. Starts at ``0`` and increments on each hop; capped at 5 to prevent infinite chaining.

Flow
----
* ``voipbin.flow.reference_data`` (String): Data passed as a reference when the flow was triggered. Contains contextual information from the triggering resource.

Call
----
* ``voipbin.call.id`` (UUID): The call's unique identifier. Obtained from ``GET /calls``.

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

Conversation
------------
Set when a flow is triggered by a message in a conversation (``reference_type`` is ``conversation``).

Self address
++++++++++++
* ``voipbin.conversation.self.name`` (String): Self address's display name.
* ``voipbin.conversation.self.detail`` (String): Self address's detail information.
* ``voipbin.conversation.self.target`` (String): Self address's target (e.g., phone number in E.164 format).
* ``voipbin.conversation.self.target_name`` (String): Self address's target name.
* ``voipbin.conversation.self.type`` (String): Self address's type (e.g., ``"tel"``).

Peer address
++++++++++++
* ``voipbin.conversation.peer.name`` (String): Peer address's display name.
* ``voipbin.conversation.peer.detail`` (String): Peer address's detail information.
* ``voipbin.conversation.peer.target`` (String): Peer address's target (e.g., phone number in E.164 format).
* ``voipbin.conversation.peer.target_name`` (String): Peer address's target name.
* ``voipbin.conversation.peer.type`` (String): Peer address's type (e.g., ``"tel"``).

Conversation info
+++++++++++++++++
* ``voipbin.conversation.id`` (UUID): The conversation's unique identifier.
* ``voipbin.conversation.owner_id`` (UUID): The customer/owner ID of the conversation.

Message
+++++++
* ``voipbin.conversation_message.id`` (UUID): The message's unique identifier.
* ``voipbin.conversation_message.text`` (String): The message's text content.
* ``voipbin.conversation_message.direction`` (enum string): The message's direction (e.g., ``"incoming"`` or ``"outgoing"``).

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
* ``voipbin.aicall.ai_engine_model`` (String): The AI engine model name (e.g., ``"openai.gpt-5"``).
* ``voipbin.aicall.confbridge_id`` (UUID): The conference bridge ID hosting the AI call.
* ``voipbin.aicall.stt_language`` (String): The AI call's speech-to-text language (e.g., ``"en-US"``).
* ``voipbin.aicall.pipecatcall_id`` (UUID): The underlying Pipecat call's unique identifier.

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