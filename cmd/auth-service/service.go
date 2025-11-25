package main

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	pb "github.com/karimiku/smart-stay-platform/pkg/genproto/auth"
	
	"github.com/karimiku/smart-stay-platform/cmd/auth-service/jwt"
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

// Login authenticates a user and returns a JWT token.
func (s *server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	log.Printf("🔑 Login request received for email: %s", req.Email)

	// TODO: Implement DB lookup and password verification here.
	// For now, we'll generate a token for any email (demo purposes)
	
	// Generate user ID (in production, get from DB)
	userID := uuid.New().String()
	
	// Generate JWT token
	expiresIn := 3600 * time.Second // 1 hour
	token, err := jwt.GenerateToken(userID, "guest", req.Email, expiresIn)
	if err != nil {
		log.Printf("❌ Failed to generate JWT token: %v", err)
		return nil, err
	}

	log.Printf("✅ JWT token generated for user: %s", userID)
	return &pb.LoginResponse{
		AccessToken: token,
		ExpiresIn:   int64(expiresIn.Seconds()),
	}, nil
}

// Validate checks if the JWT token is valid and extracts user information.
func (s *server) Validate(ctx context.Context, req *pb.ValidateRequest) (*pb.ValidateResponse, error) {
	tokenPreview := req.AccessToken
	if len(tokenPreview) > 20 {
		tokenPreview = tokenPreview[:20] + "..."
	}
	log.Printf("🛡️ Validate request received for token: %s", tokenPreview)

	// Validate JWT token
	claims, err := jwt.ValidateToken(req.AccessToken)
	if err != nil {
		log.Printf("❌ Token validation failed: %v", err)
		return &pb.ValidateResponse{
			Valid:  false,
			UserId: "",
			Role:   "",
		}, nil
	}

	log.Printf("✅ Token validated for user: %s, role: %s", claims.UserID, claims.Role)
	return &pb.ValidateResponse{
		Valid:  true,
		UserId: claims.UserID,
		Role:   claims.Role,
	}, nil
}