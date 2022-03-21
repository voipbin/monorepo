.. _number-overview:

Overview
========

The VoIPBIN number resource represents a VoIPBIN number provisioned from VoIPBIN, ported or hosted to VoIPBIN.

The Numbers list resource represents an account's VoIPBIN numbers. You can POST to the list resource to provision a new VoIPBIN number.
To find a new number to provision use the subresources of the AvailableNumbers resource.

Provisioning a number is a two-step process. First, you must find an available number to provision using the subresource of the AvailableNumbers resource.
Second, you must POST to the Numbers list resource.

.. _number-overview_flow_execution:

Flow execution
==============

VoIPBin's Number can have multiple flows. This allows execute the registered flow for variable situation.

Currently, allows call_flow_id and message_flow_id.

.. image:: _static/images/number-flow_execution.png

