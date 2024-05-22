-- +goose Up
CREATE TABLE infra_client_token (
  token uuid PRIMARY KEY,
  dest text NOT NULL,
  request bytea NOT NULL
);

-- +goose Down

DROP TABLE infra_client_token;
