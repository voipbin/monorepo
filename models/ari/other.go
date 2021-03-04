package ari

// StasisEnd ARI event struct
type StasisEnd struct {
	Event
	Channel Channel `json:"channel"`
}

// StasisStart ARI event struct
type StasisStart struct {
	Event
	Args           ArgsMap `json:"args"`
	Channel        Channel `json:"channel"`
	ReplaceChannel Channel `json:"replace_channel"`
}
