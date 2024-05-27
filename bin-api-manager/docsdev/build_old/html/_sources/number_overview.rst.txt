.. _number-overview:

Overview
========
The VoIPBIN Number resource represents a VoIPBIN number that is either provisioned directly from VoIPBIN, ported from another service provider, or hosted on VoIPBIN. These numbers are essential for establishing communication channels and enable users to make and receive calls or messages through the VoIPBIN platform.

The Numbers list resource serves as a repository for all VoIPBIN numbers associated with an account. Users can use the POST method on the list resource to provision a new VoIPBIN number. To find an available number for provisioning, users can utilize the subresources of the AvailableNumbers resource, which provides a list of numbers that can be selected for use.

Provisioning a VoIPBIN number is a two-step process. First, users need to find an available number from the list of options provided by the AvailableNumbers resource. Once a suitable number is identified, users must then proceed to the Numbers list resource and use the POST method to provision the selected number.

.. _number-overview-flow_execution:

Flow execution
--------------
VoIPBIN's Number resource offers the capability to associate multiple flows with a single number. This functionality enables users to execute different registered flows based on specific situations or criteria. Currently, the platform supports call_flow_id and message_flow_id, which allows users to define custom flows for handling incoming calls and messages, respectively.

When a call or message is received on a VoIPBIN number, the platform examines the associated flows to determine the appropriate actions to be taken. Depending on the flow's configuration and the specific situation, different actions may be triggered, such as playing a greeting message, redirecting the call, responding with an automated message, or routing the message to a specific destination.

By allowing multiple flows per number, VoIPBIN empowers users to create dynamic and customized call handling processes. This feature is particularly valuable for businesses or applications that require different call handling behaviors based on the caller's identity, time of day, or other contextual factors.

VoIPBIN's Number resource with its flow execution capabilities offers a versatile and powerful toolset for building sophisticated communication applications. It ensures efficient call and message routing, seamless flow execution, and the ability to tailor communication experiences according to specific business needs or user requirements.

.. image:: _static/images/number-flow_execution.png

