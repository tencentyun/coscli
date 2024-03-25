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

func Upload(fileUrl StorageUrl, cosUrl StorageUrl, fo *FileOperations, cpType CpType) {
	localPath := fileUrl.ToString()
	bucketName := cosUrl.(CosUrl).Bucket
	cosPath := cosUrl.(CosUrl).Object

	c := NewClient(fo.Config, fo.Param, bucketName)
	// crc64校验开关
	c.Conf.EnableCRC = fo.Operation.DisableCrc64

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

	if localPathInfo.IsDir() && !fo.Operation.Recursive {
		logger.Fatalf("localPath:%v is dir, please use --recursive option", localPath)
	}

	// 格式化路径
	if fo.Operation.Recursive {
		cosPath, localPath = formatPath(cosPath, localPath)
	}

	fo.Monitor.init(cpType)
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
		go uploadFiles(c, cosPath, fo, chFiles, chError)
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
}

func uploadFiles(c *cos.Client, cosPath string, fo *FileOperations, chFiles <-chan fileInfoType, chError chan<- error) {
	for file := range chFiles {
		if filterFile(file, fo.Operation.CheckpointDir) {
			skip, err, isDir, size, msg := SingleUpload(c, fo, file, cosPath)
			fo.Monitor.updateMonitor(skip, err, isDir, size)
			if err != nil {
				chError <- fmt.Errorf("%s failed: %w", msg, err)
				continue
			}
		}
	}

	chError <- nil
}

func SingleUpload(c *cos.Client, fo *FileOperations, file fileInfoType, cosPath string) (skip bool, rErr error, isDir bool, size int64, msg string) {
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

		//if fo.Operation.SnapshotPath != "" {
		//	fo.Operation.SnapshotDb.Put([]byte(localPath), []byte(strconv.FormatInt(fileInfo.ModTime().Unix(), 10)), nil)
		//}

	}

	return
}
