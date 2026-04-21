package util

import (
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CosCopy copies a file from srcClient to destClient using the provided URLs and FileOperations.
// srcClient and destClient are *cos.Client instances.
// srcUrl and destUrl are StorageUrl instances.
// fo is a *FileOperations instance.
func CosCopy(srcClient, destClient *cos.Client, srcUrl, destUrl StorageUrl, fo *FileOperations) error {
	startT := time.Now().UnixNano() / 1000 / 1000

	fo.Monitor.init(fo.CpType)
	chProgressSignal = make(chan chProgressSignalType, 10)
	go progressBar(fo)

	if srcUrl.(*CosUrl).Object != "" && !strings.HasSuffix(srcUrl.(*CosUrl).Object, CosSeparator) {
		// 单对象copy
		index := strings.LastIndex(srcUrl.(*CosUrl).Object, "/")
		prefix := ""
		relativeKey := srcUrl.(*CosUrl).Object
		if index > 0 {
			prefix = srcUrl.(*CosUrl).Object[:index+1]
			relativeKey = srcUrl.(*CosUrl).Object[index+1:]
		}
		// 获取文件信息
		resp, err := GetHead(srcClient, srcUrl.(*CosUrl).Object, fo.Operation.VersionId)
		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				// 源文件不在cos上
				return fmt.Errorf("Object not found : %v", err)
			}
			return fmt.Errorf("Head object err : %v", err)
		}

		// copy文件
		skip, err, isDir, size, msg := singleCopy(srcClient, destClient, fo, objectInfoType{prefix, relativeKey, resp.ContentLength, resp.Header.Get("Last-Modified"), false}, srcUrl, destUrl, fo.Operation.VersionId)

		fo.Monitor.updateMonitor(skip, err, isDir, size)
		if err != nil {
			return fmt.Errorf("%s failed: %v", msg, err)
		}

	} else {
		// 多对象copy
		batchCopyFiles(srcClient, destClient, srcUrl, destUrl, fo)
	}

	// 注意：错误输出文件与进程日志文件由 cmd 层统一关闭（见 cmd/cp.go、cmd/sync.go），
	// util 层不再重复调用 CloseErrorOutputFile / CloseProcessLoggerFile，避免重复 Close。
	closeProgress()
	fmt.Printf(fo.Monitor.progressBar(true, normalExit))

	endT := time.Now().UnixNano() / 1000 / 1000
	PrintTransferStats(startT, endT, fo)

	return nil
}

func batchCopyFiles(srcClient, destClient *cos.Client, srcUrl, destUrl StorageUrl, fo *FileOperations) {
	chObjects := make(chan objectInfoType, ChannelSize)
	chError := make(chan error, fo.Operation.Routines*10)
	chLog := make(chan string, fo.Operation.Routines)
	chListError := make(chan error, 1)

	// 启动进程日志处理协程
	var wgLogger sync.WaitGroup
	wgLogger.Add(1)
	go func() {
		defer wgLogger.Done() // 确保在退出时通知等待组
		for processMsg := range chLog {
			writeProcessLog(processMsg, fo)
		}
	}()

	if fo.BucketType == BucketTypeOfs {
		// 扫描ofs对象大小及数量
		go getOfsObjectList(srcClient, srcUrl, nil, nil, fo, true, false)
		// 获取ofs对象列表
		go getOfsObjectList(srcClient, srcUrl, chObjects, chListError, fo, false, true)
	} else {
		// 扫描cos对象大小及数量
		go getCosObjectList(srcClient, srcUrl, nil, nil, fo, true, false)
		// 获取cos对象列表
		go getCosObjectList(srcClient, srcUrl, chObjects, chListError, fo, false, true)
	}

	for i := 0; i < fo.Operation.Routines; i++ {
		go copyFiles(srcClient, destClient, srcUrl, destUrl, fo, chObjects, chError, chLog)
	}

	completed := 0
	for completed <= fo.Operation.Routines {
		select {
		case err := <-chListError:
			if err != nil {
				if fo.Operation.FailOutput {
					writeError(err.Error(), fo)
				}
			}
			completed++
		case err := <-chError:
			if err == nil {
				completed++
			} else {
				if fo.Operation.FailOutput {
					writeError(err.Error(), fo)
				}
			}
		}
	}

	close(chLog)
	wgLogger.Wait()
}

func copyFiles(srcClient, destClient *cos.Client, srcUrl, destUrl StorageUrl, fo *FileOperations, chObjects <-chan objectInfoType, chError chan<- error, chLog chan<- string) {
	for object := range chObjects {
		var skip, isDir bool
		var err error
		var size int64
		var msg string
		var processMsg string
		var sleepTime time.Duration
		for retry := 0; retry <= fo.Operation.ErrRetryNum; retry++ {
			startT := time.Now().UnixNano() / 1000 / 1000
			skip, err, isDir, size, msg = singleCopy(srcClient, destClient, fo, object, srcUrl, destUrl)
			endT := time.Now().UnixNano() / 1000 / 1000
			costTime := int(endT - startT)
			skipMsg := ""
			if skip {
				skipMsg = "(skip)"
			}
			if retry == 0 {
				if err == nil {
					processMsg += fmt.Sprintf("[%s] %s successed%s,cost %dms\n", time.Now().Format("2006-01-02 15:04:05"), msg, skipMsg, costTime)
				} else {
					processMsg += fmt.Sprintf("[%s] %s failed: %v,cost %dms\n", time.Now().Format("2006-01-02 15:04:05"), msg, err, costTime)
				}
			} else {
				if err == nil {
					processMsg += fmt.Sprintf("[%s] retry[%d] with sleep[%v] %s successed%s,cost %dms\n", time.Now().Format("2006-01-02 15:04:05"), retry, sleepTime.Seconds(), msg, skipMsg, costTime)
				} else {
					processMsg += fmt.Sprintf("[%s] retry[%d] with sleep[%v] %s failed: %v,cost %dms\n", time.Now().Format("2006-01-02 15:04:05"), retry, sleepTime.Seconds(), msg, err, costTime)
				}
			}
			if err == nil {
				break // Copy succeeded, break the loop
			} else {
				// SDK 已对 5xx 错误做过 HTTP 级重试（默认 10 次），
				// 此处应用层不再叠加重试，直接放弃并在日志中标注。
				if isSDKHandledError(err) {
					processMsg += fmt.Sprintf("[%s] %s skip coscli-retry (SDK already retried for 5xx error)\n", time.Now().Format("2006-01-02 15:04:05"), msg)
					break
				}

				if fo.Operation.ErrRetryInterval == 0 {
					// If the retry interval is not specified, retry after a random interval of 1~10 seconds.
					sleepTime = time.Duration(rand.Intn(10)+1) * time.Second
				} else {
					sleepTime = time.Duration(fo.Operation.ErrRetryInterval) * time.Second
				}

				time.Sleep(sleepTime)
			}
		}

		fo.Monitor.updateMonitor(skip, err, isDir, size)
		chLog <- processMsg
		if err != nil {
			chError <- fmt.Errorf("[%s] %s failed: %w\n", time.Now().Format("2006-01-02 15:04:05"), msg, err)
			continue
		}
	}

	chError <- nil
}

// singleCopy todo
func singleCopy(srcClient, destClient *cos.Client, fo *FileOperations, objectInfo objectInfoType, srcUrl, destUrl StorageUrl, VersionId ...string) (skip bool, rErr error, isDir bool, size int64, msg string) {
	skip = false
	rErr = nil
	isDir = false
	size = objectInfo.size
	object := objectInfo.prefix + objectInfo.relativeKey

	destPath := copyPathFixed(objectInfo.relativeKey, destUrl.(*CosUrl).Object)
	msg = fmt.Sprintf("Copy %s to %s", getCosUrl(srcUrl.(*CosUrl).Bucket, object), getCosUrl(destUrl.(*CosUrl).Bucket, destPath))

	var err error
	// 标记文件夹
	if size == 0 && strings.HasSuffix(object, "/") {
		isDir = true
	}

	// 标记跳过的对象直接跳过
	if objectInfo.skip {
		size = objectInfo.size
		skip = true
		return
	}

	// 仅sync命令执行skip
	if fo.Command == CommandSync && !isDir {
		skip, err = skipCopy(srcClient, destClient, object, destPath, fo)
		if err != nil {
			rErr = err
			return
		}
	}

	if skip {
		return
	}

	threadNum := fo.Operation.ThreadNum
	if threadNum == 0 {
		// 若未设置文件分块并发数,需要根据文件大小和分块大小计算默认分块并发数
		threadNum, err = getThreadNumByPartSize(size, fo.Operation.PartSize, fo.Operation.RateLimiting, fo.Operation.MaxThreadNum)
		if err != nil {
			rErr = err
			return
		}
	}

	url, err := GenURL(fo.Config, fo.Param, srcUrl.(*CosUrl).Bucket)

	srcURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, object)

	opt := &cos.MultiCopyOptions{
		OptCopy: &cos.ObjectCopyOptions{
			&cos.ObjectCopyHeaderOptions{
				CacheControl:             fo.Operation.Meta.CacheControl,
				ContentDisposition:       fo.Operation.Meta.ContentDisposition,
				ContentEncoding:          fo.Operation.Meta.ContentEncoding,
				ContentType:              fo.Operation.Meta.ContentType,
				Expires:                  fo.Operation.Meta.Expires,
				ContentLanguage:          fo.Operation.Meta.ContentLanguage,
				XCosStorageClass:         fo.Operation.StorageClass,
				XCosMetaXXX:              fo.Operation.Meta.XCosMetaXXX,
				XCosServerSideEncryption: fo.Operation.ServerSideEncryption,
				XCosSSECustomerAglo:      fo.Operation.SSECustomerAlgo,
				XCosSSECustomerKey:       fo.Operation.SSECustomerKey,
				XCosSSECustomerKeyMD5:    fo.Operation.SSECustomerKeyMD5,
				XOptionHeader:            &http.Header{},
			},
			&cos.ACLHeaderOptions{
				XCosACL:       fo.Operation.Acl,
				XCosGrantRead: fo.Operation.GrantRead,
				//XCosGrantWrite:       fo.Operation.GrantWrite,
				XCosGrantFullControl: fo.Operation.GrantFullControl,
				XCosGrantReadACP:     fo.Operation.GrantReadAcp,
				XCosGrantWriteACP:    fo.Operation.GrantWriteAcp,
			},
		},
		PartSize:       fo.Operation.PartSize,
		ThreadPoolSize: threadNum,
	}

	if fo.Operation.Tags != "" {
		opt.OptCopy.XOptionHeader.Add("x-cos-tagging", fo.Operation.Tags)
	}
	if fo.Operation.ForbidOverWrite {
		opt.OptCopy.XOptionHeader.Add("x-cos-forbid-overwrite", "true")
	}

	if fo.Operation.Meta.CacheControl != "" || fo.Operation.Meta.ContentDisposition != "" || fo.Operation.Meta.ContentEncoding != "" ||
		fo.Operation.Meta.ContentType != "" || fo.Operation.Meta.Expires != "" || fo.Operation.Meta.MetaChange {
	}
	{
		opt.OptCopy.ObjectCopyHeaderOptions.XCosMetadataDirective = "Replaced"
	}

	if fo.BucketType == BucketTypeOfs {
		_, _, err = destClient.Object.MultiCopy(context.Background(), destPath, srcURL, opt)
	} else {
		_, _, err = destClient.Object.MultiCopy(context.Background(), destPath, srcURL, opt, VersionId...)
	}

	if err != nil {
		rErr = err
		return
	}

	if fo.Operation.Move {
		if err == nil {
			_, err = srcClient.Object.Delete(context.Background(), object, nil)
			rErr = err
			return
		}
	}

	return
}

// CosCopyWithDelete copies files from source to destination with delete option.
// It takes srcClient and destClient as COS clients, srcKeys and copyKeys as maps of source and destination keys,
// srcUrl and destUrl as storage URLs, and fo as a FileOperations object.
func CosCopyWithDelete(srcClient, destClient *cos.Client, srcKeys, copyKeys map[string]commonInfoType, srcUrl, destUrl StorageUrl, fo *FileOperations) error {
	startT := time.Now().UnixNano() / 1000 / 1000

	fo.Monitor.init(fo.CpType)
	chProgressSignal = make(chan chProgressSignalType, 10)
	go progressBar(fo)

	// 多对象copy
	batchCopyFilesWithDelete(srcClient, destClient, srcKeys, copyKeys, srcUrl, destUrl, fo)

	// 注意：错误输出文件与进程日志文件由 cmd 层统一关闭（见 cmd/cp.go、cmd/sync.go），
	// util 层不再重复调用 CloseErrorOutputFile / CloseProcessLoggerFile，避免重复 Close。
	closeProgress()
	fmt.Printf(fo.Monitor.progressBar(true, normalExit))

	endT := time.Now().UnixNano() / 1000 / 1000
	PrintTransferStats(startT, endT, fo)

	return nil
}

// batchCopyFilesWithDelete todo
func batchCopyFilesWithDelete(srcClient, destClient *cos.Client, srcKeys, copyKeys map[string]commonInfoType, srcUrl, destUrl StorageUrl, fo *FileOperations) {
	chObjects := make(chan objectInfoType, ChannelSize)
	chError := make(chan error, fo.Operation.Routines*10)
	chLog := make(chan string, fo.Operation.Routines)
	chListError := make(chan error, 1)

	// 启动进程日志处理协程
	var wgLogger sync.WaitGroup
	wgLogger.Add(1)
	go func() {
		defer wgLogger.Done() // 确保在退出时通知等待组
		for processMsg := range chLog {
			writeProcessLog(processMsg, fo)
		}
	}()

	// 根据获取的列表统计对象大小数量并生成copy对象列表
	go getObjectListByKeys(srcKeys, copyKeys, chObjects, chListError, fo)

	for i := 0; i < fo.Operation.Routines; i++ {
		go copyFiles(srcClient, destClient, srcUrl, destUrl, fo, chObjects, chError, chLog)
	}

	completed := 0
	for completed <= fo.Operation.Routines {
		select {
		case err := <-chListError:
			if err != nil {
				if fo.Operation.FailOutput {
					writeError(err.Error(), fo)
				}
			}
			completed++
		case err := <-chError:
			if err == nil {
				completed++
			} else {
				if fo.Operation.FailOutput {
					writeError(err.Error(), fo)
				}
			}
		}
	}

	close(chLog)
	wgLogger.Wait()
}
