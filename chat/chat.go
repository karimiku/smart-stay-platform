package chat

import (
	context "context"
	"log"
)

type Server struct {
	UnimplementedChatServiceServer

}

//レシーバをs *ServerでServer用のってことね。
func (s *Server) SayHello(ctx context.Context, message *Message) (*Message, error) {
	log.Printf("Received message body from client: %s",message)
	return &Message{Body: "Hello From the server!"}, nil
}