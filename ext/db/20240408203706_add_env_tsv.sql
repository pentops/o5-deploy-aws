-- +goose Up
ALTER TABLE environment
ADD COLUMN tsv_full_name tsvector GENERATED ALWAYS AS (
	to_tsvector('english', state#>>'{config,fullName}')
) STORED;

CREATE INDEX idx_env_tsv_full_name ON environment
USING GIN(tsv_full_name);

-- +goose Down

DROP INDEX idx_env_tsv_full_name;

ALTER TABLE environment DROP COLUMN tsv_full_name;
