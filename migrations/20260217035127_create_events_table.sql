-- +goose Up
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    event_date TIMESTAMPTZ NOT NULL,
    total_spots INT NOT NULL CHECK ( total_spots > 0 ),
    requires_payment BOOLEAN NOT NULL DEFAULT true,
    booking_ttl INTERVAL NOT NULL DEFAULT '20 minutes',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS events;
