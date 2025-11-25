package main

import (
	"log"
	"net/http"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbAuth "github.com/karimiku/smart-stay-platform/pkg/genproto/auth"
	pbKey "github.com/karimiku/smart-stay-platform/pkg/genproto/key"
	pbRes "github.com/karimiku/smart-stay-platform/pkg/genproto/reservation"

	"github.com/karimiku/smart-stay-platform/cmd/api-gateway/handlers"
	"github.com/karimiku/smart-stay-platform/cmd/api-gateway/middleware"
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

	// 3. Initialize Middleware
	authMiddleware := middleware.NewAuthMiddleware(authClient)

	// 4. Initialize Handlers
	authHandler := handlers.NewAuthHandler(authClient)
	userHandler := handlers.NewUserHandler()
	reservationHandler := handlers.NewReservationHandler(resClient)
	keyHandler := handlers.NewKeyHandler(keyClient)

	// 5. Setup Router
	mux := http.NewServeMux()

	// =========================================================================
	// 🛡️ Auth Routes (Public - No authentication required)
	// =========================================================================
	mux.HandleFunc("POST /login", authHandler.Login)

	// =========================================================================
	// 👤 User Routes (Protected - Authentication required)
	// =========================================================================
	mux.HandleFunc("GET /me", authMiddleware.RequireAuth(userHandler.GetMe))

	// =========================================================================
	// 📝 Reservation Routes (Protected - Authentication required)
	// =========================================================================
	mux.HandleFunc("POST /reservations", authMiddleware.RequireAuth(reservationHandler.CreateReservation))

	// =========================================================================
	// 🔑 Key Routes (Protected - Authentication required)
	// =========================================================================
	mux.HandleFunc("POST /keys/generate", authMiddleware.RequireAuth(keyHandler.GenerateKey))

	// 6. Start Server
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