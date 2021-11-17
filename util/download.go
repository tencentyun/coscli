package util

import (
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"os"
	"path/filepath"
	"strings"
)

type DownloadOptions struct {
	RateLimiting float32
	PartSize     int64
	ThreadNum    int
}

func SingleDownload(c *cos.Client, bucketName, cosPath, localPath string, op *DownloadOptions) {
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
			XCosTrafficLimit:           (int)(op.RateLimiting * 1024 * 1024 * 8),
			Listener:                   &CosListener{},
		},
		PartSize:       op.PartSize,
		ThreadPoolSize: op.ThreadNum,
		CheckPoint:     true,
		CheckPointFile: "",
	}

	// cos://bucket/dirPath/ => ~/example/
	// Should skip
	if len(cosPath) > 1 && cosPath[len(cosPath)-1] == '/' {
		return
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
	s, err := os.Stat(localPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if s.IsDir() {
		pathList := strings.Split(cosPath, "/")
		fileName := pathList[len(pathList)-1]
		path = localPath
		localPath = localPath + fileName
	} else {
		pathList := strings.Split(localPath, "/")
		fileName := pathList[len(pathList)-1]
		path = localPath[:len(localPath)-len(fileName)]
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

func MultiDownload(c *cos.Client, bucketName, cosDir, localDir, include, exclude string, op *DownloadOptions) {
	if localDir != "" && (localDir[len(localDir)-1] != '/' && localDir[len(localDir)-1] != '\\') {
		localDir += "/"
	}
	if cosDir != "" && cosDir[len(cosDir)-1] != '/' {
		cosDir += "/"
	}

	objects := GetObjectsListRecursive(c, cosDir, 0, include, exclude)

	for _, o := range objects {
		objName := o.Key[len(cosDir):]
		localPath := localDir + objName
		SingleDownload(c, bucketName, o.Key, localPath, op)
	}
}
