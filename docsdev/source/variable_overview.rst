.. _variable_overview:

Overview
========
The VoIPBIN provides a powerful feature called "Variables" that allows users to set and use dynamic values throughout the flow execution. These variables are set by various applications and can be utilized in different actions within the flow. Variables enable users to create dynamic and context-aware flows that adapt to specific situations during execution.

.. _variable-overview-variable_use:

Variable use
------------
To use a variable in the flow, users can simply include the variable in the desired action by using the following format:

.. code::

    ${voipbin.call.source.name}

When the voipbin executes the action, it evaluates the variable at that moment, during the execution time, and replaces the variable placeholder with the actual value. This enables the flow to make decisions and perform actions based on real-time information, allowing for greater flexibility and adaptability.

.. _variable-overview-dynamic_values:

Dynamic Values
--------------
Variables can be used to capture and store various dynamic values during the flow execution. For example, the active flow that executes the call flow can set variables such as the call's source address and destination address. These values can then be used in subsequent actions, such as sending messages, updating records, or making routing decisions.

.. _variable-overview-integration_with_applications:

Integration with Applications
-----------------------------
Applications within the VoIPBIN ecosystem play a crucial role in setting and using variables. Each application is responsible for managing and providing access to specific variables that are relevant to its functionality. For example, the call application sets variables related to the call, while other applications may set variables based on user interactions, external data sources, or business logic.

By integrating with applications and utilizing variables, users can create more intelligent and responsive flows that adapt to changing conditions and user interactions.

Overall, the Variable feature in VoIPBIN adds a layer of dynamism and interactivity to the flow execution. It empowers users to build sophisticated communication workflows that can leverage real-time data and context to provide a seamless and personalized experience for callers and users alike.




