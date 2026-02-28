package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stpnv0/EventBooker/internal/domain"
	"github.com/stpnv0/EventBooker/internal/service/ports/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func TestBookingService_Book_RequiresPayment(t *testing.T) {
	bookingRepo := mocks.NewMockBookingRepo(t)
	eventRepo := mocks.NewMockEventRepo(t)
	userRepo := mocks.NewMockUserRepo(t)
	notifier := mocks.NewMockBookingNotifier(t)
	log := newTestLogger(t)

	svc := NewBookingService(bookingRepo, eventRepo, userRepo, notifier, log)

	event := &domain.Event{
		ID:              "e1",
		Title:           "Concert",
		RequiresPayment: true,
		BookingTTL:      20 * time.Minute,
	}
	user := &domain.User{ID: "u1", Username: "alice"}

	eventRepo.EXPECT().GetByID(mock.Anything, "e1").Return(event, nil)
	userRepo.EXPECT().GetByID(mock.Anything, "u1").Return(user, nil)
	bookingRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	notifier.EXPECT().NotifyBookingCreated(mock.Anything, user, event).Return()

	booking, err := svc.Book(context.Background(), "e1", "u1")

	require.NoError(t, err)
	assert.Equal(t, domain.BookingStatusPending, booking.Status)
	assert.Equal(t, "e1", booking.EventID)
	assert.Equal(t, "u1", booking.UserID)
	assert.NotEmpty(t, booking.ID)

	time.Sleep(50 * time.Millisecond) // goroutine notify
}

func TestBookingService_Book_NoPaymentRequired(t *testing.T) {
	bookingRepo := mocks.NewMockBookingRepo(t)
	eventRepo := mocks.NewMockEventRepo(t)
	userRepo := mocks.NewMockUserRepo(t)
	notifier := mocks.NewMockBookingNotifier(t)
	log := newTestLogger(t)

	svc := NewBookingService(bookingRepo, eventRepo, userRepo, notifier, log)

	event := &domain.Event{
		ID:              "e1",
		RequiresPayment: false,
	}
	user := &domain.User{ID: "u1", Username: "alice"}

	eventRepo.EXPECT().GetByID(mock.Anything, "e1").Return(event, nil)
	userRepo.EXPECT().GetByID(mock.Anything, "u1").Return(user, nil)
	bookingRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	notifier.EXPECT().NotifyBookingConfirmed(mock.Anything, user, event).Return()

	booking, err := svc.Book(context.Background(), "e1", "u1")

	require.NoError(t, err)
	assert.Equal(t, domain.BookingStatusConfirmed, booking.Status)

	time.Sleep(50 * time.Millisecond)
}

func TestBookingService_Book_EventNotFound(t *testing.T) {
	bookingRepo := mocks.NewMockBookingRepo(t)
	eventRepo := mocks.NewMockEventRepo(t)
	userRepo := mocks.NewMockUserRepo(t)
	notifier := mocks.NewMockBookingNotifier(t)
	log := newTestLogger(t)

	svc := NewBookingService(bookingRepo, eventRepo, userRepo, notifier, log)

	eventRepo.EXPECT().GetByID(mock.Anything, "missing").Return(nil, domain.ErrEventNotFound)

	_, err := svc.Book(context.Background(), "missing", "u1")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrEventNotFound)
}

func TestBookingService_Book_UserNotFound(t *testing.T) {
	bookingRepo := mocks.NewMockBookingRepo(t)
	eventRepo := mocks.NewMockEventRepo(t)
	userRepo := mocks.NewMockUserRepo(t)
	notifier := mocks.NewMockBookingNotifier(t)
	log := newTestLogger(t)

	svc := NewBookingService(bookingRepo, eventRepo, userRepo, notifier, log)

	eventRepo.EXPECT().GetByID(mock.Anything, "e1").Return(&domain.Event{ID: "e1"}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, "missing").Return(nil, domain.ErrUserNotFound)

	_, err := svc.Book(context.Background(), "e1", "missing")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestBookingService_Book_CreateError(t *testing.T) {
	bookingRepo := mocks.NewMockBookingRepo(t)
	eventRepo := mocks.NewMockEventRepo(t)
	userRepo := mocks.NewMockUserRepo(t)
	notifier := mocks.NewMockBookingNotifier(t)
	log := newTestLogger(t)

	svc := NewBookingService(bookingRepo, eventRepo, userRepo, notifier, log)

	eventRepo.EXPECT().GetByID(mock.Anything, "e1").Return(&domain.Event{ID: "e1", RequiresPayment: true}, nil)
	userRepo.EXPECT().GetByID(mock.Anything, "u1").Return(&domain.User{ID: "u1"}, nil)
	bookingRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.ErrNoAvailableSpots)

	_, err := svc.Book(context.Background(), "e1", "u1")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNoAvailableSpots)
}

func TestBookingService_Confirm_Success(t *testing.T) {
	bookingRepo := mocks.NewMockBookingRepo(t)
	eventRepo := mocks.NewMockEventRepo(t)
	userRepo := mocks.NewMockUserRepo(t)
	notifier := mocks.NewMockBookingNotifier(t)
	log := newTestLogger(t)

	svc := NewBookingService(bookingRepo, eventRepo, userRepo, notifier, log)

	event := &domain.Event{
		ID:              "e1",
		RequiresPayment: true,
		BookingTTL:      20 * time.Minute,
	}
	user := &domain.User{ID: "u1", Username: "alice"}

	eventRepo.EXPECT().GetByID(mock.Anything, "e1").Return(event, nil)
	bookingRepo.EXPECT().Confirm(mock.Anything, "e1", "u1").Return(nil)
	userRepo.EXPECT().GetByID(mock.Anything, "u1").Return(user, nil)
	notifier.EXPECT().NotifyBookingConfirmed(mock.Anything, user, event).Return()

	err := svc.Confirm(context.Background(), "e1", "u1")

	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
}

func TestBookingService_Confirm_EventNotFound(t *testing.T) {
	bookingRepo := mocks.NewMockBookingRepo(t)
	eventRepo := mocks.NewMockEventRepo(t)
	userRepo := mocks.NewMockUserRepo(t)
	notifier := mocks.NewMockBookingNotifier(t)
	log := newTestLogger(t)

	svc := NewBookingService(bookingRepo, eventRepo, userRepo, notifier, log)

	eventRepo.EXPECT().GetByID(mock.Anything, "e1").Return(nil, domain.ErrEventNotFound)

	err := svc.Confirm(context.Background(), "e1", "u1")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrEventNotFound)
}

func TestBookingService_Confirm_NoPaymentRequired(t *testing.T) {
	bookingRepo := mocks.NewMockBookingRepo(t)
	eventRepo := mocks.NewMockEventRepo(t)
	userRepo := mocks.NewMockUserRepo(t)
	notifier := mocks.NewMockBookingNotifier(t)
	log := newTestLogger(t)

	svc := NewBookingService(bookingRepo, eventRepo, userRepo, notifier, log)

	event := &domain.Event{ID: "e1", RequiresPayment: false}
	eventRepo.EXPECT().GetByID(mock.Anything, "e1").Return(event, nil)

	err := svc.Confirm(context.Background(), "e1", "u1")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestBookingService_Confirm_BookingNotPending(t *testing.T) {
	bookingRepo := mocks.NewMockBookingRepo(t)
	eventRepo := mocks.NewMockEventRepo(t)
	userRepo := mocks.NewMockUserRepo(t)
	notifier := mocks.NewMockBookingNotifier(t)
	log := newTestLogger(t)

	svc := NewBookingService(bookingRepo, eventRepo, userRepo, notifier, log)

	event := &domain.Event{ID: "e1", RequiresPayment: true, BookingTTL: 20 * time.Minute}

	eventRepo.EXPECT().GetByID(mock.Anything, "e1").Return(event, nil)
	bookingRepo.EXPECT().Confirm(mock.Anything, "e1", "u1").Return(domain.ErrBookingNotPending)

	err := svc.Confirm(context.Background(), "e1", "u1")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrBookingNotPending)
}

func TestBookingService_Confirm_Expired(t *testing.T) {
	bookingRepo := mocks.NewMockBookingRepo(t)
	eventRepo := mocks.NewMockEventRepo(t)
	userRepo := mocks.NewMockUserRepo(t)
	notifier := mocks.NewMockBookingNotifier(t)
	log := newTestLogger(t)

	svc := NewBookingService(bookingRepo, eventRepo, userRepo, notifier, log)

	event := &domain.Event{ID: "e1", RequiresPayment: true, BookingTTL: 10 * time.Minute}

	eventRepo.EXPECT().GetByID(mock.Anything, "e1").Return(event, nil)
	bookingRepo.EXPECT().Confirm(mock.Anything, "e1", "u1").Return(domain.ErrBookingExpired)

	err := svc.Confirm(context.Background(), "e1", "u1")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrBookingExpired)
}

func TestBookingService_Confirm_BookingNotFound(t *testing.T) {
	bookingRepo := mocks.NewMockBookingRepo(t)
	eventRepo := mocks.NewMockEventRepo(t)
	userRepo := mocks.NewMockUserRepo(t)
	notifier := mocks.NewMockBookingNotifier(t)
	log := newTestLogger(t)

	svc := NewBookingService(bookingRepo, eventRepo, userRepo, notifier, log)

	event := &domain.Event{ID: "e1", RequiresPayment: true}
	eventRepo.EXPECT().GetByID(mock.Anything, "e1").Return(event, nil)
	bookingRepo.EXPECT().Confirm(mock.Anything, "e1", "u1").Return(domain.ErrBookingNotFound)

	err := svc.Confirm(context.Background(), "e1", "u1")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrBookingNotFound)
}

func TestBookingService_CancelExpired_Success(t *testing.T) {
	bookingRepo := mocks.NewMockBookingRepo(t)
	eventRepo := mocks.NewMockEventRepo(t)
	userRepo := mocks.NewMockUserRepo(t)
	notifier := mocks.NewMockBookingNotifier(t)
	log := newTestLogger(t)

	svc := NewBookingService(bookingRepo, eventRepo, userRepo, notifier, log)

	cancelled := []*domain.Booking{
		{ID: "b1", EventID: "e1", UserID: "u1"},
		{ID: "b2", EventID: "e2", UserID: "u2"},
	}
	user1 := &domain.User{ID: "u1"}
	user2 := &domain.User{ID: "u2"}
	event1 := &domain.Event{ID: "e1", Title: "Event 1"}
	event2 := &domain.Event{ID: "e2", Title: "Event 2"}

	bookingRepo.EXPECT().CancelExpired(mock.Anything).Return(cancelled, nil)
	userRepo.EXPECT().GetByID(mock.Anything, "u1").Return(user1, nil)
	userRepo.EXPECT().GetByID(mock.Anything, "u2").Return(user2, nil)
	eventRepo.EXPECT().GetByID(mock.Anything, "e1").Return(event1, nil)
	eventRepo.EXPECT().GetByID(mock.Anything, "e2").Return(event2, nil)
	notifier.EXPECT().NotifyBookingCancelled(mock.Anything, user1, event1).Return()
	notifier.EXPECT().NotifyBookingCancelled(mock.Anything, user2, event2).Return()

	result, err := svc.CancelExpired(context.Background())

	require.NoError(t, err)
	assert.Len(t, result, 2)

	time.Sleep(100 * time.Millisecond) // goroutine notify
}

func TestBookingService_CancelExpired_NoneExpired(t *testing.T) {
	bookingRepo := mocks.NewMockBookingRepo(t)
	eventRepo := mocks.NewMockEventRepo(t)
	userRepo := mocks.NewMockUserRepo(t)
	notifier := mocks.NewMockBookingNotifier(t)
	log := newTestLogger(t)

	svc := NewBookingService(bookingRepo, eventRepo, userRepo, notifier, log)

	bookingRepo.EXPECT().CancelExpired(mock.Anything).Return(nil, nil)

	result, err := svc.CancelExpired(context.Background())

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestBookingService_CancelExpired_RepoError(t *testing.T) {
	bookingRepo := mocks.NewMockBookingRepo(t)
	eventRepo := mocks.NewMockEventRepo(t)
	userRepo := mocks.NewMockUserRepo(t)
	notifier := mocks.NewMockBookingNotifier(t)
	log := newTestLogger(t)

	svc := NewBookingService(bookingRepo, eventRepo, userRepo, notifier, log)

	bookingRepo.EXPECT().CancelExpired(mock.Anything).Return(nil, errors.New("db error"))

	_, err := svc.CancelExpired(context.Background())

	require.Error(t, err)
}

func TestBookingService_ListByUser_Success(t *testing.T) {
	bookingRepo := mocks.NewMockBookingRepo(t)
	eventRepo := mocks.NewMockEventRepo(t)
	userRepo := mocks.NewMockUserRepo(t)
	notifier := mocks.NewMockBookingNotifier(t)
	log := newTestLogger(t)

	svc := NewBookingService(bookingRepo, eventRepo, userRepo, notifier, log)

	bookings := []*domain.Booking{
		{ID: "b1", EventID: "e1", UserID: "u1", Status: domain.BookingStatusPending},
	}
	bookingRepo.EXPECT().ListByUser(mock.Anything, "u1").Return(bookings, nil)

	result, err := svc.ListByUser(context.Background(), "u1")

	require.NoError(t, err)
	assert.Len(t, result, 1)
}
