.. _variable_overview:

Overview
========
VoIPBIN introduces a robust feature known as "Variables," providing users with the capability to define and employ dynamic values seamlessly throughout the execution of a flow. These variables, set by various applications, serve as adaptable elements that can be harnessed across different actions within the flow. By incorporating variables, users gain the ability to construct flows that are not only dynamic but also context-aware, allowing them to adapt to specific situations in real-time during execution.

.. _variable-overview-variable_use:

Variable use
------------
Incorporating variables into a flow is a straightforward process for users. To use a variable within a specific action, simply include the variable using the following format:

.. code::

    ${voipbin.call.source.name}

During the execution of the action, VoIPBIN dynamically evaluates the variable, replacing the placeholder with the actual value. This real-time evaluation empowers the flow to make informed decisions and execute actions based on the most up-to-date information, enhancing flexibility and adaptability in response to changing conditions.

.. _variable-overview-dynamic_values:

Capturing Dynamic Values
------------------------
Variables play a pivotal role in capturing and storing a diverse range of dynamic values throughout the flow execution. For instance, the active flow responsible for executing the call flow can set variables like the call's source address and destination address. Subsequently, these captured values become valuable assets that can be leveraged in various follow-up actions, such as sending messages, updating records, or making informed routing decisions. This strategic use of variables enhances the flow's adaptability and empowers users to tailor subsequent actions based on the specifics of each call scenario.

.. _variable-overview-integration_with_applications:

Integration with Applications
-----------------------------
In the VoIPBIN ecosystem, seamless integration with applications is pivotal for the effective utilization of variables. Each application assumes a crucial role in managing and providing access to specific variables pertinent to its functionality. For instance, the call application is responsible for setting variables related to the ongoing call, while other applications may establish variables based on user interactions, external data sources, or business logic.

By strategically integrating with diverse applications and harnessing the power of variables, users gain the capability to construct more intelligent and responsive flows. These flows are designed to adapt dynamically to changing conditions and user interactions.

In essence, the Variable feature in VoIPBIN injects a layer of dynamism and interactivity into the flow execution process. This capability empowers users to craft sophisticated communication workflows capable of leveraging real-time data and context, delivering a seamless and personalized experience for both callers and users.
