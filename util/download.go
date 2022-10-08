package util

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/syndtr/goleveldb/leveldb"

	logger "github.com/sirupsen/logrus"

	"github.com/tencentyun/cos-go-sdk-v5"
)

type DownloadOptions struct {
	RateLimiting float32
	PartSize     int64
	ThreadNum    int
	SnapshotDb   *leveldb.DB
	SnapshotPath string
}

func DownloadPathFixed(localPath string, cosPath string) (string, string, error) {

	if len(cosPath) == 0 {
		logger.Warningln("Invalid cosPath")
		return "", "", errors.New("Invalid cosPath")
	}
	// cos://bucket/dirPath/ => ~/example/
	// Should skip
	if len(cosPath) >= 1 && cosPath[len(cosPath)-1] == '/' {
		logger.Warningf("Skip empty cosPath: cos://%s\n", cosPath)
		return "", "", errors.New("Skip empty cosFile")
	}

	// cos://bucket/path/123.txt => ~/example/123.txt
	// input: cos://bucket/path/123.txt => example/
	// show:  cos://bucket/path/123.txt => /Users/asdf/example/123.txt
	if !filepath.IsAbs(localPath) {
		dirPath, err := os.Getwd()
		if err != nil {
			logger.Fatalln(err)
			return "", "", err
		}
		localPath = dirPath + "/" + localPath
	}
	// 创建文件夹
	var path string
	s, _ := os.Stat(localPath)
	if (s != nil && s.IsDir()) || localPath[len(localPath)-1] == '/' {
		pathList := strings.Split(cosPath, "/")
		fileName := pathList[len(pathList)-1]
		path = localPath
		if localPath[len(localPath)-1] != '/' {
			localPath = localPath + "/"
		}
		localPath = localPath + fileName
	} else {
		pathList := strings.Split(localPath, "/")
		fileName := pathList[len(pathList)-1]
		path = localPath[:len(localPath)-len(fileName)]
	}
	err := os.MkdirAll(path, os.ModePerm)
	return localPath, cosPath, err
}

func SingleDownload(c *cos.Client, bucketName, cosPath, localPath string, op *DownloadOptions) error {
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
	localPath, cosPath, err := DownloadPathFixed(localPath, cosPath)
	if err != nil {
		logger.Errorln(err)
		return err
	}
	logger.Infof("Download cos://%s/%s => %s\n", bucketName, cosPath, localPath)

	resp, err := c.Object.Download(context.Background(), cosPath, localPath, opt)
	if err != nil {
		logger.Errorln(err)
		return err
	}

	if op.SnapshotPath != "" {
		lastModified := resp.Header.Get("Last-Modified")
		if lastModified == "" {
			return nil
		}

		var cosLastModifiedTime time.Time
		cosLastModifiedTime, err = time.Parse(time.RFC1123, lastModified)

		if err != nil {
			return err
		}
		op.SnapshotDb.Put([]byte(cosPath), []byte(strconv.FormatInt(cosLastModifiedTime.Unix(), 10)), nil)
	}

	return nil
}

func MultiDownload(c *cos.Client, bucketName, cosDir, localDir, include, exclude string, op *DownloadOptions) {
	if localDir != "" && (localDir[len(localDir)-1] != '/' && localDir[len(localDir)-1] != '\\') {
		localDir += "/"
	}
	if cosDir != "" && cosDir[len(cosDir)-1] != '/' {
		tmp := strings.Split(cosDir, "/")
		localDir = localDir + tmp[len(tmp)-1] + "/"
		cosDir += "/"
	}
	objects, commonPrefixes := GetObjectsListRecursive(c, cosDir, 0, include, exclude)
	listObjects(c, bucketName, objects, cosDir, localDir, op)

	if len(commonPrefixes) > 0 {
		for i := 0; i < len(commonPrefixes); i++ {
			localDirTemp := localDir + commonPrefixes[i]
			MultiDownload(c, bucketName, commonPrefixes[i], localDirTemp, include, exclude, op)
		}
	}
}
func listObjects(c *cos.Client, bucketName string, objects []cos.Object, cosDir string, localDir string, op *DownloadOptions) {
	failNum := 0
	successNum := 0
	if len(objects) == 0 {
		logger.Warningf("cosDir: cos://%s is empty\n", cosDir)
		return
	}
	for _, o := range objects {
		objName := o.Key[len(cosDir):]
		localPath := localDir + objName
		err := SingleDownload(c, bucketName, o.Key, localPath, op)
		if err != nil {
			failNum += 1
		} else {
			successNum += 1
		}
	}
}
