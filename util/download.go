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

func DownloadPathFixed(localPath string, cosPath string, isRecursive bool) (string, string, error) {
	if len(cosPath) == 0 {
		logger.Warningln("Invalid cosPath")
		logger.Errorln(errors.New("invalid cosPath"))
		return "", "", errors.New("invalid cosPath")
	}
	// cos://bucket/dirPath/ => ~/example/
	// Should skip

	if strings.HasSuffix(cosPath, "/") {
		if isRecursive {
			logger.Warningf("Skip empty cosPath: cos://%s \n", cosPath)
			return "", "", errors.New("Skip empty cosPath: cos://" + cosPath)
		} else {
			logger.Fatalf("CosPath: cos://%s is a dir \n", cosPath)
			return "", "", errors.New("CosPath is a dir")
		}
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
		localPath = dirPath + string(filepath.Separator) + localPath
	}

	// 创建文件夹
	var path string
	s, _ := os.Stat(localPath)
	if (s != nil && s.IsDir()) || strings.HasSuffix(localPath, string(filepath.Separator)) {
		fileName := filepath.Base(cosPath)
		path = localPath
		localPath = filepath.Join(localPath, fileName)
	} else {
		fileName := filepath.Base(localPath)
		path = localPath[:len(localPath)-len(fileName)]
	}
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		logger.Fatalln(err)
	}
	return localPath, cosPath, err
}

func SingleDownload(c *cos.Client, bucketName, cosPath, localPath string, op *DownloadOptions, isRecursive bool) error {
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
	localPath, cosPath, err := DownloadPathFixed(localPath, cosPath, isRecursive)
	if err != nil {
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
	if localDir == "" {
		logger.Fatalln("localDir is empty")
		os.Exit(1)
	}

	// 路径分隔符
	// 记录是否是代码添加的路径分隔符
	isCosAddSeparator := false
	// cos路径若不以路径分隔符结尾，则添加
	if !strings.HasSuffix(cosDir, "/") && cosDir != ""{
		isCosAddSeparator = true
		cosDir += "/"
	}
	// 判断cosDir是否是文件夹
	isDir := CheckCosPathType(c, cosDir, 0)

	if isDir {
		// cosDir是文件夹 且 localDir不以路径分隔符结尾，则添加
		if !strings.HasSuffix(localDir, string(filepath.Separator)) {
			localDir += string(filepath.Separator)
		} else {
			// 若localDir以路径分隔符结尾，且cosDir传入时不以路径分隔符结尾，则需将cos路径的最终文件拼接至local路径最后
			if isCosAddSeparator {
				fileName := filepath.Base(cosDir)
				localDir += fileName
				localDir += string(filepath.Separator)
			}
		}
	} else {
		// cosDir不是文件夹且路径分隔符为代码添加,则去掉路径分隔符
		if isCosAddSeparator {
			cosDir = strings.TrimSuffix(cosDir, "/")
		}
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
		// 跳过输入路径
		if o.Key == cosDir && strings.HasSuffix(cosDir, "/") {
			continue
		}
		objName := o.Key[len(cosDir):]
		// 兼容windows，将cos的路径分隔符 "/" 转换为 "\"
		objName = strings.ReplaceAll(objName, "/", string(filepath.Separator))
		localPath := localDir + objName

		// 格式化文件名
		if objName == "" && strings.HasSuffix(localPath, "/") {
			fileName := filepath.Base(o.Key)
			localPath = localPath + fileName
		}

		err := SingleDownload(c, bucketName, o.Key, localPath, op, true)
		if err != nil {
			failNum += 1
		} else {
			successNum += 1
		}
	}
}
