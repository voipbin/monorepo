# Add Channel SIPData from Kamailio Redis

## Problem

Kamailio writes SIP call metadata to Redis as a hash (`kamailio:<sip-call-id>`) containing fields like `source`, `transport`, `domain`, `rtpengine_address`, and `direction`. The call-manager already reads this hash in `channelhandler/db.go` via `KamailioMetadataGet()`, but only logs the data — it is not stored anywhere accessible.

## Approach

Add a `SIPData map[string]string` field to the Channel struct. When the Kamailio metadata is retrieved from Redis, store it into this field via a new DB setter. The data is then available on the Channel for any downstream logic.

- **Channel only** — no changes to the Call model. Call can access SIP data via its `ChannelID` reference.
- **`nil` when empty** — no initialization needed; the field is only populated when Kamailio metadata exists.
- **All hash fields stored** — `source`, `transport`, `domain`, `rtpengine_address`, `direction` (the full Redis hash).

## Changes

1. **Channel struct** (`models/channel/main.go`) — add `SIPData map[string]string` field
2. **DB handler** (`pkg/dbhandler/`) — add `ChannelSetSIPData` method
3. **`UpdateSIPInfo`** (`pkg/channelhandler/db.go`) — store Kamailio metadata into channel's SIPData
4. **`UpdateSIPInfoByChannelVariable`** (`pkg/channelhandler/db.go`) — same

## Not Changing

- Call model — access SIP data via Channel when needed
- Kamailio config — reading existing hash as-is
- OpenAPI/API — Channel is internal to call-manager
