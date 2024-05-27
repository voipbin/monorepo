.. _call-transfer:

Transfer
========
CPaaS, or Communications Platform as a Service, is a cloud-based technology that allows developers to add real-time communication features to their applications, such as voice and messaging capabilities.
Call transfer is a common feature in CPaaS that allows an ongoing phone call to be transferred from one person to another, or from one device to another, without disrupting the conversation.

There are two main types of call transfer in CPaaS: blind transfer and attended transfer.

In both types of call transfer, the transfer can be done manually by the person initiating the transfer, or it can be automated using CPaaS software. Automated transfer is typically done using rules-based routing, which determines the appropriate person or device to transfer the call to based on predefined rules or criteria.

Call transfer is just one of many features available in CPaaS technology, which can help improve call handling, reduce call times, and improve overall customer service.

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
    |                  | Dial           |                |
    |                  |-------------------------------->|
    |                  |                |                |
    |   Ring           |                |                |
    |<-----------------|                |                |
    |                  |                |                |
    |                  | Hangup         |                |
    |                  |--------------->|                |
    |                  |                |                |
    |                  |                |     Answer     |
    |                  |<--------------------------------|
    |                  |                |                |
    |  Stop ring       |                |                |
    |<-----------------|                |                |

* The Caller initiates a call to the VoIPBin and the call is in progress.
* The Transferer, who is already on a call, decides to transfer the Caller to the Transferee.
* The Transferer sends a transfer request to the VoIPBin, indicating the Transferee's number.
* The VoIPBin dials to the Transferee.
* The VoIPBin hangs up the transferer right after dials to the transferee.
* The Transferee answers the call and is connected to the Caller.

This is the basic process of an blind transfer using a CPaaS like VoIPBin.

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
    |                  | Dial           |                |
    |                  |-------------------------------->|
    |                  |                |                |
    |   MOH/Mute       |                |                |
    |<-----------------|                |                |
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
* The VoIPBin dials to the Transferee.
* The VoIPBin puts the Caller on music on hold and mute.
* The Transferee answers the call and is connected to the Transferer and talk to each other.
* The Transferer drops out of the call.
* The VoIPBin turn off the Caller's Music on hold and the Caller and Transferee can now hear each other.

This is the basic process of an attended transfer using a CPaaS like VoIPBin. It allows for seamless communication between parties and can help businesses manage their incoming calls more efficiently.
