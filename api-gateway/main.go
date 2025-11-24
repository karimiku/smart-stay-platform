package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbAuth "github.com/karimiku/smart-stay-platform/pkg/genproto/auth"
	pbKey "github.com/karimiku/smart-stay-platform/pkg/genproto/key"
)

func main() {
	log.Println("🚀 Starting API Gateway...")

	// 1. Get configuration from Environment Variables
	// In Docker Compose, this will be "auth-service:50051"
	authAddr := os.Getenv("AUTH_SVC_ADDR")
	if authAddr == "" {
		authAddr = "localhost:50051"
	}

	keyAddr := os.Getenv("KEY_SVC_ADDR")
	if keyAddr == "" {
		keyAddr = "localhost:50053"
	}

	// 2. Connect to Auth Service (gRPC)
	log.Printf("🔌 Connecting to Auth Service at %s...", authAddr)
	authConn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Auth Service: %v", err)
	}
	defer authConn.Close()
	authClient := pbAuth.NewAuthServiceClient(authConn)

	// 3. Connect to Key Service (gRPC)
	log.Printf("🔌 Connecting to Key Service at %s...", keyAddr)
	keyConn, err := grpc.NewClient(keyAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Key Service: %v", err)
	}
	defer keyConn.Close()
	keyClient := pbKey.NewKeyServiceClient(keyConn)

	// 4. Setup HTTP Router
	mux := http.NewServeMux()

	// POST /login Endpoint
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		// A. Validate Method
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// B. Parse JSON Body
		var reqBody struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// C. Call gRPC Service
		// Set a 5-second timeout for reliability
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Printf("[BFF] Calling Login for: %s", reqBody.Email)
		
		// The actual gRPC call
		grpcRes, err := authClient.Login(ctx, &pbAuth.LoginRequest{
			Email:    reqBody.Email,
			Password: reqBody.Password,
		})
		
		if err != nil {
			log.Printf("❌ Login failed: %v", err)
			http.Error(w, fmt.Sprintf("Login failed: %v", err), http.StatusInternalServerError)
			return
		}

		// D. Return JSON Response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token":      grpcRes.AccessToken,
			"expires_in": grpcRes.ExpiresIn,
		})
	})

	// POST /keys/generate Endpoint
	mux.HandleFunc("/keys/generate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var reqBody struct {
			ReservationID string `json:"reservation_id"`
			ValidFrom     string `json:"valid_from"`     // ISO 8601 format
			ValidUntil    string `json:"valid_until"`   // ISO 8601 format
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Parse timestamps
		validFrom, err := time.Parse(time.RFC3339, reqBody.ValidFrom)
		if err != nil {
			http.Error(w, "Invalid valid_from format (use RFC3339)", http.StatusBadRequest)
			return
		}
		validUntil, err := time.Parse(time.RFC3339, reqBody.ValidUntil)
		if err != nil {
			http.Error(w, "Invalid valid_until format (use RFC3339)", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Printf("[BFF] Generating key for reservation: %s", reqBody.ReservationID)

		grpcRes, err := keyClient.GenerateKey(ctx, &pbKey.GenerateKeyRequest{
			ReservationId: reqBody.ReservationID,
			ValidFrom:     timestamppb.New(validFrom),
			ValidUntil:    timestamppb.New(validUntil),
		})

		if err != nil {
			log.Printf("❌ GenerateKey failed: %v", err)
			http.Error(w, fmt.Sprintf("GenerateKey failed: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"key_code":  grpcRes.KeyCode,
			"device_id": grpcRes.DeviceId,
		})
	})

	// POST /keys/revoke Endpoint
	mux.HandleFunc("/keys/revoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var reqBody struct {
			ReservationID string `json:"reservation_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Printf("[BFF] Revoking key for reservation: %s", reqBody.ReservationID)

		grpcRes, err := keyClient.RevokeKey(ctx, &pbKey.RevokeKeyRequest{
			ReservationId: reqBody.ReservationID,
		})

		if err != nil {
			log.Printf("❌ RevokeKey failed: %v", err)
			http.Error(w, fmt.Sprintf("RevokeKey failed: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": grpcRes.Success,
		})
	})

	// 5. Start HTTP Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🌐 API Gateway listening on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}