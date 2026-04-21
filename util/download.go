package util

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// DownloadOptions is a struct used to configure download options.
type DownloadOptions struct {
	RateLimiting float32
	PartSize     int64
	ThreadNum    int
	SnapshotDb   *leveldb.DB
	SnapshotPath string
}

// Download downloads a file from the specified COS URL to the specified file URL.
//
// c: *cos.Client - the COS client to use for downloading.
// cosUrl: StorageUrl - the COS URL of the file to download.
// fileUrl: StorageUrl - the URL where the file should be saved.
// fo: *FileOperations - the operations object to handle file operations.
//
// Returns an error if the download fails.
func Download(c *cos.Client, cosUrl StorageUrl, fileUrl StorageUrl, fo *FileOperations) error {

	startT := time.Now().UnixNano() / 1000 / 1000

	fo.Monitor.init(fo.CpType)
	chProgressSignal = make(chan chProgressSignalType, 10)
	go progressBar(fo)

	if cosUrl.(*CosUrl).Object != "" && !strings.HasSuffix(cosUrl.(*CosUrl).Object, CosSeparator) {
		// 单对象下载
		index := strings.LastIndex(cosUrl.(*CosUrl).Object, "/")
		prefix := ""
		relativeKey := cosUrl.(*CosUrl).Object
		if index > 0 {
			prefix = cosUrl.(*CosUrl).Object[:index+1]
			relativeKey = cosUrl.(*CosUrl).Object[index+1:]
		}
		// 获取文件信息
		resp, err := GetHead(c, cosUrl.(*CosUrl).Object, fo.Operation.VersionId)
		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				// 文件不在cos上
				return fmt.Errorf("Object not found : %v", err)
			}
			return fmt.Errorf("Head object err : %v", err)
		}

		fo.Monitor.updateScanSizeNum(resp.ContentLength, 1)
		fo.Monitor.setScanEnd()
		freshProgress()

		// 下载文件
		skip, err, isDir, size, _, msg := singleDownload(c, fo, objectInfoType{prefix, relativeKey, resp.ContentLength, resp.Header.Get("Last-Modified"), false}, cosUrl, fileUrl, fo.Operation.VersionId)
		fo.Monitor.updateMonitor(skip, err, isDir, size)
		if err != nil {
			return fmt.Errorf("%s failed: %v", msg, err)
		}
	} else {
		// 多对象下载
		batchDownloadFiles(c, cosUrl, fileUrl, fo)
	}

	closeProgress()
	fmt.Printf(fo.Monitor.progressBar(true, normalExit))

	endT := time.Now().UnixNano() / 1000 / 1000
	PrintTransferStats(startT, endT, fo)

	return nil
}

func batchDownloadFiles(c *cos.Client, cosUrl StorageUrl, fileUrl StorageUrl, fo *FileOperations) {
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
		go getOfsObjectList(c, cosUrl, nil, nil, fo, true, false)
		// 获取ofs对象列表
		go getOfsObjectList(c, cosUrl, chObjects, chListError, fo, false, true)
	} else {
		// 扫描cos对象大小及数量
		go getCosObjectList(c, cosUrl, nil, nil, fo, true, false)
		// 获取cos对象列表
		go getCosObjectList(c, cosUrl, chObjects, chListError, fo, false, true)
	}

	for i := 0; i < fo.Operation.Routines; i++ {
		go downloadFiles(c, cosUrl, fileUrl, fo, chObjects, chError, chLog)
	}

	completed := 0
	for completed <= fo.Operation.Routines {
		select {
		case err := <-chListError:
			if err != nil {
				if fo.Operation.FailOutput {
					fo.Monitor.updateListErr(1)
					writeError(fmt.Sprintf("[%s] ListObjects error:%s\n", time.Now().Format("2006-01-02 15:04:05"), err.Error()), fo)
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

func downloadFiles(c *cos.Client, cosUrl, fileUrl StorageUrl, fo *FileOperations, chObjects <-chan objectInfoType, chError chan<- error, chLog chan<- string) {
	for object := range chObjects {
		var skip, isDir bool
		var err error
		var size, transferSize int64
		var msg string
		var processMsg string
		var sleepTime time.Duration
		for retry := 0; retry <= fo.Operation.ErrRetryNum; retry++ {
			startT := time.Now().UnixNano() / 1000 / 1000
			skip, err, isDir, size, transferSize, msg = singleDownload(c, fo, object, cosUrl, fileUrl)
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
				break // Download succeeded, break the loop
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

				fo.Monitor.updateDealSize(-transferSize)
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

func singleDownload(c *cos.Client, fo *FileOperations, objectInfo objectInfoType, cosUrl, fileUrl StorageUrl, VersionId ...string) (skip bool, rErr error, isDir bool, size, transferSize int64, msg string) {
	skip = false
	rErr = nil
	isDir = false
	size = objectInfo.size
	transferSize = 0
	object := objectInfo.prefix + objectInfo.relativeKey

	localFilePath := DownloadPathFixed(objectInfo.relativeKey, fileUrl.ToString())
	msg = fmt.Sprintf("Download %s to %s", getCosUrl(cosUrl.(*CosUrl).Bucket, object), localFilePath)

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

	// 是文件夹则直接创建并退出
	if isDir {
		rErr = os.MkdirAll(localFilePath, 0755)
		return
	}

	// 跳过空文件
	if fo.Operation.IgnoreEmptyFile && size == 0 {
		size = objectInfo.size
		skip = true
		return
	}

	var snapshotKey string
	if fo.Command == CommandSync {
		absLocalFilePath, _ := filepath.Abs(localFilePath)
		snapshotKey = getDownloadSnapshotKey(absLocalFilePath, cosUrl.(*CosUrl).Bucket, cosUrl.(*CosUrl).Object)
	}

	_, err := os.Stat(localFilePath)

	if err == nil {
		// 文件存在再判断是否需要跳过
		// 仅sync命令执行skip
		if fo.Command == CommandSync {
			var skipType string
			if fo.Operation.IgnoreExisting {
				skip = true
				skipType = SyncTypeIgnoreExisting
			} else {
				skip, skipType, err = skipDownload(snapshotKey, c, fo, localFilePath, objectInfo.lastModified, object)
				if err != nil {
					rErr = err
				}
			}

			if skip {

				if snapshotKey != "" && fo.Operation.SnapshotPath != "" && (skipType == SyncTypeUpdate || skipType == SyncTypeIgnoreExisting || skipType == SyncTypeCrc64) {
					lastModified := objectInfo.lastModified
					if lastModified == "" {
						return
					}

					// 解析时间字符串
					objectModifiedTime, err := time.Parse(time.RFC3339, lastModified)
					if err != nil {
						objectModifiedTime, err = time.Parse(time.RFC1123, lastModified)
						if err != nil {
							rErr = err
							return
						}

					}

					fo.SnapshotDb.Put([]byte(snapshotKey), []byte(strconv.FormatInt(objectModifiedTime.Unix(), 10)), nil)
				}
				return
			}

		}
	}

	// 不是文件夹则创建父目录
	err = createParentDirectory(localFilePath)
	if err != nil {
		rErr = err
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

	// 开始下载文件
	opt := &cos.MultiDownloadOptions{
		Opt: &cos.ObjectGetOptions{
			ResponseContentType:        "",
			ResponseContentLanguage:    "",
			ResponseExpires:            "",
			ResponseCacheControl:       "",
			ResponseContentDisposition: "",
			ResponseContentEncoding:    "",
			Range:                      "",
			IfModifiedSince:            "",
			XCosSSECustomerAglo:        "",
			XCosSSECustomerKey:         "",
			XCosSSECustomerKeyMD5:      "",
			XOptionHeader:              nil,
			XCosTrafficLimit:           (int)(fo.Operation.RateLimiting * 1024 * 1024 * 8),
		},
		PartSize:        fo.Operation.PartSize,
		ThreadPoolSize:  threadNum,
		CheckPoint:      fo.Operation.CheckPoint,
		CheckPointFile:  "",
		DisableChecksum: fo.Operation.DisableChecksum,
	}
	isPart := true
	if fo.Operation.PartSize > size {
		isPart = false
	}
	counter := &Counter{TransferSize: 0}
	// 未跳过则通过监听更新size(仅需要分块文件的通过sdk监听进度)
	opt.Opt.Listener = &CosListener{fo, counter}
	size = 0

	var resp *cos.Response

	if fo.BucketType == BucketTypeOfs {
		resp, err = c.Object.Download(context.Background(), object, localFilePath, opt)
	} else {
		resp, err = c.Object.Download(context.Background(), object, localFilePath, opt, VersionId...)
	}

	if err != nil {
		if (!isPart) || (isPart && !fo.Operation.CheckPoint) {
			transferSize = counter.TransferSize
		}
		rErr = err
		return
	}

	// 下载完成记录快照信息
	if snapshotKey != "" && fo.Operation.SnapshotPath != "" && fo.Command == CommandSync {
		lastModified := resp.Header.Get("Last-Modified")
		if lastModified == "" {
			return
		}

		// 解析时间字符串
		objectModifiedTime, err := time.Parse(time.RFC3339, lastModified)
		if err != nil {
			objectModifiedTime, err = time.Parse(time.RFC1123, lastModified)
			if err != nil {
				rErr = err
				return
			}

		}

		fo.SnapshotDb.Put([]byte(snapshotKey), []byte(strconv.FormatInt(objectModifiedTime.Unix(), 10)), nil)
	}

	return
}

// DownloadWithDelete todo
func DownloadWithDelete(c *cos.Client, srcKeys, downloadKeys map[string]commonInfoType, cosUrl StorageUrl, fileUrl StorageUrl, fo *FileOperations) error {
	startT := time.Now().UnixNano() / 1000 / 1000

	fo.Monitor.init(fo.CpType)
	chProgressSignal = make(chan chProgressSignalType, 10)
	go progressBar(fo)

	// 多对象下载
	batchDownloadFilesWithDelete(c, srcKeys, downloadKeys, cosUrl, fileUrl, fo)

	closeProgress()
	fmt.Printf(fo.Monitor.progressBar(true, normalExit))

	endT := time.Now().UnixNano() / 1000 / 1000
	PrintTransferStats(startT, endT, fo)

	return nil
}

func batchDownloadFilesWithDelete(c *cos.Client, srcKeys, downloadKeys map[string]commonInfoType, cosUrl StorageUrl, fileUrl StorageUrl, fo *FileOperations) {
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

	// 生成下载key
	go getObjectListByKeys(srcKeys, downloadKeys, chObjects, chListError, fo)

	for i := 0; i < fo.Operation.Routines; i++ {
		go downloadFiles(c, cosUrl, fileUrl, fo, chObjects, chError, chLog)
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
