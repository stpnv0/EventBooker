package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stpnv0/EventBooker/internal/domain"
	"github.com/stpnv0/EventBooker/internal/scheduler/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wb-go/wbf/logger"
)

func newTestLogger(t *testing.T) logger.Logger {
	t.Helper()
	log, err := logger.InitLogger("slog", "test", "test", logger.WithLevel(logger.ErrorLevel))
	if err != nil {
		t.Fatalf("init test logger: %v", err)
	}
	return log
}

func TestScheduler_Tick_CancelsExpired(t *testing.T) {
	canceller := mocks.NewMockBookingCanceller(t)
	log := newTestLogger(t)

	s := New(canceller, 50*time.Millisecond, log)

	cancelled := []*domain.Booking{
		{ID: "b1", EventID: "e1", UserID: "u1"},
	}
	canceller.EXPECT().CancelExpired(mock.Anything).Return(cancelled, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	s.Start(ctx)

	assert.GreaterOrEqual(t, len(canceller.Calls), 1)
}

func TestScheduler_Tick_HandlesError(t *testing.T) {
	canceller := mocks.NewMockBookingCanceller(t)
	log := newTestLogger(t)

	s := New(canceller, 50*time.Millisecond, log)

	canceller.EXPECT().CancelExpired(mock.Anything).Return(nil, errors.New("db error"))

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	s.Start(ctx)

	assert.GreaterOrEqual(t, len(canceller.Calls), 1)
}

func TestScheduler_StopsOnContextCancel(t *testing.T) {
	canceller := mocks.NewMockBookingCanceller(t)
	log := newTestLogger(t)

	s := New(canceller, time.Second, log) // interval longer than test

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		s.Start(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// success
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop on context cancel")
	}
}

func TestScheduler_MultipleTicks(t *testing.T) {
	canceller := mocks.NewMockBookingCanceller(t)
	log := newTestLogger(t)

	s := New(canceller, 30*time.Millisecond, log)

	canceller.EXPECT().CancelExpired(mock.Anything).Return(nil, nil).Times(3)

	ctx, cancel := context.WithTimeout(context.Background(), 110*time.Millisecond)
	defer cancel()

	s.Start(ctx)

	calls := len(canceller.Calls)
	assert.GreaterOrEqual(t, calls, 3)
}
