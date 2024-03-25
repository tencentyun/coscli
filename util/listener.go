package util

import (
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
)

type CosListener struct {
	cc *CopyCommand
}

//type SingleCosListener struct {
//	StartTime time.Time
//}

//func (l *SingleCosListener) ProgressChangedCallback(event *cos.ProgressEvent) {
//	switch event.EventType {
//	case cos.ProgressStartedEvent:
//		PrintTransferProcess(1, event.TotalBytes, 0, 0, event.ConsumedBytes, l.StartTime, false)
//	case cos.ProgressDataEvent:
//		PrintTransferProcess(1, event.TotalBytes, 0, 0, event.ConsumedBytes, l.StartTime, false)
//	case cos.ProgressCompletedEvent:
//		PrintTransferProcess(1, event.TotalBytes, 1, 0, event.ConsumedBytes, l.StartTime, false)
//	case cos.ProgressFailedEvent:
//		PrintTransferProcess(1, event.TotalBytes, 0, 1, event.ConsumedBytes, l.StartTime, false)
//	default:
//		fmt.Printf("Progress Changed Error: unknown progress event type\n")
//	}
//}

func (l *CosListener) ProgressChangedCallback(event *cos.ProgressEvent) {
	switch event.EventType {
	case cos.ProgressStartedEvent:
	case cos.ProgressDataEvent:
		l.cc.Monitor.updateTransferSize(event.RWBytes)
		l.cc.Monitor.updateDealSize(event.RWBytes)
	case cos.ProgressCompletedEvent:
	case cos.ProgressFailedEvent:
		l.cc.Monitor.updateDealSize(-event.ConsumedBytes)
	default:
		fmt.Printf("Progress Changed Error: unknown progress event type\n")
	}
	freshProgress()
}
