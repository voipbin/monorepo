.. _websocket_overview:

Overview
========
WebSocket, a powerful feature within VoIPBin, revolutionizes web applications by enabling them to subscribe to specific event topics. This capability facilitates real-time updates, ensuring dynamic data changes are seamlessly communicated during runtime.

WebSocket establishes a persistent and bi-directional connection with the VoIPBin server, providing web applications with real-time capabilities for constructing dynamic and interactive communication applications.

Websocket endpoint
----------------------
To leverage WebSocket in VoIPBin, connect to the following endpoint:

.. code::

    GET wss://api.voipbin.net/v1.0/ws?token=<your authtoken here>

Topic Subscription/Unsubscription
------------------------------------
WebSocket empowers users to subscribe to distinct event topics, each corresponding to a specific resource in the VoIPBin system. Users can tailor their topic subscriptions to match their application's needs, ensuring they receive updates for resources of interest.

To subscribe or unsubscribe from an event, send the following JSON message through the established WebSocket:

.. code::

    {
        "type": "<subscribe|unsubscribe>",
        "topics": ["<topic>", ...]
    }

The topic format follows:

.. code::

    <resource>:<resource-id>

Here's an example subscribe message:

.. code::

    {
        "type": "subscribe",
        "topics": [
            "activeflow:74ac5405-7c70-4184-9388-1c9f8f8ce25f"
        ]
    }

Subscribing to the topic activeflow:74ac5405-7c70-4184-9388-1c9f8f8ce25f results in receiving events related to the activeflow resource with the specified ID.

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
