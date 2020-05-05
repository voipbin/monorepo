package ari

// StasisStart ARI event struct
type StasisStart struct {
	Event
	Args           ArgsMap `json:"args"`
	Channel        Channel `json:"channel"`
	ReplaceChannel Channel `json:"replace_channel"`
}
