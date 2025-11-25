package main

import (
	"context"
	"log"

	"github.com/google/uuid"
	pb "github.com/karimiku/smart-stay-platform/pkg/genproto/auth"
)

// server implements the AuthServiceServer interface generated from protobuf.
type server struct {
	pb.UnimplementedAuthServiceServer
}

// Register creates a new user account.
func (s *server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	log.Printf("📝 Register request received for email: %s", req.Email)

	// TODO: Implement DB insertion and password hashing here.
	
	// Generate UUID for new user
	userID := uuid.New().String()
	return &pb.RegisterResponse{
		UserId: userID,
	}, nil
}

// Login authenticates a user and returns a token.
func (s *server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	log.Printf("🔑 Login request received for email: %s", req.Email)

	// TODO: Implement DB lookup and password verification here.

	// Return a dummy JWT token for now
	return &pb.LoginResponse{
		AccessToken: "dummy-jwt-token-example",
		ExpiresIn:   3600,
	}, nil
}

// Validate checks if the token is valid.
func (s *server) Validate(ctx context.Context, req *pb.ValidateRequest) (*pb.ValidateResponse, error) {
	log.Printf("🛡️ Validate request received for token: %s", req.AccessToken)

	// TODO: Implement JWT verification logic here.

	// Return valid response for dummy token
	isValid := req.AccessToken == "dummy-jwt-token-example"
	// Dummy user ID (in production, extract from JWT token)
	dummyUserID := "550e8400-e29b-41d4-a716-446655440000"
	return &pb.ValidateResponse{
		Valid:  isValid,
		UserId: dummyUserID,
		Role:   "guest",
	}, nil
}