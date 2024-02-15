-- +goose Up

CREATE TABLE environment (
  id uuid PRIMARY KEY,
  state jsonb NOT NULL
);

CREATE TABLE environment_event (
  id uuid PRIMARY KEY,
  environment_id uuid NOT NULL REFERENCES environment(id),
  data jsonb NOT NULL,
  timestamp timestamptz NOT NULL
);

ALTER TABLE stack
  ADD COLUMN github_owner TEXT,
  ADD COLUMN github_repo TEXT,
  ADD COLUMN github_ref TEXT,
  ADD COLUMN environment_id uuid REFERENCES environment(id);

-- +goose Down

DROP TABLE environment_event;
DROP TABLE environment;


ALTER TABLE stack
  DROP COLUMN github_owner,
  DROP COLUMN github_repo,
  DROP COLUMN github_ref,
  DROP COLUMN environment_id;
