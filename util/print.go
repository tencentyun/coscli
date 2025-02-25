package util

import (
	"fmt"
	"path/filepath"
)

func PrintTransferStats(startT, endT int64, fo *FileOperations) {
	if fo.Monitor.ErrNum > 0 && fo.Operation.FailOutput {
		absErrOutputPath, _ := filepath.Abs(fo.ErrOutput.Path)
		fmt.Printf("Some file upload failed, please check the detailed information in dir %s.\n", absErrOutputPath)
	}

	// 计算上传速度
	if endT-startT > 0 {
		averSpeed := (float64(fo.Monitor.TransferSize) / float64(endT-startT)) * 1000
		formattedSpeed := formatBytes(averSpeed)
		fmt.Printf("\nAvgSpeed: %s/s\n", formattedSpeed)
	}
}

func PrintCostTime(startT, endT int64) {
	// 计算并输出花费时间
	elapsedTime := float64(endT-startT) / 1000
	fmt.Printf("\ncost %.6f(s)\n", elapsedTime)
}
