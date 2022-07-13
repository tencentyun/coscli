package util

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	logger "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
)

type UploadOptions struct {
	StorageClass string
	RateLimiting float32
	PartSize     int64
	ThreadNum    int
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
				CacheControl:             "",
				ContentDisposition:       "",
				ContentEncoding:          "",
				ContentType:              "",
				ContentMD5:               "",
				ContentLength:            0,
				ContentLanguage:          "",
				Expect:                   "",
				Expires:                  "",
				XCosContentSHA1:          "",
				XCosMetaXXX:              nil,
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
		PartSize:           op.PartSize,
		ThreadPoolSize:     op.ThreadNum,
		CheckPoint:         true,
	}
	localPath, cosPath = UploadPathFixed(localPath, cosPath)
	logger.Infof("Upload %s => cos://%s/%s\n", localPath, bucketName, cosPath)
	_, _, err := c.Object.Upload(context.Background(), cosPath, localPath, opt)
	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}
}

func MultiUpload(c *cos.Client, localDir, bucketName, cosDir, include, exclude string, op *UploadOptions) {
	if localDir != "" && (localDir[len(localDir)-1] != '/' && localDir[len(localDir)-1] != '\\') {
		localDir += "/"
	}
	if cosDir != "" && cosDir[len(cosDir)-1] != '/' {
		cosDir += "/"
	}

	files := GetLocalFilesListRecursive(localDir, include, exclude)

	for _, f := range files {
		localPath := localDir + f
		cosPath := cosDir + f

		SingleUpload(c, localPath, bucketName, cosPath, op)
	}
}
