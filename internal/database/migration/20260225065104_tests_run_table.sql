-- +goose Up
-- +goose StatementBegin
CREATE TABLE
    IF NOT EXISTS test_runs (
        id TEXT NOT NULL,
        scenario_id TEXT NOT NULL,
        status TEXT NOT NULL DEFAULT 'pending',
        started_at TIMESTAMPTZ,
        finished_at TIMESTAMPTZ,
        error TEXT NOT NULL DEFAULT '',
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
        CONSTRAINT test_runs_pkey PRIMARY KEY (id),
        CONSTRAINT test_runs_scenario_fk FOREIGN KEY (scenario_id) REFERENCES scenarios (id) ON DELETE CASCADE,
        CONSTRAINT test_runs_status_check CHECK (
            status IN (
                'pending',
                'running',
                'stopped',
                'finished',
                'failed'
            )
        )
    );

CREATE INDEX IF NOT EXISTS idx_test_runs_scenario_id ON test_runs (scenario_id);

CREATE INDEX IF NOT EXISTS idx_test_runs_status ON test_runs (status);

CREATE INDEX IF NOT EXISTS idx_test_runs_started_at ON test_runs (started_at DESC NULLS LAST);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS test_runs;

-- +goose StatementEnd