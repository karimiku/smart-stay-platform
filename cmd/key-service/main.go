package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/karimiku/smart-stay-platform/internal/database"
	"github.com/karimiku/smart-stay-platform/internal/events"
	pb "github.com/karimiku/smart-stay-platform/pkg/genproto/key"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	defaultTopicID        = "reservation-events"
	defaultSubscriptionID = "key-service-subscription"
)

func main() {
	// 1. Port Configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "50053"
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

	// 5. Get Pub/Sub Subscription ID from environment
	subscriptionID := os.Getenv("PUBSUB_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		subscriptionID = defaultSubscriptionID
	}
	log.Printf("✅ Using Pub/Sub Subscription: %s", subscriptionID)

	// 6. Initialize Pub/Sub Client
	pubsubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Pub/Sub client: %v", err)
	}
	defer pubsubClient.Close()

	// 7. Create/Get Subscription
	topic := pubsubClient.Topic(topicID)
	sub := pubsubClient.Subscription(subscriptionID)
	
	exists, err := sub.Exists(ctx)
	if err != nil {
		log.Fatalf("Failed to check if subscription exists: %v", err)
	}
	if !exists {
		log.Printf("Creating subscription: %s", subscriptionID)
		sub, err = pubsubClient.CreateSubscription(ctx, subscriptionID, pubsub.SubscriptionConfig{
			Topic: topic,
		})
		if err != nil {
			log.Fatalf("Failed to create subscription: %v", err)
		}
	}

	// 8. Start Pub/Sub Listener (in background)
	queries := database.New(dbPool)
	keySvc := &server{
		queries: queries,
	}
	go func() {
		log.Printf(" Started listening to Pub/Sub subscription: %s", subscriptionID)
		err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
			log.Printf(" Received message: ID=%s", msg.ID)
			
			// Parse event payload
			var event events.EventPayload
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				log.Printf(" Failed to parse event: %v", err)
				msg.Nack()
				return
			}

			// Process event
			if event.EventType == events.EventTypeReservationCreated {
				log.Printf("🔑 Processing ReservationCreated event for reservation: %s", event.ReservationID)
				
				// Generate key for the reservation
				// Use reservation start/end dates
				// Note: UserID is retrieved from reservation in GenerateKey method
				validFrom := event.StartDate
				validUntil := event.EndDate
				
				_, err := keySvc.GenerateKey(ctx, &pb.GenerateKeyRequest{
					ReservationId: event.ReservationID,
					ValidFrom:     timestamppb.New(validFrom),
					ValidUntil:    timestamppb.New(validUntil),
				})
				
				if err != nil {
					log.Printf(" Failed to generate key: %v", err)
					msg.Nack()
					return
				}
				
				log.Printf(" Key generated successfully for reservation: %s", event.ReservationID)
			}
			
			msg.Ack()
		})
		if err != nil {
			log.Fatalf("Failed to receive messages: %v", err)
		}
	}()

	// 9. Start gRPC Server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterKeyServiceServer(grpcServer, keySvc)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("🚀 Key Service is running on port %s", port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 10. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down server...")
	grpcServer.GracefulStop()
	log.Println("✅ Server stopped")
}