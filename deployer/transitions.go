package deployer

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
)

var transitions = []ITransitionSpec{
	// Ignore Events
	TransitionSpec[*deployer_pb.DeploymentEventType_StackStatus]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_DB_MIGRATING,
		},
		EventFilter: func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
			return event.Lifecycle == deployer_pb.StackLifecycle_COMPLETE
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			return nil
		},
	},

	// [*] --> Queued : Triggered
	TransitionSpec[*deployer_pb.DeploymentEventType_Triggered]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_UNSPECIFIED,
		},
		Transition: func(ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_Triggered,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_QUEUED
			deployment.Spec = event.Spec
			deployment.StackName = fmt.Sprintf("%s-%s", event.Spec.EnvironmentName, event.Spec.AppName)

			// TODO: Wait for lock from DB, for now just pretend
			tb.ChainEvent(newEvent(deployment, &deployer_pb.DeploymentEventType_GotLock_{
				GotLock: &deployer_pb.DeploymentEventType_GotLock{},
			}))
			return nil
		},
	},
	// Queued --> Locked : GotLock
	TransitionSpec[*deployer_pb.DeploymentEventType_GotLock]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_QUEUED,
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_GotLock,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_LOCKED

			tb.ChainEvent(newEvent(deployment, &deployer_pb.DeploymentEventType_StackWait_{
				StackWait: &deployer_pb.DeploymentEventType_StackWait{},
			}))

			return nil
		},
	},
	// Locked --> Waiting : StackWait
	TransitionSpec[*deployer_pb.DeploymentEventType_StackWait]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_LOCKED,
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackWait,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_WAITING
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
	},
	// Waiting --> New : StackStatus.Missing
	TransitionSpec[*deployer_pb.DeploymentEventType_StackStatus]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_WAITING,
		},
		EventFilter: func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
			return event.Lifecycle == deployer_pb.StackLifecycle_MISSING
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {

			deployment.Status = deployer_pb.DeploymentStatus_NEW

			tb.ChainEvent(newEvent(deployment, &deployer_pb.DeploymentEventType_StackCreate_{
				StackCreate: &deployer_pb.DeploymentEventType_StackCreate{},
			}))

			return nil
		},
	},
	// New --> Creating : StackCreate
	TransitionSpec[*deployer_pb.DeploymentEventType_StackCreate]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_NEW,
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackCreate,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_CREATING

			initialParameters, err := tb.ResolveParameters(deployment.Spec.Parameters)
			if err != nil {
				return err
			}

			topicNames := make([]string, len(deployment.Spec.SnsTopics))
			for i, topic := range deployment.Spec.SnsTopics {
				topicNames[i] = fmt.Sprintf("%s-%s", deployment.Spec.EnvironmentName, topic.Name)
			}

			// Create, scale 0
			tb.SideEffect(&deployer_tpb.CreateNewStackMessage{
				StackId: &deployer_tpb.StackID{
					StackName:       deployment.StackName,
					DeploymentId:    deployment.DeploymentId,
					DeploymentPhase: "create",
				},
				Parameters:   initialParameters,
				TemplateUrl:  deployment.Spec.TemplateUrl,
				DesiredCount: 0,
				ExtraResources: &deployer_tpb.ExtraResources{
					SnsTopics: topicNames,
				},
			})
			return nil
		},
	},
	// Waiting --> Available : StackStatus.Complete
	TransitionSpec[*deployer_pb.DeploymentEventType_StackStatus]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_WAITING,
		},
		EventFilter: func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
			return event.Lifecycle == deployer_pb.StackLifecycle_COMPLETE
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_AVAILABLE
			deployment.StackOutput = event.StackOutput

			tb.ChainEvent(newEvent(deployment, &deployer_pb.DeploymentEventType_StackScale_{
				StackScale: &deployer_pb.DeploymentEventType_StackScale{
					DesiredCount: int32(0),
				},
			}))

			return nil
		},
	},
	TransitionSpec[*deployer_pb.DeploymentEventType_StackStatus]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_SCALING_DOWN,
		},
		EventFilter: func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
			return event.Lifecycle == deployer_pb.StackLifecycle_COMPLETE
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_SCALED_DOWN
			deployment.StackOutput = event.StackOutput

			tb.ChainEvent(newEvent(deployment, &deployer_pb.DeploymentEventType_StackTrigger_{
				StackTrigger: &deployer_pb.DeploymentEventType_StackTrigger{
					Phase: "update",
				},
			}))
			return nil
		},
	},
	// InfraMigrate --> InfraMigrated : StackStatus.Complete
	TransitionSpec[*deployer_pb.DeploymentEventType_StackStatus]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_INFRA_MIGRATE,
		},
		EventFilter: func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
			return event.Lifecycle == deployer_pb.StackLifecycle_COMPLETE
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_INFRA_MIGRATED
			deployment.StackOutput = event.StackOutput

			tb.ChainEvent(newEvent(deployment, &deployer_pb.DeploymentEventType_MigrateData_{
				MigrateData: &deployer_pb.DeploymentEventType_MigrateData{},
			}))

			return nil
		},
	},
	// ScalingUp --> ScaledUp : StackStatus.Complete
	TransitionSpec[*deployer_pb.DeploymentEventType_StackStatus]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_SCALING_UP,
		},
		EventFilter: func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
			return event.Lifecycle == deployer_pb.StackLifecycle_COMPLETE
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_SCALED_UP
			deployment.StackOutput = event.StackOutput

			tb.ChainEvent(newEvent(deployment, &deployer_pb.DeploymentEventType_Done_{
				Done: &deployer_pb.DeploymentEventType_Done{},
			}))
			return nil
		},
	},
	// Any Waiting --> Any Waiting : StackStatus.Progress
	TransitionSpec[*deployer_pb.DeploymentEventType_StackStatus]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_WAITING,
			deployer_pb.DeploymentStatus_CREATING,
			deployer_pb.DeploymentStatus_SCALING_UP,
			deployer_pb.DeploymentStatus_SCALING_DOWN,
			deployer_pb.DeploymentStatus_INFRA_MIGRATE,
		},
		EventFilter: func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
			return event.Lifecycle == deployer_pb.StackLifecycle_PROGRESS
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			// nothing, just log the progress
			return nil
		},
	},
	// Waiting --> Failed : StackStatus.Failed
	// Creating --> Failed : StackStatus.Failed
	// ScalingUp --> Failed : StackStatus.Failed
	// ScalingDown --> Failed : StackStatus.Failed
	// InfraMigrate --> Failed : StackStatus.Failed
	TransitionSpec[*deployer_pb.DeploymentEventType_StackStatus]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_WAITING,
			deployer_pb.DeploymentStatus_SCALING_UP,
			deployer_pb.DeploymentStatus_SCALING_DOWN,
			deployer_pb.DeploymentStatus_INFRA_MIGRATE,
		},
		EventFilter: func(event *deployer_pb.DeploymentEventType_StackStatus) bool {
			return event.Lifecycle == deployer_pb.StackLifecycle_TERMINAL ||
				event.Lifecycle == deployer_pb.StackLifecycle_CREATE_FAILED ||
				event.Lifecycle == deployer_pb.StackLifecycle_ROLLING_BACK
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackStatus,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_FAILED
			deployment.StackOutput = event.StackOutput

			return fmt.Errorf("stack failed: %s", event.FullStatus)
		},
	},
	// InfraMigrated --> DBMigrating : MigrateData
	TransitionSpec[*deployer_pb.DeploymentEventType_MigrateData]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_INFRA_MIGRATED,
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
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

				migrationMsg, err := tb.buildMigrationRequest(ctx, deployment, migration)
				if err != nil {
					return err
				}
				tb.SideEffect(migrationMsg)
			}

			if len(deployment.DataMigrations) == 0 {
				tb.ChainEvent(newEvent(deployment, &deployer_pb.DeploymentEventType_DataMigrated_{
					DataMigrated: &deployer_pb.DeploymentEventType_DataMigrated{},
				}))
			}
			return nil
		},
	},
	// Transition contains business logic.
	// When any migration is still pending, it will not transition
	// When all migrations are complete, if any failed, will transition to
	// failed
	// When all migrations are complete, and none failed, will transition to
	// DataMigrated
	TransitionSpec[*deployer_pb.DeploymentEventType_DBMigrateStatus]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_DB_MIGRATING,
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
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
				tb.ChainEvent(newEvent(deployment, &deployer_pb.DeploymentEventType_Error_{
					Error: &deployer_pb.DeploymentEventType_Error{
						Error: "database migration failed",
					},
				}))
				return nil
			}

			tb.ChainEvent(newEvent(deployment, &deployer_pb.DeploymentEventType_DataMigrated_{
				DataMigrated: &deployer_pb.DeploymentEventType_DataMigrated{},
			}))

			return nil
		},
	},
	TransitionSpec[*deployer_pb.DeploymentEventType_DataMigrated]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_DB_MIGRATING,
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_DataMigrated,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_DB_MIGRATED
			tb.ChainEvent(newEvent(deployment, &deployer_pb.DeploymentEventType_StackScale_{
				StackScale: &deployer_pb.DeploymentEventType_StackScale{
					DesiredCount: int32(1),
				},
			}))
			return nil
		},
	},
	TransitionSpec[*deployer_pb.DeploymentEventType_StackScale]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_DB_MIGRATED,
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackScale,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_SCALING_UP
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
	},
	TransitionSpec[*deployer_pb.DeploymentEventType_StackScale]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_AVAILABLE,
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackScale,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_SCALING_DOWN

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
	},
	TransitionSpec[*deployer_pb.DeploymentEventType_StackTrigger]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_SCALED_DOWN,
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackTrigger,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_INFRA_MIGRATE

			parameters, err := tb.ResolveParameters(deployment.Spec.Parameters)
			if err != nil {
				return err
			}

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
				Parameters:   parameters,
				TemplateUrl:  deployment.Spec.TemplateUrl,
				DesiredCount: 0,
				ExtraResources: &deployer_tpb.ExtraResources{
					SnsTopics: topicNames,
				},
			}

			tb.SideEffect(msg)

			return nil
		},
	},
	TransitionSpec[*deployer_pb.DeploymentEventType_Done]{
		FromStatus: []deployer_pb.DeploymentStatus{
			deployer_pb.DeploymentStatus_SCALED_UP,
		},
		Transition: func(
			ctx context.Context,
			tb TransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_Done,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_DONE
			return nil
		},
	},
}
