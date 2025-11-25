package events

import "time"

// EventPayload defines the structure of the message sent to Pub/Sub.
// This is used for communication between services via event-driven architecture.
type EventPayload struct {
	EventType     string    `json:"event_type"`
	ReservationID string    `json:"reservation_id"`
	UserID        string    `json:"user_id"` // UUID
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
}

// EventType constants for type safety
const (
	EventTypeReservationCreated = "ReservationCreated"
)

