-- +goose Up
-- +goose StatementBegin
ALTER TABLE call_results ADD COLUMN IF NOT EXISTS stage_id INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE call_results DROP COLUMN IF EXISTS stage_id;
-- +goose StatementEnd
