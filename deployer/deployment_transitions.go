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
	"gopkg.daemonl.com/sqrlx"
)

type DeploymentTransitionBaton = psm.TransitionBaton[deployer_pb.IsDeploymentEventTypeWrappedType]

type DeploymentEventer struct {
	*psm.Eventer[
		*deployer_pb.DeploymentState,
		deployer_pb.DeploymentStatus,
		*deployer_pb.DeploymentEvent,
		deployer_pb.IsDeploymentEventTypeWrappedType,
	]
}

func (se DeploymentEventer) handleEvent(
	ctx context.Context,
	tx sqrlx.Transaction,
	state *deployer_pb.DeploymentState,
	outerEvent *deployer_pb.DeploymentEvent,
) error {
	ctx = log.WithFields(ctx, map[string]interface{}{
		"entityType":   "Deployment",
		"deploymentId": state.DeploymentId,
	})
	wrapped := psm.NewSqrlxTransaction[
		*deployer_pb.DeploymentState,
		*deployer_pb.DeploymentEvent,
	](tx, storeDeploymentEvent)

	return se.Eventer.Run(ctx, wrapped, state, outerEvent)
}

func NewDeploymentEventer() (*DeploymentEventer, error) {
	deploymentEventer := &DeploymentEventer{
		Eventer: &psm.Eventer[
			*deployer_pb.DeploymentState,
			deployer_pb.DeploymentStatus,
			*deployer_pb.DeploymentEvent,
			deployer_pb.IsDeploymentEventTypeWrappedType,
		]{
			WrapEvent: func(
				ctx context.Context,
				state *deployer_pb.DeploymentState,
				event deployer_pb.IsDeploymentEventTypeWrappedType,
			) *deployer_pb.DeploymentEvent {
				wrappedEvent := &deployer_pb.DeploymentEvent{
					DeploymentId: state.DeploymentId,
					Metadata: &deployer_pb.EventMetadata{
						EventId:   uuid.NewString(),
						Timestamp: timestamppb.Now(),
					},
					Event: &deployer_pb.DeploymentEventType{},
				}
				wrappedEvent.Event.Set(event)
				return wrappedEvent
			},
			UnwrapEvent: func(event *deployer_pb.DeploymentEvent) deployer_pb.IsDeploymentEventTypeWrappedType {
				return event.Event.Get()
			},
			StateLabel: func(state *deployer_pb.DeploymentState) string {
				return state.Status.ShortString()
			},
			EventLabel: func(event deployer_pb.IsDeploymentEventTypeWrappedType) string {
				typeKey := string(event.TypeKey())

				// TODO: This by generation and annotation
				if stackStatus, ok := event.(*deployer_pb.DeploymentEventType_StackStatus); ok {
					typeKey = fmt.Sprintf("%s.%s", typeKey, stackStatus.Lifecycle.ShortString())
				}
				return typeKey
			},
		},
	}

	// [*] --> QUEUED : Created
	deploymentEventer.Register(
		psm.NewTransition([]deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_UNSPECIFIED,
		}, func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
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
	deploymentEventer.Register(
		psm.NewTransition([]deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_QUEUED,
		},
			func(ctx context.Context,
				tb DeploymentTransitionBaton,
				deployment *deployer_pb.DeploymentState,
				event *deployer_pb.DeploymentEventType_Triggered,
			) error {
				deployment.Status = deployer_pb.DeploymentStatus_TRIGGERED

				tb.ChainEvent(&deployer_pb.DeploymentEventType_StackWait{})

				return nil
			},
		))

	// TRIGGERED --> WAITING : StackWait
	deploymentEventer.Register(
		psm.NewTransition([]deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_TRIGGERED,
		},
			func(
				ctx context.Context,
				tb DeploymentTransitionBaton,
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
			},
		))

	// AVAILABLE --> CREATING : StackCreate
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_AVAILABLE,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
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

	// WAITING --> AVAILABLE : StackStatus.Complete
	// WAITING --> AVAILABLE : StackStatus.Missing
	// WAITING --> AVAILABLE : StackStatus.Terminal + UPDATE_ROLLBACK_COMPLETE
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_WAITING,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_AVAILABLE
			deployment.StackOutput = event.StackOutput
			deployment.LastStackLifecycle = event.Lifecycle
			deployment.WaitingOnRemotePhase = nil

			if deployment.Spec.QuickMode {
				tb.ChainEvent(&deployer_pb.DeploymentEventType_StackUpsert{})
			} else if event.Lifecycle == deployer_pb.StackLifecycle_MISSING {
				tb.ChainEvent(&deployer_pb.DeploymentEventType_StackCreate{})
			} else {
				tb.ChainEvent(&deployer_pb.DeploymentEventType_StackScale{
					DesiredCount: int32(0),
				})
			}

			return nil
		},
	).WithEventFilter(func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
		return event.Lifecycle == deployer_pb.StackLifecycle_COMPLETE ||
			event.Lifecycle == deployer_pb.StackLifecycle_MISSING ||
			(event.Lifecycle == deployer_pb.StackLifecycle_TERMINAL && event.FullStatus == "UPDATE_ROLLBACK_COMPLETE")
	}))

	// AVAILABLE --> UPSERTING : StackUpsert
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_AVAILABLE,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
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
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_UPSERTING,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_UPSERTED
			deployment.StackOutput = event.StackOutput
			deployment.WaitingOnRemotePhase = nil

			tb.ChainEvent(&deployer_pb.DeploymentEventType_Done{})

			return nil
		},
	).WithEventFilter(func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
		return event.Lifecycle == deployer_pb.StackLifecycle_COMPLETE
	}))

	// SCALING_DOWN --> SCALED_DOWN : StackStatus.Complete
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_SCALING_DOWN,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_SCALED_DOWN
			deployment.StackOutput = event.StackOutput
			deployment.WaitingOnRemotePhase = nil

			tb.ChainEvent(&deployer_pb.DeploymentEventType_StackTrigger{
				Phase: "update",
			})
			return nil
		},
	).WithEventFilter(func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
		return event.Lifecycle == deployer_pb.StackLifecycle_COMPLETE
	}))

	// INFRA_MIGRATE --> INFRA_MIGRATED : StackStatus.Complete
	// CREATING --> INFRA_MIGRATED : StackStatus.Complete
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_INFRA_MIGRATE,
		deployer_pb.DeploymentStatus_CREATING,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_INFRA_MIGRATED
			deployment.StackOutput = event.StackOutput
			deployment.WaitingOnRemotePhase = nil

			tb.ChainEvent(&deployer_pb.DeploymentEventType_MigrateData{})

			return nil
		},
	).WithEventFilter(func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
		return event.Lifecycle == deployer_pb.StackLifecycle_COMPLETE
	}))

	// ScalingUp --> ScaledUp : StackStatus.Complete
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_SCALING_UP,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_SCALED_UP
			deployment.StackOutput = event.StackOutput
			deployment.WaitingOnRemotePhase = nil

			tb.ChainEvent(&deployer_pb.DeploymentEventType_Done{})
			return nil
		},
	).WithEventFilter(func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
		return event.Lifecycle == deployer_pb.StackLifecycle_COMPLETE
	}))

	// Any Waiting --> Any Waiting : StackStatus.Progress
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_WAITING,
		deployer_pb.DeploymentStatus_CREATING,
		deployer_pb.DeploymentStatus_UPSERTING,
		deployer_pb.DeploymentStatus_SCALING_UP,
		deployer_pb.DeploymentStatus_SCALING_DOWN,
		deployer_pb.DeploymentStatus_INFRA_MIGRATE,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			// nothing, just log the progress
			return nil
		},
	).WithEventFilter(func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
		return event.Lifecycle == deployer_pb.StackLifecycle_PROGRESS
	}))

	// WAITING --> FAILED : StackStatus.Rollback In Progress.
	// TODO: Needs an actual key
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_WAITING,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			// nothing, just log the progress
			return nil
		},
	).WithEventFilter(func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
		return event.FullStatus == "UPDATE_ROLLBACK_IN_PROGRESS"
	}))

	// Creating --> Failed : StackStatus.Failed
	// ScalingUp --> Failed : StackStatus.Failed
	// ScalingDown --> Failed : StackStatus.Failed
	// InfraMigrate --> Failed : StackStatus.Failed
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_WAITING,
		deployer_pb.DeploymentStatus_UPSERTING,
		deployer_pb.DeploymentStatus_SCALING_UP,
		deployer_pb.DeploymentStatus_SCALING_DOWN,
		deployer_pb.DeploymentStatus_INFRA_MIGRATE,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_FAILED
			deployment.StackOutput = event.StackOutput
			deployment.WaitingOnRemotePhase = nil

			return fmt.Errorf("stack failed: %s", event.FullStatus)
		},
	).WithEventFilter(func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
		return event.Lifecycle == deployer_pb.StackLifecycle_TERMINAL ||
			event.Lifecycle == deployer_pb.StackLifecycle_CREATE_FAILED ||
			event.Lifecycle == deployer_pb.StackLifecycle_ROLLING_BACK
	}))

	// INFRA_MIGRATED --> DB_MIGRATING : MigrateData
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_INFRA_MIGRATED,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
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
				tb.ChainEvent(&deployer_pb.DeploymentEventType_DataMigrated{})
			}
			return nil
		},
	))

	// Transition contains business logic.
	// When any migration is still pending, it will not transition
	// When all migrations are complete, if any failed, will transition to
	// failed
	// When all migrations are complete, and none failed, will transition to
	// DataMigrated
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_DB_MIGRATING,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
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
				tb.ChainEvent(&deployer_pb.DeploymentEventType_Error{
					Error: "database migration failed",
				})
				return nil
			}

			tb.ChainEvent(&deployer_pb.DeploymentEventType_DataMigrated{})

			return nil
		},
	))

	// DB_MIGRATING --> DB_MIGRATED : DataMigrated
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_DB_MIGRATING,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_DataMigrated,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_DB_MIGRATED
			tb.ChainEvent(&deployer_pb.DeploymentEventType_StackScale{
				DesiredCount: int32(1),
			})
			return nil
		},
	))

	// DB_MIGRATED --> SCALING_UP : StackScale
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_DB_MIGRATED,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
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
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_AVAILABLE,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
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
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_SCALED_DOWN,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
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
	deploymentEventer.Register(psm.NewTransition([]deployer_pb.DeploymentStatus{
		deployer_pb.DeploymentStatus_SCALED_UP,
		deployer_pb.DeploymentStatus_UPSERTED,
	},
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
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
	deploymentEventer.Register(psm.NewTransition(nil,
		func(
			ctx context.Context,
			tb DeploymentTransitionBaton,
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

	return deploymentEventer, nil
}
