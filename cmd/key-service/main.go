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

	"cloud.google.com/go/pubsub"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/karimiku/smart-stay-platform/internal/events"
	pb "github.com/karimiku/smart-stay-platform/pkg/genproto/key"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	projectID      = "smart-stay-local"
	topicID        = "reservation-events"
	subscriptionID = "key-service-subscription"
)

func main() {
	// 1. Port Configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "50053"
	}

	// 2. Initialize Pub/Sub Client
	ctx := context.Background()
	pubsubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Pub/Sub client: %v", err)
	}
	defer pubsubClient.Close()

	// 3. Create/Get Subscription
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

	// 4. Start Pub/Sub Listener (in background)
	keySvc := &server{}
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
				// Use reservation start/end dates (simplified: use current time + 24h)
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

	// 5. Start gRPC Server
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

	// 6. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down server...")
	grpcServer.GracefulStop()
	log.Println("✅ Server stopped")
}