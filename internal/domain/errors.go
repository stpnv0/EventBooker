package domain

import "errors"

var (
	ErrEventNotFound   = errors.New("event not found")
	ErrUserNotFound    = errors.New("user not found")
	ErrBookingNotFound = errors.New("booking not found")
)

var (
	ErrNoAvailableSpots  = errors.New("no available spots")
	ErrAlreadyBooked     = errors.New("user already has a booking for this event")
	ErrBookingNotPending = errors.New("booking is not in pending status")
	ErrBookingExpired    = errors.New("booking has expired")
)

var (
	ErrUsernameTaken = errors.New("username is already taken")
)

var (
	ErrValidation = errors.New("validation error")
)
