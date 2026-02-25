-- +goose Up
-- +goose StatementBegin
CREATE TABLE
    IF NOT EXISTS call_results (
        id BIGSERIAL NOT NULL,
        run_id TEXT NOT NULL,
        request_name TEXT NOT NULL DEFAULT '',
        timestamp TIMESTAMPTZ NOT NULL,
        duration_ms BIGINT NOT NULL DEFAULT 0,
        status INTEGER NOT NULL DEFAULT 0,
        error TEXT NOT NULL DEFAULT '',
        bytes_out BIGINT NOT NULL DEFAULT 0,
        bytes_in BIGINT NOT NULL DEFAULT 0,
        CONSTRAINT call_results_pkey PRIMARY KEY (id),
        CONSTRAINT call_results_run_fk FOREIGN KEY (run_id) REFERENCES test_runs (id) ON DELETE CASCADE,
        CONSTRAINT call_results_duration_check CHECK (duration_ms >= 0),
        CONSTRAINT call_results_bytes_out_check CHECK (bytes_out >= 0),
        CONSTRAINT call_results_bytes_in_check CHECK (bytes_in >= 0)
    );

CREATE INDEX IF NOT EXISTS idx_call_results_run_id ON call_results (run_id);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS call_results;

-- +goose StatementEnd