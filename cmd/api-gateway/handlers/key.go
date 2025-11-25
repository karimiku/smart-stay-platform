package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	pbKey "github.com/karimiku/smart-stay-platform/pkg/genproto/key"

	"github.com/karimiku/smart-stay-platform/cmd/api-gateway/utils"
)

// KeyHandler handles key-related endpoints
type KeyHandler struct {
	keyClient pbKey.KeyServiceClient
}

// NewKeyHandler creates a new key handler
func NewKeyHandler(keyClient pbKey.KeyServiceClient) *KeyHandler {
	return &KeyHandler{
		keyClient: keyClient,
	}
}

// GenerateKey handles manual key generation (for debugging)
func (h *KeyHandler) GenerateKey(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		ReservationID string `json:"reservation_id"`
		ValidFrom     string `json:"valid_from"`
		ValidUntil    string `json:"valid_until"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse RFC3339
	validFrom, err := time.Parse(time.RFC3339, reqBody.ValidFrom)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid valid_from format (use RFC3339)")
		return
	}
	validUntil, err := time.Parse(time.RFC3339, reqBody.ValidUntil)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid valid_until format (use RFC3339)")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := h.keyClient.GenerateKey(ctx, &pbKey.GenerateKeyRequest{
		ReservationId: reqBody.ReservationID,
		ValidFrom:     timestamppb.New(validFrom),
		ValidUntil:    timestamppb.New(validUntil),
	})
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Key generation failed")
		return
	}

	utils.SuccessResponse(w, map[string]interface{}{
		"key_code":  res.KeyCode,
		"device_id": res.DeviceId,
	})
}

