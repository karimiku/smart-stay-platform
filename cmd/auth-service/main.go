package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/karimiku/smart-stay-platform/pkg/genproto/auth"
)

func main() {
	// 1. Configure the port (Default: 50051)
	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	// 2. Initialize the listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 3. Create a new gRPC server instance
	grpcServer := grpc.NewServer()

	// 4. Register the AuthService implementation
	// We pass the struct defined in service.go
	authService := &server{}
	pb.RegisterAuthServiceServer(grpcServer, authService)

	// 5. Enable Server Reflection (Useful for debugging with tools like Evans or Postman)
	reflection.Register(grpcServer)

	// 6. Start the server with graceful shutdown handling
	go func() {
		log.Printf("🚀 Auth Service is running on port %s", port)
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