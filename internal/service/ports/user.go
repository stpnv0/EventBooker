package ports

import (
	"context"

	"github.com/stpnv0/EventBooker/internal/domain"
)

type UserRepo interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	List(ctx context.Context) ([]*domain.User, error)
}
