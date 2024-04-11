-- +goose Up

-- Deployment
ALTER TABLE deployment
ADD COLUMN tsv_stack_name tsvector GENERATED ALWAYS AS (
	to_tsvector('english', state->>'stackName')
) STORED;

ALTER TABLE deployment
ADD COLUMN tsv_app_name tsvector GENERATED ALWAYS AS (
	to_tsvector('english', state#>>'{spec,appName}')
) STORED;

ALTER TABLE deployment
ADD COLUMN tsv_version tsvector GENERATED ALWAYS AS (
	to_tsvector('english', state#>>'{spec,version}')
) STORED;

ALTER TABLE deployment
ADD COLUMN tsv_environment_name tsvector GENERATED ALWAYS AS (
	to_tsvector('english', state#>>'{spec,environmentName}')
) STORED;

ALTER TABLE deployment
ADD COLUMN tsv_template_url tsvector GENERATED ALWAYS AS (
	to_tsvector('english', state#>>'{spec,templateUrl}')
) STORED;

ALTER TABLE deployment
ADD COLUMN tsv_ecs_cluster tsvector GENERATED ALWAYS AS (
	to_tsvector('english', state#>>'{spec,ecsCluster}')
) STORED;

CREATE INDEX idx_dep_tsv_stack_name ON deployment
USING GIN(tsv_stack_name);

CREATE INDEX idx_dep__tsv_app_name ON deployment
USING GIN(tsv_app_name);

CREATE INDEX idx_dep_tsv_version ON deployment
USING GIN(tsv_version);

CREATE INDEX idx_dep_tsv_environment_name ON deployment
USING GIN(tsv_environment_name);

CREATE INDEX idx_dep_tsv_template_url ON deployment
USING GIN(tsv_template_url);

CREATE INDEX idx_dep_tsv_ecs_cluster ON deployment
USING GIN(tsv_ecs_cluster);

-- Stack

ALTER TABLE stack
ADD COLUMN tsv_stack_name tsvector GENERATED ALWAYS AS (
	to_tsvector('english', state->>'stackName')
) STORED;

ALTER TABLE stack
ADD COLUMN tsv_app_name tsvector GENERATED ALWAYS AS (
	to_tsvector('english', state->>'applicationName')
) STORED;

ALTER TABLE stack
ADD COLUMN tsv_environment_name tsvector GENERATED ALWAYS AS (
	to_tsvector('english', state->>'environmentName')
) STORED;

CREATE INDEX idx_stack_tsv_stack_name ON stack
USING GIN(tsv_stack_name);

CREATE INDEX idx_stack_tsv_app_name ON stack
USING GIN(tsv_app_name);

CREATE INDEX idx_stack_tsv_environment_name ON stack
USING GIN(tsv_environment_name);

-- +goose Down

DROP INDEX idx_stack_tsv_environment_name;
DROP INDEX idx_stack_tsv_app_name;
DROP INDEX idx_stack_tsv_stack_name;

ALTER TABLE stack DROP COLUMN tsv_environment_name;
ALTER TABLE stack DROP COLUMN tsv_app_name;
ALTER TABLE stack DROP COLUMN tsv_stack_name;

DROP INDEX idx_dep_tsv_template_url;
DROP INDEX idx_dep_tsv_environment_name;
DROP INDEX idx_dep_tsv_version;
DROP INDEX idx_dep__tsv_app_name;
DROP INDEX idx_dep_tsv_stack_name;

ALTER TABLE deployment DROP COLUMN tsv_ecs_cluster;
ALTER TABLE deployment DROP COLUMN tsv_template_url;
ALTER TABLE deployment DROP COLUMN tsv_environment_name;
ALTER TABLE deployment DROP COLUMN tsv_version;
ALTER TABLE deployment DROP COLUMN tsv_app_name;
ALTER TABLE deployment DROP COLUMN tsv_stack_name;
