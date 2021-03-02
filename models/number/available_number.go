package number

// AvailableNumber struct represent number information
type AvailableNumber struct {
	Number string `json:"number"`

	Country    string   `json:"country"`
	Region     string   `json:"region"`
	PostalCode string   `json:"postal_code"`
	Features   []string `json:"features"`
}
