package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stpnv0/EventBooker/internal/domain"
	"github.com/stpnv0/EventBooker/internal/service/ports"
)

type UserService struct {
	repo ports.UserRepo
}

func NewUserService(repo ports.UserRepo) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Create(ctx context.Context, input domain.CreateUserInput) (*domain.User, error) {
	if input.Username == "" {
		return nil, fmt.Errorf("%w: username is required", domain.ErrValidation)
	}

	user := &domain.User{
		ID:             uuid.New().String(),
		Username:       input.Username,
		TelegramChatID: input.TelegramChatID,
		CreatedAt:      time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) List(ctx context.Context) ([]*domain.User, error) {
	return s.repo.List(ctx)
}
