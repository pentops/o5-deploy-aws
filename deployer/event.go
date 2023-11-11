package deployer

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.daemonl.com/sqrlx"
)

func (d *Deployer) RegisterEvent(ctx context.Context, event *deployer_pb.DeploymentEvent) error {

	deployment, err := d.getDeployment(ctx, event.DeploymentId)
	if err != nil {
		return err
	}

	log.WithFields(ctx, map[string]interface{}{
		"deploymentId": event.DeploymentId,
		"event":        event.Event,
		"state":        deployment.Status.ShortString(),
	}).Debug("Beign Deployment Event")

	var nextEvent *deployer_pb.DeploymentEvent
	type sideEffect func() error
	var sideEffects []sideEffect

	switch et := event.Event.Type.(type) {

	case *deployer_pb.DeploymentEventType_Triggered_:
		deployment.Status = deployer_pb.DeploymentStatus_QUEUED

	case *deployer_pb.DeploymentEventType_GotLock_:
		deployment.Status = deployer_pb.DeploymentStatus_LOCKED

		nextEvent, err = d.eventGotLock(ctx, deployment)
		if err != nil {
			return err
		}

	case *deployer_pb.DeploymentEventType_StackWait_:
		deployment.Status = deployer_pb.DeploymentStatus_WAITING
		sideEffects = []sideEffect{func() error {
			return d.runStackWait(ctx, deployment)
		}}

	case *deployer_pb.DeploymentEventType_StackCreate_:
		deployment.Status = deployer_pb.DeploymentStatus_CREATING
		sideEffects = []sideEffect{func() error {
			return d.createNewDeployment(ctx, deployment)
		}}

	case *deployer_pb.DeploymentEventType_StackStatus_:
		switch et.StackStatus.Lifecycle {
		case deployer_pb.StackLifecycle_COMPLETE:
			switch deployment.Status {
			case deployer_pb.DeploymentStatus_WAITING:
				deployment.Status = deployer_pb.DeploymentStatus_AVAILABLE

				nextEvent = newEvent(deployment, &deployer_pb.DeploymentEventType_StackScale_{
					StackScale: &deployer_pb.DeploymentEventType_StackScale{
						DesiredCount: int32(0),
					},
				})

			case deployer_pb.DeploymentStatus_SCALING_DOWN:
				deployment.Status = deployer_pb.DeploymentStatus_SCALED_DOWN

				nextEvent = newEvent(deployment, &deployer_pb.DeploymentEventType_StackTrigger_{
					StackTrigger: &deployer_pb.DeploymentEventType_StackTrigger{
						Phase: "update",
					},
				})

			case deployer_pb.DeploymentStatus_INFRA_MIGRATE:
				deployment.Status = deployer_pb.DeploymentStatus_INFRA_MIGRATED

				nextEvent = newEvent(deployment, &deployer_pb.DeploymentEventType_MigrateData_{
					MigrateData: &deployer_pb.DeploymentEventType_MigrateData{},
				})

			case deployer_pb.DeploymentStatus_SCALING_UP:
				deployment.Status = deployer_pb.DeploymentStatus_SCALED_UP
				nextEvent = newEvent(deployment, &deployer_pb.DeploymentEventType_Done_{
					Done: &deployer_pb.DeploymentEventType_Done{},
				})

			default:
				return fmt.Errorf("unexpected stack status event: %s", deployment.Status)
			}
		case deployer_pb.StackLifecycle_TERMINAL:
			return fmt.Errorf("stack failed: %s", et.StackStatus.FullStatus)

		case deployer_pb.StackLifecycle_PROGRESS:
			// Just log it

		case deployer_pb.StackLifecycle_CREATE_FAILED:
			return fmt.Errorf("stack failed: %s", et.StackStatus.FullStatus)

		case deployer_pb.StackLifecycle_ROLLING_BACK:
			return fmt.Errorf("stack failed: %s", et.StackStatus.FullStatus)

		default:
			return fmt.Errorf("unexpected stack lifecycle: %s", et.StackStatus.Lifecycle)
		}

	case *deployer_pb.DeploymentEventType_MigrateData_:
		deployment.Status = deployer_pb.DeploymentStatus_DB_MIGRATING
		deployment.DataMigrations = make([]*deployer_pb.DatabaseMigrationState, len(deployment.Spec.Databases))
		for i, db := range deployment.Spec.Databases {
			migration := &deployer_pb.DatabaseMigrationState{
				MigrationId:       uuid.NewString(),
				DbName:            db.Database.Name,
				Status:            deployer_pb.DatabaseMigrationStatus_PENDING,
				RotateCredentials: d.RotateSecrets,
			}
			deployment.DataMigrations[i] = migration

			sideEffects = append(sideEffects, func() error {
				return d.runMigration(ctx, deployment, migration)
			})

		}

		if len(deployment.DataMigrations) == 0 {
			nextEvent = newEvent(deployment, &deployer_pb.DeploymentEventType_DataMigrated_{
				DataMigrated: &deployer_pb.DeploymentEventType_DataMigrated{},
			})

		}

	case *deployer_pb.DeploymentEventType_DbMigrateStatus:
		anyPending := false
		anyFailed := false
		for _, migration := range deployment.DataMigrations {
			if migration.MigrationId == et.DbMigrateStatus.MigrationId {
				migration.Status = et.DbMigrateStatus.Status
				switch migration.Status {
				case deployer_pb.DatabaseMigrationStatus_COMPLETED:
					// OK
				case deployer_pb.DatabaseMigrationStatus_PENDING,
					deployer_pb.DatabaseMigrationStatus_RUNNING,
					deployer_pb.DatabaseMigrationStatus_CLEANUP:
					anyPending = true
				case deployer_pb.DatabaseMigrationStatus_FAILED:
					anyFailed = true
				default:
					return fmt.Errorf("unexpected migration status: %s", migration.Status)
				}
			}
		}
		if anyFailed {
			return fmt.Errorf("migration failed, not handled")
		}

		if !anyPending {
			nextEvent = newEvent(deployment, &deployer_pb.DeploymentEventType_DataMigrated_{
				DataMigrated: &deployer_pb.DeploymentEventType_DataMigrated{},
			})
		}

	case *deployer_pb.DeploymentEventType_DataMigrated_:
		deployment.Status = deployer_pb.DeploymentStatus_DB_MIGRATED
		nextEvent = newEvent(deployment, &deployer_pb.DeploymentEventType_StackScale_{
			StackScale: &deployer_pb.DeploymentEventType_StackScale{
				DesiredCount: int32(1),
			},
		})

	case *deployer_pb.DeploymentEventType_StackScale_:
		switch deployment.Status {
		case deployer_pb.DeploymentStatus_AVAILABLE:
			deployment.Status = deployer_pb.DeploymentStatus_SCALING_DOWN

		case deployer_pb.DeploymentStatus_DB_MIGRATED:
			deployment.Status = deployer_pb.DeploymentStatus_SCALING_UP
		}

		sideEffects = []sideEffect{func() error {
			return d.eventStackScale(ctx, deployment, int(et.StackScale.DesiredCount))
		}}

	case *deployer_pb.DeploymentEventType_StackTrigger_:
		switch deployment.Status {
		case deployer_pb.DeploymentStatus_SCALED_DOWN:
			deployment.Status = deployer_pb.DeploymentStatus_INFRA_MIGRATE

			sideEffects = []sideEffect{func() error {
				return d.migrateInfra(ctx, deployment)
			}}

		default:
			return fmt.Errorf("unexpected stack trigger event: %s", deployment.Status)
		}

	case *deployer_pb.DeploymentEventType_Done_:
		deployment.Status = deployer_pb.DeploymentStatus_DONE

	default:
		return fmt.Errorf("unknown event type: %T", et)
	}

	deploymentJSON, err := protojson.Marshal(deployment)
	if err != nil {
		return err
	}

	upsertState := sqrlx.Upsert("deployment").
		Key("id", deployment.DeploymentId).
		Set("state", deploymentJSON)

	eventJSON, err := protojson.Marshal(event)
	if err != nil {
		return err
	}

	insertEvent := sq.Insert("deployment_event").SetMap(map[string]interface{}{
		"deployment_id": deployment.DeploymentId,
		"id":            event.Metadata.EventId,
		"event":         eventJSON,
		"timestamp":     event.Metadata.Timestamp.AsTime(),
	})

	if err := d.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		Retryable: true,
		ReadOnly:  false,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {

		_, err := tx.Insert(ctx, upsertState)
		if err != nil {
			return err
		}

		_, err = tx.Insert(ctx, insertEvent)
		if err != nil {
			return err
		}

		return nil

	}); err != nil {
		return err
	}

	log.WithFields(ctx, map[string]interface{}{
		"deploymentId": event.DeploymentId,
		"event":        event.Event,
	}).Info("Deployment Event")

	for _, se := range sideEffects {
		d.eg.Go(se)
	}

	if nextEvent != nil {
		d.AsyncEvent(ctx, nextEvent)
	}

	return nil
}
