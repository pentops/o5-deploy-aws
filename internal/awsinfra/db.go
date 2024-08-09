package awsinfra

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"github.com/pentops/o5-messaging/o5msg"
	"github.com/pentops/o5-messaging/outbox"
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

func (s *Storage) PublishEvent(ctx context.Context, msg o5msg.Message) error {
	return s.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return outbox.Send(ctx, tx, msg)
	})
}

func (s *Storage) RequestToClientToken(ctx context.Context, req *messaging_j5pb.RequestMetadata) (string, error) {
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
			token = foundToken
			return nil
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

func (s *Storage) ClientTokenToRequest(ctx context.Context, token string) (*messaging_j5pb.RequestMetadata, error) {
	response := &messaging_j5pb.RequestMetadata{}

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
