package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/karimiku/smart-stay-platform/pkg/genproto/key"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {

	port := os.Getenv("PORT")
	if port == "" {
		port = "50053"
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	keyService := &server{}

	pb.RegisterKeyServiceServer(grpcServer, keyService)

	reflection.Register(grpcServer)

	go func () {
		log.Printf("🚀 Key Service is running on port %s", port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

		// Wait for interrupt signal to gracefully shutdown the server
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("🛑 Shutting down server...")
		grpcServer.GracefulStop()
		log.Println("✅ Server stopped")
}