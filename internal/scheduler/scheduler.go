package scheduler

import (
	"context"
	"time"

	"github.com/stpnv0/EventBooker/internal/domain"
	"github.com/wb-go/wbf/logger"
)

type bookingCanceller interface {
	CancelExpired(ctx context.Context) ([]*domain.Booking, error)
}

type Scheduler struct {
	bookingService bookingCanceller
	interval       time.Duration
	logger         logger.Logger
}

func New(
	bookingService bookingCanceller,
	interval time.Duration,
	logger logger.Logger,
) *Scheduler {
	return &Scheduler{
		bookingService: bookingService,
		interval:       interval,
		logger:         logger,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.logger.Info("scheduler started",
		logger.Duration("interval", s.interval),
	)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	cancelled, err := s.bookingService.CancelExpired(ctx)
	if err != nil {
		s.logger.Error("failed to cancel expired bookings",
			logger.String("error", err.Error()),
		)
		return
	}

	for _, b := range cancelled {
		s.logger.Info("booking expired",
			logger.String("booking_id", b.ID),
			logger.String("user_id", b.UserID),
			logger.String("event_id", b.EventID),
		)
	}
}
