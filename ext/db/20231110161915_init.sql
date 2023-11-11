-- +goose Up

CREATE TABLE deployment (
	id uuid NOT NULL,
	state jsonb NOT NULL,
);

CREATE TABLE deployment_event (
	id uuid NOT NULL,
	deployment_id uuid NOT NULL,
	event jsonb NOT NULL,
	timestamptz timestamp NOT NULL
);

-- +goose Down

DROP TABLE deployment;
