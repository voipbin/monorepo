package server

type Server interface{}

type server struct{}

func NewServer() Server {
	return &server{}
}
