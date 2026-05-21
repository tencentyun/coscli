package util

import (
	logger "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"sync"
)

const (
	ErrTypeUpload   string = "upload"
	ErrTypeDownload string = "download"
	ErrTypeList     string = "list"
)

// 开启错误输出
var (
	outputMu sync.Mutex
)

func writeError(errString string, fo *FileOperations) {
	// 使用 outputMu 串行化整个 init+write 过程，避免多个 goroutine 同时进入
	// "Path 为空 -> MkdirAll -> OpenFile" 的初始化阶段产生竞态
	// （之前曾出现："Failed to create error output file: ... no such file or directory"）
	outputMu.Lock()
	defer outputMu.Unlock()

	var err error
	if fo.ErrOutput.Path == "" {
		fo.ErrOutput.Path = filepath.Join(fo.Operation.FailOutputPath, fo.OutPutDirName)
		if _, statErr := os.Stat(fo.ErrOutput.Path); os.IsNotExist(statErr) {
			if mkErr := os.MkdirAll(fo.ErrOutput.Path, 0755); mkErr != nil {
				logger.Errorf("Failed to create error output dir: %v", mkErr)
				return
			}
		}
	}

	if fo.ErrOutput.outputFile == nil {
		// 创建错误日志文件
		failOutputFilePath := filepath.Join(fo.ErrOutput.Path, "error.report")
		fo.ErrOutput.outputFile, err = os.OpenFile(failOutputFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logger.Errorf("Failed to create error error output file:%v", err)
			return
		}
	}

	if _, writeErr := fo.ErrOutput.outputFile.WriteString(errString); writeErr != nil {
		logger.Errorf("Failed to write error output file : %v\n", writeErr)
	}
}

// CloseErrorOutputFile closes the error output file if it is not nil.
// 该函数是幂等的：关闭后将文件指针置为 nil，重复调用是安全的。
func CloseErrorOutputFile(fo *FileOperations) {
	if fo == nil || fo.ErrOutput == nil {
		return
	}
	if fo.ErrOutput.outputFile != nil {
		_ = fo.ErrOutput.outputFile.Close()
		fo.ErrOutput.outputFile = nil
	}
}
