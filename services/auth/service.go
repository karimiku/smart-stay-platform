package main

import (
	"context"
	"log"

	// ⚠️ Replace with your actual GitHub username
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
	
	// Return a dummy user ID for now
	return &pb.RegisterResponse{
		UserId: 1001,
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
	return &pb.ValidateResponse{
		Valid:  isValid,
		UserId: 1001,
		Role:   "guest",
	}, nil
}