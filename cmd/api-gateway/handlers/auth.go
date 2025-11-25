package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
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

