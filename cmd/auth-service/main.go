package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/karimiku/smart-stay-platform/internal/database"
	pb "github.com/karimiku/smart-stay-platform/pkg/genproto/auth"
)

func main() {
	// 1. Validate JWT secret is set (will panic if not set)
	// JWT_SECRET MUST be set via environment variable
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatalf("❌ JWT_SECRET environment variable is required. Please set it in .env file or environment.")
	}
	log.Println("✅ JWT_SECRET loaded from environment")

	// 2. Configure the port (Default: 50051)
	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	// 3. Initialize the listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 4. Connect to PostgreSQL database
	// DATABASE_URL MUST be set via environment variable
	// For local development, create .env file from .env.example
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatalf("❌ DATABASE_URL environment variable is required. Please set it in .env file or environment.")
	}
	log.Println("✅ DATABASE_URL loaded from environment")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Test database connection
	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}
	log.Println("✅ Database connection established")

	// 5. Create a new gRPC server instance
	grpcServer := grpc.NewServer()

	// 6. Register the AuthService implementation
	// We pass the database connection to the server
	queries := database.New(dbPool)
	authService := &server{
		queries: queries,
	}
	pb.RegisterAuthServiceServer(grpcServer, authService)

	// 7. Enable Server Reflection (Useful for debugging with tools like Evans or Postman)
	reflection.Register(grpcServer)

	// 8. Start the server with graceful shutdown handling
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