package main

import (
	"log"
	"net"
	"smart-stay-platform/chat"

	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Failed to listen on port 9000 : %s", err)
	}
	grpcServer := grpc.NewServer()
	s := chat.Server{}
	chat.RegisterChatServiceServer(grpcServer, &s)

	
	

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gPRC server over port 9000: %y", err)
	}
}
