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

type UserRepository struct {
	db       *dbpg.DB
	strategy retry.Strategy
}

func NewUserRepo(db *dbpg.DB) *UserRepository {
	return &UserRepository{
		db: db,
		strategy: retry.Strategy{
			Attempts: 3,
			Delay:    500 * time.Millisecond,
			Backoff:  2,
		},
	}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (id, username, telegram_chat_id, created_at)
 			  VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecWithRetry(ctx, r.strategy, query, user.ID, user.Username, user.TelegramChatID, time.Now())
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrUsernameTaken
		}
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, username, telegram_chat_id, created_at 
    		  FROM users
    		  WHERE id=$1`

	row, err := r.db.QueryRowWithRetry(ctx, r.strategy, query, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	var u domain.User
	if err = row.Scan(&u.ID, &u.Username, &u.TelegramChatID, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}

	return &u, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `SELECT id, username, telegram_chat_id, created_at 
    		  FROM users
    		  WHERE username=$1`

	row, err := r.db.QueryRowWithRetry(ctx, r.strategy, query, username)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	var u domain.User
	if err = row.Scan(&u.ID, &u.Username, &u.TelegramChatID, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}

	return &u, nil
}

func (r *UserRepository) List(ctx context.Context) ([]*domain.User, error) {
	query := `SELECT id, username, telegram_chat_id, created_at 
			  FROM users 
			  ORDER BY username DESC`

	rows, err := r.db.QueryWithRetry(ctx, r.strategy, query)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var res []*domain.User
	for rows.Next() {
		var u domain.User
		if err = rows.Scan(&u.ID, &u.Username, &u.TelegramChatID, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		res = append(res, &u)
	}

	return res, rows.Err()
}
