.. _websocket_overview:

Overview
========
The VoIPBin provides the event topic subscription via websocket.
This allows the web-application can subscribe the specific events that makes help to build the dynamic data change in a run-time.


Topic
-----
The user can subscribe specific topic events. The topic subscription allows pattern matching subscription.

The topic looks like this.

.. code::

    <resource>:<resource-id>

For example, the below event's topic will be like this `activeflow:74ac5405-7c70-4184-9388-1c9f8f8ce25f`.

.. code::

    {
        "type": "activeflow_created",
        "data": {
            "id": "74ac5405-7c70-4184-9388-1c9f8f8ce25f",
            ...
        }
    }

If the subscriber subscribes topic `activeflow:74ac5405-7c70-4184-9388-1c9f8f8ce25f`, they will receive this event.
Also the `activeflow` topic subscriber will receive the same event.

