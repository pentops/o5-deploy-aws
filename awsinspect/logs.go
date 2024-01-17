package awsinspect

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	"github.com/pentops/log.go/pretty"
)

type CloudwatchLogsClient interface {
	GetLogEvents(ctx context.Context, params *cloudwatchlogs.GetLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetLogEventsOutput, error)
}

type LogStream struct {
	Container string
	LogGroup  string
	LogStream string
}

func TailLogStream(ctx context.Context, client CloudwatchLogsClient, logGroup LogStream, fromTime time.Time) error {

	printer := pretty.NewPrinter(os.Stdout)

	fromTimeInt := fromTime.UnixNano() / int64(time.Millisecond)
	var nextToken *string
	for {

		logEvents, err := client.GetLogEvents(ctx, &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  &logGroup.LogGroup,
			LogStreamName: &logGroup.LogStream,
			StartTime:     &fromTimeInt,
			StartFromHead: aws.Bool(true),
			NextToken:     nextToken,
		})
		if err != nil {
			return err
		}
		nextToken = logEvents.NextForwardToken
		if nextToken == nil {
			return fmt.Errorf("nextToken is nil")
		}

		for _, event := range logEvents.Events {
			if event.Message != nil {
				printer.PrintRawLine(logGroup.Container, *event.Message)
			}
		}
	}
}
