package deployer

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/protostate/psm"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func chainDeploymentEvent(tb deployer_pb.DeploymentPSMTransitionBaton, event deployer_pb.IsDeploymentEventTypeWrappedType) *deployer_pb.DeploymentEvent {
	md := tb.FullCause().Metadata
	de := &deployer_pb.DeploymentEvent{
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
			Actor:     md.Actor,
		},
		DeploymentId: tb.FullCause().DeploymentId,
		Event:        &deployer_pb.DeploymentEventType{},
	}
	de.Event.Set(event)
	return de
}

func DeploymentTableSpec() deployer_pb.DeploymentPSMTableSpec {
	tableSpec := deployer_pb.DefaultDeploymentPSMTableSpec
	tableSpec.EventDataColumn = "event"
	tableSpec.StateDataColumn = "state"
	tableSpec.ExtraStateColumns = func(s *deployer_pb.DeploymentState) (map[string]interface{}, error) {
		return map[string]interface{}{
			"stack_id": s.StackId,
		}, nil
	}

	tableSpec.EventColumns = func(e *deployer_pb.DeploymentEvent) (map[string]interface{}, error) {
		return map[string]interface{}{
			"id":            e.Metadata.EventId,
			"deployment_id": e.DeploymentId,
			"event":         e,
			"timestamp":     e.Metadata.Timestamp,
		}, nil
	}

	return tableSpec
}

type deployerConversions struct {
	deployer_pb.DeploymentPSMConverter
}

func (c deployerConversions) EventLabel(event deployer_pb.DeploymentPSMEvent) string {
	typeKey := string(event.PSMEventKey())

	// TODO: This by generation and annotation
	if stackStatus, ok := event.(*deployer_pb.DeploymentEventType_StackStatus); ok {
		typeKey = fmt.Sprintf("%s.%s", typeKey, stackStatus.Lifecycle.ShortString())
	}
	return typeKey
}

func NewDeploymentEventer(db psm.Transactor) (*deployer_pb.DeploymentPSM, error) {

	sm, err := deployer_pb.NewDeploymentPSM(db,
		psm.WithTableSpec(DeploymentTableSpec()),
		psm.WithEventTypeConverter(deployerConversions{}),
	)
	if err != nil {
		return nil, err
	}

	// [*] --> QUEUED : Created
	sm.From(deployer_pb.DeploymentStatus_UNSPECIFIED).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_Created,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_QUEUED
			deployment.Spec = event.Spec
			deployment.StackName = fmt.Sprintf("%s-%s", event.Spec.EnvironmentName, event.Spec.AppName)
			deployment.StackId = StackID(event.Spec.EnvironmentName, event.Spec.AppName)

			// No follow on, the stack state will trigger

			return nil
		}))

		// QUEUED --> TRIGGERED : Trigger
	sm.From(deployer_pb.DeploymentStatus_QUEUED).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_Triggered,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_TRIGGERED

			tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_StackWait{}))

			return nil
		}))

		// TRIGGERED --> WAITING : StackWait
	sm.From(deployer_pb.DeploymentStatus_TRIGGERED).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackWait,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_WAITING
			deployment.WaitingOnRemotePhase = proto.String("wait")

			tb.SideEffect(&deployer_tpb.StabalizeStackMessage{
				StackId: &deployer_tpb.StackID{
					StackName:       deployment.StackName,
					DeploymentId:    deployment.DeploymentId,
					DeploymentPhase: "wait",
				},
				CancelUpdate: deployment.Spec.CancelUpdates,
			})

			return nil
		}))

	// AVAILABLE --> CREATING : StackCreate
	sm.From(deployer_pb.DeploymentStatus_AVAILABLE).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackCreate,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_CREATING

			topicNames := make([]string, len(deployment.Spec.SnsTopics))
			for i, topic := range deployment.Spec.SnsTopics {
				topicNames[i] = fmt.Sprintf("%s-%s", deployment.Spec.EnvironmentName, topic.Name)
			}

			deployment.WaitingOnRemotePhase = proto.String("create")

			// Create, scale 0
			tb.SideEffect(&deployer_tpb.CreateNewStackMessage{
				StackId: &deployer_tpb.StackID{
					StackName:       deployment.StackName,
					DeploymentId:    deployment.DeploymentId,
					DeploymentPhase: "create",
				},
				Parameters:   deployment.Spec.Parameters,
				TemplateUrl:  deployment.Spec.TemplateUrl,
				DesiredCount: 0,
				ExtraResources: &deployer_tpb.ExtraResources{
					SnsTopics: topicNames,
				},
			})
			return nil
		},
		))

	stackLifecycleIs := func(wantStatuses ...deployer_pb.StackLifecycle) func(deployer_pb.DeploymentPSMEvent) bool {
		return func(event deployer_pb.DeploymentPSMEvent) bool {
			stackStatus, ok := event.(*deployer_pb.DeploymentEventType_StackStatus)
			if !ok {
				return false
			}

			for _, wantStatus := range wantStatuses {
				if stackStatus.Lifecycle == wantStatus {
					return true
				}
			}

			return false
		}
	}
	// WAITING --> AVAILABLE : StackStatus.Complete
	// WAITING --> AVAILABLE : StackStatus.Missing
	// WAITING --> AVAILABLE : StackStatus.Terminal + UPDATE_ROLLBACK_COMPLETE
	sm.From(deployer_pb.DeploymentStatus_WAITING).
		Where(stackLifecycleIs(
			deployer_pb.StackLifecycle_COMPLETE,
			deployer_pb.StackLifecycle_MISSING,
			deployer_pb.StackLifecycle_ROLLED_BACK,
		)).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_AVAILABLE
			deployment.StackOutput = event.StackOutput
			deployment.LastStackLifecycle = event.Lifecycle
			deployment.WaitingOnRemotePhase = nil

			if deployment.Spec.QuickMode {
				tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_StackUpsert{}))
			} else if event.Lifecycle == deployer_pb.StackLifecycle_MISSING {
				tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_StackCreate{}))
			} else {
				tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_StackScale{
					DesiredCount: int32(0),
				}))
			}

			return nil
		},
		))

	// AVAILABLE --> UPSERTING : StackUpsert
	sm.From(deployer_pb.DeploymentStatus_AVAILABLE).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackUpsert,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_UPSERTING

			topicNames := make([]string, len(deployment.Spec.SnsTopics))
			for i, topic := range deployment.Spec.SnsTopics {
				topicNames[i] = fmt.Sprintf("%s-%s", deployment.Spec.EnvironmentName, topic.Name)
			}

			if deployment.LastStackLifecycle == deployer_pb.StackLifecycle_MISSING {
				// Create a new stack
				deployment.WaitingOnRemotePhase = proto.String("create")
				tb.SideEffect(&deployer_tpb.CreateNewStackMessage{
					StackId: &deployer_tpb.StackID{
						StackName:       deployment.StackName,
						DeploymentId:    deployment.DeploymentId,
						DeploymentPhase: "create",
					},
					Parameters:   deployment.Spec.Parameters,
					TemplateUrl:  deployment.Spec.TemplateUrl,
					DesiredCount: 1, // TODO: Pull from spec
					ExtraResources: &deployer_tpb.ExtraResources{
						SnsTopics: topicNames,
					},
				})
			} else {
				deployment.WaitingOnRemotePhase = proto.String("infra-migrate")
				tb.SideEffect(&deployer_tpb.UpdateStackMessage{
					StackId: &deployer_tpb.StackID{
						StackName:       deployment.StackName,
						DeploymentId:    deployment.DeploymentId,
						DeploymentPhase: "infra-migrate",
					},
					Parameters:   deployment.Spec.Parameters,
					TemplateUrl:  deployment.Spec.TemplateUrl,
					DesiredCount: 1, // TODO: Pull from spec
					ExtraResources: &deployer_tpb.ExtraResources{
						SnsTopics: topicNames,
					},
				})
			}

			return nil
		},
		))

	// UPSERTING --> UPSERTED : StackStatus.Complete
	sm.From(deployer_pb.DeploymentStatus_UPSERTING).
		Where(stackLifecycleIs(deployer_pb.StackLifecycle_COMPLETE)).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_UPSERTED
			deployment.StackOutput = event.StackOutput
			deployment.WaitingOnRemotePhase = nil

			tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_Done{}))

			return nil
		},
		))

	// SCALING_DOWN --> SCALED_DOWN : StackStatus.Complete
	sm.From(deployer_pb.DeploymentStatus_SCALING_DOWN).
		Where(stackLifecycleIs(deployer_pb.StackLifecycle_COMPLETE)).
		Do(deployer_pb.DeploymentPSMFunc(
			func(
				ctx context.Context,
				tb deployer_pb.DeploymentPSMTransitionBaton,
				deployment *deployer_pb.DeploymentState,
				event *deployer_pb.DeploymentEventType_StackStatus,
			) error {
				deployment.Status = deployer_pb.DeploymentStatus_SCALED_DOWN
				deployment.StackOutput = event.StackOutput
				deployment.WaitingOnRemotePhase = nil

				tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_StackTrigger{
					Phase: "update",
				}))
				return nil
			},
		))

	// INFRA_MIGRATE --> INFRA_MIGRATED : StackStatus.Complete
	// CREATING --> INFRA_MIGRATED : StackStatus.Complete
	sm.From(
		deployer_pb.DeploymentStatus_INFRA_MIGRATE,
		deployer_pb.DeploymentStatus_CREATING,
	).
		Where(stackLifecycleIs(deployer_pb.StackLifecycle_COMPLETE)).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_INFRA_MIGRATED
			deployment.StackOutput = event.StackOutput
			deployment.WaitingOnRemotePhase = nil

			tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_MigrateData{}))

			return nil
		}))

	// ScalingUp --> ScaledUp : StackStatus.Complete
	sm.From(deployer_pb.DeploymentStatus_SCALING_UP).
		Where(stackLifecycleIs(deployer_pb.StackLifecycle_COMPLETE)).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_SCALED_UP
			deployment.StackOutput = event.StackOutput
			deployment.WaitingOnRemotePhase = nil

			tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_Done{}))
			return nil
		}))

	// Any Waiting --> Any Waiting : StackStatus.Progress
	sm.From(
		deployer_pb.DeploymentStatus_WAITING,
		deployer_pb.DeploymentStatus_CREATING,
		deployer_pb.DeploymentStatus_UPSERTING,
		deployer_pb.DeploymentStatus_SCALING_UP,
		deployer_pb.DeploymentStatus_SCALING_DOWN,
		deployer_pb.DeploymentStatus_INFRA_MIGRATE,
	).
		Where(stackLifecycleIs(deployer_pb.StackLifecycle_PROGRESS)).
		Do(deployer_pb.DeploymentPSMFunc(
			func(
				ctx context.Context,
				tb deployer_pb.DeploymentPSMTransitionBaton,
				deployment *deployer_pb.DeploymentState,
				event *deployer_pb.DeploymentEventType_StackStatus,
			) error {
				// nothing, just log the progress
				return nil
			},
		))

	// WAITING --> WAITING : StackStatus.Rollback In Progress.
	// TODO: Needs an actual key
	sm.From(deployer_pb.DeploymentStatus_WAITING).
		Where(stackLifecycleIs(deployer_pb.StackLifecycle_ROLLING_BACK)).
		Do(deployer_pb.DeploymentPSMFunc(func(ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			// nothing, just log the progress
			return nil
		}))

	// Creating --> Failed : StackStatus.Failed
	// ScalingUp --> Failed : StackStatus.Failed
	// ScalingDown --> Failed : StackStatus.Failed
	// InfraMigrate --> Failed : StackStatus.Failed
	sm.From(
		deployer_pb.DeploymentStatus_WAITING,
		deployer_pb.DeploymentStatus_UPSERTING,
		deployer_pb.DeploymentStatus_SCALING_UP,
		deployer_pb.DeploymentStatus_SCALING_DOWN,
		deployer_pb.DeploymentStatus_INFRA_MIGRATE,
	).
		Where(stackLifecycleIs(
			deployer_pb.StackLifecycle_TERMINAL,
			deployer_pb.StackLifecycle_CREATE_FAILED,
			deployer_pb.StackLifecycle_ROLLING_BACK,
			deployer_pb.StackLifecycle_ROLLED_BACK,
		)).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_FAILED
			deployment.StackOutput = event.StackOutput
			deployment.WaitingOnRemotePhase = nil

			return fmt.Errorf("stack failed: %s", event.FullStatus)
		}))

	// INFRA_MIGRATED --> DB_MIGRATING : MigrateData
	sm.From(deployer_pb.DeploymentStatus_INFRA_MIGRATED).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_MigrateData,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_DB_MIGRATING
			deployment.DataMigrations = make([]*deployer_pb.DatabaseMigrationState, len(deployment.Spec.Databases))
			for i, db := range deployment.Spec.Databases {
				migration := &deployer_pb.DatabaseMigrationState{
					MigrationId:       uuid.NewString(),
					DbName:            db.Database.Name,
					Status:            deployer_pb.DatabaseMigrationStatus_PENDING,
					RotateCredentials: deployment.Spec.RotateCredentials,
				}
				deployment.DataMigrations[i] = migration

				ctx = log.WithFields(ctx, map[string]interface{}{
					"database":    db.Database.Name,
					"serverGroup": db.Database.GetPostgres().ServerGroup,
				})
				log.Debug(ctx, "Upsert Database")
				var migrationTaskARN string
				var secretARN string
				for _, output := range deployment.StackOutput {
					if *db.MigrationTaskOutputName == output.Name {
						migrationTaskARN = output.Value
					}
					if *db.SecretOutputName == output.Name {
						secretARN = output.Value
					}
				}

				if db.MigrationTaskOutputName != nil {
					if migrationTaskARN == "" {
						return fmt.Errorf("no migration task found for database %s", db.Database.Name)
					}
					if secretARN == "" {
						return fmt.Errorf("no migration secret found for database %s", db.Database.Name)
					}
				}

				secretName := db.RdsHost.SecretName
				if secretName == "" {
					return fmt.Errorf("no host found for server group %q", db.Database.GetPostgres().ServerGroup)
				}

				migrationMsg := &deployer_tpb.RunDatabaseMigrationMessage{
					MigrationId:       migration.MigrationId,
					DeploymentId:      deployment.DeploymentId,
					MigrationTaskArn:  migrationTaskARN,
					SecretArn:         secretARN,
					Database:          db,
					RotateCredentials: migration.RotateCredentials,
					EcsClusterName:    deployment.Spec.EcsCluster,
					RootSecretName:    secretName,
				}

				tb.SideEffect(migrationMsg)
			}

			if len(deployment.DataMigrations) == 0 {
				tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_DataMigrated{}))
			}
			return nil
		}))

	// Transition contains business logic.
	// When any migration is still pending, it will not transition
	// When all migrations are complete, if any failed, will transition to
	// failed
	// When all migrations are complete, and none failed, will transition to
	// DataMigrated
	sm.From(deployer_pb.DeploymentStatus_DB_MIGRATING).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_DBMigrateStatus,
		) error {
			anyPending := false
			anyFailed := false
			for _, migration := range deployment.DataMigrations {
				if migration.MigrationId == event.MigrationId {
					migration.Status = event.Status
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

			if anyPending {
				return nil
			}

			if anyFailed {
				tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_Error{
					Error: "database migration failed",
				}))
				return nil
			}

			tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_DataMigrated{}))

			return nil
		},
		))

	// DB_MIGRATING --> DB_MIGRATED : DataMigrated
	sm.From(deployer_pb.DeploymentStatus_DB_MIGRATING).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_DataMigrated,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_DB_MIGRATED
			tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_StackScale{
				DesiredCount: int32(1),
			}))
			return nil
		},
		))

	// DB_MIGRATED --> SCALING_UP : StackScale
	sm.From(deployer_pb.DeploymentStatus_DB_MIGRATED).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackScale,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_SCALING_UP
			deployment.WaitingOnRemotePhase = proto.String("scale-up")

			tb.SideEffect(&deployer_tpb.ScaleStackMessage{
				StackId: &deployer_tpb.StackID{
					StackName:       deployment.StackName,
					DeploymentId:    deployment.DeploymentId,
					DeploymentPhase: "scale-up",
				},
				DesiredCount: event.DesiredCount,
			})
			return nil
		},
		))

	// AVAILABLE --> SCALING_DOWN : StackScale
	sm.From(deployer_pb.DeploymentStatus_AVAILABLE).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackScale,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_SCALING_DOWN
			deployment.WaitingOnRemotePhase = proto.String("scale-down")

			tb.SideEffect(&deployer_tpb.ScaleStackMessage{
				DesiredCount: event.DesiredCount,
				StackId: &deployer_tpb.StackID{
					StackName:       deployment.StackName,
					DeploymentId:    deployment.DeploymentId,
					DeploymentPhase: "scale-down",
				},
			})
			return nil
		},
		))

	// SCALED_DOWN --> INFRA_MIGRATE : StackTrigger
	sm.From(deployer_pb.DeploymentStatus_SCALED_DOWN).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackTrigger,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_INFRA_MIGRATE
			deployment.WaitingOnRemotePhase = proto.String("infra-migrate")

			topicNames := make([]string, len(deployment.Spec.SnsTopics))
			for i, topic := range deployment.Spec.SnsTopics {
				topicNames[i] = fmt.Sprintf("%s-%s", deployment.Spec.EnvironmentName, topic.Name)
			}

			msg := &deployer_tpb.UpdateStackMessage{
				StackId: &deployer_tpb.StackID{
					StackName:       deployment.StackName,
					DeploymentId:    deployment.DeploymentId,
					DeploymentPhase: "infra-migrate",
				},
				Parameters:   deployment.Spec.Parameters,
				TemplateUrl:  deployment.Spec.TemplateUrl,
				DesiredCount: 0,
				ExtraResources: &deployer_tpb.ExtraResources{
					SnsTopics: topicNames,
				},
			}

			tb.SideEffect(msg)

			return nil
		},
		))

	// SCALED_UP --> DONE : Done
	// UPSERTED --> DONE : Done
	sm.From(
		deployer_pb.DeploymentStatus_SCALED_UP,
		deployer_pb.DeploymentStatus_UPSERTED,
	).Do(deployer_pb.DeploymentPSMFunc(func(
		ctx context.Context,
		tb deployer_pb.DeploymentPSMTransitionBaton,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Done,
	) error {
		deployment.Status = deployer_pb.DeploymentStatus_DONE

		tb.SideEffect(&deployer_tpb.DeploymentCompleteMessage{
			DeploymentId:    deployment.DeploymentId,
			StackId:         deployment.StackId,
			EnvironmentName: deployment.Spec.EnvironmentName,
			ApplicationName: deployment.Spec.AppName,
		})

		return nil
	},
	))

	// * --> FAILED : Error
	sm.From().Do(deployer_pb.DeploymentPSMFunc(func(
		ctx context.Context,
		tb deployer_pb.DeploymentPSMTransitionBaton,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Error,
	) error {
		deployment.Status = deployer_pb.DeploymentStatus_FAILED

		tb.SideEffect(&deployer_tpb.DeploymentCompleteMessage{
			DeploymentId:    deployment.DeploymentId,
			StackId:         deployment.StackId,
			EnvironmentName: deployment.Spec.EnvironmentName,
			ApplicationName: deployment.Spec.AppName,
		})
		return nil
	},
	))

	return sm, nil
}
