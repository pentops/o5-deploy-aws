package deployer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/encoding/protojson"
)

var DeploymentNotFoundError = fmt.Errorf("deployment not found")
var StackNotFoundError = fmt.Errorf("stack not found")

var namespaceStackID = uuid.MustParse("C27983FD-BC4B-493F-A056-CC8C869A1999")

func StackID(envName, appName string) string {
	return uuid.NewMD5(namespaceStackID, []byte(fmt.Sprintf("%s-%s", envName, appName))).String()
}

func getDeployment(ctx context.Context, tx sqrlx.Transaction, id string) (*deployer_pb.DeploymentState, error) {
	var deploymentJSON []byte
	err := tx.SelectRow(ctx, sq.Select("state").From("deployment").Where("id = ?", id)).Scan(&deploymentJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, DeploymentNotFoundError
	} else if err != nil {
		return nil, err
	}
	var deployment deployer_pb.DeploymentState
	if err := protojson.Unmarshal(deploymentJSON, &deployment); err != nil {
		return nil, fmt.Errorf("unmarshal deployment: %w", err)
	}
	return &deployment, nil
}
