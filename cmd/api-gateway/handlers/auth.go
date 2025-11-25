package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	pbAuth "github.com/karimiku/smart-stay-platform/pkg/genproto/auth"

	"github.com/karimiku/smart-stay-platform/cmd/api-gateway/utils"
)

// AuthHandler handles authentication-related endpoints
type AuthHandler struct {
	authClient pbAuth.AuthServiceClient
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authClient pbAuth.AuthServiceClient) *AuthHandler {
	return &AuthHandler{
		authClient: authClient,
	}
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate email
	reqBody.Email = strings.TrimSpace(reqBody.Email)
	if reqBody.Email == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Email is required")
		return
	}
	if !isValidEmail(reqBody.Email) {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid email format")
		return
	}

	// Validate password
	if strings.TrimSpace(reqBody.Password) == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Password is required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("[BFF] Calling Login for: %s", reqBody.Email)
	res, err := h.authClient.Login(ctx, &pbAuth.LoginRequest{
		Email:    reqBody.Email,
		Password: reqBody.Password,
	})
	if err != nil {
		log.Printf("❌ Login failed: %v", err)
		utils.ErrorResponse(w, http.StatusInternalServerError, "Login failed")
		return
	}

	utils.SuccessResponse(w, map[string]interface{}{
		"token":      res.AccessToken,
		"expires_in": res.ExpiresIn,
	})
}

// isValidEmail validates email format using regex
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

