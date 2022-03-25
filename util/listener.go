package util

import (
	"fmt"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func drawBar(percent int) (bar string) {
	bar1 := "##############################"
	bar2 := "------------------------------"
	total := 30
	doneNum := total * percent / 100
	remainNum := total - doneNum
	bar = bar1[:doneNum] + bar2[:remainNum]
	return bar
}

type CosListener struct {
}

func (l *CosListener) ProgressChangedCallback(event *cos.ProgressEvent) {
	switch event.EventType {
	case cos.ProgressStartedEvent:
		fmt.Printf("%3d%% [%s] %d/%d Bytes", 0, drawBar(0), 0, event.TotalBytes)
	case cos.ProgressDataEvent:
		percent := int(event.ConsumedBytes * 100 / event.TotalBytes)
		fmt.Printf("\r%3d%% [%s] %d/%d Bytes", percent, drawBar(percent), event.ConsumedBytes, event.TotalBytes)
	case cos.ProgressCompletedEvent:
		fmt.Printf("\r%3d%% [%s] %d/%d Bytes\n", 100, drawBar(100), event.TotalBytes, event.TotalBytes)
	case cos.ProgressFailedEvent:
		fmt.Printf("\nTransfer Failed!\n")
	default:
		fmt.Printf("Progress Changed Error: unknown progress event type\n")
	}
}
