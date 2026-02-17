package notification

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stpnv0/EventBooker/internal/domain"
	"github.com/wb-go/wbf/logger"
)

type TelegramNotifier struct {
	bot    *tgbotapi.BotAPI
	logger logger.Logger
}

func NewTelegramNotifier(token string, logger logger.Logger) (*TelegramNotifier, error) {
	if token == "" {
		logger.Warn("telegram bot token is empty, notifications disabled")
		return &TelegramNotifier{bot: nil, logger: logger}, nil
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}

	return &TelegramNotifier{bot: bot, logger: logger}, nil
}

func (n *TelegramNotifier) NotifyBookingConfirmed(ctx context.Context, user *domain.User, event *domain.Event) {
	text := fmt.Sprintf(
		"*Бронирование подтверждено!*\n\n"+"Мероприятие: %s\n"+"Дата (время указано в UTC): %s",
		event.Title, event.EventDate.Format("02.01.2006 15:04"),
	)
	n.send(ctx, user.TelegramChatID, text)
}

func (n *TelegramNotifier) NotifyBookingCancelled(ctx context.Context, user *domain.User, event *domain.Event) {
	text := fmt.Sprintf(
		"*Бронирование отменено (истекло время оплаты)*\n\n"+"Мероприятие: %s\n"+"Дата (время указано в UTC): %s",
		event.Title, event.EventDate.Format("02.01.2006 15:04"),
	)
	n.send(ctx, user.TelegramChatID, text)
}

func (n *TelegramNotifier) NotifyBookingCreated(ctx context.Context, user *domain.User, event *domain.Event) {
	text := fmt.Sprintf(
		"*Место забронировано!*\n\n"+"Мероприятие: %s\n"+"Дата (время указано в UTC): %s\n"+"Подтвердите бронь в течение %s, иначе она будет отменена.",
		event.Title,
		event.EventDate.Format("02.01.2006 15:04"),
		event.BookingTTL.String(),
	)
	n.send(ctx, user.TelegramChatID, text)
}

func (n *TelegramNotifier) send(ctx context.Context, chatID *int64, text string) {
	if n.bot == nil {
		n.logger.Debug("notification skipped (bot disabled)", logger.String("text", text))
		return
	}

	if chatID == nil {
		n.logger.Debug("notification skipped (no chat_id)", logger.String("text", text))
		return
	}

	if err := ctx.Err(); err != nil {
		n.logger.Debug("notification skipped (context cancelled)",
			logger.Int64("chat_id", *chatID),
		)
		return
	}

	msg := tgbotapi.NewMessage(*chatID, text)
	msg.ParseMode = "Markdown"

	if _, err := n.bot.Send(msg); err != nil {
		n.logger.Error("failed to send telegram notification",
			logger.Int64("chat_id", *chatID),
			logger.String("error", err.Error()),
		)
	}
}
