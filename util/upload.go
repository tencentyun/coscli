package util

import (
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	mu sync.Mutex
)

func Upload(c *cos.Client, fileUrl StorageUrl, cosUrl StorageUrl, fo *FileOperations) {
	startT := time.Now().UnixNano() / 1000 / 1000
	localPath := fileUrl.ToString()

	fo.Monitor.init(fo.CpType)
	chProgressSignal = make(chan chProgressSignalType, 10)
	go progressBar(fo)

	chFiles := make(chan fileInfoType, ChannelSize)
	chError := make(chan error, fo.Operation.Routines)
	chListError := make(chan error, 1)
	// 统计文件数量及大小数据
	go fileStatistic(localPath, fo)
	// 生成文件列表
	go generateFileList(localPath, chFiles, chListError, fo)

	for i := 0; i < fo.Operation.Routines; i++ {
		go uploadFiles(c, cosUrl, fo, chFiles, chError)
	}

	completed := 0
	for completed <= fo.Operation.Routines {
		select {
		case err := <-chListError:
			if err != nil {
				if fo.Operation.FailOutput {
					writeError(ErrTypeList, err.Error(), fo)
				}
			}
			completed++
		case err := <-chError:
			if err == nil {
				completed++
			} else {
				if fo.Operation.FailOutput {
					writeError(ErrTypeUpload, err.Error(), fo)
				}
			}
		}
	}

	closeProgress()
	fmt.Printf(fo.Monitor.progressBar(true, normalExit))

	endT := time.Now().UnixNano() / 1000 / 1000
	PrintTransferStats(startT, endT, fo)
}

func uploadFiles(c *cos.Client, cosUrl StorageUrl, fo *FileOperations, chFiles <-chan fileInfoType, chError chan<- error) {
	for file := range chFiles {
		skip, err, isDir, size, msg := SingleUpload(c, fo, file, cosUrl)
		fo.Monitor.updateMonitor(skip, err, isDir, size)
		if err != nil {
			chError <- fmt.Errorf("%s failed: %w", msg, err)
			continue
		}
	}

	chError <- nil
}

func SingleUpload(c *cos.Client, fo *FileOperations, file fileInfoType, cosUrl StorageUrl) (skip bool, rErr error, isDir bool, size int64, msg string) {
	skip = false
	rErr = nil
	isDir = false
	size = 0

	localFilePath, cosPath := UploadPathFixed(file, cosUrl.(*CosUrl).Object)

	fileInfo, err := os.Stat(localFilePath)
	if err != nil {
		rErr = err
		return
	}

	var snapshotKey string

	msg = fmt.Sprintf("Upload %s to %s", localFilePath, getCosUrl(cosUrl.(*CosUrl).Bucket, cosPath))
	if fileInfo.IsDir() {
		isDir = true
		// 在cos创建文件夹
		_, err = c.Object.Put(context.Background(), cosPath, strings.NewReader(""), nil)
		if err != nil {
			rErr = err
			return
		}
	} else {
		size = fileInfo.Size()

		// 仅sync命令执行skip
		if fo.Command == CommandSync {
			absLocalFilePath, _ := filepath.Abs(localFilePath)
			snapshotKey = getUploadSnapshotKey(absLocalFilePath, cosUrl.(*CosUrl).Bucket, cosUrl.(*CosUrl).Object)
			skip, err = skipUpload(snapshotKey, c, fo, fileInfo.ModTime().Unix(), cosPath, localFilePath)
			if err != nil {
				rErr = err
				return
			}
		}

		if skip {
			return
		}

		// 未跳过则通过监听更新size
		size = 0

		opt := &cos.MultiUploadOptions{
			OptIni: &cos.InitiateMultipartUploadOptions{
				ACLHeaderOptions: &cos.ACLHeaderOptions{
					XCosACL:              "",
					XCosGrantRead:        "",
					XCosGrantWrite:       "",
					XCosGrantFullControl: "",
					XCosGrantReadACP:     "",
					XCosGrantWriteACP:    "",
				},
				ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
					CacheControl:             fo.Operation.Meta.CacheControl,
					ContentDisposition:       fo.Operation.Meta.ContentDisposition,
					ContentEncoding:          fo.Operation.Meta.ContentEncoding,
					ContentType:              fo.Operation.Meta.ContentType,
					ContentMD5:               fo.Operation.Meta.ContentMD5,
					ContentLength:            fo.Operation.Meta.ContentLength,
					ContentLanguage:          fo.Operation.Meta.ContentLanguage,
					Expect:                   "",
					Expires:                  fo.Operation.Meta.Expires,
					XCosContentSHA1:          "",
					XCosMetaXXX:              fo.Operation.Meta.XCosMetaXXX,
					XCosStorageClass:         fo.Operation.StorageClass,
					XCosServerSideEncryption: "",
					XCosSSECustomerAglo:      "",
					XCosSSECustomerKey:       "",
					XCosSSECustomerKeyMD5:    "",
					XOptionHeader:            nil,
					XCosTrafficLimit:         (int)(fo.Operation.RateLimiting * 1024 * 1024 * 8),
					Listener:                 &CosListener{fo},
				},
			},
			PartSize:       fo.Operation.PartSize,
			ThreadPoolSize: fo.Operation.ThreadNum,
			CheckPoint:     true,
		}
		_, _, err = c.Object.Upload(context.Background(), cosPath, localFilePath, opt)
		if err != nil {
			rErr = err
			return
		}
	}

	if snapshotKey != "" && fo.Operation.SnapshotPath != "" {
		// 上传成功后添加快照
		fo.SnapshotDb.Put([]byte(snapshotKey), []byte(strconv.FormatInt(fileInfo.ModTime().Unix(), 10)), nil)
	}

	return
}
