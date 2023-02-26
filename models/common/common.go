package common

// StasisStart event's context types
const (
	ContextIncomingCall  = "call-in"            // context for the incoming channel
	ContextOutgoingCall  = "call-out"           // context for the outgoing channel
	ContextRecording     = "call-record"        // context for the channel which created only for recording
	ContextServiceCall   = "call-svc"           // context for the channel where it came back to stasis from the other asterisk application
	ContextJoinCall      = "call-join"          // context for the channel for conference joining
	ContextExternalMedia = "call-externalmedia" // context for the external media channel. this channel will get the media from the external
	ContextExternalSoop  = "call-externalsnoop" // context for the external snoop channel
	ContextApplication   = "call-application"   // context for dialplan application execution
)

// list of domain defines
const (
	DomainConference = "conference.voipbin.net"
	DomainPSTN       = "pstn.voipbin.net"
	DomainSIPSuffix  = ".sip.voipbin.net"
)
