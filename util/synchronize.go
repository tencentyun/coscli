package util

import (
	"fmt"
	logger "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tencentyun/cos-go-sdk-v5"
	"strconv"
	"time"
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
					fo.SnapshotDb.Put([]byte(snapshotKey), []byte(strconv.FormatInt(localFileModifiedTime, 10)), nil)
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

func getUploadSnapshotKey(absLocalFilePath string, bucket string, object string) string {
	return absLocalFilePath + SnapshotConnector + getCosUrl(bucket, object)
}

func skipDownload(c *cos.Client, fo *FileOperations, localPath string, objectModifiedTimeStr string, object string) (bool, error) {
	// 解析时间字符串
	objectModifiedTime, err := time.Parse(time.RFC3339, objectModifiedTimeStr)
	if err != nil {
		return false, err
	}
	if fo.Operation.SnapshotPath != "" {
		timeStr, err := fo.SnapshotDb.Get([]byte(object), nil)
		if err == nil {
			modifiedTime, _ := strconv.ParseInt(string(timeStr), 10, 64)
			if modifiedTime == objectModifiedTime.Unix() {
				return true, nil
			}
		}
	}

	localCrc, _ := CalculateHash(localPath, "crc64")
	resp, err := getHead(c, object)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			// 文件不在cos上
			return false, fmt.Errorf("Object not found")
		} else {
			return false, err
		}
	} else {
		cosCrc := resp.Header.Get("x-cos-hash-crc64ecma")
		if cosCrc == localCrc {
			// 本地校验通过后，添加快照记录
			if fo.Operation.SnapshotPath != "" {
				fo.SnapshotDb.Put([]byte(object), []byte(strconv.FormatInt(objectModifiedTime.Unix(), 10)), nil)
			}
			return true, nil
		} else {
			return false, nil
		}
	}

	return false, nil
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

func SyncDownload(c *cos.Client, cosUrl StorageUrl, fileUrl StorageUrl, fo *FileOperations) {
	var err error
	keysToDelete := make(map[string]string)
	if fo.Operation.Delete {
		keysToDelete, err = getDeleteKeys(c, cosUrl, fileUrl, fo)
		if err != nil {
			logger.Fatalf("get delete keys error : %v", err)
		}
	}

	// 下载
	Download(c, cosUrl, fileUrl, fo)

	if len(keysToDelete) > 0 {
		// 删除源位置没有而目标位置有的cos对象或本地文件
		err = deleteKeys(c, keysToDelete, fileUrl, fo)
	}

	if err != nil {
		logger.Fatalf("delete keys error : %v", err)
	}
}
