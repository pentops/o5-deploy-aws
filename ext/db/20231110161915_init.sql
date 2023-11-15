-- +goose Up

CREATE TABLE deployment (
	id uuid PRIMARY KEY,
	state jsonb NOT NULL
);

CREATE TABLE deployment_event (
	id uuid PRIMARY KEY,
	deployment_id uuid NOT NULL,
	event jsonb NOT NULL,
	timestamp timestamptz NOT NULL
);

-- +goose Down

DROP TABLE deployment_event;
DROP TABLE deployment;

