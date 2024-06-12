package mocks

import (
	"context"
	"fmt"

	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
)

type Github struct {
	Configs map[string][]*application_pb.Application
}

func NewGithub() *Github {
	return &Github{
		Configs: map[string][]*application_pb.Application{},
	}
}

func (gm *Github) PullO5Configs(ctx context.Context, org string, repo string, ref string) ([]*application_pb.Application, error) {
	key := fmt.Sprintf("%s/%s/%s", org, repo, ref)
	if configs, ok := gm.Configs[key]; ok {
		return configs, nil
	}
	return []*application_pb.Application{}, nil
}
