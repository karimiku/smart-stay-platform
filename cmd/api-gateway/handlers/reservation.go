package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	pbRes "github.com/karimiku/smart-stay-platform/pkg/genproto/reservation"

	"github.com/karimiku/smart-stay-platform/cmd/api-gateway/middleware"
	"github.com/karimiku/smart-stay-platform/cmd/api-gateway/utils"
)

// ReservationHandler handles reservation-related endpoints
type ReservationHandler struct {
	resClient pbRes.ReservationServiceClient
}

// NewReservationHandler creates a new reservation handler
func NewReservationHandler(resClient pbRes.ReservationServiceClient) *ReservationHandler {
	return &ReservationHandler{
		resClient: resClient,
	}
}

// CreateReservation handles reservation creation
func (h *ReservationHandler) CreateReservation(w http.ResponseWriter, r *http.Request) {
	// Get user_id from JWT (not from request body)
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.ErrorResponse(w, http.StatusUnauthorized, "User ID not found")
		return
	}

	var reqBody struct {
		RoomID    int64  `json:"room_id"`
		StartDate string `json:"start_date"` // Format: YYYY-MM-DD
		EndDate   string `json:"end_date"`   // Format: YYYY-MM-DD
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse Date Strings to Time
	layout := "2006-01-02"
	start, err := time.Parse(layout, reqBody.StartDate)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid start_date format (use YYYY-MM-DD)")
		return
	}
	end, err := time.Parse(layout, reqBody.EndDate)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid end_date format (use YYYY-MM-DD)")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("[BFF] Creating Reservation for User %s", userID)

	// Call gRPC (Trigger Pub/Sub event inside Reservation Service)
	res, err := h.resClient.CreateReservation(ctx, &pbRes.CreateReservationRequest{
		UserId:    userID, // Use user_id from JWT
		RoomId:    reqBody.RoomID,
		StartDate: timestamppb.New(start),
		EndDate:   timestamppb.New(end),
	})
	if err != nil {
		log.Printf("❌ Reservation failed: %v", err)
		utils.ErrorResponse(w, http.StatusInternalServerError, "Reservation failed")
		return
	}

	// Return response (Status should be PENDING)
	utils.SuccessResponse(w, map[string]interface{}{
		"reservation_id": res.ReservationId,
		"status":         res.Status.String(),
	})
}

// ListReservations handles listing all reservations for the current user
func (h *ReservationHandler) ListReservations(w http.ResponseWriter, r *http.Request) {
	// Get user_id from JWT
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.ErrorResponse(w, http.StatusUnauthorized, "User ID not found")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("[BFF] Listing Reservations for User %s", userID)

	// Call gRPC
	res, err := h.resClient.ListReservations(ctx, &pbRes.ListReservationsRequest{
		UserId: userID,
	})
	if err != nil {
		log.Printf("❌ List reservations failed: %v", err)
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to list reservations")
		return
	}

	// Convert reservations to JSON format
	var reservations []map[string]interface{}
	for _, reservation := range res.Reservations {
		reservations = append(reservations, map[string]interface{}{
			"id":          reservation.Id,
			"user_id":     reservation.UserId,
			"room_id":     reservation.RoomId,
			"start_date":  reservation.StartDate.AsTime().Format("2006-01-02"),
			"end_date":    reservation.EndDate.AsTime().Format("2006-01-02"),
			"total_price": reservation.TotalPrice,
			"status":      reservation.Status.String(),
		})
	}

	utils.SuccessResponse(w, map[string]interface{}{
		"reservations": reservations,
	})
}

