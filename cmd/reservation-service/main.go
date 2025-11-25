package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"cloud.google.com/go/pubsub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	// ⚠️ Replace [yourusername]
	pb "github.com/karimiku/smart-stay-platform/pkg/genproto/reservation"
)

const (
	projectID = "smart-stay-local"
	topicID   = "reservation-events" // イベントを流すトピック名
)

func main() {
	// 1. Port Configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "50052"
	}

	// 2. Initialize Pub/Sub Client
	// PUBSUB_EMULATOR_HOST environment variable is automatically handled by the client library.
	ctx := context.Background()
	pubsubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Pub/Sub client: %v", err)
	}
	defer pubsubClient.Close()

	// 3. Create Topic if not exists (Idempotent)
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

	// 4. Start TCP Listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 5. Create gRPC Server & Register Service
	grpcServer := grpc.NewServer()
	
	// Pass the topic to the service implementation
	svc := &server{
		pubsubTopic: topic,
	}
	pb.RegisterReservationServiceServer(grpcServer, svc)

	reflection.Register(grpcServer)

	// 6. Start Server
	go func() {
		log.Printf("📝 Reservation Service is running on port %s", port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 7. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down Reservation Service...")
	grpcServer.GracefulStop()
	log.Println("✅ Reservation Service stopped")
}