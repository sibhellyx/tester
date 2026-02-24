-- +goose Up
-- +goose StatementBegin
CREATE TABLE
    IF NOT EXISTS scenarios (
        id TEXT NOT NULL,
        name TEXT NOT NULL DEFAULT '',
        base_url TEXT NOT NULL,
        total_duration INTEGER NOT NULL DEFAULT 0,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
        CONSTRAINT scenarios_pkey PRIMARY KEY (id),
        CONSTRAINT scenarios_base_url_check CHECK (base_url <> ''),
        CONSTRAINT scenarios_duration_check CHECK (total_duration >= 0)
    );

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS scenarios;

-- +goose StatementEnd