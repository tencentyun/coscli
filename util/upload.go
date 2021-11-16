package util

import (
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"os"
	"regexp"
	"strings"
)

func SingleUpload(c *cos.Client, localPath string, bucketName string, cosPath string, storageClass string, rateLimiting float32, partSize int64, threadNum int) {
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
				XCosStorageClass:         storageClass,
				XCosServerSideEncryption: "",
				XCosSSECustomerAglo:      "",
				XCosSSECustomerKey:       "",
				XCosSSECustomerKeyMD5:    "",
				XOptionHeader:            nil,
				XCosTrafficLimit:         (int)(rateLimiting * 1024 * 1024 * 8),
				Listener:                 &CosListener{},
			},
		},
		PartSize:           partSize,
		ThreadPoolSize:     threadNum,
		CheckPoint:         true,
		EnableVerification: false,
	}

	// eg:~/example/123.txt => cos://bucket/path/123.txt
	// 0. ~/example/123.txt => cos://bucket
	if cosPath == "" {
		pathList := strings.Split(localPath, "/")
		fileName := pathList[len(pathList)-1]
		cosPath = fileName
	}
	// 1. ~/example/123.txt => cos://bucket/path/
	if cosPath[len(cosPath)-1] == '/' || cosPath[len(cosPath)-1] == '\\' {
		fileNames := strings.Split(localPath, "/")
		fileName := fileNames[len(fileNames)-1]
		cosPath = cosPath + fileName
	}
	// 2. 123.txt => cos://bucket/path/
	isWindowsAbsolute, err := regexp.MatchString(WindowsAbsolutePattern, localPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if localPath[0] != '/' && !isWindowsAbsolute{
		dirPath, err := os.Getwd()
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		localPath = dirPath + "/" + localPath
	}
	fmt.Printf("Upload %s => cos://%s/%s\n", localPath, bucketName, cosPath)
	_, _, err = c.Object.Upload(context.Background(), cosPath, localPath, opt)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func MultiUpload(c *cos.Client, localDir string, bucketName string, cosDir string, include string, exclude string, storageClass string, rateLimiting float32, partSize int64, threadNum int) {
	if localDir != "" && (localDir[len(localDir)-1] != '/' || localDir[len(localDir)-1] != '\\') {
		localDir += "/"
	}
	if cosDir != "" && cosDir[len(cosDir)-1] != '/' {
		cosDir += "/"
	}

	files := GetLocalFilesListRecursive(localDir, include, exclude)

	for _, f := range files {
		localPath := localDir + f
		cosPath := cosDir + f

		SingleUpload(c, localPath, bucketName, cosPath, storageClass, rateLimiting, partSize, threadNum)
	}
}
