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

type BookingRepository struct {
	db       *dbpg.DB
	strategy retry.Strategy
}

func NewBookingRepo(db *dbpg.DB) *BookingRepository {
	return &BookingRepository{
		db: db,
		strategy: retry.Strategy{
			Attempts: 3,
			Delay:    500 * time.Millisecond,
			Backoff:  2,
		},
	}
}

func (r *BookingRepository) Create(ctx context.Context, b *domain.Booking) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Проверяем наличие мест
	spotQuery := `SELECT total_spots FROM events WHERE id = $1 FOR UPDATE`
	var totalSpots int
	var activeBookings int
	if err = tx.QueryRowContext(ctx, spotQuery, b.EventID).Scan(&totalSpots); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrEventNotFound
		}
		return fmt.Errorf("get total spots: %w", err)
	}

	activeQuery := `SELECT COUNT(*) FROM bookings
              WHERE event_id = $1 AND status = ANY($2)`
	if err = tx.QueryRowContext(
		ctx, activeQuery, b.EventID,
		pq.Array(domain.ActiveStatuses),
	).Scan(&activeBookings); err != nil {
		return fmt.Errorf("count bookings: %w", err)
	}

	if activeBookings >= totalSpots {
		return domain.ErrNoAvailableSpots
	}

	// Создаем бронь
	query := `INSERT INTO bookings (id, event_id, user_id, status, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = tx.ExecContext(
		ctx, query, b.ID, b.EventID,
		b.UserID, b.Status, b.CreatedAt, b.UpdatedAt,
	)

	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyBooked
		}
		return fmt.Errorf("insert booking: %w", err)
	}

	return tx.Commit()
}

func (r *BookingRepository) GetByEventAndUser(ctx context.Context, eventID, userID string) (*domain.Booking, error) {
	query := `SELECT id, event_id, user_id, status, created_at, updated_at
			  FROM bookings
			  WHERE event_id=$1 AND user_id=$2  AND status = ANY($3)
			  ORDER BY created_at DESC
              LIMIT 1`

	row, err := r.db.QueryRowWithRetry(ctx, r.strategy, query, eventID, userID, pq.Array(domain.ActiveStatuses))
	if err != nil {
		return nil, fmt.Errorf("get booking: %w", err)
	}

	var b domain.Booking
	if err = row.Scan(&b.ID, &b.EventID, &b.UserID, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrBookingNotFound
		}
		return nil, fmt.Errorf("scan booking: %w", err)
	}

	return &b, nil
}

func (r *BookingRepository) ListByUser(ctx context.Context, userID string) ([]*domain.Booking, error) {
	query := `SELECT id, event_id, user_id, status, created_at, updated_at
              FROM bookings
              WHERE user_id = $1
              ORDER BY created_at DESC`

	rows, err := r.db.QueryWithRetry(ctx, r.strategy, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list bookings by user: %w", err)
	}
	defer rows.Close()

	var res []*domain.Booking
	for rows.Next() {
		var b domain.Booking
		if err = rows.Scan(&b.ID, &b.EventID, &b.UserID, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		res = append(res, &b)
	}

	return res, rows.Err()
}

func (r *BookingRepository) Confirm(ctx context.Context, eventID, userID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Получаем TTL мероприятия
	var ttlSeconds int64
	ttlQuery := `SELECT EXTRACT(EPOCH FROM booking_ttl)::bigint FROM events WHERE id = $1`
	if err = tx.QueryRowContext(ctx, ttlQuery, eventID).Scan(&ttlSeconds); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrEventNotFound
		}
		return fmt.Errorf("get event ttl: %w", err)
	}

	// Атомарно проверяем статус и TTL, обновляем бронь
	query := `UPDATE bookings
			  SET status = $4, updated_at = now()
			  WHERE event_id = $1
			    AND user_id = $2
			    AND status = $3
			    AND created_at + make_interval(secs => $5) >= now()`
	res, err := tx.ExecContext(
		ctx, query, eventID, userID,
		domain.BookingStatusPending, domain.BookingStatusConfirmed,
		ttlSeconds,
	)
	if err != nil {
		return fmt.Errorf("confirm booking: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("booking rows affected: %w", err)
	}
	if rows == 0 {
		// Определяем причину: бронь не найдена, не pending, или истекла
		var status string
		var createdAt time.Time
		checkQuery := `SELECT status, created_at FROM bookings
					   WHERE event_id = $1 AND user_id = $2 AND status = ANY($3)
					   ORDER BY created_at DESC LIMIT 1`
		scanErr := tx.QueryRowContext(ctx, checkQuery, eventID, userID, pq.Array(domain.ActiveStatuses)).
			Scan(&status, &createdAt)
		if scanErr != nil {
			return domain.ErrBookingNotFound
		}
		if status != string(domain.BookingStatusPending) {
			return domain.ErrBookingNotPending
		}
		ttl := time.Duration(ttlSeconds) * time.Second
		if time.Since(createdAt) > ttl {
			return domain.ErrBookingExpired
		}
		return domain.ErrBookingNotFound
	}

	return tx.Commit()
}

func (r *BookingRepository) CancelExpired(ctx context.Context) ([]*domain.Booking, error) {
	query := `
        UPDATE bookings b
        SET status = $2, updated_at = NOW()
        FROM events e
        WHERE b.event_id = e.id
          AND b.status = $1
          AND b.created_at + e.booking_ttl < NOW()
        RETURNING b.id, b.event_id, b.user_id,
                  b.status, b.created_at, b.updated_at`

	rows, err := r.db.QueryWithRetry(
		ctx, r.strategy, query,
		domain.BookingStatusPending, domain.BookingStatusCancelled,
	)
	if err != nil {
		return nil, fmt.Errorf("cancel expired: %w", err)
	}
	defer rows.Close()

	var res []*domain.Booking
	for rows.Next() {
		var b domain.Booking
		if err = rows.Scan(
			&b.ID, &b.EventID, &b.UserID,
			&b.Status, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		res = append(res, &b)
	}

	return res, rows.Err()
}

func (r *BookingRepository) ListByEvent(ctx context.Context, eventID string) ([]*domain.Booking, error) {
	query := `SELECT id, event_id, user_id, status, created_at, updated_at
              FROM bookings
              WHERE event_id = $1 AND status = ANY($2)`
	rows, err := r.db.QueryWithRetry(ctx, r.strategy, query, eventID, pq.Array(domain.ActiveStatuses))
	if err != nil {
		return nil, fmt.Errorf("list bookings by event: %w", err)
	}
	defer rows.Close()

	var res []*domain.Booking
	for rows.Next() {
		var b domain.Booking
		if err = rows.Scan(&b.ID, &b.EventID, &b.UserID, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan booking by event: %w", err)
		}
		res = append(res, &b)
	}

	return res, rows.Err()
}
