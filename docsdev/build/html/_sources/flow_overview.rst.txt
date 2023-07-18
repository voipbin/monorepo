.. _flow-overview:

Overview
========
The Flow in VoIPBIN is a powerful set of instructions that guides the platform on how to handle incoming calls. When someone calls one of your VoIPBIN numbers or destinations, the platform looks up the associated URL and sends a request. The Flow instructions are then read to determine the appropriate actions to be taken, such as recording the call, playing a message, or prompting the caller to press digits on their keypad.

At its core, the Flow is an array of JSON objects with special tags defined by VoIPBIN, enabling users to build their own Programmable Voice applications.

.. _flow-overview-actions:


How the flow works
------------------
When an incoming call is received on one of your VoIPBIN numbers or destinations, the platform identifies the associated URL and sends a request.

The Flow instructions are then processed, providing a step-by-step guide on how to handle the call. This might involve greeting the caller, presenting options, gathering input, or executing various actions based on the caller's response.

Unified flow
------------
VoIPBIN's Unified Flow feature is a game-changer, allowing users to create a single flow that can be attached to multiple communication channels, including voice and video calls, SMS, and RESTful API triggers.

With Unified Flow, users can design a custom flow that defines the actions to be taken when a specific channel request is received. VoIPBIN then automatically executes this flow, ensuring a consistent experience for customers across various channels.

Non-linear action execution
---------------------------
VoIPBIN's Flow provides the flexibility of Non-Linear action execution.

Users can customize their actions in both linear and non-linear ways, allowing them to define complex logic and decision trees to handle various call scenarios.

.. image:: _static/images/flow_overview_non_linear.png

Flow fork
------------
Certain flow actions, such as fetch, fetch_flow, queue_join, and more, enable flow forking.

When the flow is forked, the execution cursor moves to the forked flow and starts executing the actions within it. Once the forked flow reaches its end, the execution cursor moves back to the following action of the forking action, continuing the flow execution.

.. image:: _static/images/flow_overview_fork.png

Actions
-------
Actions in VoIPBIN tell the platform what actions to take on a given flow, such as making a call, playing sounds or prompts, or initiating text-to-speech. These actions are fundamental building blocks in constructing custom voice applications and creating personalized call experiences for users.

VoIPBIN's Flow offers a versatile and powerful toolset for developers to create dynamic voice applications, handle incoming calls efficiently, and deliver tailored call experiences. With its non-linear execution, unified flow capabilities, and various actions, VoIPBIN's Flow opens up endless possibilities for building sophisticated and interactive voice applications tailored to specific business needs.
