.. _variable-variable:

Variable
========

Activeflow
----------
* voipbin.activeflow.id: Activeflow's ID.
* voipbin.activeflow.reference_type: Activeflow's reference type.
* voipbin.activeflow.reference_id: Activeflow's reference ID.
* voipbin.activeflow.reference_activeflow_id: Activeflow's reference activeflow id.
* voipbin.activeflow.flow_id: Activeflow's flow id.

Call
----

Source address
++++++++++++++
* voipbin.call.source.name: Source address's name.
* voipbin.call.source.detail: Source address's detail.
* voipbin.call.source.target: Source address's target.
* voipbin.call.source.target_name: Source address's target name.
* voipbin.call.source.type: Source address's type.

Destination address
+++++++++++++++++++
* voipbin.call.destination.name: Destination address's name.
* voipbin.call.destination.detail: Destination address's detail.
* voipbin.call.destination.target: Destination address's target.
* voipbin.call.destination.target_name: Destination address's target name.
* voipbin.call.destination.type: Destination address's type.

Others
++++++
* voipbin.call.direction: Call's direction.
* voipbin.call.master_call_id: Call's master call id.
* voipbin.call.digits: Call's received digits.

Message
-------

Source address
++++++++++++++
* voipbin.message.source.name: Source address's name.
* voipbin.message.source.detail: Source address's detail.
* voipbin.message.source.target: Source address's target.
* voipbin.message.source.target_name: Source address's target name.
* voipbin.message.source.type: Source address's type.

Target destination address
++++++++++++++++++++++++++
* voipbin.message.target.destination.name: Destination address's name.
* voipbin.message.target.destination.detail: Destination address's detail.
* voipbin.message.target.destination.target: Destination address's target.
* voipbin.message.target.destination.target_name: Destination address's target name.
* voipbin.message.target.destination.type: Destination address's type.

Message
+++++++
* voipbin.message.id: Message's id.
* voipbin.message.text: Message's text.
* voipbin.message.direction: Message's direction.

Queue
-----

Queue info
++++++++++
* voipbin.queue.id: Entered Queue's ID.
* voipbin.queue.name: Entered Queue's name.
* voipbin.queue.detail: Entered Queue's detail.

Queuecall info
++++++++++++++
* voipbin.queuecall.id: Created Queuecall's ID.
* voipbin.queuecall.timeout_wait: Created Queuecall's wait timeout
* voipbin.queuecall.timeout_service: Created Queuecall's service timeout.

AI Call
-------
* voipbin.aicall.id: Created AI Call's ID.
* voipbin.aicall.ai_id: Created AI Call's AI ID.
* voipbin.aicall.ai_engine_model: Created AI Call's AI engine model.
* voipbin.aicall.confbridge_id: Created AI Call's confbridge ID.
* voipbin.aicall.gender: Created AI call's voice gender.
* voipbin.aicall.language: Created AI call's voice language.

AI Summary
----------
* voipbin.ai_summary.id: Created AI Summary's ID.
* voipbin.ai_summary.reference_type: Created AI Summary's reference type.
* voipbin.ai_summary.reference_id: Created AI Summary's reference ID.
* voipbin.ai_summary.language: Created AI Summary's language.
* voipbin.ai_summary.content: Created AI Summary's content.

Recording
---------
* voipbin.recording.id: Created Recording's ID.
* voipbin.recording.reference_type: Created Recording's reference type.
* voipbin.recording.reference_id: Created Recording's reference ID.
* voipbin.recording.format: Created Recording's format.
* voipbin.recording.recording_name: Created Recording's name.
* voipbin.recording.filenames: Created Recording's filenames.

Transcribe
----------
* voipbin.transcribe.id: Created Transcribe's ID.
* voipbin.transcribe.language: Created Transcribe's language.
* voipbin.transcribe.direction: Created Transcribe's direction.

Transcript
----------
* voipbin.transcript.id: Created Transcript's ID.
* voipbin.transcript.transcribe_id: Parent Transcribe's ID.
* voipbin.transcript.direction: Transcript's direction (in/out).
* voipbin.transcript.message: Transcript's text content.

Conference
----------
* voipbin.conference.id: Created Conference's ID.
* voipbin.conference.name: Conference's name.
* voipbin.conference.type: Conference's type (connect/confbridge).
* voipbin.conference.status: Conference's status.

Confbridge
++++++++++
* voipbin.confbridge.id: Created Confbridge's ID.
* voipbin.confbridge.type: Confbridge's type.
* voipbin.confbridge.status: Confbridge's status.

Agent
-----
* voipbin.agent.id: Agent's ID.
* voipbin.agent.name: Agent's name.
* voipbin.agent.detail: Agent's detail.
* voipbin.agent.status: Agent's status.

Webhook Response
----------------
Variables available after a webhook_send action with sync=true.

* voipbin.webhook.status_code: HTTP response status code.
* voipbin.webhook.body: HTTP response body as string.

Email
-----
* voipbin.email.id: Created Email's ID.
* voipbin.email.status: Email's status.
* voipbin.email.subject: Email's subject.

Outdial
-------
* voipbin.outdial.id: Created Outdial's ID.
* voipbin.outdial.status: Outdial's status.

Outdialtarget
+++++++++++++
* voipbin.outdialtarget.id: Created Outdialtarget's ID.
* voipbin.outdialtarget.status: Outdialtarget's status.
* voipbin.outdialtarget.try_count: Current try count.

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