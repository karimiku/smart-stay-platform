package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/karimiku/smart-stay-platform/internal/database"
	pb "github.com/karimiku/smart-stay-platform/pkg/genproto/key"
)

type server struct {
	pb.UnimplementedKeyServiceServer
	queries *database.Queries
}

// GenerateKey generates a time-sensitive PIN code for a specific reservation.
// In a real implementation, this would call an external Smart Lock API (e.g., RemoteLock, NinjaLock).
func (s *server) GenerateKey(ctx context.Context, req *pb.GenerateKeyRequest) (*pb.GenerateKeyResponse, error) {
	log.Printf("🔑 Generating Key for Reservation: %s (Valid: %s - %s)",
		req.ReservationId, req.ValidFrom.AsTime(), req.ValidUntil.AsTime())

	// Parse reservation_id and user_id from request
	resUUID, err := stringToUUID(req.ReservationId)
	if err != nil {
		return nil, errors.New("invalid reservation_id format")
	}

	// Get reservation to get user_id
	reservation, err := s.queries.GetReservation(ctx, resUUID)
	if err != nil {
		return nil, errors.New("reservation not found")
	}

	// TODO: Integrate with actual Smart Lock API here.
	
	// Generate secure PIN code (4-digit code: 1000-9999)
	// Using crypto/rand for cryptographically secure random number generation
	maxPin := big.NewInt(9000) // 0-8999 range
	randomNum, err := rand.Int(rand.Reader, maxPin)
	if err != nil {
		log.Printf("❌ Failed to generate secure PIN: %v", err)
		return nil, errors.New("failed to generate secure key code")
	}
	pin := 1000 + int(randomNum.Int64()) // Generates a number between 1000 and 9999

	keyCode := strconv.Itoa(pin)
	// Get device ID from environment variable, with default fallback
	deviceID := os.Getenv("SMART_LOCK_DEVICE_ID")
	if deviceID == "" {
		deviceID = "smart-lock-device-001"
	}

	// Convert timestamps
	validFrom := pgtype.Timestamp{
		Time:  req.ValidFrom.AsTime(),
		Valid: true,
	}
	validUntil := pgtype.Timestamp{
		Time:  req.ValidUntil.AsTime(),
		Valid: true,
	}

	// Store key in database
	_, err = s.queries.CreateKey(ctx, database.CreateKeyParams{
		ReservationID: resUUID,
		UserID:        reservation.UserID,
		KeyCode:       keyCode,
		DeviceID:      deviceID,
		ValidFrom:     validFrom,
		ValidUntil:    validUntil,
	})
	if err != nil {
		log.Printf("❌ Failed to create key in database: %v", err)
		return nil, errors.New("failed to create key")
	}

	return &pb.GenerateKeyResponse{
		KeyCode:  keyCode,
		DeviceId: deviceID,
	}, nil
}

// RevokeKey immediately invalidates a key for a given reservation.
// This is critical for security scenarios like check-out or cancellation.
func (s *server) RevokeKey(ctx context.Context, req *pb.RevokeKeyRequest) (*pb.RevokeKeyResponse, error) {
	log.Printf("🚫 Revoking Key for Reservation: %s", req.ReservationId)

	// TODO: Call Smart Lock API to delete/disable the key.
	// For now, we just return success (in production, we'd update the key status in DB)

	return &pb.RevokeKeyResponse{
		Success: true,
	}, nil
}

// ListKeys retrieves all keys for a user
func (s *server) ListKeys(ctx context.Context, req *pb.ListKeysRequest) (*pb.ListKeysResponse, error) {
	userUUID, err := stringToUUID(req.UserId)
	if err != nil {
		return nil, errors.New("invalid user_id format")
	}

	dbKeys, err := s.queries.ListActiveKeysByUserID(ctx, userUUID)
	if err != nil {
		log.Printf("❌ Failed to list keys: %v", err)
		return nil, errors.New("failed to list keys")
	}

	var keys []*pb.Key
	for _, dbKey := range dbKeys {
		keys = append(keys, dbKeyToProto(dbKey))
	}

	return &pb.ListKeysResponse{
		Keys: keys,
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

// dbKeyToProto converts database Key to protobuf Key
func dbKeyToProto(dbKey database.Key) *pb.Key {
	var validFrom, validUntil *timestamppb.Timestamp
	if dbKey.ValidFrom.Valid {
		validFrom = timestamppb.New(dbKey.ValidFrom.Time)
	}
	if dbKey.ValidUntil.Valid {
		validUntil = timestamppb.New(dbKey.ValidUntil.Time)
	}

	return &pb.Key{
		KeyCode:       dbKey.KeyCode,
		DeviceId:      dbKey.DeviceID,
		ReservationId: uuidToString(dbKey.ReservationID),
		ValidFrom:     validFrom,
		ValidUntil:    validUntil,
	}
}