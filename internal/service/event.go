package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stpnv0/EventBooker/internal/domain"
	"github.com/stpnv0/EventBooker/internal/service/ports"
)

const defaultBookingTTL = 20 * time.Minute

type EventService struct {
	repo        ports.EventRepo
	bookingRepo ports.BookingRepo
}

func NewEventService(repo ports.EventRepo, bookingRepo ports.BookingRepo) *EventService {
	return &EventService{
		repo:        repo,
		bookingRepo: bookingRepo,
	}
}

func (s *EventService) CreateEvent(ctx context.Context, input domain.CreateEventInput) (*domain.Event, error) {
	if input.Title == "" {
		return nil, fmt.Errorf("%w: title is required", domain.ErrValidation)
	}
	if input.TotalSpots <= 0 {
		return nil, fmt.Errorf("%w: total_spots must be positive", domain.ErrValidation)
	}
	if input.EventDate.Before(time.Now()) {
		return nil, fmt.Errorf("%w: event_date must be in the future", domain.ErrValidation)
	}
	requiresPayment := true
	if input.RequiresPayment != nil {
		requiresPayment = *input.RequiresPayment
	}

	ttl := input.BookingTTL
	if ttl == 0 {
		ttl = defaultBookingTTL
	}
	event := &domain.Event{
		ID:              uuid.New().String(),
		Title:           input.Title,
		Description:     input.Description,
		EventDate:       input.EventDate,
		RequiresPayment: requiresPayment,
		TotalSpots:      input.TotalSpots,
		BookingTTL:      ttl,
	}

	if err := s.repo.Create(ctx, event); err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}

	return event, nil
}

func (s *EventService) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *EventService) GetDetails(ctx context.Context, id string) (*domain.EventDetails, error) {
	details, err := s.repo.GetDetails(ctx, id)
	if err != nil {
		return nil, err
	}

	bookings, err := s.bookingRepo.ListByEvent(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}

	details.Bookings = make([]domain.Booking, len(bookings))
	for i, b := range bookings {
		details.Bookings[i] = *b
	}

	return details, nil
}

func (s *EventService) List(ctx context.Context) ([]*domain.Event, error) {
	return s.repo.List(ctx)
}
