package address

// DeriveEndpoints maps an absolute (source, destination) address pair to a
// relative (peer, local) pair using direction. This is the single shared
// authority for the direction-swap rule used across the monorepo wherever an
// absolute source/destination event pair (call, conversation message, etc.)
// needs to be reinterpreted from "our own" perspective (see VOIP-1215 §3.0a
// for the original contract definition on conversation messages; the same
// rule applies identically to call-manager's Source/Destination).
//
//   - incoming: the remote party is the source (they called/wrote us), so
//     peer = source, local = destination.
//   - outgoing: the remote party is the destination (we called/wrote them),
//     so peer = destination, local = source.
//   - anything else (empty string, unrecognized value): both return values
//     are zero. The caller decides whether to still persist the row (the
//     direction field itself carries the raw value for diagnostics) or skip
//     it — this function never guesses.
func DeriveEndpoints(direction string, source, destination Address) (peer Address, local Address) {
	switch direction {
	case "incoming":
		return source, destination
	case "outgoing":
		return destination, source
	default:
		return Address{}, Address{}
	}
}
