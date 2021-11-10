package util

import (
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"os"
	"path/filepath"
	"strings"
)

func SingleDownload(c *cos.Client, bucketName string, cosPath string, localPath string) {
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
			XCosTrafficLimit:           0,
			Listener:                   &CosListener{},
		},
		PartSize:       32,
		ThreadPoolSize: 5,
		CheckPoint:     true,
		CheckPointFile: "",
	}

	// cos://bucket/path/123.txt => ~/example/123.txt
	// input: cos://bucket/path/123.txt => example/
	// show:  cos://bucket/path/123.txt => /Users/asdf/example/123.txt
	if !filepath.IsAbs(localPath) {
		dirPath, err := os.Getwd()
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		localPath = dirPath + "/" + localPath
	}

	// 创建文件夹
	var path string
	if localPath[len(localPath) - 1] == '/' {
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

	err := os.MkdirAll(path, os.ModePerm)
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

func MultiDownload(c *cos.Client, bucketName string, cosDir string, localDir string, include string, exclude string) {
	if localDir != "" && localDir[len(localDir)-1] != '/' {
		localDir += "/"
	}
	if cosDir != "" && cosDir[len(cosDir)-1] != '/' {
		cosDir += "/"
	}

	objects := GetObjectsListRecursive(c, cosDir, 0, include, exclude)

	for _, o := range objects {
		objName := o.Key[len(cosDir):]
		localPath := localDir + objName
		SingleDownload(c, bucketName, o.Key, localPath)
	}
}
