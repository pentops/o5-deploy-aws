package awsinfra

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-messaging/gen/o5/messaging/v1/messaging_tpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RawWorkerCFHandler interface {
	HandleCloudFormationEvent(ctx context.Context, payload map[string]string) error
}

type RawWorkerECSHandler interface {
	HandleECSTaskEvent(ctx context.Context, eventID string, event *ECSTaskStateChangeEvent) error
}

type RawMessageWorker struct {
	messaging_tpb.UnimplementedRawMessageTopicServer

	cfHandler  RawWorkerCFHandler
	ecsHandler RawWorkerECSHandler
}

func NewRawMessageWorker(cfHandler RawWorkerCFHandler, ecsHandler RawWorkerECSHandler) *RawMessageWorker {
	return &RawMessageWorker{
		cfHandler:  cfHandler,
		ecsHandler: ecsHandler,
	}
}

func (worker *RawMessageWorker) Raw(ctx context.Context, msg *messaging_tpb.RawMessage) (*emptypb.Empty, error) {

	log.WithField(ctx, "topic", msg.Topic).Debug("RawMessage")

	if strings.HasSuffix(msg.Topic, "-o5-infra") {

		if msg.Payload[0] == '{' {
			if err := worker.HandleInfraJSON(ctx, msg.Payload); err != nil {
				return nil, err
			}
			return &emptypb.Empty{}, nil
		}

		if err := worker.HandleInfraNonJSON(ctx, msg.Payload); err != nil {
			return nil, err
		}
		return &emptypb.Empty{}, nil
	}

	return nil, fmt.Errorf("unknown topic: %s", msg.Topic)
}

func (worker *RawMessageWorker) HandleInfraJSON(ctx context.Context, payload []byte) error {
	infraEvent := &InfraEvent{}
	if err := json.Unmarshal(payload, infraEvent); err == nil && infraEvent.Valid() {
		log.WithFields(ctx, map[string]interface{}{
			"eventId":     infraEvent.ID,
			"eventSource": infraEvent.Source,
			"eventType":   infraEvent.DetailType,
		}).Debug("infra event")

		if !strings.HasPrefix(infraEvent.Source, "aws.") {
			return fmt.Errorf("unknown event-bridge source: %s", infraEvent.Source)
		}
		if err := worker.handleAWSInfraEvent(ctx, infraEvent); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("unknown JSON-like message: %s", string(payload))
}

func (worker *RawMessageWorker) HandleInfraNonJSON(ctx context.Context, payload []byte) error {
	fields, err := parseAWSRawMessage(payload)
	if err != nil {
		return err
	}
	if err := worker.cfHandler.HandleCloudFormationEvent(ctx, fields); err != nil {
		return err
	}
	return nil
}

func parseAWSRawMessage(raw []byte) (map[string]string, error) {

	lines := strings.Split(string(raw), "\n")
	fields := map[string]string{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line: %s", line)
		}
		s := parts[1]
		if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
			s = s[1 : len(s)-1]
		}
		fields[parts[0]] = s
	}
	return fields, nil
}

type InfraEvent struct {
	Source     string          `json:"source"`
	DetailType string          `json:"detail-type"`
	Detail     json.RawMessage `json:"detail"`
	ID         string          `json:"id"`
}

func (ie *InfraEvent) Valid() bool {
	return ie.Source != "" && ie.DetailType != ""
}

func (worker *RawMessageWorker) handleAWSInfraEvent(ctx context.Context, infraEvent *InfraEvent) error {
	if infraEvent.Source == "aws.ecs" && infraEvent.DetailType == "ECS Task State Change" {
		taskEvent := &ECSTaskStateChangeEvent{}
		if err := json.Unmarshal(infraEvent.Detail, taskEvent); err != nil {
			return err
		}

		if err := worker.ecsHandler.HandleECSTaskEvent(ctx, infraEvent.ID, taskEvent); err != nil {
			return fmt.Errorf("failed to handle ECS task event: %w", err)
		}
		return nil
	} else {
		return fmt.Errorf("unhandled AWS Infra event: %s %s", infraEvent.Source, infraEvent.DetailType)
	}
}
