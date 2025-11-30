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

	"cloud.google.com/go/pubsub"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/karimiku/smart-stay-platform/pkg/genproto/reservation"
	"github.com/karimiku/smart-stay-platform/internal/database"
)

const (
	defaultTopicID = "reservation-events" // デフォルトのトピック名
)

func main() {
	// 1. Port Configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "50052"
	}

	// 2. Connect to PostgreSQL database
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

	// 3. Get Google Cloud Project ID from environment
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("GCP_PROJECT")
	}
	if projectID == "" {
		log.Fatalf("❌ GOOGLE_CLOUD_PROJECT or GCP_PROJECT environment variable is required")
	}
	log.Printf("✅ Using Google Cloud Project: %s", projectID)

	// 4. Get Pub/Sub Topic ID from environment
	topicID := os.Getenv("PUBSUB_TOPIC_ID")
	if topicID == "" {
		topicID = defaultTopicID
	}
	log.Printf("✅ Using Pub/Sub Topic: %s", topicID)

	// 5. Initialize Pub/Sub Client
	// PUBSUB_EMULATOR_HOST environment variable is automatically handled by the client library.
	pubsubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Pub/Sub client: %v", err)
	}
	defer pubsubClient.Close()

	// 6. Create Topic if not exists (Idempotent)
	topic := pubsubClient.Topic(topicID)
	exists, err := topic.Exists(ctx)
	if err != nil {
		log.Fatalf("Failed to check if topic exists: %v", err)
	}
	if !exists {
		log.Printf("📢 Creating topic: %s", topicID)
		_, err = pubsubClient.CreateTopic(ctx, topicID)
		if err != nil {
			log.Fatalf("Failed to create topic: %v", err)
		}
	}

	// 7. Start TCP Listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 8. Create gRPC Server & Register Service
	grpcServer := grpc.NewServer()
	
	// Pass the topic and database queries to the service implementation
	queries := database.New(dbPool)
	svc := &server{
		pubsubTopic: topic,
		queries:     queries,
	}
	pb.RegisterReservationServiceServer(grpcServer, svc)

	reflection.Register(grpcServer)

	// 9. Start Server
	go func() {
		log.Printf("📝 Reservation Service is running on port %s", port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 10. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down Reservation Service...")
	grpcServer.GracefulStop()
	log.Println("✅ Reservation Service stopped")
}