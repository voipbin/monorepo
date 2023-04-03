.. _flow-overview:

Overview
========
The flow is a set of instructions you can use to tell VoIPBIN what to do when you receive an incoming call.

.. _flow-overview-actions:


How the flow works
------------------
When someone makes a call to one of your VoIPBIN numbers or destinations, VoIPBIN looks up the URL associated with that phone number and sends it a request.
VoIPBIN then reads the Flow instructions to determine what to do, whether it's recording the call, playing a message for the caller, or prompting the caller to press digits on their keypad.

At its core, Flow is an array of json objects with special tags defined by VoIPBIN to help you build your Programmable Voice application.

Unified flow
------------
VoIPBin's unified flow feature allows users to create and attach a single flow to multiple communication channels, including voice and video calls, SMS, and RESTful API triggers. With unified flow, users can design a custom flow that defines the actions to be taken when a specific channel request is received, and VoIPBin will execute this flow automatically upon incoming requests.

Currently, the unified flow feature supports the channels of voice and video calls, SMS, and RESTful API triggers. In the future, VoIPBin plans to add support for more channel handlers, enabling users to design even more complex communication flows that cover a wider range of channels.

Despite the current limited set of channels, users can still define their own logic and actions based on specific criteria, such as caller ID, time of day, or message content, and can incorporate third-party APIs and services into their flows to extend their functionality and integrate with external systems.

Overall, VoIPBin's unified flow feature simplifies the process of designing and managing communication flows across multiple channels, while providing a powerful and customizable tool for delivering a seamless omnichannel experience to customers.

Non-linear action execution
---------------------------
The VoIPBIN's flow provides Non-linear type of action execution. The user can customize their own actions in linear or non-linear way.

.. image:: _static/images/flow_overview_non_linear.png


Flow fork
------------
Some flow actions(fetch, fetch_flow, queue_join, …) fork the flow.

The execution cursor moves to the forked flow once the flow is forked. And starts to execute the actions. Then when it reaches the end of the forked flow, the execution cursor moves back to the following action of the forking action.

.. image:: _static/images/flow_overview_fork.png


Actions
-------
In the VoIPBIN, the users. The action tells VoIPBIN what should take on a given flow such as making a call or speaking sound or talk etc...
