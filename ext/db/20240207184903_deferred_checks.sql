-- +goose Up


ALTER TABLE deployment ALTER CONSTRAINT deployment_stack_id_fkey DEFERRABLE INITIALLY DEFERRED;

-- +goose Down

ALTER TABLE deployment ALTER CONSTRAINT deployment_stack_id_fkey NOT DEFERRABLE;
