.. _call-transfer:

Transfer
========
CPaaS, or Communications Platform as a Service, is a cloud-based technology that allows developers to add real-time communication features to their applications, such as voice and messaging capabilities.
Call transfer is a common feature in CPaaS that allows an ongoing phone call to be transferred from one person to another, or from one device to another, without disrupting the conversation.

There are two main types of call transfer in CPaaS: blind transfer and attended transfer.

In both types of call transfer, the transfer can be done manually by the person initiating the transfer, or it can be automated using CPaaS software. Automated transfer is typically done using rules-based routing, which determines the appropriate person or device to transfer the call to based on predefined rules or criteria.

Call transfer is just one of many features available in CPaaS technology, which can help improve call handling, reduce call times, and improve overall customer service.

.. note:: **AI Implementation Hint**

   Call transfers involve active calls, which are chargeable. A transfer creates a new outbound call to the transfer destination. Initiate a transfer via ``POST /transfers`` with the ``transferer_call_id`` (UUID, obtained from ``GET /calls``) of an active call in ``progressing`` status, along with ``transferee_addresses`` (array of addresses) and ``transfer_type`` (enum: ``attended`` or ``blind``).

   Agents of a customer can transfer a call they own through ``POST /service_agents/transfers``. The ``transferee_addresses`` takes exactly one address of type ``extension`` with the extension ID as ``target``, and the extension must belong to the same customer.

.. _call-transfer-blind_transfer:

Blind Transfer
--------------
Blind transfer is the simplest type of call transfer. In this type of transfer, the person initiating the transfer simply transfers the call to another person or phone number without first speaking to them. This is useful when the person receiving the call is known to be available and ready to take the call. Blind transfer is commonly used in call center environments where a caller needs to be routed to the appropriate agent or department.

.. code::

    Caller           VoIPBin        Transferer        Transferee
    |                  |                |                |
    |    Call in       | Call in        |                |
    |    progress      | progress       |                |
    |<---------------->|<-------------->|                |
    |                  |                |                |
    |                  | Send transfer  |                |
    |                  | Request        |                |
    |                  |<---------------|                |
    |                  |                |                |
    |                  | Hangup         |                |
    |                  |--------------->|                |
    |                  |                |                |
    |   Ring           |                |                |
    |<-----------------|                |                |
    |                  |                |                |
    |                  | Dial           |                |
    |                  |-------------------------------->|
    |                  |                |                |
    |                  |                |     Answer     |
    |                  |<--------------------------------|
    |                  |                |                |
    |  Stop ring       |                |                |
    |<-----------------|                |                |

* The Caller initiates a call to the VoIPBin and the call is in progress.
* The Transferer, who is already on a call, decides to transfer the Caller to the Transferee.
* The Transferer sends a transfer request to the VoIPBin, indicating the Transferee's number.
* The VoIPBin immediately hangs up the Transferer's call.
* The VoIPBin rings the Caller (who now hears ringback) and dials the Transferee.
* The Transferee answers the call and is connected to the Caller.

This is the basic process of an blind transfer using a CPaaS like VoIPBin.

.. note:: **AI Implementation Hint**

   In a blind transfer, the transferer is disconnected immediately after the transfer is initiated. If the transferee does not answer, the caller may be left with no connection. For critical calls, use attended transfer instead. The caller will hear ringing while waiting for the transferee to answer.

.. _call-transfer-attended_transfer:

Attended Transfer
-----------------
Attended transfer, also known as consultative transfer, involves the person initiating the transfer first speaking to the person who will be taking the call. This allows the person initiating the transfer to provide context or information about the caller or the reason for the transfer. Once the person who will be taking the call is ready, the transfer is initiated and the original caller is connected to the new person or device. Attended transfer is commonly used in situations where the person receiving the call may need more information before taking the call, such as when transferring a call to a supervisor or manager.

.. code::

    Caller           VoIPBin        Transferer        Transferee
    |                  |                |                |
    |    Call in       | Call in        |                |
    |    progress      | progress       |                |
    |<---------------->|<-------------->|                |
    |                  |                |                |
    |                  | Send transfer  |                |
    |                  | Request        |                |
    |                  |<---------------|                |
    |                  |                |                |
    |   MOH/Mute       |                |                |
    |<-----------------|                |                |
    |                  |                |                |
    |                  | Dial           |                |
    |                  |-------------------------------->|
    |                  |                |                |
    |                  |                |    Answer      |
    |                  |<--------------------------------|
    |                  |                |                |
    |                  | Call in        |                |
    |                  | progress       |                |
    |                  |<-------------->|                |
    |                  |                |                |
    |                  |                | Call in        |
    |                  |                | progress       |
    |                  |<------------------------------->|
    |                  |                |                |
    |                  | Hangup         |                |
    |                  |<---------------|                |
    |                  |                |                |
    |  MOH off/Unmute  |                |                |
    |<-----------------|                |                |
    |                  |                |                |
    |    Call in       |                |                |
    |    progress      |                |                |
    |<---------------->|                |                |

* The Caller initiates a call to the VoIPBin, and the call is in progress with transferer.
* The Transferer, who is already on a call, decides to transfer the Caller to the Transferee.
* The Transferer sends a transfer request to the VoIPBin, indicating the Transferee's number.
* The VoIPBin puts the Caller on music on hold and mute (direction ``in``).
* The VoIPBin dials to the Transferee.
* The Transferee answers the call and is connected to the Transferer and talk to each other.
* The Transferer drops out of the call.
* The VoIPBin turn off the Caller's Music on hold and the Caller and Transferee can now hear each other.

This is the basic process of an attended transfer using a CPaaS like VoIPBin. It allows for seamless communication between parties and can help businesses manage their incoming calls more efficiently.

.. note:: **AI Implementation Hint**

   During an attended transfer, the caller is placed on hold with music. The transferer and transferee can speak privately before completing the transfer. The transfer is only completed when the transferer hangs up. If the transferee does not answer or the consultation fails, the transferer can cancel the transfer and resume the original call with the caller.

.. _call-transfer-add_participant:

Adding a Participant
--------------------
An agent can bring a colleague into a call the agent owns, so that the two-party call becomes a three-party call. Unlike a transfer, nobody is put on hold or muted, and nobody leaves the call: the colleague's phone rings, and when the colleague answers, the colleague joins the call.

Send ``POST /service_agents/calls/{id}/participants`` with an ``address`` of type ``extension``, whose ``target`` is the extension ID of the colleague (the ``target`` of the extension address returned by ``GET /service_agents/agents``). The extension must belong to the same customer as the call, and the call must be in ``progressing`` status and have exactly two parties.

.. code::

    POST /service_agents/calls/{id}/participants
    {
        "address": {
            "type": "extension",
            "target": "c5e7f18c-fc5a-4520-8326-e534e2ca0b8f"
        }
    }

The response is the group call of the new leg. Its ``call_ids`` hold the IDs of the call legs. The legs are created right after the response, and an extension with several registered phones has one leg per phone, of which only the one that answers joins the call. The legs belong to the colleague, so they are not listed by ``GET /service_agents/calls`` of the calling agent. Follow them from the call events.

**When the call ends**

- A call that came in through a queue ends for everyone when the customer leaves. When the agent leaves, the customer and the colleague stay connected.
- A call that the agent placed ends for everyone when the agent leaves, including the colleague. When the other party leaves, the agent and the colleague stay connected.
- In both cases, when only one party is left, the call ends.
- Calls that reached the agent in another way, for example a direct call to the agent's extension, were not measured. Do not rely on a particular behavior when the agent leaves such a call.

**Limits**

- Only a two-party call can take a participant, and only one participant can be added. A request while another participant is being added is refused with ``409`` (``PARTICIPANT_ADD_IN_PROGRESS``). Once a ring has started, a new request is also refused for up to 90 seconds, even when the colleague declined or did not answer, so such a ring can be retried after that time. A request that failed before any ring started, for example for an extension without a registered phone, does not hold the call.
- Only extensions of the same customer can be added. Other address types and unknown, deleted or foreign extensions return ``400``. An extension that has no registered phone returns ``409``.
- A recording of a leg that was started before the colleague joined continues. Insight listening cannot be started while the call has three parties.

.. note:: **AI Implementation Hint**

   Add a participant only to a call in ``progressing`` status that has two parties. Do not retry ``PARTICIPANT_ADD_IN_PROGRESS`` immediately: wait up to 90 seconds or until the call state changes. ``PARTICIPANT_COUNT_INVALID`` and ``CALL_NOT_JOINABLE`` do not clear by waiting, and ``EXTENSION_NOT_REGISTERED`` clears when the colleague's phone registers. The colleague answers on the phone registered for the extension, and the call grows to three parties only after the colleague answers.
