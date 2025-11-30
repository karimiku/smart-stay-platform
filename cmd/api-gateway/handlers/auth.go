package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
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

	log.Printf("[BFF] Calling Login")
	res, err := h.authClient.Login(ctx, &pbAuth.LoginRequest{
		Email:    reqBody.Email,
		Password: reqBody.Password,
	})
	if err != nil {
		log.Printf("❌ Login failed: %v", err)
		utils.ErrorResponse(w, http.StatusInternalServerError, "Login failed")
		return
	}

	// Set httpOnly Cookie for secure token storage
	// For cross-origin requests (production), use SameSite=None with Secure=true
	// For same-origin requests (development), use SameSite=Lax
	isSecure := os.Getenv("ENVIRONMENT") == "production" || os.Getenv("HTTPS_ENABLED") == "true"
	sameSite := http.SameSiteLaxMode
	if isSecure {
		// Cross-origin requests require SameSite=None with Secure=true
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    res.AccessToken,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: sameSite,
		MaxAge:   int(res.ExpiresIn),
		Path:     "/",
	})

	utils.SuccessResponse(w, map[string]interface{}{
		"message":    "Login successful",
		"expires_in": res.ExpiresIn,
	})
}

// Signup handles user registration
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
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
	reqBody.Password = strings.TrimSpace(reqBody.Password)
	if reqBody.Password == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Password is required")
		return
	}
	if err := validatePassword(reqBody.Password); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate name
	reqBody.Name = strings.TrimSpace(reqBody.Name)
	if reqBody.Name == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Name is required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("[BFF] Calling Register for: %s", reqBody.Email)
	res, err := h.authClient.Register(ctx, &pbAuth.RegisterRequest{
		Email:    reqBody.Email,
		Password: reqBody.Password,
		Name:     reqBody.Name,
	})
	if err != nil {
		log.Printf("❌ Signup failed: %v", err)
		// Check for specific error types
		if strings.Contains(err.Error(), "already registered") || strings.Contains(err.Error(), "duplicate") {
			utils.ErrorResponse(w, http.StatusConflict, "Email already registered")
			return
		}
		utils.ErrorResponse(w, http.StatusInternalServerError, "Registration failed")
		return
	}

	log.Printf("✅ User registered successfully: %s", res.UserId)
	utils.SuccessResponse(w, map[string]interface{}{
		"user_id": res.UserId,
		"message": "User registered successfully",
	})
}

// Logout handles user logout by clearing the auth cookie
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear the auth cookie
	// For cross-origin requests (production), use SameSite=None with Secure=true
	// For same-origin requests (development), use SameSite=Lax
	isSecure := os.Getenv("ENVIRONMENT") == "production" || os.Getenv("HTTPS_ENABLED") == "true"
	sameSite := http.SameSiteLaxMode
	if isSecure {
		// Cross-origin requests require SameSite=None with Secure=true
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: sameSite,
		MaxAge:   -1, // Delete the cookie
		Path:     "/",
	})

	utils.SuccessResponse(w, map[string]interface{}{
		"message": "Logout successful",
	})
}

// isValidEmail validates email format using regex
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
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

