package deployer

import (
	"fmt"

	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/infra/v1/awsinfra_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
)

func buildDatabaseSpecs(databases []*awsdeployer_pb.PostgresDatabaseResource, awsCluster *environment_pb.AWSCluster, envName, appName string) ([]*awsdeployer_pb.PostgresSpec, error) {

	auroraHosts := map[string]*awsinfra_pb.AuroraConnection{}
	dbSpecs := make([]*awsdeployer_pb.PostgresSpec, 0, len(databases))
	for _, db := range databases {

		fullName := safeDBName(fmt.Sprintf("%s_%s_%s", envName, appName, db.DbNameSuffix))

		var host *environment_pb.RDSHost
		for _, search := range awsCluster.RdsHosts {
			if search.ServerGroupName == db.ServerGroup {
				host = search
				break
			}
		}
		if host == nil {
			return nil, fmt.Errorf("no RDS host found for database %s", db.AppKey)
		}

		dbSpec := &awsdeployer_pb.PostgresSpec{
			AppKey:                db.AppKey,
			FullDbName:            fullName,
			DbExtensions:          db.DbExtensions,
			AppConnection:         &awsdeployer_pb.PostgresConnectionType{},
			AdminConnection:       &awsinfra_pb.RDSHostType{},
			ClientSecurityGroupId: host.ClientSecurityGroupId,
		}
		if db.MigrationTaskOutputName != nil {
			dbSpec.Migrate = &awsdeployer_pb.PostgresMigrateSpec{
				Type: &awsdeployer_pb.PostgresMigrateSpec_Ecs{
					Ecs: &awsdeployer_pb.PostgresMigrateSpec_ECS{
						TaskOutputName: *db.MigrationTaskOutputName,
						SubnetIds:      awsCluster.EcsCluster.SubnetIds,
					},
				},
			}
		}

		switch hostType := host.Auth.Get().(type) {
		case *environment_pb.RDSAuthType_SecretsManager:
			secret := db.GetSecretOutputName()
			if secret == "" {
				panic(fmt.Sprintf("no secret name found for database %s", db.AppKey))
			}

			dbSpec.AppConnection.Set(&awsdeployer_pb.PostgresConnectionType_SecretsManager{
				AppSecretOutputName: secret,
			})

			dbSpec.AdminConnection.Set(&awsinfra_pb.RDSHostType_SecretsManager{
				SecretName: hostType.SecretName,
			})

		case *environment_pb.RDSAuthType_IAM:
			clientConn := &awsinfra_pb.AuroraConnection{
				Endpoint:   host.Endpoint,
				Port:       host.Port,
				DbUser:     fullName,
				DbName:     fullName,
				Identifier: host.Identifier,
			}
			auroraHosts[db.ServerGroup] = clientConn

			dbSpec.AppConnection.Set(&awsdeployer_pb.PostgresConnectionType_Aurora{
				Conn: clientConn,
			})

			dbSpec.AdminConnection.Set(&awsinfra_pb.RDSHostType_Aurora{
				Conn: &awsinfra_pb.AuroraConnection{
					Endpoint: host.Endpoint,
					Port:     host.Port,
					DbUser:   hostType.DbUser,
					DbName:   coalesce(hostType.DbUser, hostType.DbName),
				},
			})

		}

		dbSpecs = append(dbSpecs, dbSpec)
	}
	return dbSpecs, nil
}
