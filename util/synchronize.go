package util

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	logger "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func SyncSingleUpload(c *cos.Client, localPath, bucketName, cosPath string, op *UploadOptions) {
	//localPath, cosPath, err := UploadPathFixed(localPath, cosPath)
	//skip, err := skipUpload(c, op.SnapshotPath, op.SnapshotDb, localPath, cosPath)
	//if err != nil {
	//	logger.Errorf("Sync LocalPath:%s, err:%s", localPath, err.Error())
	//	return
	//}
	//
	//if skip {
	//	logger.Infof("Sync upload file localPath skip, %s", localPath)
	//} else {
	//	SingleUpload(c, localPath, bucketName, cosPath, &CosListener{}, op)
	//}
}

func skipUpload(c *cos.Client, snapshotPath string, snapshotDb *leveldb.DB, localPath string,
	cosPath string) (skip bool, err error) {

	var localPathInfo os.FileInfo
	localPathInfo, err = os.Stat(localPath)
	// 直接和本地的snapshot作对比
	if snapshotPath != "" {
		if err != nil {
			return
		}
		var info []byte
		info, err = snapshotDb.Get([]byte(localPath), nil)
		if err == nil {
			t, _ := strconv.ParseInt(string(info), 10, 64)
			if t == localPathInfo.ModTime().Unix() {
				return true, nil
			} else {
				return false, nil
			}
		}
	}

	headOpt := &cos.ObjectHeadOptions{
		IfModifiedSince:       "",
		XCosSSECustomerAglo:   "",
		XCosSSECustomerKey:    "",
		XCosSSECustomerKeyMD5: "",
		XOptionHeader:         nil,
	}
	resp, err := c.Object.Head(context.Background(), cosPath, headOpt)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			// 文件不在cos上，上传
			return false, nil
		} else {
			return false, err
		}
	} else {
		if resp.StatusCode != 404 {
			cosCrc := resp.Header.Get("x-cos-hash-crc64ecma")
			localCrc, _ := CalculateHash(localPath, "crc64")
			if cosCrc == localCrc {
				// 本地校验通过后，若未记录快照。则添加
				if snapshotPath != "" {
					snapshotDb.Put([]byte(localPath), []byte(strconv.FormatInt(localPathInfo.ModTime().Unix(), 10)), nil)
				}
				return true, nil
			} else {
				return false, nil
			}
		} else {
			return false, nil
		}
	}
}

func SyncMultiUpload(c *cos.Client, localDir, bucketName, cosDir, include, exclude string, op *UploadOptions) {
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
			SyncSingleUpload(c, localPath, bucketName, cosPath, op)
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

		SyncSingleUpload(c, localPath, bucketName, cosPath, op)
	}
}

func SyncSingleDownload(c *cos.Client, bucketName, cosPath, localPath string, op *DownloadOptions,
	cosLastModified string, isRecursive bool) error {
	localPath, cosPath, err := DownloadPathFixed(localPath, cosPath, isRecursive)
	if err != nil {
		return err
	}
	_, err = os.Stat(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不在本地，下载
			err = SingleDownload(c, bucketName, cosPath, localPath, op, isRecursive)
		} else {
			logger.Fatalln(err)
			return err
		}
	} else {
		var skip bool
		skip, err = skipDownload(c, op.SnapshotPath, op.SnapshotDb, localPath, cosPath, cosLastModified)
		if err != nil {
			logger.Errorf("Sync cosPath, err:%s", err.Error())
			return err
		}

		if skip {
			logger.Infof("Sync skip download, localPath:%s, cosPath:%s", localPath, cosPath)
			return nil
		}
		err = SingleDownload(c, bucketName, cosPath, localPath, op, isRecursive)
	}
	return err
}

func skipDownload(c *cos.Client, snapshotPath string, snapshotDb *leveldb.DB, localPath string,
	cosPath string, cosLastModified string) (skip bool, err error) {
	// 直接和本地的snapshot作对比
	if snapshotPath != "" {
		if cosLastModified == "" {
			cosLastModified, err = getCosLastModified(c, cosPath)
			if err != nil {
				return
			}
		}
		var cosLastModifiedTime time.Time
		cosLastModifiedTime, err = time.Parse(time.RFC3339, cosLastModified)
		if err != nil {
			cosLastModifiedTime, err = time.Parse(time.RFC1123, cosLastModified)
			if err != nil {
				return
			}
		}
		var info []byte
		info, err = snapshotDb.Get([]byte(cosPath), nil)
		if err == nil {
			t, _ := strconv.ParseInt(string(info), 10, 64)
			if t == cosLastModifiedTime.Unix() {
				return true, nil
			} else {
				return false, nil
			}
		}
	}

	localCrc, _ := CalculateHash(localPath, "crc64")
	cosCrc, _, resp := ShowHash(c, cosPath, "crc64")
	if cosCrc == localCrc {
		// 本地校验通过后，若未记录快照。则添加
		if snapshotPath != "" {
			lastModified := resp.Header.Get("Last-Modified")
			if lastModified == "" {
				return false, nil
			}
			var cosLastModifiedTime time.Time
			cosLastModifiedTime, err = time.Parse(time.RFC1123, lastModified)
			if err != nil {
				return false, nil
			}
			snapshotDb.Put([]byte(cosPath), []byte(strconv.FormatInt(cosLastModifiedTime.Unix(), 10)), nil)
		}
		return true, nil
	} else {
		return false, nil
	}
}

func getCosLastModified(c *cos.Client, cosPath string) (lmt string, err error) {
	headOpt := &cos.ObjectHeadOptions{
		IfModifiedSince:       "",
		XCosSSECustomerAglo:   "",
		XCosSSECustomerKey:    "",
		XCosSSECustomerKeyMD5: "",
		XOptionHeader:         nil,
	}
	resp, err := c.Object.Head(context.Background(), cosPath, headOpt)
	if err != nil {
		return "", err
	} else {
		return resp.Header.Get("Last-Modified"), nil
	}
}

func SyncMultiDownload(c *cos.Client, bucketName, cosDir, localDir, include, exclude string, retryNum int, op *DownloadOptions) {
	if localDir == "" {
		logger.Fatalln("localDir is empty")
		os.Exit(1)
	}
	// 记录是否是代码添加的路径分隔符
	isCosAddSeparator := false
	// cos路径若不以路径分隔符结尾，则添加
	if !strings.HasSuffix(cosDir, "/") && cosDir != "" {
		isCosAddSeparator = true
		cosDir += "/"
	}
	// 判断cosDir是否是文件夹
	isDir := CheckCosPathType(c, cosDir, 0, retryNum)

	if isDir {
		// cosDir是文件夹
		if !strings.HasSuffix(localDir, string(filepath.Separator)) {
			// 若localDir不以路径分隔符结尾，则添加
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

	objects, _ := GetObjectsListRecursive(c, cosDir, 0, include, exclude, retryNum)
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
		localPath := localDir + objName
		SyncSingleDownload(c, bucketName, o.Key, localPath, op, o.LastModified, true)

	}
}
