package ports

import (
	"context"

	"github.com/stpnv0/EventBooker/internal/domain"
)

type BookingRepo interface {
	Create(ctx context.Context, b *domain.Booking) error
	GetByEventAndUser(ctx context.Context, eventID, userID string) (*domain.Booking, error)
	Confirm(ctx context.Context, eventID, userID string) error
	CancelExpired(ctx context.Context) ([]*domain.Booking, error)
	ListByEvent(ctx context.Context, eventID string) ([]*domain.Booking, error)
	ListByUser(ctx context.Context, userID string) ([]*domain.Booking, error)
}
