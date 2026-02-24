-- +goose Up
-- +goose StatementBegin
CREATE TABLE
    IF NOT EXISTS chaos_params (
        id SERIAL NOT NULL,
        stage_id INTEGER NOT NULL,
        scenario_id TEXT NOT NULL,
        type TEXT NOT NULL,
        target_container_id TEXT NOT NULL,
        start_delay INTEGER NOT NULL DEFAULT 0,
        duration INTEGER NOT NULL,
        delay TEXT NOT NULL DEFAULT '',
        jitter TEXT NOT NULL DEFAULT '',
        packet_loss INTEGER NOT NULL DEFAULT 0,
        cpu_quota BIGINT NOT NULL DEFAULT 0,
        memory_bytes BIGINT NOT NULL DEFAULT 0,
        CONSTRAINT chaos_params_pkey PRIMARY KEY (id),
        CONSTRAINT chaos_params_stage_fk FOREIGN KEY (scenario_id, stage_id) REFERENCES stages (scenario_id, id) ON DELETE CASCADE,
        CONSTRAINT chaos_params_duration_check CHECK (duration > 0),
        CONSTRAINT chaos_params_start_delay_check CHECK (start_delay >= 0),
        CONSTRAINT chaos_params_packet_loss_check CHECK (packet_loss BETWEEN 0 AND 100),
        CONSTRAINT chaos_params_type_check CHECK (
            type IN (
                'component_shutdown',
                'network_delay',
                'packet_loss',
                'resource_limit'
            )
        )
    );

CREATE INDEX IF NOT EXISTS idx_chaos_params_stage ON chaos_params (scenario_id, stage_id);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS chaos_params;

-- +goose StatementEnd