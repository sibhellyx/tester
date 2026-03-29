-- +goose Up
-- +goose StatementBegin
ALTER TABLE stages
    DROP CONSTRAINT stages_type_check,
    ADD CONSTRAINT stages_type_check CHECK (type IN ('ramp_up', 'steady', 'ramp_down', 'spike'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE stages
    DROP CONSTRAINT stages_type_check,
    ADD CONSTRAINT stages_type_check CHECK (type IN ('ramp_up', 'steady', 'ramp_down'));
-- +goose StatementEnd
