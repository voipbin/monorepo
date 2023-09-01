package helphandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package helphandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

// HelpHandler define
type HelpHandler interface {
	HashGenerate(password string) (string, error)
	HashCheck(password, hashString string) bool
}

type helpHandler struct {
}

// NewHelpHandler returns HelpHandler
func NewHelpHandler() HelpHandler {
	return &helpHandler{}
}
