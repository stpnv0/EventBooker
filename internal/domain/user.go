package domain

import "time"

type User struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	TelegramChatID *int64    `json:"telegram_chat_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateUserInput struct {
	Username       string
	TelegramChatID *int64
}
