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

func writeError(errorType, errString string, cc *CopyCommand) {
	var err error
	if cc.ErrOutput.Path == "" {
		cc.ErrOutput.Path = filepath.Join(cc.CpParams.FailOutputPath, time.Now().Format("20060102_150405"))
		_, err := os.Stat(cc.ErrOutput.Path)
		if os.IsNotExist(err) {
			err := os.MkdirAll(cc.ErrOutput.Path, 0755)
			if err != nil {
				logger.Fatalf("Failed to create error output dir: %v", err)
			}
		}
	}

	if errorType == ErrTypeList && cc.ErrOutput.ListOutput == nil {
		// 创建错误日志文件
		listFailOutputFilePath := filepath.Join(cc.ErrOutput.Path, "list_err.report")
		cc.ErrOutput.ListOutput, err = os.OpenFile(listFailOutputFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logger.Fatal("Failed to create error list output file:", err)
		}
		defer cc.ErrOutput.ListOutput.Close()
	}

	if errorType == ErrTypeUpload && cc.ErrOutput.UploadOutput == nil {
		uploadFailOutputFilePath := filepath.Join(cc.ErrOutput.Path, "upload_err.report")
		cc.ErrOutput.UploadOutput, err = os.OpenFile(uploadFailOutputFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logger.Fatal("Failed to create error upload output file:", err)
		}
		defer cc.ErrOutput.UploadOutput.Close()
	}

	outputMu.Lock()
	var writeErr error
	if errorType == ErrTypeList {
		_, writeErr = cc.ErrOutput.ListOutput.WriteString(errString)
	} else {
		_, writeErr = cc.ErrOutput.UploadOutput.WriteString(errString)
	}

	if writeErr != nil {
		logger.Printf("Failed to write error output file : %v\n", writeErr)
	}
	outputMu.Unlock()
}
