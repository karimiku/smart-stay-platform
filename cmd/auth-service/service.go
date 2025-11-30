package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	pb "github.com/karimiku/smart-stay-platform/pkg/genproto/auth"
	"golang.org/x/crypto/bcrypt"

	"github.com/karimiku/smart-stay-platform/cmd/auth-service/jwt"
	"github.com/karimiku/smart-stay-platform/internal/database"
)

// server implements the AuthServiceServer interface generated from protobuf.
type server struct {
	pb.UnimplementedAuthServiceServer
	queries *database.Queries
}

// Register creates a new user account.
func (s *server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	log.Printf("📝 Register request received for email: %s", req.Email)

	// Validate input
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		return nil, errors.New("email is required")
	}
	if strings.TrimSpace(req.Password) == "" {
		return nil, errors.New("password is required")
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("name is required")
	}

	// Hash password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("❌ Failed to hash password: %v", err)
		return nil, errors.New("failed to process password")
	}

	// Create user in database
	user, err := s.queries.CreateUser(ctx, database.CreateUserParams{
		Email:          req.Email,
		HashedPassword: hashedPassword,
		Name:           strings.TrimSpace(req.Name),
		Role:           "guest", // Default role
	})
	if err != nil {
		log.Printf("❌ Failed to create user: %v", err)
		// Check if it's a duplicate email error
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return nil, errors.New("email already registered")
		}
		return nil, errors.New("failed to create user")
	}

	// Convert UUID to string
	userID := uuidToString(user.ID)
	log.Printf("✅ User registered: %s", userID)
	return &pb.RegisterResponse{
		UserId: userID,
	}, nil
}

// Login authenticates a user and returns a JWT token.
func (s *server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	log.Printf("🔑 Login request received for email: %s", req.Email)

	// Validate input
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		return nil, errors.New("email is required")
	}
	if strings.TrimSpace(req.Password) == "" {
		return nil, errors.New("password is required")
	}

	// Lookup user from database
	user, err := s.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		log.Printf("❌ User not found: %s", req.Email)
		return nil, errors.New("invalid credentials")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(req.Password))
	if err != nil {
		log.Printf("❌ Invalid password for email: %s", req.Email)
		return nil, errors.New("invalid credentials")
	}

	// Convert UUID to string
	userID := uuidToString(user.ID)

	// Generate JWT token
	expiresIn := 3600 * time.Second // 1 hour
	token, err := jwt.GenerateToken(userID, user.Role, user.Email, expiresIn)
	if err != nil {
		log.Printf("❌ Failed to generate JWT token: %v", err)
		return nil, errors.New("failed to generate token")
	}

	log.Printf("✅ JWT token generated for user: %s (role: %s)", userID, user.Role)
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

// validatePassword validates password strength
// Requirements:
// - At least 8 characters
// - Contains at least one uppercase letter
// - Contains at least one lowercase letter
// - Contains at least one number
// - Contains at least one special character
func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>/?]`).MatchString(password)

	var missing []string
	if !hasUpper {
		missing = append(missing, "uppercase letter")
	}
	if !hasLower {
		missing = append(missing, "lowercase letter")
	}
	if !hasNumber {
		missing = append(missing, "number")
	}
	if !hasSpecial {
		missing = append(missing, "special character")
	}

	if len(missing) > 0 {
		return fmt.Errorf("password must contain at least one %s", strings.Join(missing, ", "))
	}

	return nil
}