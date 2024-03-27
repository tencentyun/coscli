package util

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	logger "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func SyncUpload(c *cos.Client, fileUrl StorageUrl, cosUrl StorageUrl, fo *FileOperations) {
	var err error
	keysToDelete := make(map[string]string)
	if fo.Operation.Delete {
		keysToDelete, err = getDeleteKeys(c, fileUrl, cosUrl, fo)
		if err != nil {
			logger.Fatalf("get delete keys error : %v", err)
		}
	}

	// 上传
	Upload(c, fileUrl, cosUrl, fo)
	if len(keysToDelete) > 0 {
		// 删除源位置没有而目标位置有的cos对象或本地文件
		err = deleteKeys(c, keysToDelete, cosUrl, fo)
	}

	if err != nil {
		logger.Fatalf("delete keys error : %v", err)
	}
}

func skipUpload(snapshotKey string, c *cos.Client, fo *FileOperations, localFileModifiedTime int64, cosPath string, localPath string) (bool, error) {

	if fo.Operation.SnapshotPath != "" {
		timeStr, err := fo.SnapshotDb.Get([]byte(snapshotKey), nil)
		if err == nil {
			modifiedTime, _ := strconv.ParseInt(string(timeStr), 10, 64)
			if modifiedTime == localFileModifiedTime {
				return true, nil
			}
		}
	}

	resp, err := getHead(c, cosPath)
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
				if fo.Operation.SnapshotPath != "" {
					fo.SnapshotDb.Put([]byte(localPath), []byte(strconv.FormatInt(localFileModifiedTime, 10)), nil)
				}
				return true, nil
			} else {
				return false, nil
			}
		} else {
			return false, nil
		}
	}
	return false, nil
}

func getSnapshotKey(absLocalFilePath string, bucket string, object string) string {
	return absLocalFilePath + SnapshotConnector + getCosUrl(bucket, object)
}

func InitSnapshotDb(srcUrl, destUrl StorageUrl, fo *FileOperations) {
	if fo.Operation.SnapshotPath == "" {
		return
	}

	var err error
	if fo.CpType == CpTypeUpload {
		err = CheckPath(srcUrl, fo, TypeSnapshotPath)
	} else if fo.CpType == CpTypeDownload {
		err = CheckPath(destUrl, fo, TypeSnapshotPath)
	} else {
		logger.Fatalln("copy object doesn't support option --snapshot-path")
	}
	if err != nil {
		logger.Fatalln(err)
	}

	if fo.SnapshotDb, err = leveldb.OpenFile(fo.Operation.SnapshotPath, nil); err != nil {
		logger.Fatalln("Sync load snapshot error, reason: " + err.Error())
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
