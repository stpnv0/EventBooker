package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/stpnv0/EventBooker/internal/domain"
	"github.com/stpnv0/EventBooker/internal/handler/dto"
	"github.com/wb-go/wbf/ginext"
)

type EventSvc interface {
	CreateEvent(ctx context.Context, input domain.CreateEventInput) (*domain.Event, error)
	GetDetails(ctx context.Context, id string) (*domain.EventDetails, error)
	List(ctx context.Context) ([]*domain.Event, error)
}

type BookingSvc interface {
	Book(ctx context.Context, eventID, userID string) (*domain.Booking, error)
	Confirm(ctx context.Context, eventID, userID string) error
	ListByUser(ctx context.Context, userID string) ([]*domain.Booking, error)
}

type UserSvc interface {
	Create(ctx context.Context, input domain.CreateUserInput) (*domain.User, error)
	List(ctx context.Context) ([]*domain.User, error)
}

type Handler struct {
	eventService   EventSvc
	bookingService BookingSvc
	userService    UserSvc
}

func NewHandler(eventService EventSvc, bookingService BookingSvc, userService UserSvc) *Handler {
	return &Handler{
		eventService:   eventService,
		bookingService: bookingService,
		userService:    userService,
	}
}

// Events
func (h *Handler) CreateEvent(c *ginext.Context) {
	var req dto.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	eventDate, err := time.Parse(time.RFC3339, req.EventDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid event_date format, expected RFC3339",
		})
		return
	}

	input := domain.CreateEventInput{
		Title:           req.Title,
		Description:     req.Description,
		EventDate:       eventDate,
		TotalSpots:      req.TotalSpots,
		RequiresPayment: req.RequiresPayment,
		BookingTTL:      time.Duration(req.BookingTTL) * time.Minute,
	}

	event, err := h.eventService.CreateEvent(c.Request.Context(), input)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToEventResponse(event))
}

func (h *Handler) GetEvent(c *ginext.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid event id"})
		return
	}

	details, err := h.eventService.GetDetails(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToEventDetailsResponse(details))
}

func (h *Handler) ListEvents(c *ginext.Context) {
	events, err := h.eventService.List(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}

	resp := make([]dto.EventResponse, 0, len(events))
	for _, e := range events {
		resp = append(resp, dto.ToEventResponse(e))
	}

	c.JSON(http.StatusOK, resp)
}

// Bookings

func (h *Handler) BookEvent(c *ginext.Context) {
	eventID := c.Param("id")
	if _, err := uuid.Parse(eventID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid event id"})
		return
	}

	var req dto.BookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	booking, err := h.bookingService.Book(c.Request.Context(), eventID, req.UserID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToBookingResponse(booking))
}

func (h *Handler) ConfirmBooking(c *ginext.Context) {
	eventID := c.Param("id")
	if _, err := uuid.Parse(eventID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid event id"})
		return
	}

	var req dto.ConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.bookingService.Confirm(c.Request.Context(), eventID, req.UserID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ginext.H{"status": "confirmed"})
}

func (h *Handler) GetUserBookings(c *ginext.Context) {
	userID := c.Param("id")
	if _, err := uuid.Parse(userID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid user id"})
		return
	}
	bookings, err := h.bookingService.ListByUser(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	resp := make([]dto.BookingResponse, 0, len(bookings))
	for _, b := range bookings {
		resp = append(resp, dto.ToBookingResponse(b))
	}

	c.JSON(http.StatusOK, resp)
}

// Users

func (h *Handler) CreateUser(c *ginext.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	input := domain.CreateUserInput{
		Username:       req.Username,
		TelegramChatID: req.TelegramChatID,
	}

	user, err := h.userService.Create(c.Request.Context(), input)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToUserResponse(user))
}

func (h *Handler) ListUsers(c *ginext.Context) {
	users, err := h.userService.List(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}

	resp := make([]dto.UserResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, dto.ToUserResponse(u))
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) handleError(c *ginext.Context, err error) {
	c.Set("error", err.Error())

	switch {
	case errors.Is(err, domain.ErrEventNotFound),
		errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrBookingNotFound):
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: err.Error()})

	case errors.Is(err, domain.ErrNoAvailableSpots),
		errors.Is(err, domain.ErrAlreadyBooked),
		errors.Is(err, domain.ErrBookingNotPending),
		errors.Is(err, domain.ErrBookingExpired):
		c.JSON(http.StatusConflict, dto.ErrorResponse{Error: err.Error()})

	case errors.Is(err, domain.ErrValidation),
		errors.Is(err, domain.ErrUsernameTaken):
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})

	default:
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "internal server error"})
	}
}
