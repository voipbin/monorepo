.. _websocket-tutorial:

Tutorial
========

Connect to WebSocket
--------------------

Establish a WebSocket connection to receive real-time event updates from VoIPBIN. The WebSocket provides bi-directional communication for subscribing to specific event topics.

**WebSocket Endpoint:**

.. code::

    wss://api.voipbin.net/v1.0/ws?token=<YOUR_AUTH_TOKEN>

Subscribe to Events
-------------------

After connecting, send a subscription message to receive events for specific resources.

**Subscription Message Format:**

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:<your-customer-id>:call:<call-id>",
            "customer_id:<your-customer-id>:activeflow:<activeflow-id>",
            "agent_id:<your-agent-id>:queue:<queue-id>"
        ]
    }

**Example Subscription:**

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:12345678-1234-1234-1234-123456789012:call:*",
            "customer_id:12345678-1234-1234-1234-123456789012:message:*"
        ]
    }

This subscribes to all call and message events for your customer account.

Unsubscribe from Events
------------------------

Stop receiving events for specific topics by sending an unsubscribe message.

.. code::

    {
        "type": "unsubscribe",
        "topics": [
            "customer_id:12345678-1234-1234-1234-123456789012:call:*"
        ]
    }

WebSocket Client Examples
--------------------------

**Python Example:**

Using the ``websocket-client`` library:

.. code::

    import websocket
    import json
    import time

    def on_message(ws, message):
        """Handle incoming WebSocket messages"""
        data = json.loads(message)
        event_type = data.get('event_type')
        resource_data = data.get('data')

        print(f"Received event: {event_type}")

        if event_type == 'call.status':
            call_id = resource_data['id']
            status = resource_data['status']
            print(f"Call {call_id} status: {status}")

        elif event_type == 'message.received':
            message_text = resource_data['text']
            from_number = resource_data['source']['target']
            print(f"Message from {from_number}: {message_text}")

        elif event_type == 'activeflow.updated':
            activeflow_id = resource_data['id']
            current_action = resource_data['current_action']['type']
            print(f"Activeflow {activeflow_id} executing: {current_action}")

    def on_error(ws, error):
        """Handle WebSocket errors"""
        print(f"Error: {error}")

    def on_close(ws, close_status_code, close_msg):
        """Handle WebSocket connection close"""
        print(f"Connection closed: {close_status_code} - {close_msg}")

    def on_open(ws):
        """Handle WebSocket connection open"""
        print("WebSocket connection established")

        # Subscribe to events after connection opens
        subscription = {
            "type": "subscribe",
            "topics": [
                "customer_id:12345678-1234-1234-1234-123456789012:call:*",
                "customer_id:12345678-1234-1234-1234-123456789012:message:*",
                "customer_id:12345678-1234-1234-1234-123456789012:activeflow:*"
            ]
        }
        ws.send(json.dumps(subscription))
        print("Subscribed to events")

    if __name__ == "__main__":
        # Replace with your actual token
        token = "<YOUR_AUTH_TOKEN>"
        ws_url = f"wss://api.voipbin.net/v1.0/ws?token={token}"

        # Create WebSocket connection
        ws = websocket.WebSocketApp(
            ws_url,
            on_open=on_open,
            on_message=on_message,
            on_error=on_error,
            on_close=on_close
        )

        # Run forever (blocks)
        ws.run_forever()

**JavaScript (Browser) Example:**

.. code::

    const token = '<YOUR_AUTH_TOKEN>';
    const wsUrl = `wss://api.voipbin.net/v1.0/ws?token=${token}`;

    // Create WebSocket connection
    const ws = new WebSocket(wsUrl);

    ws.onopen = function(event) {
        console.log('WebSocket connection established');

        // Subscribe to events
        const subscription = {
            type: 'subscribe',
            topics: [
                'customer_id:12345678-1234-1234-1234-123456789012:call:*',
                'customer_id:12345678-1234-1234-1234-123456789012:message:*'
            ]
        };
        ws.send(JSON.stringify(subscription));
        console.log('Subscribed to events');
    };

    ws.onmessage = function(event) {
        const data = JSON.parse(event.data);
        const eventType = data.event_type;

        console.log(`Received event: ${eventType}`);

        switch(eventType) {
            case 'call.status':
                handleCallStatus(data.data);
                break;
            case 'message.received':
                handleMessageReceived(data.data);
                break;
            case 'activeflow.updated':
                handleActiveflowUpdate(data.data);
                break;
            default:
                console.log('Unknown event type:', eventType);
        }
    };

    ws.onerror = function(error) {
        console.error('WebSocket error:', error);
    };

    ws.onclose = function(event) {
        console.log('WebSocket connection closed:', event.code, event.reason);

        // Implement reconnection logic if needed
        setTimeout(function() {
            console.log('Reconnecting...');
            // Recreate connection
        }, 5000);
    };

    function handleCallStatus(callData) {
        console.log(`Call ${callData.id} status: ${callData.status}`);
        // Update UI
        document.getElementById('call-status').textContent = callData.status;
    }

    function handleMessageReceived(messageData) {
        console.log(`Message from ${messageData.source.target}: ${messageData.text}`);
        // Display message in UI
        const messageList = document.getElementById('messages');
        const messageElement = document.createElement('div');
        messageElement.textContent = `${messageData.source.target}: ${messageData.text}`;
        messageList.appendChild(messageElement);
    }

    function handleActiveflowUpdate(activeflowData) {
        console.log(`Activeflow ${activeflowData.id} action: ${activeflowData.current_action.type}`);
        // Update flow visualization
    }

**Node.js Example:**

Using the ``ws`` library:

.. code::

    const WebSocket = require('ws');

    const token = '<YOUR_AUTH_TOKEN>';
    const wsUrl = `wss://api.voipbin.net/v1.0/ws?token=${token}`;

    // Create WebSocket connection
    const ws = new WebSocket(wsUrl);

    ws.on('open', function() {
        console.log('WebSocket connection established');

        // Subscribe to events
        const subscription = {
            type: 'subscribe',
            topics: [
                'customer_id:12345678-1234-1234-1234-123456789012:call:*',
                'customer_id:12345678-1234-1234-1234-123456789012:message:*',
                'customer_id:12345678-1234-1234-1234-123456789012:queue:*'
            ]
        };
        ws.send(JSON.stringify(subscription));
        console.log('Subscribed to events');
    });

    ws.on('message', function(data) {
        const message = JSON.parse(data);
        const eventType = message.event_type;

        console.log(`Received event: ${eventType}`);

        // Process events
        switch(eventType) {
            case 'call.status':
                console.log(`Call ${message.data.id}: ${message.data.status}`);
                break;

            case 'queue.joined':
                console.log(`Caller ${message.data.call_id} joined queue ${message.data.queue_id}`);
                // Notify agents
                notifyAgents(message.data);
                break;

            case 'message.received':
                console.log(`Message: ${message.data.text}`);
                // Process message
                processIncomingMessage(message.data);
                break;
        }
    });

    ws.on('error', function(error) {
        console.error('WebSocket error:', error);
    });

    ws.on('close', function(code, reason) {
        console.log(`WebSocket closed: ${code} - ${reason}`);

        // Implement reconnection
        setTimeout(function() {
            console.log('Reconnecting...');
            // Recreate connection
        }, 5000);
    });

    function notifyAgents(queueData) {
        // Notify available agents about new call in queue
        console.log(`Notifying agents about queue entry`);
    }

    function processIncomingMessage(messageData) {
        // Auto-respond or route to agent
        console.log(`Processing message from ${messageData.source.target}`);
    }

Topic Pattern Matching
-----------------------

WebSocket supports wildcard subscriptions using ``*`` to match multiple resources.

**Subscribe to all calls:**

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:12345678-1234-1234-1234-123456789012:call:*"
        ]
    }

**Subscribe to specific call:**

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:12345678-1234-1234-1234-123456789012:call:a1b2c3d4-e5f6-7890-abcd-ef1234567890"
        ]
    }

**Subscribe to multiple resource types:**

.. code::

    {
        "type": "subscribe",
        "topics": [
            "customer_id:12345678-1234-1234-1234-123456789012:call:*",
            "customer_id:12345678-1234-1234-1234-123456789012:message:*",
            "customer_id:12345678-1234-1234-1234-123456789012:conference:*",
            "customer_id:12345678-1234-1234-1234-123456789012:queue:*"
        ]
    }

**Agent-level subscription:**

.. code::

    {
        "type": "subscribe",
        "topics": [
            "agent_id:98765432-4321-4321-4321-210987654321:queue:*",
            "agent_id:98765432-4321-4321-4321-210987654321:call:*"
        ]
    }

Permission Requirements
-----------------------

**Customer-Level Topics:**

To subscribe to customer-level topics, you need:
- Admin permission, OR
- Manager permission

.. code::

    customer_id:<customer-id>:<resource>:<resource-id>

**Agent-Level Topics:**

Only the owner of the agent can subscribe:

.. code::

    agent_id:<agent-id>:<resource>:<resource-id>

Event Message Format
--------------------

All WebSocket events follow this structure:

.. code::

    {
        "event_type": "call.status",
        "timestamp": "2026-01-20T10:30:00.000000Z",
        "topic": "customer_id:12345678-1234-1234-1234-123456789012:call:a1b2c3d4",
        "data": {
            // Event-specific resource data
        }
    }

**Fields:**
- ``event_type``: Type of event (e.g., ``call.status``, ``message.received``)
- ``timestamp``: When the event occurred (ISO 8601 in UTC)
- ``topic``: The topic that triggered this event
- ``data``: Resource-specific data (call, message, activeflow, etc.)

Common Use Cases
----------------

**Real-Time Call Monitoring:**

Monitor all active calls in real-time:

.. code::

    # Python example
    def on_message(ws, message):
        data = json.loads(message)

        if data['event_type'] == 'call.status':
            call = data['data']

            if call['status'] == 'answered':
                print(f"Call {call['id']} answered - Duration tracking started")
                start_call_timer(call['id'])

            elif call['status'] == 'hangup':
                print(f"Call {call['id']} ended")
                stop_call_timer(call['id'])
                calculate_call_metrics(call)

**Live Agent Dashboard:**

Update agent status and queue information in real-time:

.. code::

    # JavaScript example
    ws.onmessage = function(event) {
        const data = JSON.parse(event.data);

        if (data.event_type === 'queue.joined') {
            // Update queue count
            updateQueueCount(data.data.queue_id, '+1');
            // Show notification to agents
            notifyAgents(`New caller in queue`);
        }

        if (data.event_type === 'agent.status') {
            // Update agent availability display
            updateAgentStatus(data.data.agent_id, data.data.status);
        }
    };

**Flow Execution Visualization:**

Visualize flow execution in real-time:

.. code::

    # Node.js example
    ws.on('message', function(data) {
        const message = JSON.parse(data);

        if (message.event_type === 'activeflow.updated') {
            const activeflow = message.data;

            // Update flow diagram
            highlightCurrentAction(
                activeflow.flow_id,
                activeflow.current_action.id
            );

            // Log action execution
            console.log(`Flow ${activeflow.flow_id}: Executing ${activeflow.current_action.type}`);
        }
    });

**Auto-Reply to Messages:**

Respond to messages immediately when received:

.. code::

    # Python example
    def on_message(ws, message):
        data = json.loads(message)

        if data['event_type'] == 'message.received':
            msg = data['data']

            # Auto-reply logic
            if 'help' in msg['text'].lower():
                send_auto_reply(
                    to=msg['source']['target'],
                    text="Thanks for reaching out! Our team will respond soon. For urgent matters, please call us at +15551234567."
                )

Best Practices
--------------

**1. Connection Management:**
- Implement automatic reconnection with exponential backoff
- Handle connection drops gracefully
- Monitor connection health with ping/pong

**2. Subscription Management:**
- Subscribe only to events you need
- Use wildcards (``*``) for broad monitoring
- Unsubscribe from unused topics to reduce load

**3. Error Handling:**
- Catch and log all WebSocket errors
- Implement timeout handling
- Validate incoming message format

**4. Performance:**
- Process messages asynchronously if handling is time-consuming
- Batch UI updates to avoid excessive rendering
- Use message queues for high-volume scenarios

**5. Security:**
- Always use WSS (WebSocket Secure) in production
- Rotate authentication tokens regularly
- Validate message sources

**6. Testing:**
- Test reconnection logic thoroughly
- Simulate connection failures
- Verify subscription/unsubscription works correctly

Connection Lifecycle
--------------------

**1. Connect:**

.. code::

    ws = new WebSocket('wss://api.voipbin.net/v1.0/ws?token=<YOUR_AUTH_TOKEN>');

**2. Subscribe on Open:**

.. code::

    ws.onopen = function() {
        ws.send(JSON.stringify({
            type: 'subscribe',
            topics: ['customer_id:<id>:call:*']
        }));
    };

**3. Handle Messages:**

.. code::

    ws.onmessage = function(event) {
        const data = JSON.parse(event.data);
        // Process event
    };

**4. Handle Disconnection:**

.. code::

    ws.onclose = function(event) {
        console.log('Disconnected');
        // Implement reconnection
        setTimeout(reconnect, 5000);
    };

**5. Graceful Shutdown:**

.. code::

    // Unsubscribe before closing
    ws.send(JSON.stringify({
        type: 'unsubscribe',
        topics: ['customer_id:<id>:call:*']
    }));

    // Close connection
    ws.close();

For more details about WebSocket topics and event types, see :ref:`WebSocket Overview <websocket_overview>`.
