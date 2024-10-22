package deployer

import (
	"context"
	"fmt"

	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/infra/v1/awsinfra_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder"
)

func buildDatabaseSpecs(ctx context.Context, databases []appbuilder.DatabaseRef, awsCluster *environment_pb.AwsCluster, environment *environment_pb.Environment, app *application_pb.Application) ([]*awsdeployer_pb.PostgresSpec, error) {

	auroraHosts := map[string]*awsinfra_pb.AuroraConnection{}
	dbSpecs := make([]*awsdeployer_pb.PostgresSpec, len(app.Databases))
	for idx, db := range app.Databases {
		fullName := safeDBName(fmt.Sprintf("%s_%s_%s", environment.FullName, app.Name, db.DbName))

		var host *environment_pb.RDSHost
		for _, search := range awsCluster.RdsHosts {
			if search.ServerGroupName == db.ServerGroup {
				host = search
				break
			}
		}
		if host == nil {
			return nil, fmt.Errorf("no RDS host found for database %s", db.DbName)
		}

		dbSpec := &awsdeployer_pb.PostgresSpec{
			DbName:                  fullName,
			DbExtensions:            db.DbExtensions,
			MigrationTaskOutputName: db.MigrationTaskOutputName,
			AppConnection:           &awsdeployer_pb.PostgresConnectionType{},
			AdminConnection:         &awsinfra_pb.RDSHostType{},
			ClientSecurityGroupId:   host.ClientSecurityGroupId,
		}
		dbSpecs[idx] = dbSpec

		switch hostType := host.Auth.Get().(type) {
		case *environment_pb.RDSAuthType_SecretsManager:
			secret := db.GetSecretOutputName()
			if secret == "" {
				panic(fmt.Sprintf("no secret name found for database %s", db.DbName))
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

	}
}
