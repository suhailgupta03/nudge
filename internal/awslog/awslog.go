package awslog

import (
	"bytes"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	time2 "nudge/internal/time"
	"strings"
	"sync"
)

// AWSLog implements a simple log buffer that can be supplied to a std
// log instance. It stores logs up to N lines.
type AWSLog struct {
	maxLines int
	buf      *bytes.Buffer
	lines    []types.InputLogEvent
	aws      AWS
	sync.RWMutex
}

// New returns a new log buffer that stores up to maxLines lines.
func New(maxLines int, aws AWS) *AWSLog {
	return &AWSLog{
		maxLines: maxLines,
		buf:      &bytes.Buffer{},
		lines:    make([]types.InputLogEvent, 0, maxLines),
		aws:      aws,
	}
}

// Write writes a log item to the buffer with an auto flush to
// aws log stream when MAX capacity is reached
func (awsLog *AWSLog) Write(b []byte) (n int, err error) {
	awsLog.Lock()
	logLine := strings.TrimSpace(string(b))
	nudgeTime := new(time2.NudgeTime)
	timestamp := nudgeTime.NudgeTime().UnixMilli()

	if len(awsLog.lines) >= awsLog.maxLines {
		awsLog.aws.submitLog(awsLog.lines)
		awsLog.lines = nil
	}

	awsLog.lines = append(awsLog.lines, types.InputLogEvent{
		Message:   &logLine,
		Timestamp: &timestamp,
	})
	awsLog.Unlock()
	return len(b), nil
}
