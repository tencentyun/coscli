package util

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	logger "github.com/sirupsen/logrus"
	leveldb "github.com/syndtr/goleveldb/leveldb"
	"github.com/tencentyun/cos-go-sdk-v5"
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

func UploadPathFixed(localPath string, cosPath string) (string, string) {
	// eg:~/example/123.txt => cos://bucket/path/123.txt
	// 0. ~/example/123.txt => cos://bucket
	if cosPath == "" {
		pathList := strings.Split(localPath, "/")
		fileName := pathList[len(pathList)-1]
		cosPath = fileName
	}
	// 1. ~/example/123.txt => cos://bucket/path/
	s, err := os.Stat(localPath)
	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}
	if s.IsDir() {
		fileNames := strings.Split(localPath, "/")
		fileName := fileNames[len(fileNames)-1]
		cosPath = cosPath + fileName
	}
	// 2. 123.txt => cos://bucket/path/
	if !filepath.IsAbs(localPath) {
		dirPath, err := os.Getwd()
		if err != nil {
			logger.Fatalln(err)
			os.Exit(1)
		}
		localPath = dirPath + "/" + localPath
	}
	return localPath, cosPath
}
func SingleUpload(c *cos.Client, localPath, bucketName, cosPath string, op *UploadOptions) {
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
				Listener:                 &CosListener{},
			},
		},
		PartSize:       op.PartSize,
		ThreadPoolSize: op.ThreadNum,
		CheckPoint:     true,
	}
	localPath, cosPath = UploadPathFixed(localPath, cosPath)
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return
	}

	if fileInfo.Mode().IsRegular() {
	} else if fileInfo.IsDir() {
	} else if fileInfo.Mode()&os.ModeSymlink == fs.ModeSymlink { // 软链接
		logger.Infoln(fmt.Sprintf("List %s file is Symlink, will be excluded, "+
			"please list or upload it from realpath",
			localPath))
		return
	} else {
		logger.Infoln(fmt.Sprintf("file %s is not regular file, will be excluded", localPath))
		return
	}

	logger.Infof("Upload %s => cos://%s/%s\n", localPath, bucketName, cosPath)
	_, _, err = c.Object.Upload(context.Background(), cosPath, localPath, opt)
	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}

	if op.SnapshotPath != "" {
		op.SnapshotDb.Put([]byte(localPath), []byte(strconv.FormatInt(fileInfo.ModTime().Unix(), 10)), nil)
	}
}

func MultiUpload(c *cos.Client, localDir, bucketName, cosDir, include, exclude string, op *UploadOptions) {

	if cosDir != "" && cosDir[len(cosDir)-1] != '/' {
		cosDir += "/"
	}
	if localDir != "" && (localDir[len(localDir)-1] != '/' && localDir[len(localDir)-1] != '\\') {
		tmp := strings.Split(localDir, "/")
		cosDir = cosDir + tmp[len(tmp)-1] + "/"
		localDir += "/"
	}

	files := GetLocalFilesListRecursive(localDir, include, exclude)

	for _, f := range files {
		localPath := localDir + f
		cosPath := cosDir + f
		SingleUpload(c, localPath, bucketName, cosPath, op)
	}
}
