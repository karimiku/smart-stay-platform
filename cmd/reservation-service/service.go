package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"cloud.google.com/go/pubsub"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/karimiku/smart-stay-platform/internal/database"
	"github.com/karimiku/smart-stay-platform/internal/events"
	pb "github.com/karimiku/smart-stay-platform/pkg/genproto/reservation"
)

// server implements the ReservationServiceServer interface.
type server struct {
	pb.UnimplementedReservationServiceServer
	pubsubTopic *pubsub.Topic // Reference to the Pub/Sub topic
	queries     *database.Queries
}

// CreateReservation handles new booking requests.
func (s *server) CreateReservation(ctx context.Context, req *pb.CreateReservationRequest) (*pb.CreateReservationResponse, error) {
	log.Printf("📝 Received CreateReservation request. User: %s, Room: %d", req.UserId, req.RoomId)

	// 1. Parse user_id from string to UUID
	userUUID, err := stringToUUID(req.UserId)
	if err != nil {
		return nil, errors.New("invalid user_id format")
	}

	// 2. Calculate total price (simplified: 1 night = 50000 yen)
	days := int(req.EndDate.AsTime().Sub(req.StartDate.AsTime()).Hours() / 24)
	if days < 1 {
		days = 1
	}
	totalPrice := int64(days * 50000)

	// 3. Convert timestamps
	startTimestamp := pgtype.Timestamp{
		Time:  req.StartDate.AsTime(),
		Valid: true,
	}
	endTimestamp := pgtype.Timestamp{
		Time:  req.EndDate.AsTime(),
		Valid: true,
	}

	// 4. Create reservation in database
	dbReservation, err := s.queries.CreateReservation(ctx, database.CreateReservationParams{
		UserID:     userUUID,
		RoomID:     req.RoomId,
		StartDate:  startTimestamp,
		EndDate:    endTimestamp,
		TotalPrice: totalPrice,
		Status:     "PENDING",
	})
	if err != nil {
		log.Printf("❌ Failed to create reservation: %v", err)
		return nil, errors.New("failed to create reservation")
	}

	resID := uuidToString(dbReservation.ID)

	// 5. Publish Event to Pub/Sub (Asynchronous)
	// We don't wait for Key Service here. We just shout "Created!" and return.
	event := events.EventPayload{
		EventType:     events.EventTypeReservationCreated,
		ReservationID: resID,
		UserID:        req.UserId,
		StartDate:     req.StartDate.AsTime(),
		EndDate:       req.EndDate.AsTime(),
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

	// 6. Return Response (Immediately PENDING)
	return &pb.CreateReservationResponse{
		ReservationId: resID,
		Status:        pb.ReservationStatus_PENDING,
	}, nil
}

// GetReservation retrieves a reservation by ID
func (s *server) GetReservation(ctx context.Context, req *pb.GetReservationRequest) (*pb.GetReservationResponse, error) {
	resUUID, err := stringToUUID(req.ReservationId)
	if err != nil {
		return nil, errors.New("invalid reservation_id format")
	}

	dbReservation, err := s.queries.GetReservation(ctx, resUUID)
	if err != nil {
		return nil, errors.New("reservation not found")
	}

	reservation := dbReservationToProto(dbReservation)
	return &pb.GetReservationResponse{
		Reservation: reservation,
	}, nil
}

// ListReservations retrieves all reservations for a user
func (s *server) ListReservations(ctx context.Context, req *pb.ListReservationsRequest) (*pb.ListReservationsResponse, error) {
	userUUID, err := stringToUUID(req.UserId)
	if err != nil {
		return nil, errors.New("invalid user_id format")
	}

	dbReservations, err := s.queries.ListReservationsByUserID(ctx, userUUID)
	if err != nil {
		log.Printf("❌ Failed to list reservations: %v", err)
		return nil, errors.New("failed to list reservations")
	}

	var reservations []*pb.Reservation
	for _, dbRes := range dbReservations {
		reservations = append(reservations, dbReservationToProto(dbRes))
	}

	return &pb.ListReservationsResponse{
		Reservations: reservations,
	}, nil
}

// Helper functions

// stringToUUID converts string UUID to pgtype.UUID
func stringToUUID(s string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	err := uuid.Scan(s)
	return uuid, err
}

// uuidToString converts pgtype.UUID to string format
func uuidToString(uuid pgtype.UUID) string {
	if !uuid.Valid {
		return ""
	}
	if len(uuid.Bytes) == 16 {
		// Format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			uuid.Bytes[0:4],
			uuid.Bytes[4:6],
			uuid.Bytes[6:8],
			uuid.Bytes[8:10],
			uuid.Bytes[10:16])
	}
	// Fallback: use hex encoding
	return hex.EncodeToString(uuid.Bytes[:])
}

// dbReservationToProto converts database Reservation to protobuf Reservation
func dbReservationToProto(dbRes database.Reservation) *pb.Reservation {
	status := pb.ReservationStatus_PENDING
	switch dbRes.Status {
	case "CONFIRMED":
		status = pb.ReservationStatus_CONFIRMED
	case "CANCELLED":
		status = pb.ReservationStatus_CANCELLED
	case "COMPLETED":
		status = pb.ReservationStatus_COMPLETED
	}

	var startDate, endDate *timestamppb.Timestamp
	if dbRes.StartDate.Valid {
		startDate = timestamppb.New(dbRes.StartDate.Time)
	}
	if dbRes.EndDate.Valid {
		endDate = timestamppb.New(dbRes.EndDate.Time)
	}

	return &pb.Reservation{
		Id:         uuidToString(dbRes.ID),
		UserId:     uuidToString(dbRes.UserID),
		RoomId:     dbRes.RoomID,
		StartDate:  startDate,
		EndDate:    endDate,
		TotalPrice: dbRes.TotalPrice,
		Status:     status,
	}
}