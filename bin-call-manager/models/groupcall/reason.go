package groupcall

// ReasonExtensionNotRegistered is the reason of the typed error that the dial returns when the destination extension has
// no registered contact. The caller of a call creation reads it to tell this failure from the others.
const ReasonExtensionNotRegistered = "EXTENSION_NOT_REGISTERED"
