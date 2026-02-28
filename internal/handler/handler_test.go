package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stpnv0/EventBooker/internal/domain"
	"github.com/stpnv0/EventBooker/internal/handler/dto"
	hmocks "github.com/stpnv0/EventBooker/internal/handler/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wb-go/wbf/ginext"
)

func setupRouter(t *testing.T) (*hmocks.MockEventSvc, *hmocks.MockBookingSvc, *hmocks.MockUserSvc, http.Handler) {
	t.Helper()
	eventSvc := hmocks.NewMockEventSvc(t)
	bookingSvc := hmocks.NewMockBookingSvc(t)
	userSvc := hmocks.NewMockUserSvc(t)

	h := NewHandler(eventSvc, bookingSvc, userSvc)

	r := ginext.New("test")
	api := r.Group("/api")
	{
		api.POST("/events", h.CreateEvent)
		api.GET("/events", h.ListEvents)
		api.GET("/events/:id", h.GetEvent)
		api.POST("/events/:id/book", h.BookEvent)
		api.POST("/events/:id/confirm", h.ConfirmBooking)
		api.POST("/users", h.CreateUser)
		api.GET("/users", h.ListUsers)
		api.GET("/users/:id/bookings", h.GetUserBookings)
	}

	return eventSvc, bookingSvc, userSvc, r
}

// --- Events ---

func TestHandler_CreateEvent_Success(t *testing.T) {
	eventSvc, _, _, r := setupRouter(t)

	now := time.Now().Add(24 * time.Hour)
	event := &domain.Event{
		ID:              uuid.New().String(),
		Title:           "Concert",
		Description:     "Live music",
		EventDate:       now,
		TotalSpots:      100,
		RequiresPayment: true,
		BookingTTL:      20 * time.Minute,
		CreatedAt:       time.Now(),
	}

	eventSvc.EXPECT().CreateEvent(mock.Anything, mock.Anything).Return(event, nil)

	body, _ := json.Marshal(dto.CreateEventRequest{
		Title:       "Concert",
		Description: "Live music",
		EventDate:   now.Format(time.RFC3339),
		TotalSpots:  100,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp dto.EventResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Concert", resp.Title)
}

func TestHandler_CreateEvent_BadRequest(t *testing.T) {
	_, _, _, r := setupRouter(t)

	body := []byte(`{"title":""}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateEvent_InvalidDate(t *testing.T) {
	_, _, _, r := setupRouter(t)

	body := []byte(`{"title":"X","description":"Y","event_date":"not-a-date","total_spots":10}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetEvent_Success(t *testing.T) {
	eventSvc, _, _, r := setupRouter(t)

	eventID := uuid.New().String()
	details := &domain.EventDetails{
		Event:          domain.Event{ID: eventID, Title: "Concert", TotalSpots: 100, EventDate: time.Now(), CreatedAt: time.Now()},
		AvailableSpots: 95,
		Bookings:       []domain.Booking{},
	}

	eventSvc.EXPECT().GetDetails(mock.Anything, eventID).Return(details, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/events/"+eventID, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.EventDetailsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 95, resp.AvailableSpots)
}

func TestHandler_GetEvent_InvalidID(t *testing.T) {
	_, _, _, r := setupRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/events/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetEvent_NotFound(t *testing.T) {
	eventSvc, _, _, r := setupRouter(t)

	eventID := uuid.New().String()
	eventSvc.EXPECT().GetDetails(mock.Anything, eventID).Return(nil, domain.ErrEventNotFound)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/events/"+eventID, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ListEvents_Success(t *testing.T) {
	eventSvc, _, _, r := setupRouter(t)

	events := []*domain.Event{
		{ID: "e1", Title: "Event 1", EventDate: time.Now(), CreatedAt: time.Now()},
		{ID: "e2", Title: "Event 2", EventDate: time.Now(), CreatedAt: time.Now()},
	}
	eventSvc.EXPECT().List(mock.Anything).Return(events, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []dto.EventResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
}

// --- Bookings ---

func TestHandler_BookEvent_Success(t *testing.T) {
	_, bookingSvc, _, r := setupRouter(t)

	eventID := uuid.New().String()
	userID := uuid.New().String()
	booking := &domain.Booking{
		ID:        uuid.New().String(),
		EventID:   eventID,
		UserID:    userID,
		Status:    domain.BookingStatusPending,
		CreatedAt: time.Now(),
	}

	bookingSvc.EXPECT().Book(mock.Anything, eventID, userID).Return(booking, nil)

	body, _ := json.Marshal(dto.BookRequest{UserID: userID})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/book", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp dto.BookingResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "pending", resp.Status)
}

func TestHandler_BookEvent_InvalidEventID(t *testing.T) {
	_, _, _, r := setupRouter(t)

	body := []byte(`{"user_id":"` + uuid.New().String() + `"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/events/bad-id/book", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_BookEvent_NoSpots(t *testing.T) {
	_, bookingSvc, _, r := setupRouter(t)

	eventID := uuid.New().String()
	userID := uuid.New().String()

	bookingSvc.EXPECT().Book(mock.Anything, eventID, userID).Return(nil, domain.ErrNoAvailableSpots)

	body, _ := json.Marshal(dto.BookRequest{UserID: userID})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/book", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_ConfirmBooking_Success(t *testing.T) {
	_, bookingSvc, _, r := setupRouter(t)

	eventID := uuid.New().String()
	userID := uuid.New().String()

	bookingSvc.EXPECT().Confirm(mock.Anything, eventID, userID).Return(nil)

	body, _ := json.Marshal(dto.ConfirmRequest{UserID: userID})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ConfirmBooking_InvalidEventID(t *testing.T) {
	_, _, _, r := setupRouter(t)

	body := []byte(`{"user_id":"` + uuid.New().String() + `"}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/events/bad-id/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ConfirmBooking_Expired(t *testing.T) {
	_, bookingSvc, _, r := setupRouter(t)

	eventID := uuid.New().String()
	userID := uuid.New().String()

	bookingSvc.EXPECT().Confirm(mock.Anything, eventID, userID).Return(domain.ErrBookingExpired)

	body, _ := json.Marshal(dto.ConfirmRequest{UserID: userID})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// --- Users ---

func TestHandler_CreateUser_Success(t *testing.T) {
	_, _, userSvc, r := setupRouter(t)

	user := &domain.User{
		ID:        uuid.New().String(),
		Username:  "alice",
		CreatedAt: time.Now(),
	}
	userSvc.EXPECT().Create(mock.Anything, mock.Anything).Return(user, nil)

	body, _ := json.Marshal(dto.CreateUserRequest{Username: "alice"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp dto.UserResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "alice", resp.Username)
}

func TestHandler_CreateUser_BadRequest(t *testing.T) {
	_, _, _, r := setupRouter(t)

	body := []byte(`{}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateUser_UsernameTaken(t *testing.T) {
	_, _, userSvc, r := setupRouter(t)

	userSvc.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, domain.ErrUsernameTaken)

	body, _ := json.Marshal(dto.CreateUserRequest{Username: "taken"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListUsers_Success(t *testing.T) {
	_, _, userSvc, r := setupRouter(t)

	users := []*domain.User{
		{ID: "u1", Username: "alice", CreatedAt: time.Now()},
	}
	userSvc.EXPECT().List(mock.Anything).Return(users, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []dto.UserResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
}

func TestHandler_GetUserBookings_Success(t *testing.T) {
	_, bookingSvc, _, r := setupRouter(t)

	userID := uuid.New().String()
	bookings := []*domain.Booking{
		{ID: "b1", EventID: "e1", UserID: userID, Status: domain.BookingStatusPending, CreatedAt: time.Now()},
	}

	bookingSvc.EXPECT().ListByUser(mock.Anything, userID).Return(bookings, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users/"+userID+"/bookings", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []dto.BookingResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
}

func TestHandler_GetUserBookings_InvalidID(t *testing.T) {
	_, _, _, r := setupRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users/bad-id/bookings", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_HandleError_InternalError(t *testing.T) {
	eventSvc, _, _, r := setupRouter(t)

	eventID := uuid.New().String()
	eventSvc.EXPECT().GetDetails(mock.Anything, eventID).Return(nil, assert.AnError)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/events/"+eventID, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
