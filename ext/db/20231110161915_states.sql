-- +goose Up

CREATE TABLE cluster (
	id uuid PRIMARY KEY,
	state jsonb NOT NULL
);

CREATE TABLE cluster_event (
	id uuid PRIMARY KEY,
	cluster_id uuid NOT NULL,
  sequence int NOT NULL,
	data jsonb NOT NULL,
  cause jsonb NOT NULL,
	timestamp timestamptz NOT NULL
);

CREATE TABLE environment (
  id uuid PRIMARY KEY,
  cluster_id uuid NOT NULL REFERENCES cluster(id),
  state jsonb NOT NULL
);

CREATE TABLE environment_event (
  id uuid PRIMARY KEY,
  environment_id uuid NOT NULL REFERENCES environment(id),
  cluster_id uuid NOT NULL REFERENCES cluster(id),

  sequence int NOT NULL,
  data jsonb NOT NULL,
  cause jsonb NOT NULL,
  timestamp timestamptz NOT NULL
);

CREATE TABLE stack (
	id uuid PRIMARY KEY,
  environment_id uuid REFERENCES environment(id),
  cluster_id uuid REFERENCES cluster(id),

  app_name TEXT NOT NULL,
  env_name TEXT NOT NULL,

	state jsonb NOT NULL,
  github_owner TEXT,
  github_repo TEXT,
  github_ref TEXT,
  CONSTRAINT stack_unique UNIQUE (env_name, app_name)
);


CREATE TABLE stack_event (
	id uuid PRIMARY KEY,
	stack_id uuid NOT NULL,
  environment_id uuid NOT NULL,
  cluster_id uuid NOT NULL,

  sequence int NOT NULL,
	data jsonb NOT NULL,
  cause jsonb NOT NULL,
	timestamp timestamptz NOT NULL
);

CREATE TABLE deployment (
	id uuid PRIMARY KEY,
	stack_id uuid NOT NULL REFERENCES stack(id) DEFERRABLE INITIALLY DEFERRED,
  environment_id uuid NOT NULL REFERENCES environment(id) DEFERRABLE INITIALLY DEFERRED,
  cluster_id uuid NOT NULL REFERENCES cluster(id) DEFERRABLE INITIALLY DEFERRED,
	state jsonb NOT NULL
);


CREATE TABLE deployment_event (
	id uuid PRIMARY KEY,
	deployment_id uuid NOT NULL,
	stack_id uuid NOT NULL REFERENCES stack(id) DEFERRABLE INITIALLY DEFERRED,
  environment_id uuid NOT NULL REFERENCES environment(id) DEFERRABLE INITIALLY DEFERRED,
  cluster_id uuid NOT NULL REFERENCES cluster(id) DEFERRABLE INITIALLY DEFERRED,

  sequence int NOT NULL,
	data jsonb NOT NULL,
  cause jsonb NOT NULL,
	timestamp timestamptz NOT NULL
);
-- +goose Down


DROP TABLE deployment_event;
DROP TABLE deployment;

DROP TABLE stack_event;
DROP TABLE stack;

DROP TABLE environment_event;
DROP TABLE environment;

DROP TABLE cluster_event;
DROP TABLE cluster;






