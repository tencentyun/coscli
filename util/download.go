package util

import (
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"os"
	"regexp"
	"strings"
)

func SingleDownload(c *cos.Client, bucketName string, cosPath string, localPath string, rateLimiting float32, partSize int64, threadNum int) {
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
			XCosTrafficLimit:           (int)(rateLimiting * 1024 * 1024 * 8),
			Listener:                   &CosListener{},
		},
		PartSize:       partSize,
		ThreadPoolSize: threadNum,
		CheckPoint:     true,
		CheckPointFile: "",
	}

	// cos://bucket/path/123.txt => ~/example/123.txt
	// input: cos://bucket/path/123.txt => example/
	// show:  cos://bucket/path/123.txt => /Users/asdf/example/123.txt
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

	// 创建文件夹
	var path string
	if localPath[len(localPath) - 1] == '/' || localPath[len(localPath) - 1] == '\\' {
		pathList := strings.Split(cosPath, "/")
		fileName := pathList[len(pathList)-1]
		path = localPath
		localPath = localPath + fileName
	} else {
		pathList := strings.Split(localPath, "/")
		fileName := pathList[len(pathList)-1]
		path = localPath[:len(localPath) - len(fileName)]
	}
	fmt.Printf("Download cos://%s/%s => %s\n", bucketName, cosPath, localPath)

	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = c.Object.Download(context.Background(), cosPath, localPath, opt)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func MultiDownload(c *cos.Client, bucketName string, cosDir string, localDir string, include string, exclude string, rateLimiting float32, partSize int64, threadNum int) {
	if localDir != "" && (localDir[len(localDir)-1] != '/' || localDir[len(localDir)-1] != '\\') {
		localDir += "/"
	}
	if cosDir != "" && cosDir[len(cosDir)-1] != '/' {
		cosDir += "/"
	}

	objects := GetObjectsListRecursive(c, cosDir, 0, include, exclude)

	for _, o := range objects {
		objName := o.Key[len(cosDir):]
		localPath := localDir + objName
		SingleDownload(c, bucketName, o.Key, localPath, rateLimiting, partSize, threadNum)
	}
}
