package awsinfra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/o5-go/messaging/v1/messaging_pb"
	"github.com/pentops/outbox.pg.go/outbox"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type Storage struct {
	db *sqrlx.Wrapper
}

func NewStorage(conn sqrlx.Connection) (*Storage, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	return &Storage{
		db: db,
	}, nil
}

func (s *Storage) PublishEvent(ctx context.Context, msg outbox.OutboxMessage) error {
	return s.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return outbox.Send(ctx, tx, msg)
	})
}

type DuplicateRequestContextError string

func (e DuplicateRequestContextError) Error() string {
	return fmt.Sprintf("duplicate request context: %s", string(e))
}

func (s *Storage) RequestToClientToken(ctx context.Context, req *messaging_pb.RequestMetadata) (string, error) {
	token := uuid.NewString()

	return token, s.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		var foundToken string
		err := tx.SelectRow(ctx, sq.Select("token").From("infra_client_token").Where(
			sq.Eq{
				"request": req.Context,
				"dest":    req.ReplyTo,
			})).Scan(&foundToken)
		if err == nil {
			return DuplicateRequestContextError(foundToken)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		didInsert, err := tx.InsertRow(ctx, sq.Insert("infra_client_token").
			SetMap(map[string]interface{}{
				"token":   token,
				"dest":    req.ReplyTo,
				"request": req.Context,
			}))
		if err != nil {
			return err
		}
		if !didInsert {
			return errors.New("client token not unique")
		}
		return nil
	})
}

func (s *Storage) ClientTokenToRequest(ctx context.Context, token string) (*messaging_pb.RequestMetadata, error) {
	response := &messaging_pb.RequestMetadata{}

	err := s.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return tx.QueryRow(ctx, sq.Select("dest", "request").
			From("infra_client_token").
			Where(sq.Eq{"token": token})).
			Scan(&response.ReplyTo, &response.Context)
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}
