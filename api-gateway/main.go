package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	// ⚠️ Replace [yourusername] with your actual GitHub username
	pbAuth "github.com/karimiku/smart-stay-platform/pkg/genproto/auth"
	pbKey "github.com/karimiku/smart-stay-platform/pkg/genproto/key"
	pbRes "github.com/karimiku/smart-stay-platform/pkg/genproto/reservation"
)

func main() {
	log.Println("🚀 Starting API Gateway...")

	// 1. Configuration (Env vars)
	authAddr := getEnv("AUTH_SVC_ADDR", "localhost:50051")
	resAddr := getEnv("RESERVATION_SVC_ADDR", "localhost:50052")
	keyAddr := getEnv("KEY_SVC_ADDR", "localhost:50053")

	// 2. Connect to Services (gRPC)
	// Auth Service
	authConn := mustConnectGrpc("Auth Service", authAddr)
	defer authConn.Close()
	authClient := pbAuth.NewAuthServiceClient(authConn)

	// Reservation Service
	resConn := mustConnectGrpc("Reservation Service", resAddr)
	defer resConn.Close()
	resClient := pbRes.NewReservationServiceClient(resConn)

	// Key Service
	keyConn := mustConnectGrpc("Key Service", keyAddr)
	defer keyConn.Close()
	keyClient := pbKey.NewKeyServiceClient(keyConn)

	// 3. Setup Router
	mux := http.NewServeMux()

	// =========================================================================
	// 🛡️ Auth Routes
	// =========================================================================
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var reqBody struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Printf("[BFF] Calling Login for: %s", reqBody.Email)
		res, err := authClient.Login(ctx, &pbAuth.LoginRequest{
			Email:    reqBody.Email,
			Password: reqBody.Password,
		})
		if err != nil {
			log.Printf("❌ Login failed: %v", err)
			http.Error(w, "Login failed", http.StatusInternalServerError)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"token":      res.AccessToken,
			"expires_in": res.ExpiresIn,
		})
	})

	// =========================================================================
	// 📝 Reservation Routes (Saga Start)
	// =========================================================================
	mux.HandleFunc("/reservations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var reqBody struct {
			UserID    string `json:"user_id"` // UUID
			RoomID    int64  `json:"room_id"`
			StartDate string `json:"start_date"` // Format: YYYY-MM-DD
			EndDate   string `json:"end_date"`   // Format: YYYY-MM-DD
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Parse Date Strings to Time
		layout := "2006-01-02"
		start, err := time.Parse(layout, reqBody.StartDate)
		if err != nil {
			http.Error(w, "Invalid start_date format (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		end, err := time.Parse(layout, reqBody.EndDate)
		if err != nil {
			http.Error(w, "Invalid end_date format (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Printf("[BFF] Creating Reservation for User %s", reqBody.UserID)
		
		// Call gRPC (Trigger Pub/Sub event inside Reservation Service)
		res, err := resClient.CreateReservation(ctx, &pbRes.CreateReservationRequest{
			UserId:    reqBody.UserID,
			RoomId:    reqBody.RoomID,
			StartDate: timestamppb.New(start),
			EndDate:   timestamppb.New(end),
		})
		if err != nil {
			log.Printf("❌ Reservation failed: %v", err)
			http.Error(w, "Reservation failed", http.StatusInternalServerError)
			return
		}

		// Return response (Status should be PENDING)
		jsonResponse(w, map[string]interface{}{
			"reservation_id": res.ReservationId,
			"status":         res.Status.String(), // Convert Enum to String
		})
	})

	// =========================================================================
	// 🔑 Key Routes (Debug/Manual)
	// =========================================================================
	mux.HandleFunc("/keys/generate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var reqBody struct {
			ReservationID string `json:"reservation_id"`
			ValidFrom     string `json:"valid_from"`
			ValidUntil    string `json:"valid_until"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Parse RFC3339
		validFrom, _ := time.Parse(time.RFC3339, reqBody.ValidFrom)
		validUntil, _ := time.Parse(time.RFC3339, reqBody.ValidUntil)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		res, err := keyClient.GenerateKey(ctx, &pbKey.GenerateKeyRequest{
			ReservationId: reqBody.ReservationID,
			ValidFrom:     timestamppb.New(validFrom),
			ValidUntil:    timestamppb.New(validUntil),
		})
		if err != nil {
			http.Error(w, "Key generation failed", http.StatusInternalServerError)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"key_code":  res.KeyCode,
			"device_id": res.DeviceId,
		})
	})

	// 4. Start Server
	port := getEnv("PORT", "8080")
	log.Printf("🌐 API Gateway listening on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// --- Helpers ---

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustConnectGrpc(name, addr string) *grpc.ClientConn {
	log.Printf("🔌 Connecting to %s at %s...", name, addr)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to %s: %v", name, err)
	}
	return conn
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}