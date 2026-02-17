package domain

import "time"

type BookingStatus string

const (
	BookingStatusPending   BookingStatus = "pending"
	BookingStatusConfirmed BookingStatus = "confirmed"
	BookingStatusCancelled BookingStatus = "cancelled"
)

var ActiveStatuses = []BookingStatus{BookingStatusPending, BookingStatusConfirmed}

type Booking struct {
	ID        string        `json:"id"`
	EventID   string        `json:"event_id"`
	UserID    string        `json:"user_id"`
	Status    BookingStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}
