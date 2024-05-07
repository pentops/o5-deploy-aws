package github

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/o5-deploy-aws/gen/o5/github/v1/github_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type RefMatcher interface {
	PushTargets(context.Context, *github_pb.PushMessage) ([]string, error)
}

type RefStore struct {
	db *sqrlx.Wrapper
}

func NewRefStore(conn sqrlx.Connection) (*RefStore, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	return &RefStore{
		db: db,
	}, nil
}

func (rs *RefStore) PushTargets(ctx context.Context, push *github_pb.PushMessage) ([]string, error) {
	qq := sq.Select("environment_id").
		From("stack").Where(sq.Eq{
		"github_owner": push.Owner,
		"github_repo":  push.Repo,
		"github_ref":   push.Ref,
	})

	environments := []string{}

	err := rs.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {

		rows, err := tx.Select(ctx, qq)
		if err != nil {
			return err
		}

		defer rows.Close()

		for rows.Next() {
			environment := ""
			if err := rows.Scan(&environment); err != nil {
				return err
			}
			environments = append(environments, environment)
		}

		if err := rows.Err(); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("selecting push targets: %w", err)
	}

	return environments, nil
}
