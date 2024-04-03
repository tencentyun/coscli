package util

import (
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
)

type CosListener struct {
	fo *FileOperations
}

func (l *CosListener) ProgressChangedCallback(event *cos.ProgressEvent) {
	switch event.EventType {
	case cos.ProgressStartedEvent:
	case cos.ProgressDataEvent:
		l.fo.Monitor.updateTransferSize(event.RWBytes)
		l.fo.Monitor.updateDealSize(event.RWBytes)
	case cos.ProgressCompletedEvent:
	case cos.ProgressFailedEvent:
		l.fo.Monitor.updateDealSize(-event.ConsumedBytes)
	default:
		fmt.Printf("Progress Changed Error: unknown progress event type\n")
	}
	freshProgress()
}
