package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/stpnv0/EventBooker/internal/domain"
	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/retry"
)

type EventRepository struct {
	db       *dbpg.DB
	strategy retry.Strategy
}

func NewEventRepo(db *dbpg.DB) *EventRepository {
	return &EventRepository{
		db: db,
		strategy: retry.Strategy{
			Attempts: 3,
			Delay:    500 * time.Millisecond,
			Backoff:  2,
		},
	}
}

func (r *EventRepository) Create(ctx context.Context, e *domain.Event) error {
	query := `INSERT into events (id, title, description, event_date, total_spots, requires_payment, booking_ttl, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, make_interval(secs => $7), $8, $9)`
	now := time.Now().UTC()
	_, err := r.db.ExecWithRetry(
		ctx, r.strategy, query,
		e.ID, e.Title, e.Description, e.EventDate,
		e.TotalSpots, e.RequiresPayment, e.BookingTTL.Seconds(), now, now,
	)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	return nil
}

func (r *EventRepository) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	query := `SELECT id, title, description, event_date, total_spots, requires_payment,
       		  		EXTRACT(EPOCH FROM booking_ttl)::bigint, created_at, updated_at
			  FROM events 
			  WHERE id=$1`
	row, err := r.db.QueryRowWithRetry(ctx, r.strategy, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrEventNotFound
		}
		return nil, fmt.Errorf("get event: %w", err)
	}

	var e domain.Event
	var ttlSeconds int64
	if err = row.Scan(
		&e.ID, &e.Title, &e.Description, &e.EventDate, &e.TotalSpots,
		&e.RequiresPayment, &ttlSeconds, &e.CreatedAt, &e.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan event: %w", err)
	}
	e.BookingTTL = time.Duration(ttlSeconds) * time.Second

	return &e, nil
}

func (r *EventRepository) List(ctx context.Context) ([]*domain.Event, error) {
	query := `SELECT id, title, description, event_date, total_spots, requires_payment,
		      		EXTRACT(EPOCH FROM booking_ttl)::bigint, created_at, updated_at
			  FROM events 
			  ORDER BY event_date DESC`

	rows, err := r.db.QueryWithRetry(ctx, r.strategy, query)
	if err != nil {
		return nil, fmt.Errorf("list event: %w", err)
	}
	defer rows.Close()

	var res []*domain.Event
	for rows.Next() {
		var e domain.Event
		var ttlSeconds int64
		if err = rows.Scan(
			&e.ID, &e.Title, &e.Description, &e.EventDate, &e.TotalSpots,
			&e.RequiresPayment, &ttlSeconds, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.BookingTTL = time.Duration(ttlSeconds) * time.Second
		res = append(res, &e)
	}

	return res, rows.Err()
}

func (r *EventRepository) GetDetails(ctx context.Context, eventID string) (*domain.EventDetails, error) {
	query := `
		SELECT
            e.id, e.title, e.description, e.event_date,
            e.total_spots, e.requires_payment, EXTRACT(EPOCH FROM e.booking_ttl)::bigint, 
            e.created_at, e.updated_at,
            e.total_spots - COUNT(b.id) AS available_spots
        FROM events e
        LEFT JOIN bookings b
            ON b.event_id = e.id
            AND b.status = ANY($2)
        WHERE e.id = $1
        GROUP BY e.id`

	row, err := r.db.QueryRowWithRetry(ctx, r.strategy, query, eventID, pq.Array(domain.ActiveStatuses))
	if err != nil {
		return nil, fmt.Errorf("get details: %w", err)
	}
	var e domain.EventDetails
	var ttlSeconds int64
	err = row.Scan(
		&e.Event.ID, &e.Event.Title, &e.Event.Description,
		&e.Event.EventDate, &e.Event.TotalSpots, &e.Event.RequiresPayment, &ttlSeconds,
		&e.Event.CreatedAt, &e.Event.UpdatedAt,
		&e.AvailableSpots,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrEventNotFound
		}
		return nil, fmt.Errorf("get event details: %w", err)
	}
	e.Event.BookingTTL = time.Duration(ttlSeconds) * time.Second

	return &e, nil
}
