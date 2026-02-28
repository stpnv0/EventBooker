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
)

func TestEventService_CreateEvent_Success(t *testing.T) {
	eventRepo := mocks.NewMockEventRepo(t)
	bookingRepo := mocks.NewMockBookingRepo(t)
	svc := NewEventService(eventRepo, bookingRepo)

	eventRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	requiresPayment := true
	input := domain.CreateEventInput{
		Title:           "Concert",
		Description:     "Live music",
		EventDate:       time.Now().Add(24 * time.Hour),
		TotalSpots:      100,
		BookingTTL:      30 * time.Minute,
		RequiresPayment: &requiresPayment,
	}

	event, err := svc.CreateEvent(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "Concert", event.Title)
	assert.Equal(t, "Live music", event.Description)
	assert.Equal(t, 100, event.TotalSpots)
	assert.True(t, event.RequiresPayment)
	assert.Equal(t, 30*time.Minute, event.BookingTTL)
	assert.NotEmpty(t, event.ID)
}

func TestEventService_CreateEvent_DefaultTTL(t *testing.T) {
	eventRepo := mocks.NewMockEventRepo(t)
	bookingRepo := mocks.NewMockBookingRepo(t)
	svc := NewEventService(eventRepo, bookingRepo)

	eventRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	input := domain.CreateEventInput{
		Title:      "Workshop",
		EventDate:  time.Now().Add(time.Hour),
		TotalSpots: 10,
	}

	event, err := svc.CreateEvent(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, defaultBookingTTL, event.BookingTTL)
}

func TestEventService_CreateEvent_DefaultRequiresPayment(t *testing.T) {
	eventRepo := mocks.NewMockEventRepo(t)
	bookingRepo := mocks.NewMockBookingRepo(t)
	svc := NewEventService(eventRepo, bookingRepo)

	eventRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	input := domain.CreateEventInput{
		Title:      "Meetup",
		EventDate:  time.Now().Add(time.Hour),
		TotalSpots: 50,
	}

	event, err := svc.CreateEvent(context.Background(), input)

	require.NoError(t, err)
	assert.True(t, event.RequiresPayment)
}

func TestEventService_CreateEvent_NoPaymentRequired(t *testing.T) {
	eventRepo := mocks.NewMockEventRepo(t)
	bookingRepo := mocks.NewMockBookingRepo(t)
	svc := NewEventService(eventRepo, bookingRepo)

	eventRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	f := false
	input := domain.CreateEventInput{
		Title:           "Free Event",
		EventDate:       time.Now().Add(time.Hour),
		TotalSpots:      10,
		RequiresPayment: &f,
	}

	event, err := svc.CreateEvent(context.Background(), input)

	require.NoError(t, err)
	assert.False(t, event.RequiresPayment)
}

func TestEventService_CreateEvent_EmptyTitle(t *testing.T) {
	svc := NewEventService(nil, nil)

	input := domain.CreateEventInput{
		EventDate:  time.Now().Add(time.Hour),
		TotalSpots: 10,
	}

	_, err := svc.CreateEvent(context.Background(), input)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestEventService_CreateEvent_ZeroSpots(t *testing.T) {
	svc := NewEventService(nil, nil)

	input := domain.CreateEventInput{
		Title:      "Test",
		EventDate:  time.Now().Add(time.Hour),
		TotalSpots: 0,
	}

	_, err := svc.CreateEvent(context.Background(), input)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestEventService_CreateEvent_PastDate(t *testing.T) {
	svc := NewEventService(nil, nil)

	input := domain.CreateEventInput{
		Title:      "Test",
		EventDate:  time.Now().Add(-time.Hour),
		TotalSpots: 10,
	}

	_, err := svc.CreateEvent(context.Background(), input)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestEventService_CreateEvent_RepoError(t *testing.T) {
	eventRepo := mocks.NewMockEventRepo(t)
	bookingRepo := mocks.NewMockBookingRepo(t)
	svc := NewEventService(eventRepo, bookingRepo)

	repoErr := errors.New("db error")
	eventRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(repoErr)

	input := domain.CreateEventInput{
		Title:      "Test",
		EventDate:  time.Now().Add(time.Hour),
		TotalSpots: 10,
	}

	_, err := svc.CreateEvent(context.Background(), input)

	require.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
}

func TestEventService_GetDetails_Success(t *testing.T) {
	eventRepo := mocks.NewMockEventRepo(t)
	bookingRepo := mocks.NewMockBookingRepo(t)
	svc := NewEventService(eventRepo, bookingRepo)

	eventID := "event-123"
	details := &domain.EventDetails{
		Event:          domain.Event{ID: eventID, Title: "Concert", TotalSpots: 100},
		AvailableSpots: 98,
	}
	bookings := []*domain.Booking{
		{ID: "b1", EventID: eventID, UserID: "u1", Status: domain.BookingStatusPending},
		{ID: "b2", EventID: eventID, UserID: "u2", Status: domain.BookingStatusConfirmed},
	}

	eventRepo.EXPECT().GetDetails(mock.Anything, eventID).Return(details, nil)
	bookingRepo.EXPECT().ListByEvent(mock.Anything, eventID).Return(bookings, nil)

	result, err := svc.GetDetails(context.Background(), eventID)

	require.NoError(t, err)
	assert.Equal(t, eventID, result.Event.ID)
	assert.Equal(t, 98, result.AvailableSpots)
	assert.Len(t, result.Bookings, 2)
}

func TestEventService_GetDetails_NotFound(t *testing.T) {
	eventRepo := mocks.NewMockEventRepo(t)
	bookingRepo := mocks.NewMockBookingRepo(t)
	svc := NewEventService(eventRepo, bookingRepo)

	eventRepo.EXPECT().GetDetails(mock.Anything, "missing").Return(nil, domain.ErrEventNotFound)

	_, err := svc.GetDetails(context.Background(), "missing")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrEventNotFound)
}

func TestEventService_List_Success(t *testing.T) {
	eventRepo := mocks.NewMockEventRepo(t)
	bookingRepo := mocks.NewMockBookingRepo(t)
	svc := NewEventService(eventRepo, bookingRepo)

	events := []*domain.Event{
		{ID: "e1", Title: "Event 1"},
		{ID: "e2", Title: "Event 2"},
	}
	eventRepo.EXPECT().List(mock.Anything).Return(events, nil)

	result, err := svc.List(context.Background())

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestEventService_List_Error(t *testing.T) {
	eventRepo := mocks.NewMockEventRepo(t)
	bookingRepo := mocks.NewMockBookingRepo(t)
	svc := NewEventService(eventRepo, bookingRepo)

	eventRepo.EXPECT().List(mock.Anything).Return(nil, errors.New("db error"))

	_, err := svc.List(context.Background())

	require.Error(t, err)
}
