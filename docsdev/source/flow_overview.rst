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
