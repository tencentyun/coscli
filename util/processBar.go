package util

import (
	"fmt"
)

// 文件上传进度
func drawBar(percent int) (bar string) {
	bar1 := "##############################"
	bar2 := "------------------------------"
	total := 30
	doneNum := total * percent / 100
	remainNum := total - doneNum
	bar = bar1[:doneNum] + bar2[:remainNum]
	return bar
}

func freshProgress() {
	if len(chProgressSignal) <= signalNum {
		chProgressSignal <- chProgressSignalType{false, normalExit}
	}
}

func progressBar(fo *FileOperations) {
	for signal := range chProgressSignal {
		fmt.Printf(fo.Monitor.progressBar(signal.finish, signal.exitStat))
	}
}

func closeProgress() {
	signalNum = -1
}
