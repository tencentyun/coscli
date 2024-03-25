package util

import (
	logger "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	ErrTypeUpload string = "upload"
	ErrTypeList   string = "list"
)

// 开启错误输出
var (
	outputMu sync.Mutex
)

func writeError(errorType, errString string, fo *FileOperations) {
	var err error
	if fo.ErrOutput.Path == "" {
		fo.ErrOutput.Path = filepath.Join(fo.Operation.FailOutputPath, time.Now().Format("20060102_150405"))
		_, err := os.Stat(fo.ErrOutput.Path)
		if os.IsNotExist(err) {
			err := os.MkdirAll(fo.ErrOutput.Path, 0755)
			if err != nil {
				logger.Fatalf("Failed to create error output dir: %v", err)
			}
		}
	}

	if errorType == ErrTypeList && fo.ErrOutput.ListOutput == nil {
		// 创建错误日志文件
		listFailOutputFilePath := filepath.Join(fo.ErrOutput.Path, "list_err.report")
		fo.ErrOutput.ListOutput, err = os.OpenFile(listFailOutputFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logger.Fatal("Failed to create error list output file:", err)
		}
		defer fo.ErrOutput.ListOutput.Close()
	}

	if errorType == ErrTypeUpload && fo.ErrOutput.UploadOutput == nil {
		uploadFailOutputFilePath := filepath.Join(fo.ErrOutput.Path, "upload_err.report")
		fo.ErrOutput.UploadOutput, err = os.OpenFile(uploadFailOutputFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logger.Fatal("Failed to create error upload output file:", err)
		}
		defer fo.ErrOutput.UploadOutput.Close()
	}

	outputMu.Lock()
	var writeErr error
	if errorType == ErrTypeList {
		_, writeErr = fo.ErrOutput.ListOutput.WriteString(errString)
	} else {
		_, writeErr = fo.ErrOutput.UploadOutput.WriteString(errString)
	}

	if writeErr != nil {
		logger.Printf("Failed to write error output file : %v\n", writeErr)
	}
	outputMu.Unlock()
}
