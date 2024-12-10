package util

import (
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"math/rand"
	"strings"
	"time"
)

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
		resp, err := getHead(srcClient, srcUrl.(*CosUrl).Object, fo.Operation.VersionId)
		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				// 源文件不在cos上
				return fmt.Errorf("Object not found : %v", err)
			}
			return fmt.Errorf("Head object err : %v", err)
		}

		// copy文件
		skip, err, isDir, size, msg := singleCopy(srcClient, destClient, fo, objectInfoType{prefix, relativeKey, resp.ContentLength, resp.Header.Get("Last-Modified")}, srcUrl, destUrl, fo.Operation.VersionId)

		fo.Monitor.updateMonitor(skip, err, isDir, size)
		if err != nil {
			return fmt.Errorf("%s failed: %v", msg, err)
		}

	} else {
		// 多对象copy
		batchCopyFiles(srcClient, destClient, srcUrl, destUrl, fo)
	}

	CloseErrorOutputFile(fo)
	closeProgress()
	fmt.Printf(fo.Monitor.progressBar(true, normalExit))

	endT := time.Now().UnixNano() / 1000 / 1000
	PrintTransferStats(startT, endT, fo)

	return nil
}

func batchCopyFiles(srcClient, destClient *cos.Client, srcUrl, destUrl StorageUrl, fo *FileOperations) {
	chObjects := make(chan objectInfoType, ChannelSize)
	chError := make(chan error, fo.Operation.Routines)
	chListError := make(chan error, 1)

	if fo.BucketType == "OFS" {
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
		go copyFiles(srcClient, destClient, srcUrl, destUrl, fo, chObjects, chError)
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
}

func copyFiles(srcClient, destClient *cos.Client, srcUrl, destUrl StorageUrl, fo *FileOperations, chObjects <-chan objectInfoType, chError chan<- error) {
	for object := range chObjects {
		var skip, isDir bool
		var err error
		var size int64
		var msg string
		for retry := 0; retry <= fo.Operation.ErrRetryNum; retry++ {
			skip, err, isDir, size, msg = singleCopy(srcClient, destClient, fo, object, srcUrl, destUrl)
			if err == nil {
				break // Copy succeeded, break the loop
			} else {
				if retry < fo.Operation.ErrRetryNum {
					if fo.Operation.ErrRetryInterval == 0 {
						// If the retry interval is not specified, retry after a random interval of 1~10 seconds.
						time.Sleep(time.Duration(rand.Intn(10)+1) * time.Second)
					} else {
						time.Sleep(time.Duration(fo.Operation.ErrRetryInterval) * time.Second)
					}
				}
			}
		}

		fo.Monitor.updateMonitor(skip, err, isDir, size)
		if err != nil {
			chError <- fmt.Errorf("%s failed: %w", msg, err)
			continue
		}
	}

	chError <- nil
}

func singleCopy(srcClient, destClient *cos.Client, fo *FileOperations, objectInfo objectInfoType, srcUrl, destUrl StorageUrl, VersionId ...string) (skip bool, rErr error, isDir bool, size int64, msg string) {
	skip = false
	rErr = nil
	isDir = false
	size = objectInfo.size
	object := objectInfo.prefix + objectInfo.relativeKey

	destPath := copyPathFixed(objectInfo.relativeKey, destUrl.(*CosUrl).Object)
	msg = fmt.Sprintf("\nCopy %s to %s", getCosUrl(srcUrl.(*CosUrl).Bucket, object), getCosUrl(destUrl.(*CosUrl).Bucket, destPath))

	var err error
	// 是文件夹则直接创建并退出
	if size == 0 && strings.HasSuffix(object, "/") {
		isDir = true
	}

	// 仅sync命令执行skip
	if fo.Command == CommandSync && !isDir {
		skip, err = skipCopy(srcClient, destClient, object, destPath)
		if err != nil {
			rErr = err
			return
		}
	}

	if skip {
		return
	}

	// copy暂不支持监听进度
	// size = 0

	url, err := GenURL(fo.Config, fo.Param, srcUrl.(*CosUrl).Bucket)

	srcURL := fmt.Sprintf("%s/%s", url.BucketURL.Host, object)

	opt := &cos.MultiCopyOptions{
		OptCopy: &cos.ObjectCopyOptions{
			&cos.ObjectCopyHeaderOptions{
				CacheControl:       fo.Operation.Meta.CacheControl,
				ContentDisposition: fo.Operation.Meta.ContentDisposition,
				ContentEncoding:    fo.Operation.Meta.ContentEncoding,
				ContentType:        fo.Operation.Meta.ContentType,
				Expires:            fo.Operation.Meta.Expires,
				XCosStorageClass:   fo.Operation.StorageClass,
				XCosMetaXXX:        fo.Operation.Meta.XCosMetaXXX,
			},
			nil,
		},
		PartSize:       fo.Operation.PartSize,
		ThreadPoolSize: fo.Operation.ThreadNum,
	}
	if fo.Operation.Meta.CacheControl != "" || fo.Operation.Meta.ContentDisposition != "" || fo.Operation.Meta.ContentEncoding != "" ||
		fo.Operation.Meta.ContentType != "" || fo.Operation.Meta.Expires != "" || fo.Operation.Meta.MetaChange {
	}
	{
		opt.OptCopy.ObjectCopyHeaderOptions.XCosMetadataDirective = "Replaced"
	}

	_, _, err = destClient.Object.MultiCopy(context.Background(), destPath, srcURL, opt, VersionId...)

	if err != nil {
		rErr = err
		return
	}

	return
}
