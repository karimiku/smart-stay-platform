package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	pb "github.com/karimiku/smart-stay-platform/pkg/genproto/reservation"
)

// server implements the ReservationServiceServer interface.
type server struct {
	pb.UnimplementedReservationServiceServer
	pubsubTopic *pubsub.Topic // Reference to the Pub/Sub topic
}

// EventPayload defines the structure of the message sent to Pub/Sub.
type EventPayload struct {
	EventType     string 	`json:"event_type"`
	ReservationID string 	`json:"reservation_id"`
	UserID        string 	`json:"user_id"` // UUID
	StartDate	  time.Time `json:"start_date"`
	EndDate		  time.Time `json:"end_date"`
}

// CreateReservation handles new booking requests.
func (s *server) CreateReservation(ctx context.Context, req *pb.CreateReservationRequest) (*pb.CreateReservationResponse, error) {
	log.Printf("📝 Received CreateReservation request. User: %s, Room: %d", req.UserId, req.RoomId)

	// 1. Simulate DB Insert (Generate a dummy ID)
	resID := uuid.New().String()

	// 2. Publish Event to Pub/Sub (Asynchronous)
	// We don't wait for Key Service here. We just shout "Created!" and return.
	event := EventPayload{
		EventType:     "ReservationCreated",
		ReservationID: resID,
		UserID:        req.UserId,
		StartDate:     req.StartDate.AsTime(),
		EndDate:  	   req.EndDate.AsTime(),
	}
	eventData, err := json.Marshal(event)
	if err != nil {
		log.Printf("failed to marshal event: %v", err)
	}

	// Publish the message
	result := s.pubsubTopic.Publish(ctx, &pubsub.Message{
		Data: eventData,
		Attributes: map[string]string{
			"origin": "reservation-service",
		},
	})

	// Optionally wait for the publish result (or handle it in background)
	// For high throughput, you might just fire and forget, but here we check for errors.
	id, err := result.Get(ctx)
	if err != nil {
		log.Printf("❌ Failed to publish event: %v", err)
		// Note: Even if publish fails, we might still return success if DB commit worked,
		// but in a robust Saga, we'd need an outbox pattern.
	} else {
		log.Printf("📢 Published event ID: %s", id)
	}

	// 3. Return Response (Immediately PENDING)
	return &pb.CreateReservationResponse{
		ReservationId: resID,
		Status:        pb.ReservationStatus_PENDING,
	}, nil
}

// GetReservation (Dummy implementation)
func (s *server) GetReservation(ctx context.Context, req *pb.GetReservationRequest) (*pb.GetReservationResponse, error) {
	return &pb.GetReservationResponse{
		Reservation: &pb.Reservation{
			Id:         req.ReservationId,
			UserId:     "550e8400-e29b-41d4-a716-446655440000", // Dummy UUID
			RoomId:     505,
			StartDate:  timestamppb.New(time.Now()),
			EndDate:    timestamppb.New(time.Now().Add(24 * time.Hour)),
			TotalPrice: 50000,
			Status:     pb.ReservationStatus_CONFIRMED,
		},
	}, nil
}