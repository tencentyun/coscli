package util

import (
	"fmt"
	"sync"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

type CosListener struct {
	TotalUploadedBytes int64
	mu                 sync.Mutex
}
type SingleCosListener struct {
	StartTime time.Time
}

func (l *SingleCosListener) ProgressChangedCallback(event *cos.ProgressEvent) {
	switch event.EventType {
	case cos.ProgressStartedEvent:
		PrintTransferProcess(1, event.TotalBytes, 0, 0, event.ConsumedBytes, l.StartTime, false)
	case cos.ProgressDataEvent:
		PrintTransferProcess(1, event.TotalBytes, 0, 0, event.ConsumedBytes, l.StartTime, false)
	case cos.ProgressCompletedEvent:
		PrintTransferProcess(1, event.TotalBytes, 1, 0, event.ConsumedBytes, l.StartTime, false)
	case cos.ProgressFailedEvent:
		PrintTransferProcess(1, event.TotalBytes, 0, 1, event.ConsumedBytes, l.StartTime, false)
	default:
		fmt.Printf("Progress Changed Error: unknown progress event type\n")
	}
}

func (l *CosListener) ProgressChangedCallback(event *cos.ProgressEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	switch event.EventType {
	case cos.ProgressStartedEvent:
	case cos.ProgressDataEvent:
		l.TotalUploadedBytes += event.RWBytes
	case cos.ProgressCompletedEvent:
	case cos.ProgressFailedEvent:
		fmt.Printf("\nTransfer Failed!\n")
	default:
		fmt.Printf("Progress Changed Error: unknown progress event type\n")
	}
}
