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

	pb "github.com/karimiku/smart-stay-platform/pkg/genproto/auth"
)

func main() {
	log.Println("🚀 Starting API Gateway...")

	// 1. Get configuration from Environment Variables
	// In Docker Compose, this will be "auth-service:50051"
	authAddr := os.Getenv("AUTH_SVC_ADDR")
	if authAddr == "" {
		authAddr = "localhost:50051"
	}

	// 2. Connect to Auth Service (gRPC)
	log.Printf("🔌 Connecting to Auth Service at %s...", authAddr)
	// Use insecure credentials for internal container communication
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Auth Service: %v", err)
	}
	defer conn.Close()

	// Create the gRPC client 
	authClient := pb.NewAuthServiceClient(conn)

	// 3. Setup HTTP Router
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
		
		//  The actual gRPC call
		grpcRes, err := authClient.Login(ctx, &pb.LoginRequest{
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

	// 4. Start HTTP Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🌐 API Gateway listening on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}