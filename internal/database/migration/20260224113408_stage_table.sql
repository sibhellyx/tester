-- +goose Up
-- +goose StatementBegin
CREATE TABLE
    IF NOT EXISTS stages (
        id INTEGER NOT NULL,
        scenario_id TEXT NOT NULL,
        type TEXT NOT NULL,
        duration INTEGER NOT NULL,
        target_users INTEGER NOT NULL,
        CONSTRAINT stages_pkey PRIMARY KEY (scenario_id, id),
        CONSTRAINT stages_scenario_fk FOREIGN KEY (scenario_id) REFERENCES scenarios (id) ON DELETE CASCADE,
        CONSTRAINT stages_duration_check CHECK (duration > 0),
        CONSTRAINT stages_users_check CHECK (target_users > 0),
        CONSTRAINT stages_type_check CHECK (type IN ('ramp_up', 'steady', 'ramp_down'))
    );

CREATE INDEX IF NOT EXISTS idx_stages_scenario_id ON stages (scenario_id);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS stages;

-- +goose StatementEnd