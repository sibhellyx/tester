-- +goose Up
-- +goose StatementBegin
ALTER TABLE stages
DROP CONSTRAINT stages_users_check,
ADD CONSTRAINT stages_users_check CHECK (
    (
        type = 'ramp_down'
        AND target_users >= 0
    )
    OR (
        type != 'ramp_down'
        AND target_users > 0
    )
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
ALTER TABLE stages
DROP CONSTRAINT stages_users_check,
ADD CONSTRAINT stages_users_check CHECK (target_users > 0);

-- +goose StatementEnd