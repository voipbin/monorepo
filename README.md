# flow-manager

The flow-manager manages the flows.

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
    "flow_revision": "d96e918e-880f-11ea-accd-f3a1e1ceaddc",
    "data": {
      "action_version": "0.1"
    }
  }
}
```
