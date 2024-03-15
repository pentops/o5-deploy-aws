-- +goose Up

CREATE TABLE stack (
	id uuid PRIMARY KEY,
	env_name text NOT NULL,
	app_name text NOT NULL,
	state jsonb NOT NULL
);

CREATE TABLE stack_event (
	id uuid PRIMARY KEY,
	stack_id uuid NOT NULL,
	data jsonb NOT NULL,
	timestamp timestamptz NOT NULL
);

CREATE TABLE deployment (
	id uuid PRIMARY KEY,
	stack_id uuid NOT NULL REFERENCES stack(id),
	state jsonb NOT NULL
);

ALTER TABLE stack ADD COLUMN current_deployment UUID REFERENCES deployment(id);

CREATE TABLE deployment_event (
	id uuid PRIMARY KEY,
	deployment_id uuid NOT NULL,
	data jsonb NOT NULL,
	timestamp timestamptz NOT NULL
);

-- +goose Down

DROP TABLE deployment_event;
ALTER TABLE stack DROP COLUMN current_deployment;

DROP TABLE deployment;

DROP TABLE stack_event;
DROP TABLE stack;
