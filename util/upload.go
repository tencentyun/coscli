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
	// 若local路径不是绝对路径，则补齐
	if !filepath.IsAbs(localPath) {
		dirPath, err := os.Getwd()
		if err != nil {
			logger.Fatalln(err)
			os.Exit(1)
		}
		localPath = filepath.Join(dirPath, localPath)
	}

	fileInfo, err := os.Stat(localPath)
	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}
	// 若local路径为文件夹则报错
	if fileInfo.IsDir() || strings.HasSuffix(localPath, string(filepath.Separator)) {
		logger.Fatalf("path %s : is a dir", localPath)
		os.Exit(1)
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
	if localDir == "" {
		logger.Fatalln("localDir is empty")
		os.Exit(1)
	}

	// 格式化本地路径
	localDir = strings.TrimPrefix(localDir, "./")

	// 判断local路径是文件还是文件夹
	localDirInfo, err := os.Stat(localDir)
	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}
	var files []string
	if localDirInfo.IsDir() {
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

		files = GetLocalFilesListRecursive(localDir, include, exclude)
		for _, f := range files {
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

			SingleUpload(c, localPath, bucketName, cosPath, op)
		}
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

		// 若cos路径为空或以路径分隔符结尾，则需拼接文件名
		cosPath := cosDir
		if cosDir == "" || strings.HasSuffix(cosDir, "/") {
			cosPath = cosDir + fileName
		}
		localPath := localDir

		SingleUpload(c, localPath, bucketName, cosPath, op)
	}

}
