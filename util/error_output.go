package util

import (
	logger "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"sync"
	"time"
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
	var err error
	if fo.ErrOutput.Path == "" {
		fo.ErrOutput.Path = filepath.Join(fo.Operation.FailOutputPath, time.Now().Format("20060102_150405"))
		_, err := os.Stat(fo.ErrOutput.Path)
		if os.IsNotExist(err) {
			err := os.MkdirAll(fo.ErrOutput.Path, 0755)
			if err != nil {
				logger.Errorf("Failed to create error output dir: %v", err)
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

	outputMu.Lock()
	_, writeErr := fo.ErrOutput.outputFile.WriteString(errString)

	if writeErr != nil {
		logger.Errorf("Failed to write error output file : %v\n", writeErr)
	}
	outputMu.Unlock()
}

func CloseErrorOutputFile(fo *FileOperations) {
	if fo.ErrOutput.outputFile != nil {
		defer fo.ErrOutput.outputFile.Close()
	}
}
