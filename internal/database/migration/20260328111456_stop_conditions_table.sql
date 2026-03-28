-- +goose Up
-- +goose StatementBegin
CREATE TABLE
    IF NOT EXISTS stop_conditions (
        id SERIAL NOT NULL,
        scenario_id TEXT NOT NULL,
        target_container_id TEXT NOT NULL DEFAULT '',
        error_rate_percent DOUBLE PRECISION,
        max_response_time_sec DOUBLE PRECISION,
        max_cpu_percent DOUBLE PRECISION,
        max_ram_percent DOUBLE PRECISION,
        CONSTRAINT stop_conditions_pkey PRIMARY KEY (id),
        CONSTRAINT stop_conditions_scenario_fk FOREIGN KEY (scenario_id) REFERENCES scenarios (id) ON DELETE CASCADE,
        CONSTRAINT stop_conditions_error_rate_check CHECK (
            error_rate_percent IS NULL
            OR error_rate_percent BETWEEN 1 AND 100
        ),
        CONSTRAINT stop_conditions_cpu_check CHECK (
            max_cpu_percent IS NULL
            OR max_cpu_percent BETWEEN 1 AND 95
        ),
        CONSTRAINT stop_conditions_ram_check CHECK (
            max_ram_percent IS NULL
            OR max_ram_percent BETWEEN 1 AND 95
        ),
        CONSTRAINT stop_conditions_response_time_check CHECK (
            max_response_time_sec IS NULL
            OR max_response_time_sec > 0
        )
    );

CREATE UNIQUE INDEX IF NOT EXISTS idx_stop_conditions_scenario_id ON stop_conditions (scenario_id);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS stop_conditions;

-- +goose StatementEnd