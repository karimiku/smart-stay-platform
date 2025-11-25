package handlers

import (
	"net/http"

	"github.com/karimiku/smart-stay-platform/cmd/api-gateway/middleware"
	"github.com/karimiku/smart-stay-platform/cmd/api-gateway/utils"
)

// UserHandler handles user-related endpoints
type UserHandler struct{}

// NewUserHandler creates a new user handler
func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// GetMe returns the current user's information
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.ErrorResponse(w, http.StatusUnauthorized, "User ID not found")
		return
	}

	role, ok := middleware.GetRole(r)
	if !ok {
		role = "guest" // Default role
	}

	utils.SuccessResponse(w, map[string]interface{}{
		"user_id": userID,
		"role":    role,
	})
}

