package util

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	logger "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func SyncSingleUpload(c *cos.Client, localPath, bucketName, cosPath string, op *UploadOptions) {
	localPath, cosPath = UploadPathFixed(localPath, cosPath)
	skip, err := skipUpload(c, op.SnapshotPath, op.SnapshotDb, localPath, cosPath)
	if err != nil {
		logger.Errorf("Sync LocalPath:%s, err:%s", localPath, err.Error())
		return
	}

	if skip {
		logger.Infof("Sync upload file localPath skip, %s", localPath)
	} else {
		SingleUpload(c, localPath, bucketName, cosPath, op)
	}
}

func skipUpload(c *cos.Client, snapshotPath string, snapshotDb *leveldb.DB, localPath string,
	cosPath string) (skip bool, err error) {
	// 直接和本地的snapshot作对比
	if snapshotPath != "" {
		var localPathInfo os.FileInfo
		localPathInfo, err = os.Stat(localPath)
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
		} else {
			return false, nil
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
	if cosDir != "" && cosDir[len(cosDir)-1] != '/' {
		cosDir += "/"
	}
	if localDir != "" && (localDir[len(localDir)-1] != '/' && localDir[len(localDir)-1] != '\\') {
		tmp := strings.Split(localDir, "/")
		cosDir = cosDir + tmp[len(tmp)-1] + "/"
		localDir += "/"
	}

	files := GetLocalFilesListRecursive(localDir, include, exclude)

	for _, f := range files {
		localPath := localDir + f
		cosPath := cosDir + f

		SyncSingleUpload(c, localPath, bucketName, cosPath, op)
	}
}

func SyncSingleDownload(c *cos.Client, bucketName, cosPath, localPath string, op *DownloadOptions,
	cosLastModified string) error {
	localPath, cosPath, err := DownloadPathFixed(localPath, cosPath)
	if err != nil {
		return err
	}
	_, err = os.Stat(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不在本地，下载
			err = SingleDownload(c, bucketName, cosPath, localPath, op)
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
		err = SingleDownload(c, bucketName, cosPath, localPath, op)
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
		} else {
			return false, nil
		}
	}

	localCrc, _ := CalculateHash(localPath, "crc64")
	cosCrc, _ := ShowHash(c, cosPath, "crc64")
	if cosCrc == localCrc {
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

func SyncMultiDownload(c *cos.Client, bucketName, cosDir, localDir, include, exclude string, op *DownloadOptions) {
	if localDir != "" && (localDir[len(localDir)-1] != '/' && localDir[len(localDir)-1] != '\\') {
		localDir += "/"
	}
	if cosDir != "" && cosDir[len(cosDir)-1] != '/' {
		tmp := strings.Split(cosDir, "/")
		localDir = localDir + tmp[len(tmp)-1] + "/"
		cosDir += "/"
	}
	objects, _ := GetObjectsListRecursive(c, cosDir, 0, include, exclude)
	if len(objects) == 0 {
		logger.Warningf("cosDir: cos://%s is empty\n", cosDir)
		return
	}
	for _, o := range objects {
		objName := o.Key[len(cosDir):]
		localPath := localDir + objName
		SyncSingleDownload(c, bucketName, o.Key, localPath, op, o.LastModified)

	}
}
