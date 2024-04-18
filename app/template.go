package app

import (
	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/pentops/o5-deploy-aws/cf"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
)

type BuiltApplication struct {
	Template *cloudformation.Template
	*deployer_pb.BuiltApplication
}

func (ba *BuiltApplication) TemplateJSON() ([]byte, error) {
	return ba.Template.JSON()
}

type SNSTopic struct {
	Name string
}

type Application struct {
	*cf.Template
	appName           string
	version           string
	quickMode         bool
	postgresDatabases []*deployer_pb.PostgresDatabaseResource
	snsTopics         map[string]*SNSTopic

	runtimes map[string]*RuntimeService
}

func NewApplication(name, version string) *Application {
	return &Application{
		Template:  cf.NewTemplate(),
		appName:   name,
		version:   version,
		snsTopics: map[string]*SNSTopic{},
		runtimes:  map[string]*RuntimeService{},
	}
}

func (ss *Application) Build() *BuiltApplication {
	snsToipcs := []string{}
	for _, topic := range ss.snsTopics {
		snsToipcs = append(snsToipcs, topic.Name)
	}

	template, parameterSlice := ss.Template.Build()

	return &BuiltApplication{
		Template: template,
		BuiltApplication: &deployer_pb.BuiltApplication{
			Parameters:        parameterSlice,
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
