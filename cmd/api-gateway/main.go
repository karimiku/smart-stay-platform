package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	mux.HandleFunc("POST /signup", authHandler.Signup)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("POST /logout", authHandler.Logout)

	// =========================================================================
	// 👤 User Routes (Protected - Authentication required)
	// =========================================================================
	mux.HandleFunc("GET /me", authMiddleware.RequireAuth(userHandler.GetMe))

	// =========================================================================
	// 📝 Reservation Routes (Protected - Authentication required)
	// =========================================================================
	mux.HandleFunc("POST /reservations", authMiddleware.RequireAuth(reservationHandler.CreateReservation))
	mux.HandleFunc("GET /reservations", authMiddleware.RequireAuth(reservationHandler.ListReservations))

	// =========================================================================
	// 🔑 Key Routes (Protected - Authentication required)
	// =========================================================================
	mux.HandleFunc("POST /keys/generate", authMiddleware.RequireAuth(keyHandler.GenerateKey))
	mux.HandleFunc("GET /keys", authMiddleware.RequireAuth(keyHandler.ListKeys))

	// 6. Apply CORS middleware
	handler := middleware.CORS(mux)

	// 7. Start Server
	port := getEnv("PORT", "8080")
	log.Printf("🌐 API Gateway listening on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
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
	
	// Parse address to handle both localhost:port and https://hostname formats
	grpcAddr, useTLS := parseGrpcAddress(addr)
	
	var creds credentials.TransportCredentials
	if useTLS {
		// Use TLS for Cloud Run (production)
		creds = credentials.NewTLS(nil) // nil means use system's root CA certificates
		log.Printf("🔒 Using TLS for %s", name)
	} else {
		// Use insecure for local development
		creds = insecure.NewCredentials()
		log.Printf("⚠️  Using insecure connection for %s (local development)", name)
	}
	
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("Failed to connect to %s: %v", name, err)
	}
	return conn
}

// parseGrpcAddress parses the address and returns the gRPC address and whether to use TLS
// Handles:
// - localhost:50051 -> localhost:50051, false
// - https://auth-service-xxb5f653ia-an.a.run.app -> auth-service-xxb5f653ia-an.a.run.app:443, true
func parseGrpcAddress(addr string) (string, bool) {
	// If it's already in host:port format (no scheme), use as-is
	if !strings.Contains(addr, "://") {
		return addr, false
	}
	
	// Parse URL
	parsedURL, err := url.Parse(addr)
	if err != nil {
		log.Printf("⚠️  Failed to parse URL %s, using as-is: %v", addr, err)
		return addr, false
	}
	
	host := parsedURL.Hostname()
	if host == "" {
		host = parsedURL.Host
	}
	
	// Extract port from URL or use default
	port := parsedURL.Port()
	if port == "" {
		if parsedURL.Scheme == "https" {
			port = "443" // Default HTTPS port
		} else {
			port = "80" // Default HTTP port
		}
	}
	
	grpcAddr := host + ":" + port
	useTLS := parsedURL.Scheme == "https"
	
	return grpcAddr, useTLS
}