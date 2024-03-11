package util

import (
	"fmt"
	"time"
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

// 文件上传信息
func PrintTransferProcess(fileNum int, totalSize int64, successNum int, failNum int, totalUploadedBytes int64, startTime time.Time, isFirst bool) {
	if !isFirst {
		// 清空内容
		fmt.Print("\033[1A\033[K\033[1A\033[K")
	}
	totalProgress := float64(totalUploadedBytes) / float64(totalSize) * 100
	speed := float64(totalUploadedBytes) / 1024 / 1024 / time.Since(startTime).Seconds()
	fmt.Printf("Total num: %d, Total size: %s. Dealed num: %d(upload %d files), OK size: %s, Progress: %.3f%%, AvgSpeed: %.2fMB/s\n[%s] %s/%s \n",
		fileNum, formatBytes(totalSize), successNum+failNum, successNum, formatBytes(totalUploadedBytes), totalProgress, speed, drawBar(int(totalProgress)), formatBytes(totalUploadedBytes), formatBytes(totalSize))
}

func CleanTransferProcess() {
	// 清空内容
	fmt.Print("\033[1A\033[K\033[1A\033[K")
}
