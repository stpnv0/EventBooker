package dto

import (
	"time"

	"github.com/stpnv0/EventBooker/internal/domain"
)

type EventResponse struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	EventDate       string `json:"event_date"`
	TotalSpots      int    `json:"total_spots"`
	BookingTTL      string `json:"booking_ttl"`
	RequiresPayment bool   `json:"requires_payment"`
	CreatedAt       string `json:"created_at"`
}

type EventDetailsResponse struct {
	Event          EventResponse     `json:"event"`
	AvailableSpots int               `json:"available_spots"`
	Bookings       []BookingResponse `json:"bookings"`
}

type BookingResponse struct {
	ID        string `json:"id"`
	EventID   string `json:"event_id"`
	UserID    string `json:"user_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type UserResponse struct {
	ID             string `json:"id"`
	Username       string `json:"username"`
	TelegramChatID *int64 `json:"telegram_chat_id,omitempty"`
	CreatedAt      string `json:"created_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func ToEventResponse(e *domain.Event) EventResponse {
	return EventResponse{
		ID:              e.ID,
		Title:           e.Title,
		Description:     e.Description,
		EventDate:       e.EventDate.Format(time.RFC3339),
		TotalSpots:      e.TotalSpots,
		RequiresPayment: e.RequiresPayment,
		BookingTTL:      e.BookingTTL.String(),
		CreatedAt:       e.CreatedAt.Format(time.RFC3339),
	}
}

func ToEventDetailsResponse(d *domain.EventDetails) EventDetailsResponse {
	bookings := make([]BookingResponse, 0, len(d.Bookings))
	for _, b := range d.Bookings {
		bookings = append(bookings, ToBookingResponse(&b))
	}

	return EventDetailsResponse{
		Event:          ToEventResponse(&d.Event),
		AvailableSpots: d.AvailableSpots,
		Bookings:       bookings,
	}
}

func ToBookingResponse(b *domain.Booking) BookingResponse {
	return BookingResponse{
		ID:        b.ID,
		EventID:   b.EventID,
		UserID:    b.UserID,
		Status:    string(b.Status),
		CreatedAt: b.CreatedAt.Format(time.RFC3339),
	}
}

func ToUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:             u.ID,
		Username:       u.Username,
		TelegramChatID: u.TelegramChatID,
		CreatedAt:      u.CreatedAt.Format(time.RFC3339),
	}
}
