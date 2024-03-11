package util

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	logger "github.com/sirupsen/logrus"
	leveldb "github.com/syndtr/goleveldb/leveldb"
	"github.com/tencentyun/cos-go-sdk-v5"
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

func UploadPathFixed(localPath string, cosPath string) (string, string, error) {
	// 若local路径不是绝对路径，则补齐
	if !filepath.IsAbs(localPath) {
		dirPath, err := os.Getwd()
		if err != nil {
			return "", "", err
		}
		localPath = dirPath + string(filepath.Separator) + localPath
	}

	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return "", "", err
	}
	// 若local路径为文件夹则报错
	if fileInfo.IsDir() || strings.HasSuffix(localPath, string(filepath.Separator)) {
		errMsg := fmt.Sprintf("path %s : is a dir", localPath)
		return "", "", fmt.Errorf(errMsg)
	}

	// 文件名称
	fileName := filepath.Base(localPath)
	// 若cos路径为空，则直接赋值为文件名
	if cosPath == "" {
		cosPath = fileName
	} else {
		// 若cos路径不为空且以路径分隔符结尾，则拼接文件名
		if strings.HasSuffix(cosPath, "/") {
			cosPath += fileName
		}
	}

	return localPath, cosPath, nil
}
func SingleUpload(c *cos.Client, localPath, bucketName, cosPath string, listener interface{}, op *UploadOptions) error {

	localPath, cosPath, err := UploadPathFixed(localPath, cosPath)
	if err != nil {
		return err
	}

	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return err
	}

	if fileInfo.Mode().IsRegular() {
	} else if fileInfo.IsDir() {
	} else if fileInfo.Mode()&os.ModeSymlink == fs.ModeSymlink { // 软链接
		errMsg := fmt.Sprintf("List %s file is Symlink, will be excluded, "+
			"please list or upload it from realpath",
			localPath)
		return fmt.Errorf(errMsg)
	} else {
		errMsg := fmt.Sprintf("file %s is not regular file, will be excluded", localPath)
		return fmt.Errorf(errMsg)
	}

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
				CacheControl:             op.Meta.CacheControl,
				ContentDisposition:       op.Meta.ContentDisposition,
				ContentEncoding:          op.Meta.ContentEncoding,
				ContentType:              op.Meta.ContentType,
				ContentMD5:               op.Meta.ContentMD5,
				ContentLength:            op.Meta.ContentLength,
				ContentLanguage:          op.Meta.ContentLanguage,
				Expect:                   "",
				Expires:                  op.Meta.Expires,
				XCosContentSHA1:          "",
				XCosMetaXXX:              op.Meta.XCosMetaXXX,
				XCosStorageClass:         op.StorageClass,
				XCosServerSideEncryption: "",
				XCosSSECustomerAglo:      "",
				XCosSSECustomerKey:       "",
				XCosSSECustomerKeyMD5:    "",
				XOptionHeader:            nil,
				XCosTrafficLimit:         (int)(op.RateLimiting * 1024 * 1024 * 8),
			},
		},
		PartSize:       op.PartSize,
		ThreadPoolSize: op.ThreadNum,
		CheckPoint:     true,
	}

	switch l := listener.(type) {
	case *CosListener:
		opt.OptIni.ObjectPutHeaderOptions.Listener = l
	case *SingleCosListener:
		opt.OptIni.ObjectPutHeaderOptions.Listener = l
	default:
		errMsg := fmt.Sprintf("system error")
		return fmt.Errorf(errMsg)
	}

	_, _, err = c.Object.Upload(context.Background(), cosPath, localPath, opt)
	if err != nil {
		return err
	}

	if op.SnapshotPath != "" {
		op.SnapshotDb.Put([]byte(localPath), []byte(strconv.FormatInt(fileInfo.ModTime().Unix(), 10)), nil)
	}

	return nil
}

func MultiUpload(c *cos.Client, localDir string, localPathInfo os.FileInfo, bucketName, cosDir, include, exclude string, op *UploadOptions, startTime time.Time) {
	// 判断local路径是文件还是文件夹
	if localPathInfo.IsDir() {
		if cosDir != "" {
			if strings.HasSuffix(cosDir, "/") {
				// cos路径若以路径分隔符结尾，且 local路径若不以路径分隔符结尾，则需将local路径的最终文件拼接至cos路径最后
				if !strings.HasSuffix(localDir, string(filepath.Separator)) {
					fileName := filepath.Base(localDir)
					cosDir += fileName
				}
			} else {
				// cos路径若不以路径分隔符结尾，则添加路径分隔符
				cosDir += "/"
			}
		} else {
			// cos路径为空，且 local路径若不以路径分隔符结尾，则需将local路径的最终文件拼接至cos路径最后
			if !strings.HasSuffix(localDir, string(filepath.Separator)) {
				fileName := filepath.Base(localDir)
				cosDir += fileName
			}
		}

		// local路径若不以路径分隔符结尾，则添加
		if !strings.HasSuffix(localDir, string(filepath.Separator)) {
			localDir += string(filepath.Separator)
		}

		files := GetLocalFilesListRecursive(localDir, include, exclude)

		var wg sync.WaitGroup
		totalSize := int64(0)
		for _, f := range files {
			path := filepath.Join(localDir, f)
			file, _ := os.Stat(path)
			totalSize += file.Size()
		}
		// 获取文件总数
		fileNum := len(files)
		failNum := 0
		successNum := 0

		// 控制并发数
		concurrency := 10
		semaphore := make(chan struct{}, concurrency)
		listener := &CosListener{}
		for _, f := range files {
			wg.Add(1)

			go func(f string) {
				defer wg.Done()

				localPath := filepath.Join(localDir, f)
				// 兼容windows，将windows的路径分隔符 "\" 转换为 "/"
				f = strings.ReplaceAll(f, string(filepath.Separator), "/")
				// 格式化cos路径
				cosPath := f
				if cosDir != "" {
					if !strings.HasSuffix(cosDir, "/") {
						cosPath = cosDir + "/" + f
					} else {
						cosPath = cosDir + f
					}
				}
				// 获取信号量
				semaphore <- struct{}{}
				err := SingleUpload(c, localPath, bucketName, cosPath, listener, op)
				if err != nil {
					// 在修改 failNum 之前锁定互斥锁
					mu.Lock()
					failNum += 1
					// 修改完成后解锁
					mu.Unlock()
				} else {
					// 在修改 successNum 之前锁定互斥锁
					mu.Lock()
					successNum += 1
					// 修改完成后解锁
					mu.Unlock()
				}

				// 释放信号量
				<-semaphore
			}(f)
		}

		// 初始化进度
		PrintTransferProcess(fileNum, totalSize, successNum, failNum, listener.TotalUploadedBytes, startTime, true)

		// 定期输出总进度
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				mu.Lock()
				PrintTransferProcess(fileNum, totalSize, successNum, failNum, listener.TotalUploadedBytes, startTime, false)
				mu.Unlock()
			}
		}()
		wg.Wait()

		// 输出最终进度
		PrintTransferProcess(fileNum, totalSize, successNum, failNum, totalSize, startTime, false)
		elapsedTime := time.Since(startTime)
		logger.Infof("Upload %d files completed. %d successed, %d failed. cost %v", fileNum, successNum, failNum, elapsedTime)

	} else {
		// 若是文件直接取出文件名
		fileName := filepath.Base(localDir)

		// 匹配规则
		if len(include) > 0 {
			re := regexp.MustCompile(include)
			match := re.MatchString(fileName)
			if !match {
				logger.Warningf("skip file %s due to not matching \"%s\" pattern ", localDir, include)
				os.Exit(1)
			}
		}

		if len(exclude) > 0 {
			re := regexp.MustCompile(exclude)
			match := re.MatchString(fileName)
			if match {
				logger.Warningf("skip file %s due to matching \"%s\" pattern ", localDir, exclude)
				os.Exit(1)
			}
		}

		// 初始化进度
		PrintTransferProcess(1, localPathInfo.Size(), 0, 0, 0, startTime, true)
		err := SingleUpload(c, localDir, bucketName, cosDir, &SingleCosListener{StartTime: startTime}, op)
		if err != nil {
			// 清空进度条
			CleanTransferProcess()
			logger.Fatalln(err)
			os.Exit(1)
		}
		elapsedTime := time.Since(startTime)
		logger.Infof("Upload file successed.  cost %v", elapsedTime)
	}
}
