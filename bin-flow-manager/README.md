# flow-manager

The flow-manager manages the flows.

# Usage
```
$ ./flow-manager -h
Usage of ./flow-manager:
  -dbDSN string
        database dsn for flow-manager. (default "testid:testpassword@tcp(127.0.0.1:3306)/test")
  -prom_endpoint string
        endpoint for prometheus metric collecting. (default "/metrics")
  -prom_listen_addr string
        endpoint for prometheus metric collecting. (default ":2112")
  -rabbit_addr string
        rabbitmq service address. (default "amqp://guest:guest@localhost:5672")
  -rabbit_exchange_delay string
        rabbitmq exchange name for delayed messaging. (default "bin-manager.delay")
  -rabbit_queue_event string
        rabbitmq queue name for event notify (default "bin-manager.flow-manager.event")
  -rabbit_queue_listen string
        rabbitmq queue name for request listen (default "bin-manager.flow-manager.request")
  -redis_addr string
        redis address. (default "127.0.0.1:6379")
  -redis_db int
        redis database. (default 1)
  -redis_password string
        redis password
```

## Example
```
$ ./flow-manager \
-prom_endpoint /metrics \
-prom_listen_addr :2112 \
-dbDSN 'bin-manager:398e02d8-8aaa-11ea-b1f6-9b65a2a4f3a3@tcp(10.126.80.5:3306)/bin_manager' \
-rabbit_addr amqp://guest:guest@rabbitmq.voipbin.net:5672 \
-rabbit_queue_listen bin-manager.flow-manager.request \
-rabbit_queue_event bin-manager.flow-manager.event \
-rabbit_exchange_delay bin-manager.delay \
-redis_addr 10.164.15.220:6379 \
-redis_db 1
```

# RabbitMQ RPC

## Qeueue
Queue name: flow_manager-request

## Request
RPC request
```
{
  "uri": "<string>",
  "method": "<string>",
  "data_type": "<string>"
  "data": {...},
}
```
* uri: The target uri destination.
* method: Capitalized http methods. POST, GET, PUT, DELETE, ...
* data_type: Type of data. Mostly, "application/json".
* data: data

## Response
RPC response
```
{
  "status_code": <number>,
  "data_type": "<string>"
  "data": "{...}"
}
```
* status_code: Status code.
* data_type: Type of data.
* data: data.

# URI string

# Restful APIs

## /activeflows POST
Creates a new activeflow.

### example
```
request
{
  "uri": "/v1/activeflows",
  "method": "POST",
  "data_type": "application/json"
  "data": {"call_id": "1eb4ed62-05ef-11eb-9354-eb6fe8497be5", "flow_id": "2f68edd4-05ef-11eb-8beb-0f9f9c21b69c"}
}

response
{
  ...
}
```

## /activeflows/<id>/next GET
Returns next action of the given activeflow.

### example
```
request
{
  "uri": "/v1/activeflows/cec5b926-06a7-11eb-967e-fb463343f0a5/next",
  "method": "GET",
  "data_type": "application/json"
  "data": {"current_action_id": "6a1ce642-06a8-11eb-a632-978be835f982"}
}

response
{
  ...
}
```

## /flows/<flow-id>
Returns registered flow info.

## /flows/<flow-id>/actions
Returns registered actions.

## /flows/<flow-id>/actions/<action-id>
Returns registered action.

### example
```
request
{
  "uri": "/v1/flows/3271831e-880f-11ea-bc66-4f3de31bc41e/actions",
  "method": "GET",
  "data_type": "application/json"
  "data": {},
}

response
{
  "status_code": 200,
  "data": {
    "id": "7e4ae910-880f-11ea-b08b-dbc017c70055",
    "action": "answer",
    "next_action": "93835308-880f-11ea-97f7-f71252fc3528",
    "flow_id": "3271831e-880f-11ea-bc66-4f3de31bc41e",
    "data": {
      "action_version": "0.1"
    }
  }
}
```

# Build

Update git config
```
$ git config --global url.git@gitlab.com:.insteadOf https://gitlab.com/
or
$ git config --global url."https://<$GL_DEPLOY_USER>:<$GL_DEPLOY_TOKEN@gitlab.com>".insteadOf "https://gitlab.com"
```

Set golang
```
$ export GOPRIVATE="gitlab.com/voipbin"
```

```
$ go mod vendor
$ go build ./cmd/...
```

# Resources

## flow

```
	ID     uuid.UUID `json:"id"`
	UserID uint64    `json:"user_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Persist    bool   `json:"persist"`
	WebhookURI string `json:"webhook_uri"`

	Actions []action.Action `json:"actions"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
```

## activeflow

```
	CallID     uuid.UUID `json:"call_id"`
	FlowID     uuid.UUID `json:"flow_id"`
	UserID     uint64    `json:"user_id"`
	WebhookURI string    `json:"webhook_uri"`

	CurrentAction   action.Action `json:"current_action"`
	ExecuteCount    uint64        `json:"execute_count"`
	ForwardActionID uuid.UUID     `json:"forward_action_id"`

	Actions []action.Action `json:"actions"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
```

## action
```
	TypeAgentCall           Type = "agent_call"           // agent-manager. make a all to the agent.
	TypeAMD                 Type = "amd"                  // call-manager. answering machine detection.
	TypeAnswer              Type = "answer"               // call-manager. answer the call.
	TypeConfbridgeJoin      Type = "confbridge_join"      // call-manager. join to the confbridge.
	TypeConferenceJoin      Type = "conference_join"      // conference-manager. join to the conference.
	TypeConnect             Type = "connect"              // flow-manager. connect to the other destination.
	TypeDTMFReceive         Type = "dtmf_receive"         // call-manager. receive the dtmfs.
	TypeDTMFSend            Type = "dtmf_send"            // call-manager. send the dtmfs.
	TypeEcho                Type = "echo"                 // call-manager.
	TypeExternalMediaStart  Type = "external_media_start" // call-manager.
	TypeExternalMediaStop   Type = "external_media_stop"  // call-manager.
	TypeGoto                Type = "goto"                 // flow-manager.
	TypeHangup              Type = "hangup"               // call-manager.
	TypePatch               Type = "patch"                // flow-manager.
	TypePatchFlow           Type = "patch_flow"           // flow-manager.
	TypePlay                Type = "play"                 // call-manager.
	TypeQueueJoin           Type = "queue_join"           // flow-manager. put the call into the queue.
	TypeRecordingStart      Type = "recording_start"      // call-manager. startr the record of the given call.
	TypeRecordingStop       Type = "recording_stop"       // call-manager. stop the record of the given call.
	TypeStreamEcho          Type = "stream_echo"          // call-manager.
	TypeTalk                Type = "talk"                 // call-manager. generate audio from the given text(ssml or plain text) and play it.
	TypeTranscribeStart     Type = "transcribe_start"     // transcribe-manager. start transcribe the call
	TypeTranscribeStop      Type = "transcribe_stop"      // transcribe-manager. stop transcribe the call
	TypeTranscribeRecording Type = "transcribe_recording" // transcribe-manager. transcribe the recording and send it to webhook.
```
