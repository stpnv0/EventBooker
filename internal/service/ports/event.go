package ports

import (
	"context"

	"github.com/stpnv0/EventBooker/internal/domain"
)

type EventRepo interface {
	Create(ctx context.Context, e *domain.Event) error
	GetByID(ctx context.Context, id string) (*domain.Event, error)
	List(ctx context.Context) ([]*domain.Event, error)
	GetDetails(ctx context.Context, eventID string) (*domain.EventDetails, error)
}
