package deployer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.daemonl.com/sqrlx"
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

func getStack(ctx context.Context, tx sqrlx.Transaction, stackID string) (*deployer_pb.StackState, error) {
	var stackJSON []byte
	err := tx.SelectRow(ctx, sq.Select("state").From("stack").Where("id = ?", stackID)).Scan(&stackJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, StackNotFoundError
	}
	if err != nil {
		return nil, err
	}
	var stack deployer_pb.StackState
	if err := protojson.Unmarshal(stackJSON, &stack); err != nil {
		return nil, err
	}
	return &stack, nil
}

func storeStackEvent(ctx context.Context, tx sqrlx.Transaction, stack *deployer_pb.StackState, event *deployer_pb.StackEvent) error {
	stackJSON, err := protojson.Marshal(stack)
	if err != nil {
		return err
	}

	upsertStack := sqrlx.Upsert("stack").
		Key("id", stack.StackId).SetMap(map[string]interface{}{
		"env_name": stack.EnvironmentName,
		"app_name": stack.ApplicationName,
		"state":    stackJSON,
	})

	_, err = tx.Insert(ctx, upsertStack)
	if err != nil {
		return err
	}

	eventJSON, err := protojson.Marshal(event)
	if err != nil {
		return err
	}

	insertEvent := sq.Insert("stack_event").SetMap(map[string]interface{}{
		"stack_id":  stack.StackId,
		"id":        event.Metadata.EventId,
		"event":     eventJSON,
		"timestamp": event.Metadata.Timestamp.AsTime(),
	})

	_, err = tx.Insert(ctx, insertEvent)
	if err != nil {
		return err
	}

	return nil
}

func storeDeploymentEvent(ctx context.Context, tx sqrlx.Transaction, state *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error {
	deploymentJSON, err := protojson.Marshal(state)
	if err != nil {
		return err
	}

	upsertState := sqrlx.Upsert("deployment").
		Key("id", state.DeploymentId).
		Set("state", deploymentJSON).
		Set("stack_id", state.StackId)

	eventJSON, err := protojson.Marshal(event)
	if err != nil {
		return err
	}

	_, err = tx.Insert(ctx, upsertState)
	if err != nil {
		return err
	}

	insertEvent := sq.Insert("deployment_event").SetMap(map[string]interface{}{
		"deployment_id": state.DeploymentId,
		"id":            event.Metadata.EventId,
		"event":         eventJSON,
		"timestamp":     event.Metadata.Timestamp.AsTime(),
	})

	_, err = tx.Insert(ctx, insertEvent)
	if err != nil {
		return err
	}

	return nil

}
