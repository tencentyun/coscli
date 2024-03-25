package util

import (
	"fmt"
	"path/filepath"
)

func PrintCpStats(startT, endT int64, cc *CopyCommand) {
	if cc.Monitor.errNum > 0 && cc.CpParams.FailOutput {
		absErrOutputPath, _ := filepath.Abs(cc.ErrOutput.Path)
		fmt.Printf("Some file upload failed, please check the detailed information in dir %s.\n", absErrOutputPath)
	}

	// 计算上传速度
	if endT-startT > 0 {
		averSpeed := (float64(cc.Monitor.TransferSize) / float64(endT-startT)) * 1000
		formattedSpeed := formatBytes(averSpeed)
		fmt.Printf("\nAvgSpeed: %s/s\n", formattedSpeed)
	}

	// 计算并输出花费时间
	elapsedTime := float64(endT-startT) / 1000
	fmt.Printf("\ncost %.6f(s)\n", elapsedTime)

}
