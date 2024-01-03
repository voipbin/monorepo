.. _websocket_overview:

Overview
========
WebSocket, a formidable feature within VoIPBin, empowers web applications to subscribe to specific event topics, facilitating real-time updates for dynamic data changes during runtime.

WebSocket establishes a persistent and bi-directional connection with the VoIPBin server, fostering seamless communication and event notification, ultimately bestowing web applications with real-time capabilities. This makes WebSocket an indispensable tool for constructing dynamic and interactive communication applications using VoIPBin.

Topic Subscription
------------------
WebSocket enables users to subscribe to distinct event topics, each corresponding to a specific resource in the VoIPBin system. Subscribers can tailor their topic subscriptions to match their application's needs, ensuring they receive updates for resources of interest.

The topic format follows:

.. code::

    <resource>:<resource-id>

For instance, subscribing to the topic `activeflow:74ac5405-7c70-4184-9388-1c9f8f8ce25f` results in receiving events related to the activeflow resource with the ID 74ac5405-7c70-4184-9388-1c9f8f8ce25f. Additionally, subscribing to the broader topic activeflow yields events for all activeflow resources in the system.

.. code::

    {
        "type": "activeflow_created",
        "data": {
            "id": "74ac5405-7c70-4184-9388-1c9f8f8ce25f",
            ...
        }
    }

Pattern Matching Subscription
-----------------------------
WebSocket supports pattern matching subscriptions, allowing users to subscribe to multiple topics based on a pattern. For example, subscribing to the <resource> topic encompasses all events related to a specific resource type.

This pattern matching approach streamlines and enhances topic subscriptions, enabling users to receive updates for multiple resources without the need for individual subscriptions.

Real-Time Event Updates
-----------------------
When an event transpires for a subscribed topic, VoIPBin dispatches the corresponding event data to the WebSocket connection. Web applications receive this data in real-time, facilitating immediate updates and dynamic data changes within the application.

Web applications can seamlessly process these event updates to trigger actions, update UI elements, or reflect changes in the user interface without resorting to manual page refreshes. WebSocket ensures a seamless and interactive user experience, keeping users consistently informed with the latest information.

Benefits of WebSocket
---------------------
WebSocket extends various benefits to web applications leveraging VoIPBin:

* Real-Time Communication: WebSocket fosters instant updates and event notifications, establishing real-time communication between the web application and the VoIPBin server.
* Dynamic Data Updates: WebSocket facilitates the handling of dynamic data changes, enabling the creation of dynamic and interactive user interfaces.
* Efficient Subscription: Pattern matching subscription efficiency allows users to subscribe to multiple resources without the need for individual subscriptions.
* Reduced Latency: By eliminating the need for repeated HTTP requests, WebSocket reduces latency, enhancing the overall responsiveness of the application.
