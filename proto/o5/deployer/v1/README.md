Deployer
========


## Stack State

```mermaid
stateDiagram-v2
    classDef auto fill:#002222,stroke-dasharray:4
	classDef failed stroke:red,fill:#330000

	[*] --> CREATING : Triggered
	CREATING --> CREATING : Triggered<br>(Adds to the queue)
	CREATING --> STABLE : DeploymentCompleted
	state stable <<choice>>
	STABLE:::auto --> stable
	stable --> AVAILABLE : Nothing Queued
	stable --> MIGRATING : Pops from queue
	AVAILABLE --> MIGRATING : Triggered<br>(External)
	MIGRATING --> STABLE : DeploymentCompleted
	MIGRATING --> MIGRATING : Triggered<br>(Adds to the queue)
	CREATING --> BROKEN:::failed : DeploymentFailed
	MIGRATING --> BROKEN : DeploymentFailed
```

## Deployment State

```mermaid
stateDiagram-v2
    classDef auto fill:#002222,stroke-dasharray:4
	classDef failed stroke:red,fill:#330000

    [*] --> QUEUED : Requested
	QUEUED --> TRIGGERED : Triggered
    TRIGGERED:::auto --> WAITING : StackWait

    WAITING --> WAITING : StackStatus.Progress
	FAILED_1:::failed : FAILED
    WAITING --> FAILED_1:::failed : Stack Failure
    WAITING --> AVAILABLE : StackStatus.Complete<br>StackStatus.Missing<br>StackStatus.Terminal (rolled back)
	AVAILABLE --> RUNNING : Running
	state stepStatus <<choice>>
	RUNNING --> stepStatus : StepStatus
	stepStatus --> RUNNING : Any Running
	state noRunning <<choice>>
	stepStatus --> noRunning : All Stopped
	noRunning --> FAILED_1 : Any Error
	noRunning --> DONE : All OK
```

## Database Deriving

```mermaid
flowchart

	Application[o5.application.v1.Application]
	Environment[o5.environment.v1.Environment]
	Built[BuiltApplication]
	DeploymentSpec[o5.deployer.v1.DeploymentSpec]

	Application --> BuildApp(Build Application) --> Built --> Build(Build Spec)
	Environment --> Build
	Build --> CFTemplate
	Build --> DeploymentSpec
```

# Sequences

## Github Trigger

```mermaid
sequenceDiagram
	autonumber
	participant github as GitHub
	participant webhook as GH Webhook<br>Worker
	participant db as Deployer<br>DB
	participant worker as Deployer<br>Worker

	github ->> webhook : Push
	webhook ->> db : Get Push Targets
	db -->> webhook : Push Targets
	webhook ->> github : Pull O5 Configs
	github -->> webhook : O5 Configs
	loop push targets
	webhook ->> worker : RequestDeployment
	end
```

## Manual Trigger

```mermaid
sequenceDiagram
	autonumber
	actor user
	participant github as GitHub
	participant command as Deployer<br>Command
	participant db as Deployer<br>DB
	participant worker as Deployer<br>Worker

	user ->> command : TriggerDeployment
	command ->> github : Pull O5 Configs
	github -->> command : O5 Configs
	command ->> db : Lookup Environment
	db -->> command : Matching Env
	command ->> worker : RequestDeployment
```

## Deployment

```mermaid
sequenceDiagram
	autonumber
	participant bus as Event Bus
	participant worker as Deployer<br>Worker
	participant db as Deployer<br>DB
	participant deployment as Deployment<br>PSM
	participant stack as Stack<br>PSM
	participant migration as PG Migration<br>PSM
	participant aws as AWS

	bus ->> worker : RequestDeployment
	activate worker
	worker ->> db : GetEnvironment
	db -->> worker : Environment
	worker ->> worker : Build Spec
	worker ->> aws : CF Template JSON (S3)
	worker ->> deployment : Created
	deployment ->> deployment : Status -> QUEUED
	deactivate worker

	deployment ->> stack : Triggered
	stack ->> worker : TriggerDeployment
	activate worker
	worker ->> deployment : Triggered
	note over deployment : TRIGGERED

	deployment ->> deployment : StackWait
	note over deployment : WAITING
	deployment ->> aws : StabalizeStack
	aws ->> worker : StackStatus.Missing
	deactivate worker

	activate worker
	worker ->> deployment : StackStatus.Missing
	deployment ->> deployment : StackCreate
	deployment ->> aws : CreateNewStack
	deactivate worker

	aws ->> worker : StackStatus.Complete
	worker ->> deployment : StackStatus.Complete
	note over deployment : INFRA_MIGRATED

	deployment ->> deployment : MigrateData
	note over deployment : DB_MIGRATING

	deployment ->> migration : MigratePostgresDatabase
	migration ->> aws : Upsert DB
	migration ->> aws : Cleanup Permissions
	migration ->> aws : Run Migration ECS Task
	migration ->> aws : Cleanup Permissions
	migration ->> worker : MigrationStatus.Completed
	worker ->> deployment : MigrationStatus.Completed
	deployment ->> deployment : DataMigrated
	note over deployment : DB_MIGRATED
	deployment ->> deployment : StackScale:1
	note over deployment : SCALING_UP
	deployment ->> aws : ScaleStack

	aws ->> worker : StackStatus.Complete
	worker ->> deployment : StackStatus.Complete
	note over deployment : SCALED_UP
	deployment ->> deployment : Done
	note over deployment : DONE

```



# Deployment Plans

## Current

Current state of the code, does not evaluate, and quick is sequential.

### Quick

When 'quick' is set on deployer, does not support DB migration

```mermaid
flowchart LR
  Start --> q1[CFUpsert] --> Done
```


### Full
```mermaid
flowchart LR
  Start --> ScaleDown
  ScaleDown --> d1[InfraMigrate] --> d2[PGMigrate] --> ScaleUp
  ScaleUp --> Done
```

## Future / Plan

### Quick

When 'quick' is set on deployer, everything runs in parallel

```mermaid
flowchart LR
  Start --> q1[CFUpsert] --> Done
  Start --> q2[PGUpsert] --> q3[PGMigrate] --> q4[PGCleanup] --> Done
```


### Full

Once evaluations are built in, the plan will depend on deployment config as well
as the current state of the infra.

```mermaid
flowchart
  Start --> PGUpsert --> PGEval --> EvalJoin
  Start --> CFPlan --> EvalJoin{{EvalJoin}}
  EvalJoin --DOWNTIME--> d0( )
  EvalJoin --BEFORE--> b0( )
  EvalJoin --NOP--> n0( )
  EvalJoin --PARALLEL--> p0( )

  d0 --> ScaleDown
  ScaleDown --> d1[InfraMigrate] --> ScaleUp
  ScaleDown --> d2[PGMigrate] --> d3[PGCleanup] --> ScaleUp
  ScaleUp --> Done

  b2[VersionMigrate]--> Done
  b0 --> b1[InfraMigrate] --> b2
  b0 --> b3[PGMigrate] --> b4[PGCleanup] --> b2

  p0 --> p1[CFUpsert] --> Done
  p0 --> p2[PGMigrate] --> p3[PGCleanup] --> Done

  n0 --> n1[VersionMigrate] --> Done

```

