package domain

import "time"

type Event struct {
	ID              string        `json:"id"`
	Title           string        `json:"title"`
	Description     string        `json:"description"`
	EventDate       time.Time     `json:"event_date"`
	TotalSpots      int           `json:"total_spots"`
	RequiresPayment bool          `json:"requires_payment"`
	BookingTTL      time.Duration `json:"booking_ttl"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type EventDetails struct {
	Event          Event     `json:"event"`
	AvailableSpots int       `json:"available_spots"`
	Bookings       []Booking `json:"bookings"`
}

type CreateEventInput struct {
	Title           string
	Description     string
	EventDate       time.Time
	TotalSpots      int
	BookingTTL      time.Duration
	RequiresPayment *bool
}
