package tokenstore

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"github.com/pentops/o5-messaging/o5msg"
	"github.com/pentops/o5-messaging/outbox"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type DBLite interface {
	PublishEvent(context.Context, o5msg.Message) error
	RequestToClientToken(context.Context, *messaging_j5pb.RequestMetadata) (string, error)
	ClientTokenToRequest(context.Context, string) (*messaging_j5pb.RequestMetadata, error)
}

var ErrRequestTokenNotFound = errors.New("request token not found")

type Storage struct {
	db sqrlx.Transactor
}

func NewStorage(db sqrlx.Transactor) (*Storage, error) {
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

const tokenPrefix = "o5-deploy-"

func (s *Storage) RequestToClientToken(ctx context.Context, req *messaging_j5pb.RequestMetadata) (string, error) {
	token := uuid.NewString()
	tokenStr := tokenPrefix + token
	return tokenStr, s.db.Transact(ctx, &sqrlx.TxOptions{
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
			SetMap(map[string]any{
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
	if !strings.HasPrefix(token, tokenPrefix) {
		return nil, ErrRequestTokenNotFound
	}
	token = strings.TrimPrefix(token, tokenPrefix)

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
