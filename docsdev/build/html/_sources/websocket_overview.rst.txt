.. _websocket_overview:

Overview
========
WebSocket is a powerful feature provided by VoIPBin that allows web applications to subscribe to specific event topics and receive real-time updates for dynamic data changes during runtime.

With WebSocket, users can establish a persistent and bi-directional connection with the VoIPBin server, enabling seamless communication and event notification.

Overall, WebSocket empowers web applications with real-time capabilities, making it an essential tool for building dynamic and interactive communication applications with VoIPBin.

Topic Subscription
------------------
WebSocket allows users to subscribe to specific event topics. Each event topic corresponds to a particular resource in the VoIPBin system. Users can subscribe to topics that are relevant to their application's requirements, enabling them to receive updates for specific resources of interest.

The topic format is as follows:

.. code::

    <resource>:<resource-id>

For example, if the user subscribes to the topic `activeflow:74ac5405-7c70-4184-9388-1c9f8f8ce25f`, they will receive events related to the specific activeflow resource with the ID 74ac5405-7c70-4184-9388-1c9f8f8ce25f. Additionally, if the user subscribes to the broader topic activeflow, they will receive events for all activeflow resources in the system.

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
WebSocket supports pattern matching subscription, allowing users to subscribe to multiple topics based on a pattern. For example, a user can subscribe to all events related to a specific resource type by subscribing to the <resource> topic.

This pattern matching approach enables more flexible and efficient topic subscription, as users can receive updates for multiple resources without the need to subscribe to each resource individually.

Real-Time Event Updates
-----------------------
When an event occurs for a subscribed topic, VoIPBin sends the corresponding event data to the WebSocket connection. The web application receives this data in real-time, allowing for immediate updates and dynamic data changes within the application.

Web applications can process these event updates to trigger actions, update UI elements, or reflect changes to the user interface without the need for manual page refreshes. WebSocket provides a seamless and interactive experience for users, ensuring that they are always up-to-date with the latest information.

Benefits of WebSocket
---------------------
WebSocket offers several benefits for web applications using VoIPBin:

* Real-Time Communication: WebSocket provides instant updates and event notifications, enabling real-time communication between the web application and the VoIPBin server.
* Dynamic Data Updates: With WebSocket, web applications can easily handle dynamic data changes, allowing for dynamic and interactive user interfaces.
* Efficient Subscription: Pattern matching subscription makes it efficient to subscribe to multiple resources without the need for individual subscriptions.
* Reduced Latency: WebSocket reduces latency by eliminating the need for repeated HTTP requests, improving the overall responsiveness of the application.





