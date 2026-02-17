package ports

import (
	"context"

	"github.com/stpnv0/EventBooker/internal/domain"
)

type BookingNotifier interface {
	NotifyBookingCreated(ctx context.Context, user *domain.User, event *domain.Event)
	NotifyBookingConfirmed(ctx context.Context, user *domain.User, event *domain.Event)
	NotifyBookingCancelled(ctx context.Context, user *domain.User, event *domain.Event)
}
