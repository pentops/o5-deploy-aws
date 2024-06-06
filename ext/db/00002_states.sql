-- +goose Up

CREATE TABLE stack (
  stack_id uuid,
  environment_id uuid NOT NULL,
  cluster_id uuid NOT NULL,
  state jsonb NOT NULL,
  CONSTRAINT stack_pk PRIMARY KEY (stack_id)
);

ALTER TABLE stack ADD COLUMN tsv_stack_name tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(state, '$.data.stackName'))) STORED;

CREATE INDEX stack_tsv_stack_name_idx ON stack USING GIN (tsv_stack_name);

ALTER TABLE stack ADD COLUMN tsv_app_name tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(state, '$.data.applicationName'))) STORED;

CREATE INDEX stack_tsv_app_name_idx ON stack USING GIN (tsv_app_name);

ALTER TABLE stack ADD COLUMN tsv_environment_name tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(state, '$.data.environmentName'))) STORED;

CREATE INDEX stack_tsv_environment_name_idx ON stack USING GIN (tsv_environment_name);

CREATE TABLE stack_event (
  id uuid,
  stack_id uuid NOT NULL,
  environment_id uuid NOT NULL,
  cluster_id uuid NOT NULL,
  timestamp timestamptz NOT NULL,
  sequence int NOT NULL,
  data jsonb NOT NULL,
  state jsonb NOT NULL,
  CONSTRAINT stack_event_pk PRIMARY KEY (id),
  CONSTRAINT stack_event_fk_state FOREIGN KEY (stack_id) REFERENCES stack(stack_id)
);

CREATE TABLE deployment (
  deployment_id uuid,
  stack_id uuid NOT NULL,
  environment_id uuid NOT NULL,
  cluster_id uuid NOT NULL,
  state jsonb NOT NULL,
  CONSTRAINT deployment_pk PRIMARY KEY (deployment_id)
);

ALTER TABLE deployment ADD COLUMN tsv_app_name tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(state, '$.data.spec.appName'))) STORED;

CREATE INDEX deployment_tsv_app_name_idx ON deployment USING GIN (tsv_app_name);

ALTER TABLE deployment ADD COLUMN tsv_version tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(state, '$.data.spec.version'))) STORED;

CREATE INDEX deployment_tsv_version_idx ON deployment USING GIN (tsv_version);

ALTER TABLE deployment ADD COLUMN tsv_environment_name tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(state, '$.data.spec.environmentName'))) STORED;

CREATE INDEX deployment_tsv_environment_name_idx ON deployment USING GIN (tsv_environment_name);

ALTER TABLE deployment ADD COLUMN tsv_ecs_cluster tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(state, '$.data.spec.ecsCluster'))) STORED;

CREATE INDEX deployment_tsv_ecs_cluster_idx ON deployment USING GIN (tsv_ecs_cluster);

ALTER TABLE deployment ADD COLUMN tsv_stack_name tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(state, '$.data.stackName'))) STORED;

CREATE INDEX deployment_tsv_stack_name_idx ON deployment USING GIN (tsv_stack_name);

CREATE TABLE deployment_event (
  id uuid,
  deployment_id uuid NOT NULL,
  stack_id uuid NOT NULL,
  environment_id uuid NOT NULL,
  cluster_id uuid NOT NULL,
  timestamp timestamptz NOT NULL,
  sequence int NOT NULL,
  data jsonb NOT NULL,
  state jsonb NOT NULL,
  CONSTRAINT deployment_event_pk PRIMARY KEY (id),
  CONSTRAINT deployment_event_fk_state FOREIGN KEY (deployment_id) REFERENCES deployment(deployment_id)
);

ALTER TABLE deployment_event ADD COLUMN tsv_app_name tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(data, '$.event.created.spec.source'))) STORED;

CREATE INDEX deployment_event_tsv_app_name_idx ON deployment_event USING GIN (tsv_app_name);

ALTER TABLE deployment_event ADD COLUMN tsv_version tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(data, '$.event.created.spec.source'))) STORED;

CREATE INDEX deployment_event_tsv_version_idx ON deployment_event USING GIN (tsv_version);

ALTER TABLE deployment_event ADD COLUMN tsv_environment_name tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(data, '$.event.created.spec.source'))) STORED;

CREATE INDEX deployment_event_tsv_environment_name_idx ON deployment_event USING GIN (tsv_environment_name);

ALTER TABLE deployment_event ADD COLUMN tsv_ecs_cluster tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(data, '$.event.created.spec.source'))) STORED;

CREATE INDEX deployment_event_tsv_ecs_cluster_idx ON deployment_event USING GIN (tsv_ecs_cluster);

CREATE TABLE environment (
  environment_id uuid,
  cluster_id uuid NOT NULL,
  state jsonb NOT NULL,
  CONSTRAINT environment_pk PRIMARY KEY (environment_id)
);

ALTER TABLE environment ADD COLUMN tsv_full_name tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(state, '$.data.config.fullName'))) STORED;

CREATE INDEX environment_tsv_full_name_idx ON environment USING GIN (tsv_full_name);

CREATE TABLE environment_event (
  id uuid,
  environment_id uuid NOT NULL,
  cluster_id uuid NOT NULL,
  timestamp timestamptz NOT NULL,
  sequence int NOT NULL,
  data jsonb NOT NULL,
  state jsonb NOT NULL,
  CONSTRAINT environment_event_pk PRIMARY KEY (id),
  CONSTRAINT environment_event_fk_state FOREIGN KEY (environment_id) REFERENCES environment(environment_id)
);

ALTER TABLE environment_event ADD COLUMN tsv_full_name tsvector GENERATED ALWAYS
  AS (to_tsvector('english', jsonb_path_query_array(data, '$.event.configured.config.aws'))) STORED;

CREATE INDEX environment_event_tsv_full_name_idx ON environment_event USING GIN (tsv_full_name);

CREATE TABLE cluster (
  cluster_id uuid,
  state jsonb NOT NULL,
  CONSTRAINT cluster_pk PRIMARY KEY (cluster_id)
);

CREATE TABLE cluster_event (
  id uuid,
  cluster_id uuid NOT NULL,
  timestamp timestamptz NOT NULL,
  sequence int NOT NULL,
  data jsonb NOT NULL,
  state jsonb NOT NULL,
  CONSTRAINT cluster_event_pk PRIMARY KEY (id),
  CONSTRAINT cluster_event_fk_state FOREIGN KEY (cluster_id) REFERENCES cluster(cluster_id)
);

-- +goose Down

DROP TABLE cluster_event;
DROP TABLE cluster;

DROP INDEX environment_event_tsv_full_name_idx;
ALTER TABLE environment_event DROP COLUMN tsv_full_name;

DROP TABLE environment_event;

DROP INDEX environment_tsv_full_name_idx;
ALTER TABLE environment DROP COLUMN tsv_full_name;

DROP TABLE environment;

DROP INDEX deployment_event_tsv_ecs_cluster_idx;
ALTER TABLE deployment_event DROP COLUMN tsv_ecs_cluster;

DROP INDEX deployment_event_tsv_environment_name_idx;
ALTER TABLE deployment_event DROP COLUMN tsv_environment_name;

DROP INDEX deployment_event_tsv_version_idx;
ALTER TABLE deployment_event DROP COLUMN tsv_version;

DROP INDEX deployment_event_tsv_app_name_idx;
ALTER TABLE deployment_event DROP COLUMN tsv_app_name;

DROP TABLE deployment_event;

DROP INDEX deployment_tsv_stack_name_idx;
ALTER TABLE deployment DROP COLUMN tsv_stack_name;

DROP INDEX deployment_tsv_ecs_cluster_idx;
ALTER TABLE deployment DROP COLUMN tsv_ecs_cluster;

DROP INDEX deployment_tsv_environment_name_idx;
ALTER TABLE deployment DROP COLUMN tsv_environment_name;

DROP INDEX deployment_tsv_version_idx;
ALTER TABLE deployment DROP COLUMN tsv_version;

DROP INDEX deployment_tsv_app_name_idx;
ALTER TABLE deployment DROP COLUMN tsv_app_name;

DROP TABLE deployment;
DROP TABLE stack_event;

DROP INDEX stack_tsv_environment_name_idx;
ALTER TABLE stack DROP COLUMN tsv_environment_name;

DROP INDEX stack_tsv_app_name_idx;
ALTER TABLE stack DROP COLUMN tsv_app_name;

DROP INDEX stack_tsv_stack_name_idx;
ALTER TABLE stack DROP COLUMN tsv_stack_name;

DROP TABLE stack;

