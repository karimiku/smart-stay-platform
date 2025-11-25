package main

import (
	"context"
	"log"
	"math/rand"
	"strconv"
	"time"

	pb "github.com/karimiku/smart-stay-platform/pkg/genproto/key"
)

type server struct {
	pb.UnimplementedKeyServiceServer
}

// GenerateKey generates a time-sensitive PIN code for a specific reservation.
// In a real implementation, this would call an external Smart Lock API (e.g., RemoteLock, NinjaLock).
func (s *server) GenerateKey(ctx context.Context, req *pb.GenerateKeyRequest) (*pb.GenerateKeyResponse, error) {
	log.Printf("🔑 Generating Key for Reservation: %s (Valid: %s - %s)",
		req.ReservationId, req.ValidFrom.AsTime(), req.ValidUntil.AsTime())

	// TODO: Integrate with actual Smart Lock API here.
	
	// Simulate PIN generation (4-digit code)
	// Seed the random number generator (in production, use crypto/rand)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pin := 1000 + r.Intn(9000) // Generates a number between 1000 and 9999

	// Return the dummy key credentials
	return &pb.GenerateKeyResponse{
		KeyCode:  strconv.Itoa(pin),
		DeviceId: "smart-lock-device-001",
	}, nil
}

// RevokeKey immediately invalidates a key for a given reservation.
// This is critical for security scenarios like check-out or cancellation.
func (s *server) RevokeKey(ctx context.Context, req *pb.RevokeKeyRequest) (*pb.RevokeKeyResponse, error) {
	log.Printf("🚫 Revoking Key for Reservation: %s", req.ReservationId)

	// TODO: Call Smart Lock API to delete/disable the key.

	return &pb.RevokeKeyResponse{
		Success: true,
	}, nil
}