package awslog

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"log"
	"time"
)

type AWS struct {
	cfg           aws.Config
	logGroupName  string
	logStreamName string
}

func AWSInit(logGroupName, logStreamName string) *AWS {
	// Load the Shared AWS Configuration (~/.aws/config)
	// If the file is not present, will try to read the standard ENV variables
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to read the AWS configuration %v", err)
	}
	return &AWS{
		cfg,
		logStreamName,
		logGroupName,
	}
}

func (aws *AWS) submitLog(logLines []types.InputLogEvent) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	client := cloudwatchlogs.NewFromConfig(aws.cfg)
	r, err := client.PutLogEvents(ctx, &cloudwatchlogs.PutLogEventsInput{
		LogEvents:     logLines,
		LogGroupName:  &aws.logGroupName,
		LogStreamName: &aws.logStreamName,
	})

	if err != nil || r.RejectedLogEventsInfo != nil {
		log.Printf("Failed to insert logs to AWS log group %s log stream %s - %v", aws.logGroupName, aws.logStreamName, err)
	}

}

func (aws *AWS) DoesLogGroupExist(logGroupName string) bool {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	client := cloudwatchlogs.NewFromConfig(aws.cfg)
	params := &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: &logGroupName,
	}
	logGroups, err := client.DescribeLogGroups(ctx, params)
	if err != nil {
		log.Printf("Failed to read the existing log groups %v", err)
		return false
	}

	found := false
	if logGroups != nil {
		for _, lg := range logGroups.LogGroups {
			if *lg.LogGroupName == logGroupName {
				found = true
				break
			}
		}
	}

	return found
}

func (aws *AWS) DoesLogStreamExist(logGroupName, logStreamName string) bool {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	client := cloudwatchlogs.NewFromConfig(aws.cfg)
	params := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        &logGroupName,
		LogStreamNamePrefix: &logStreamName,
	}
	streams, err := client.DescribeLogStreams(ctx, params)
	if err != nil {
		log.Printf("Failed to read the existing log streams for logGroup %s - %v", logGroupName, err)
	}

	found := false
	if streams != nil {
		for _, stream := range streams.LogStreams {
			if *stream.LogStreamName == logStreamName {
				found = true
				break
			}
		}
	}

	return found
}
