package app

import (
	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/cf"
)

type BuiltApplication struct {
	Template *cloudformation.Template
	*awsdeployer_pb.BuiltApplication
}

func (ba *BuiltApplication) TemplateJSON() ([]byte, error) {
	return ba.Template.JSON()
}

type SNSTopic struct {
	Name string
}

type Application struct {
	*cf.TemplateBuilder
	appName           string
	version           string
	quickMode         bool
	postgresDatabases []*awsdeployer_pb.PostgresDatabaseResource
	snsTopics         map[string]*SNSTopic

	runtimes map[string]*RuntimeService
}

func NewApplication(name, version string) *Application {
	return &Application{
		TemplateBuilder: cf.NewTemplateBuilder(),
		appName:         name,
		version:         version,
		snsTopics:       map[string]*SNSTopic{},
		runtimes:        map[string]*RuntimeService{},
	}
}

func (ss *Application) Build() *BuiltApplication {
	snsToipcs := []string{}
	for _, topic := range ss.snsTopics {
		snsToipcs = append(snsToipcs, topic.Name)
	}

	template := ss.TemplateBuilder.Build()

	return &BuiltApplication{
		Template: template.Template,
		BuiltApplication: &awsdeployer_pb.BuiltApplication{
			Parameters:        template.Parameters,
			PostgresDatabases: ss.postgresDatabases,
			SnsTopics:         snsToipcs,
			Name:              ss.appName,
			Version:           ss.version,
			QuickMode:         ss.quickMode,
		},
	}
}

func (ss *Application) AddSNSTopic(name string) {
	for _, topic := range ss.snsTopics {
		if topic.Name == name {
			return
		}
	}
	ss.snsTopics[name] = &SNSTopic{
		Name: name,
	}
}
