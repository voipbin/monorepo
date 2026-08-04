package dbhandler

// Empty-collection normalization table for the VOIP-1078 squirrel migration.
//
// This is the plan's §5 R-A deliverable (Phase 2.5). It is keyed by
// (field × call site), NOT by field alone, because the same field behaves
// differently at different call sites — outboundconfig.DestinationWhitelist
// normalizes at Create but not at Update, and collapsing the two rows would
// silently change one of them.
//
// Phases 3, 5, 6 and 7 must consult this table field-by-field. Do NOT apply a
// blanket "normalize everything" or "normalize nothing" rule: blanket
// normalization would change channel.SIPData's current (non-normalized)
// behavior, and blanket skipping would break the read sites that do normalize.
//
// ---------------------------------------------------------------------------
// Background: what the shared helpers do with an empty/nil collection
// ---------------------------------------------------------------------------
//
// Read (commondatabasehandler.ScanRow -> copyJSON, mapping.go:426-435):
//   SQL NULL or empty string -> field left at its Go zero value (nil), no error.
//   Note this is a deliberate behavior change for outbound_config, whose
//   hand-written scan helper unconditionally json.Unmarshal'd and therefore
//   ERRORED on NULL/empty. See the outbound_config rows below.
//
// Write, map-based PrepareFields (processMapValues -> convertValueForDB with an
// empty conversion type, mapping.go:335-360 / :184-209):
//   nil slice  -> json.Marshal -> the JSON literal "null"
//   nil map    -> json.Marshal -> the JSON literal "null"
//
// Write, struct-based PrepareFields with a `,json` tag (convertValueForDB
// mapping.go:153-167):
//   nil SLICE  -> SQL NULL   (the :158-161 nil guard checks reflect.Slice)
//   nil MAP    -> "null"     (no equivalent guard for maps)
//
// So a nil slice has THREE possible stored representations ("[]" if normalized
// before the call, "null" via the map path, SQL NULL via the struct path) while
// a nil map has only TWO ("{}" if normalized, "null" otherwise).
//
// ---------------------------------------------------------------------------
// READ SIDE — current behavior that must be preserved
// ---------------------------------------------------------------------------
//
//	field                                | read call site           | normalize? | note
//	-------------------------------------+--------------------------+------------+------------------------------
//	bridge.ChannelIDs                    | bridgeGetFromRow         | YES nil->[]| was bridge.go:76-78
//	channel.Data                         | channelGetFromRow        | YES nil->{}| was channel.go:117-119
//	channel.StasisData                   | channelGetFromRow        | YES nil->{}| was channel.go:134-136
//	channel.SIPData                      | channelGetFromRow        | NO         | left nil today - do NOT add
//	outboundconfig.DestinationWhitelist  | outboundConfigGetFromRow | NO (see ->)| today ERRORS on NULL/empty
//	                                     |                          |            | (unconditional json.Unmarshal).
//	                                     |                          |            | ScanRow makes it a non-error nil.
//	                                     |                          |            | Intentional, tested (Phase 7).
//	confbridge.Flags                     | confbridgeGetFromRow     | n/a        | already ScanRow-based, untouched
//	groupcall.CallIDs / GroupcallIDs     | groupcallGetFromRow      | YES nil->[]| already ScanRow-based, untouched
//
// Because every read site that feeds the Redis cache normalizes (except
// SIPData, which is nil today and stays nil), the cache mirror keeps its
// current shape regardless of which of the three write representations a row
// was stored with. This is what bounds plan risk R4 (mixed old/new pods during
// a rolling deploy): the normalized shape is produced on read, not on write.
//
// ---------------------------------------------------------------------------
// WRITE SIDE — current behavior that must be preserved
// ---------------------------------------------------------------------------
//
//	field                                | write call site                                        | normalize? | nil result
//	-------------------------------------+--------------------------------------------------------+------------+-----------
//	confbridge.Flags                     | ConfbridgeSetFlags                                     | YES nil->[]| "[]"
//	groupcall.CallIDs                    | GroupcallSetCallIDsAndCallCountAndDialIndex            | YES nil->[]| "[]"
//	groupcall.GroupcallIDs               | GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex  | YES nil->[]| "[]"
//	outboundconfig.DestinationWhitelist  | OutboundConfigCreate                                   | YES nil->[]| "[]"
//	outboundconfig.DestinationWhitelist  | OutboundConfigUpdate                                   | NO         | "null"
//	bridge.ChannelIDs                    | BridgeCreate                                           | NO         | see (a)
//	channel.Data / SIPData / StasisData  | ChannelCreate                                          | NO         | "null"
//	channel.Data                         | ChannelSetData                                         | NO         | "null"
//	channel.SIPData                      | ChannelSetSIPData                                      | NO         | "null"
//	channel.StasisData                   | ChannelSetStasisInfo                                   | NO         | "null"
//
// (a) bridge.ChannelIDs at BridgeCreate is the one site whose STORED
// representation changes: it is a nil SLICE going through struct-based
// PrepareFields with a `,json` tag, so it now stores SQL NULL where the
// pre-migration json.Marshal stored the literal "null". This is not
// observable through any dbhandler read path or the cache, because
// bridgeGetFromRow normalizes both NULL and "null" to []string{} (see the read
// table above). Called out explicitly so a reviewer evaluates it deliberately
// rather than discovering it later.
//
// The nil-map sites (channel.Data/SIPData/StasisData) keep storing the literal
// "null" exactly as before, since convertValueForDB has no nil-map guard.
//
// ---------------------------------------------------------------------------
//
// Write sites marked "YES" above normalize through normalizeSlice below so the
// decision is made in one named, greppable place instead of an inline nil check
// repeated per call site.

// normalizeSlice returns an empty, non-nil slice when s is nil, so the value
// marshals to the JSON literal "[]" rather than "null" (map path) or SQL NULL
// (struct path with a ,json tag).
//
// Only call this at sites marked "YES" in the write table above. Adding it to a
// "NO" site is a behavior change, not a cleanup.
func normalizeSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
