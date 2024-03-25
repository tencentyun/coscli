package util

import (
	"context"
	"fmt"
	logger "github.com/sirupsen/logrus"
	leveldb "github.com/syndtr/goleveldb/leveldb"
	"github.com/tencentyun/cos-go-sdk-v5"
	"os"
	"strings"
	"sync"
)

type ProgressInfo struct {
	Info       string
	Percent    int
	Consumed   int64
	Total      int64
	IsFinished bool
	Index      int
}

var (
	mu sync.Mutex
)

type UploadOptions struct {
	StorageClass string
	RateLimiting float32
	PartSize     int64
	ThreadNum    int
	Meta         Meta
	SnapshotDb   *leveldb.DB
	SnapshotPath string
}

type Message struct {
	message string
	args    []interface{}
}

func Upload(fileUrl StorageUrl, cosUrl StorageUrl, cc *CopyCommand, cpType CpType) {
	localPath := fileUrl.ToString()
	bucketName := cosUrl.(CosUrl).Bucket
	cosPath := cosUrl.(CosUrl).Object

	c := NewClient(cc.Config, cc.Param, bucketName)

	if localPath == "" {
		logger.Fatalln("localPath is empty")
	}

	// 格式化本地路径
	localPath = strings.TrimPrefix(localPath, "./")
	// 获取本地文件/文件夹信息
	localPathInfo, err := os.Stat(localPath)
	if err != nil {
		logger.Fatalln(err)
	}

	if localPathInfo.IsDir() && !cc.CpParams.Recursive {
		logger.Fatalf("localPath:%v is dir, please use --recursive option", localPath)
	}

	// 格式化路径
	if cc.CpParams.Recursive {
		cosPath, localPath = formatPath(cosPath, localPath)
	}

	cc.Monitor.init(cpType)
	chProgressSignal = make(chan chProgressSignalType, 10)
	go progressBar(cc)

	chFiles := make(chan fileInfoType, ChannelSize)
	chError := make(chan error, cc.CpParams.Routines)
	chListError := make(chan error, 1)
	// 统计文件数量及大小数据
	go fileStatistic(localPath, cc)
	// 生成文件列表
	go generateFileList(localPath, chFiles, chListError, cc)

	for i := 0; i < cc.CpParams.Routines; i++ {
		go uploadFiles(c, cosPath, cc, chFiles, chError)
	}

	completed := 0
	for completed <= cc.CpParams.Routines {
		select {
		case err := <-chListError:
			if err != nil {
				if cc.CpParams.FailOutput {
					writeError(ErrTypeList, err.Error(), cc)
				}
			}
			completed++
		case err := <-chError:
			if err == nil {
				completed++
			} else {
				if cc.CpParams.FailOutput {
					writeError(ErrTypeUpload, err.Error(), cc)
				}
			}
		}
	}

	closeProgress()
	fmt.Printf(cc.Monitor.progressBar(true, normalExit))
}

func uploadFiles(c *cos.Client, cosPath string, cc *CopyCommand, chFiles <-chan fileInfoType, chError chan<- error) {
	for file := range chFiles {
		if filterFile(file, cc.CpParams.CheckpointDir) {
			skip, err, isDir, size, msg := SingleUpload(c, cc, file, cosPath)
			cc.Monitor.updateMonitor(skip, err, isDir, size)
			if err != nil {
				chError <- fmt.Errorf("%s failed: %w", msg, err)
				continue
			}
		}
	}

	chError <- nil

	//files := GetLocalFilesListRecursive(localDir, include, exclude)
	//
	//var wg sync.WaitGroup
	//totalSize := int64(0)
	//for _, f := range files {
	//	path := filepath.Join(localDir, f)
	//	file, _ := os.Stat(path)
	//	totalSize += file.Size()
	//}
	//// 获取文件总数
	//fileNum := len(files)
	//failNum := 0
	//successNum := 0
	//
	//// 控制并发数
	//concurrency := fileThreadNum
	//semaphore := make(chan struct{}, concurrency)
	//listener := &CosListener{}
	//
	//// 开启错误输出
	//var outputFile *os.File
	//if failOutput {
	//	// 创建错误日志目录
	//	_, err := os.Stat(failOutputPath)
	//	if os.IsNotExist(err) {
	//		err := os.MkdirAll(failOutputPath, 0755)
	//		if err != nil {
	//			logger.Fatalf("Failed to create error output dir: %v", err)
	//		}
	//	}
	//	// 创建错误日志文件
	//	failOutputFilePath := filepath.Join(failOutputPath, time.Now().Format("20060102_150405")+".report")
	//	outputFile, err = os.OpenFile(failOutputFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	//	if err != nil {
	//		logger.Fatal("Failed to create error output file:", err)
	//	}
	//	defer outputFile.Close()
	//}
	//
	//for _, f := range files {
	//	wg.Add(1)
	//
	//	go func(f string) {
	//		defer wg.Done()
	//
	//		localPath := filepath.Join(localDir, f)
	//		// 兼容windows，将windows的路径分隔符 "\" 转换为 "/"
	//		f = strings.ReplaceAll(f, string(filepath.Separator), "/")
	//		// 格式化cos路径
	//		cosPath := f
	//		if cosDir != "" {
	//			if !strings.HasSuffix(cosDir, "/") {
	//				cosPath = cosDir + "/" + f
	//			} else {
	//				cosPath = cosDir + f
	//			}
	//		}
	//		// 获取信号量
	//		semaphore <- struct{}{}
	//		err := SingleUpload(c, localPath, bucketName, cosPath, listener, op)
	//		if err != nil {
	//			// 在修改 failNum 之前锁定互斥锁
	//			mu.Lock()
	//			failNum += 1
	//			// 修改完成后解锁
	//			mu.Unlock()
	//
	//			// 记录失败原因
	//			if failOutput {
	//				outputMu.Lock()
	//				_, writeErr := outputFile.WriteString(fmt.Sprintf("[Upload error]Failed to upload %s: %v\n", localPath, err))
	//				if writeErr != nil {
	//					logger.Printf("Failed to write error output file for %s: %v\n", localPath, writeErr)
	//				}
	//				outputMu.Unlock()
	//			}
	//		} else {
	//			// 在修改 successNum 之前锁定互斥锁
	//			mu.Lock()
	//			successNum += 1
	//			// 修改完成后解锁
	//			mu.Unlock()
	//		}
	//
	//		// 释放信号量
	//		<-semaphore
	//	}(f)
	//}
	//
	//// 初始化进度
	//PrintTransferProcess(fileNum, totalSize, successNum, failNum, listener.TotalUploadedBytes, startTime, true)
	//
	//// 定期输出总进度
	//go func() {
	//	ticker := time.NewTicker(1 * time.Second)
	//	defer ticker.Stop()
	//	for range ticker.C {
	//		mu.Lock()
	//		PrintTransferProcess(fileNum, totalSize, successNum, failNum, listener.TotalUploadedBytes, startTime, false)
	//		mu.Unlock()
	//	}
	//}()
	//wg.Wait()
	//
	//// 输出最终进度
	//PrintTransferProcess(fileNum, totalSize, successNum, failNum, totalSize, startTime, false)
	//elapsedTime := time.Since(startTime)
	//logger.Infof("Upload %d files completed. %d successed, %d failed. cost %v", fileNum, successNum, failNum, elapsedTime)
	//if failNum > 0 {
	//	absOutputFile, _ := filepath.Abs(outputFile.Name())
	//	logger.Warnf("Some file upload failed, please check the detailed information in the %s.", absOutputFile)
	//}

}

func SingleUpload(c *cos.Client, cc *CopyCommand, file fileInfoType, cosPath string) (skip bool, rErr error, isDir bool, size int64, msg string) {
	skip = false
	rErr = nil
	isDir = false
	size = 0

	localFilePath, cosPath := UploadPathFixed(file, cosPath)

	msg = fmt.Sprintf("Upload %s to %s", localFilePath, SchemePrefix+cosPath)

	fileInfo, err := os.Stat(localFilePath)
	if err != nil {
		rErr = err
		return
	}

	size = fileInfo.Size()

	if fileInfo.IsDir() {
		isDir = true
		// 在cos创建文件夹
		_, err = c.Object.Put(context.Background(), cosPath, strings.NewReader(""), nil)
		if err != nil {
			rErr = err
			return
		}
	} else {
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
					CacheControl:             cc.CpParams.Meta.CacheControl,
					ContentDisposition:       cc.CpParams.Meta.ContentDisposition,
					ContentEncoding:          cc.CpParams.Meta.ContentEncoding,
					ContentType:              cc.CpParams.Meta.ContentType,
					ContentMD5:               cc.CpParams.Meta.ContentMD5,
					ContentLength:            cc.CpParams.Meta.ContentLength,
					ContentLanguage:          cc.CpParams.Meta.ContentLanguage,
					Expect:                   "",
					Expires:                  cc.CpParams.Meta.Expires,
					XCosContentSHA1:          "",
					XCosMetaXXX:              cc.CpParams.Meta.XCosMetaXXX,
					XCosStorageClass:         cc.CpParams.StorageClass,
					XCosServerSideEncryption: "",
					XCosSSECustomerAglo:      "",
					XCosSSECustomerKey:       "",
					XCosSSECustomerKeyMD5:    "",
					XOptionHeader:            nil,
					XCosTrafficLimit:         (int)(cc.CpParams.RateLimiting * 1024 * 1024 * 8),
					Listener:                 &CosListener{cc},
				},
			},
			PartSize:       cc.CpParams.PartSize,
			ThreadPoolSize: cc.CpParams.ThreadNum,
			CheckPoint:     true,
		}
		_, _, err = c.Object.Upload(context.Background(), cosPath, localFilePath, opt)
		if err != nil {
			rErr = err
			return
		}

		//if cc.CpParams.SnapshotPath != "" {
		//	cc.CpParams.SnapshotDb.Put([]byte(localPath), []byte(strconv.FormatInt(fileInfo.ModTime().Unix(), 10)), nil)
		//}

	}

	return
}
