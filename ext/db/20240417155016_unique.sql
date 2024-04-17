-- +goose Up

ALTER TABLE stack ADD CONSTRAINT stack_unique UNIQUE (env_name, app_name);

-- +goose Down

ALTER TABLE stack DROP CONSTRAINT stack_unique;
