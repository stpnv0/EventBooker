package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stpnv0/EventBooker/internal/domain"
	"github.com/stpnv0/EventBooker/internal/service/ports"
	"github.com/wb-go/wbf/logger"
)

type BookingService struct {
	bookingRepo ports.BookingRepo
	eventRepo   ports.EventRepo
	userRepo    ports.UserRepo
	notifier    ports.BookingNotifier
	logger      logger.Logger
}

func NewBookingService(
	bookingRepo ports.BookingRepo,
	eventRepo ports.EventRepo,
	userRepo ports.UserRepo,
	notifier ports.BookingNotifier,
	logger logger.Logger,
) *BookingService {
	return &BookingService{
		bookingRepo: bookingRepo,
		eventRepo:   eventRepo,
		userRepo:    userRepo,
		notifier:    notifier,
		logger:      logger,
	}
}

func (s *BookingService) Book(ctx context.Context, eventID, userID string) (*domain.Booking, error) {
	// проверка, что eventID, userID exist
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("check event: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check user: %w", err)
	}

	status := domain.BookingStatusPending
	if !event.RequiresPayment {
		status = domain.BookingStatusConfirmed
	}

	booking := &domain.Booking{
		ID:        uuid.New().String(),
		EventID:   eventID,
		UserID:    userID,
		Status:    status,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err = s.bookingRepo.Create(ctx, booking); err != nil {
		return nil, fmt.Errorf("create booking: %w", err)
	}

	s.logger.Info("booking created",
		logger.String("booking_id", booking.ID),
		logger.String("event_id", eventID),
		logger.String("user_id", userID),
	)

	if event.RequiresPayment {
		go s.notifier.NotifyBookingCreated(context.WithoutCancel(ctx), user, event)
	} else {
		go s.notifier.NotifyBookingConfirmed(context.WithoutCancel(ctx), user, event)
	}

	return booking, nil
}

func (s *BookingService) Confirm(ctx context.Context, eventID, userID string) error {
	// проверка TTL и подтверждения
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("get event: %w", err)
	}

	if !event.RequiresPayment {
		return fmt.Errorf("%w: this event does not require payment", domain.ErrValidation)
	}

	// проверка брони
	booking, err := s.bookingRepo.GetByEventAndUser(ctx, eventID, userID)
	if err != nil {
		return fmt.Errorf("get booking: %w", err)
	}

	if booking.Status != domain.BookingStatusPending {
		return domain.ErrBookingNotPending
	}

	if time.Since(booking.CreatedAt) > event.BookingTTL {
		return domain.ErrBookingExpired
	}

	if err = s.bookingRepo.Confirm(ctx, eventID, userID); err != nil {
		return fmt.Errorf("confirm booking: %w", err)
	}

	s.logger.Info("booking confirmed",
		logger.String("booking_id", booking.ID),
		logger.String("event_id", eventID),
		logger.String("user_id", userID),
	)

	// notify
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user for notification",
			logger.String("user_id", userID),
			logger.String("error", err.Error()),
		)
		return nil
	}

	go s.notifier.NotifyBookingConfirmed(context.WithoutCancel(ctx), user, event)

	return nil
}

func (s *BookingService) CancelExpired(ctx context.Context) ([]*domain.Booking, error) {
	cancelled, err := s.bookingRepo.CancelExpired(ctx)
	if err != nil {
		return nil, fmt.Errorf("cancel expired: %w", err)
	}

	if len(cancelled) > 0 {
		s.logger.Info("expired bookings cancelled",
			logger.Int("count", len(cancelled)),
		)

		go s.notifyCancelled(context.WithoutCancel(ctx), cancelled)
	}

	return cancelled, nil
}

func (s *BookingService) notifyCancelled(ctx context.Context, bookings []*domain.Booking) {
	for _, b := range bookings {
		user, err := s.userRepo.GetByID(ctx, b.UserID)
		if err != nil {
			s.logger.Error("failed to get user for cancel notification",
				logger.String("user_id", b.UserID),
			)
			continue
		}

		event, err := s.eventRepo.GetByID(ctx, b.EventID)
		if err != nil {
			s.logger.Error("failed to get event for cancel notification",
				logger.String("event_id", b.EventID),
			)
			continue
		}

		s.notifier.NotifyBookingCancelled(ctx, user, event)
	}
}
func (s *BookingService) ListByUser(ctx context.Context, userID string) ([]*domain.Booking, error) {
	return s.bookingRepo.ListByUser(ctx, userID)
}
