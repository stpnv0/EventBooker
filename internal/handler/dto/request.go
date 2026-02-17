package dto

type CreateEventRequest struct {
	Title           string `json:"title" binding:"required"`
	Description     string `json:"description" binding:"required"`
	EventDate       string `json:"event_date" binding:"required"`
	TotalSpots      int    `json:"total_spots" binding:"required,gt=0"`
	BookingTTL      int    `json:"booking_ttl_minutes"`
	RequiresPayment *bool  `json:"requires_payment"`
}

type BookRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

type ConfirmRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

type CreateUserRequest struct {
	Username       string `json:"username" binding:"required"`
	TelegramChatID *int64 `json:"telegram_chat_id"`
}
