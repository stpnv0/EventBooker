-- +goose Up
CREATE TABLE IF NOT EXISTS bookings (
    id           UUID PRIMARY KEY,
    event_id     UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       VARCHAR(35) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'confirmed', 'cancelled')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW ()
);

CREATE UNIQUE INDEX idx_bookings_active_unique
    ON bookings (event_id, user_id)
    WHERE status IN ('pending', 'confirmed');

CREATE INDEX idx_bookings_pending_created
    ON bookings (created_at)
    WHERE status = 'pending';

CREATE INDEX idx_bookings_event_id ON bookings (event_id);
CREATE INDEX idx_bookings_user_id ON bookings (user_id);

-- +goose Down
DROP TABLE IF EXISTS bookings;
