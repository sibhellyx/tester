-- +goose Up
-- +goose StatementBegin
CREATE TABLE
    IF NOT EXISTS test_requests (
        id SERIAL NOT NULL,
        stage_id INTEGER NOT NULL,
        scenario_id TEXT NOT NULL,
        name TEXT NOT NULL DEFAULT '',
        method TEXT NOT NULL,
        path TEXT NOT NULL DEFAULT '/',
        headers TEXT NOT NULL DEFAULT '{}',
        body TEXT NOT NULL DEFAULT '',
        weight INTEGER NOT NULL DEFAULT 1,
        expected_codes INT ARRAY NOT NULL DEFAULT '{}',
        CONSTRAINT test_requests_pkey PRIMARY KEY (id),
        CONSTRAINT test_requests_stage_fk FOREIGN KEY (scenario_id, stage_id) REFERENCES stages (scenario_id, id) ON DELETE CASCADE,
        CONSTRAINT test_requests_method_check CHECK (
            method IN (
                'GET',
                'POST',
                'PUT',
                'PATCH',
                'DELETE',
                'HEAD',
                'OPTIONS'
            )
        ),
        CONSTRAINT test_requests_weight_check CHECK (weight >= 0)
    );

CREATE INDEX IF NOT EXISTS idx_test_requests_stage ON test_requests (scenario_id, stage_id);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS test_requests;

-- +goose StatementEnd