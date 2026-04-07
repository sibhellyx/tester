-- +goose Up
-- +goose StatementBegin
ALTER TABLE chaos_params
    RENAME COLUMN cpu_quota TO cpu_percent;

ALTER TABLE chaos_params
    ALTER COLUMN cpu_percent TYPE DOUBLE PRECISION USING cpu_percent::DOUBLE PRECISION;

ALTER TABLE chaos_params
    RENAME COLUMN memory_bytes TO memory_mb;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE chaos_params
    RENAME COLUMN cpu_percent TO cpu_quota;

ALTER TABLE chaos_params
    ALTER COLUMN cpu_quota TYPE BIGINT USING cpu_quota::BIGINT;

ALTER TABLE chaos_params
    RENAME COLUMN memory_mb TO memory_bytes;
-- +goose StatementEnd
