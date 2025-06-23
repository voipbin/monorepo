.. _flow-overview:

Overview
========
The Flow in VoIPBIN is a powerful set of instructions that guides the platform on how to handle incoming calls. When someone calls one of your VoIPBIN numbers or destinations, the platform looks up the associated URL and sends a request. The Flow instructions are then read to determine the appropriate actions to be taken, such as recording the call, playing a message, or prompting the caller to press digits on their keypad.

At its core, the Flow is an array of JSON objects with special tags defined by VoIPBIN, enabling users to build their own Programmable Voice applications.

.. _flow-overview-how_the_flow_works:

How the flow works
------------------
When an incoming call is received on one of your VoIPBIN numbers or destinations, the platform identifies the associated URL and sends a request.

The Flow instructions are then processed, providing a step-by-step guide on how to handle the call. This might involve greeting the caller, presenting options, gathering input, or executing various actions based on the caller's response.

.. _flow-overview-unified_flow:

Unified flow
------------
VoIPBIN's Unified Flow feature is a game-changer, allowing users to create a single flow that can be attached to multiple communication channels, including voice and video calls, SMS, and RESTful API triggers.

With Unified Flow, users can design a custom flow that defines the actions to be taken when a specific channel request is received. VoIPBIN then automatically executes this flow, ensuring a consistent experience for customers across various channels.

.. _flow-overview-non_linear_action_execution:

Non-linear action execution
---------------------------
VoIPBIN's Flow provides the flexibility of Non-Linear action execution.

Users can customize their actions in both linear and non-linear ways, allowing them to define complex logic and decision trees to handle various call scenarios.

.. image:: _static/images/flow_overview_non_linear.png

.. _flow-overview-flow_fork:

Flow fork
------------
Certain flow actions, such as fetch, fetch_flow, queue_join, and more, enable flow forking.

When the flow is forked, the execution cursor moves to the forked flow and starts executing the actions within it. Once the forked flow reaches its end, the execution cursor moves back to the following action of the forking action, continuing the flow execution.

.. image:: _static/images/flow_overview_fork.png

.. _flow-overview-actions:

Actions
-------
Within VoIPBIN, actions dictate the course of the platform's response in a given flow, encompassing tasks such as initiating calls, playing sounds or prompts, activating text-to-speech, sending SMS or DTMFs, and dispatching SNS messages. These diverse actions serve as indispensable building blocks for crafting customized voice applications, ensuring the creation of uniquely tailored call experiences for users.

VoIPBIN's Flow provides developers with a versatile and robust toolkit, empowering them to forge dynamic voice applications, efficiently manage incoming calls, and curate personalized call encounters. Through its non-linear execution, cohesive flow capabilities, and an array of actions including SMS, DTMFs, SNS messages, and more, VoIPBIN's Flow unveils limitless possibilities for the development of sophisticated and interactive voice applications, precisely tailored to meet specific business needs.

.. _flow-overview-flow_execution:

Flow execution
--------------
In VoIPBin, the execution of flows is versatile and can be initiated through various methods:

Incoming/Outgoing Calls
+++++++++++++++++++++++

VoIPBin facilitates flow execution with both incoming and outgoing calls. Users have the option to register a flow to a specific number. Upon an incoming call to the registered number, VoIPBin will seamlessly execute the associated flow.
Users can also initiate a call with a predefined flow. Once the outgoing call is answered, the attached flow will be triggered.

Received Messages
++++++++++++++++++

VoIPBin allows users to register flows to received messages. When a message is received, the registered flow will be executed.
For example, you can register the flow like the below.

.. code::

  ...
  {
    "type": "branch",
    "option": {
      "variable": "voipbin.message.text",
      "target_ids": {
        "call me": "7a1e8c4c-aaea-11ed-9f0f-bffcc31ca3c3",
        "text me": "79fd8c9a-aaea-11ed-a238-0b094a49c637",
        "text developer": "73107b70-d56b-4b47-b331-dc4f2ac43a5b"
      }
    }
  }
  ...

In the provided example, the system examines the content of the received message using **voipbin.message.text** and proceeds to compare the message text.
Should the message text match the specified criterion, such as "call me," the flow execution cursor will be directed to the designated action ID (7a1e8c4c-aaea-11ed-9f0f-bffcc31ca3c3) for subsequent execution.
See detail about available variables at :ref:`here <variable-main>`.

Outbound Campaigns
++++++++++++++++++

The VoIPBin platform integrates flows into outbound campaigns, providing a structured and automated approach to execution.

API
+++

Users have the flexibility to trigger flows through API calls. This API integration empowers developers to seamlessly incorporate and activate flows within their applications.
These diverse methods offer users and developers flexibility in integrating and executing flows based on their specific needs and scenarios.
See detail about API calling at `here <https://api.voipbin.net/redoc/#tag/Activeflow>`_.
And due to the absence of any media-based incoming or outgoing elements in the API, certain actions within the flow—such as answer, talk, transcribe, and others—will be ignored. Instead, the execution cursor will proceed to the next action in the sequence.

Flow Actions and Interrupt Actions
----------------------------------

VoIPBIN's Flow is built on an action-based engine. Each flow is defined as a JSON array of actions, where a "cursor" tracks the current execution point. Under normal circumstances, the cursor advances sequentially from one action to the next based on the flow's logic. This modular design lets you compose a variety of call-handling scenarios—from answering calls and playing prompts to branching based on caller input.

Standard Flow Actions
++++++++++++++++++++++
Standard flow actions are the building blocks of your call handling logic. Examples include:

* Answer: Picks up the incoming call.
* Talk: Plays a TTS message or pre-recorded audio.
* Digits Receive: Gathers DTMF input from the caller.
* Branch: Directs the flow based on collected input.

Each action is executed in sequence, and upon completion, the flow cursor moves to the next action defined in the JSON array. For example:

.. code::

  {
    "id": "9bc851be-f05e-11ef-a1e0-77ec628521e0",
    "type": "flow",
    "name": "Sample Call Flow",
    "actions": [
      {
        "id": "9c8ebac0-f05e-11ef-b9ae-4b8aecc935fe",
        "type": "answer"
      },
      {
        "id": "9c30642a-f05e-11ef-8674-ebba44c5cb0f2",
        "type": "talk",
        "option": {
          "text": "Welcome to VoIPBIN",
          "language": "en-US"
        }
      },
      {
        "id": "9c659370-f05e-11ef-b141-63708262508b",
        "type": "digits_receive",
        "option": {
          "duration": 5000,
          "length": 1
        }
      },
      {
        "id": "9cb2962a-f05e-11ef-ae35-2374e1ae16df",
        "type": "branch",
        "option": {
          "variable": "voipbin.call.digits",
          "target_ids": {
            "1": "9c30642a-f05e-11ef-8674-ebba44c5cb0f2",
          }
        }
      }
    ]
  }

In this example, the flow proceeds step-by-step—from answering the call to playing a welcome message, receiving DTMF input, and branching accordingly.

Interrupt Actions
++++++++++++++++++

Interrupt actions are specialized commands that can be triggered asynchronously—at any point during the flow—to handle unexpected events or to initiate high-priority tasks. Unlike standard actions, interrupt actions are designed to override or temporarily suspend the normal flow progression.

Common interrupt actions include:

* Attended Transfer: Initiates a transfer even if the call is mid-flow. This allows an operator to consult with a transferee before completing the transfer.
* Transcribe: Starts real-time transcription of the call. This can be triggered on demand without waiting for the current flow action to complete.
* Recording: Begins or stops call recording independently of the flow's sequential actions.
* Text-to-Speech (TTS): Plays a TTS message at any moment, regardless of the flow's current state.

How Interrupt Actions Work
+++++++++++++++++++++++++++

Interrupt actions are implemented through dedicated API endpoints. When an interrupt is invoked, the platform either temporarily pauses the current flow or dynamically inserts the interrupt action into the execution sequence. Once the interrupt is handled, the flow resumes from the appropriate point.

For example, while the call is waiting for DTMF input in the standard flow, an attended transfer interrupt may be triggered via a separate API call. This immediate action takes precedence over the flow's waiting state, ensuring that time-sensitive operations are executed without delay.

Integration and Best Practices
+++++++++++++++++++++++++++++++

* Flexibility: Use standard flow actions to define your routine call-handling steps, and rely on interrupt actions for events that require immediate attention.
* API Integration: Interrupt actions can be initiated via dedicated API calls, enabling your application to react in real time to dynamic call events.
* Flow Resumption: After an interrupt action completes, VoIPBIN resumes the flow from the designated point, ensuring a seamless call experience.

By combining sequential flow actions with the ability to trigger interrupt actions at any time, VoIPBIN provides a robust, programmable environment that can handle both predictable call scenarios and unexpected events with equal ease.

