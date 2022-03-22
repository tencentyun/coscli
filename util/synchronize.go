package util

import (
	"context"
	"os"

	logger "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func SyncSingleUpload(c *cos.Client, localPath, bucketName, cosPath string, op *UploadOptions) {
	headOpt := &cos.ObjectHeadOptions{
		IfModifiedSince:       "",
		XCosSSECustomerAglo:   "",
		XCosSSECustomerKey:    "",
		XCosSSECustomerKeyMD5: "",
		XOptionHeader:         nil,
	}
	localPath, cosPath = UploadPathFixed(localPath, cosPath)
	resp, err := c.Object.Head(context.Background(), cosPath, headOpt)

	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			// 文件不在cos上，上传
			SingleUpload(c, localPath, bucketName, cosPath, op)
		} else {
			logger.Fatalln(err)
			os.Exit(1)
		}
	} else {
		if resp.StatusCode != 404 {
			cosCrc := resp.Header.Get("x-cos-hash-crc64ecma")
			localCrc, _ := CalculateHash(localPath, "crc64")
			if cosCrc == localCrc {
				logger.Infoln("Skip", localPath)
				return
			}
		}

		SingleUpload(c, localPath, bucketName, cosPath, op)
	}
}

func SyncMultiUpload(c *cos.Client, localDir, bucketName, cosDir, include, exclude string, op *UploadOptions) {
	if localDir != "" && (localDir[len(localDir)-1] != '/' && localDir[len(localDir)-1] != '\\') {
		localDir += "/"
	}
	if cosDir != "" && cosDir[len(cosDir)-1] != '/' {
		cosDir += "/"
	}

	files := GetLocalFilesListRecursive(localDir, include, exclude)

	for _, f := range files {
		localPath := localDir + f
		cosPath := cosDir + f

		SyncSingleUpload(c, localPath, bucketName, cosPath, op)
	}
}

func SyncSingleDownload(c *cos.Client, bucketName, cosPath, localPath string, op *DownloadOptions) error {
	localPath, cosPath, err := DownloadPathFixed(localPath, cosPath)
	if err != nil {
		return err
	}
	_, err = os.Stat(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不在本地，下载
			SingleDownload(c, bucketName, cosPath, localPath, op)
		} else {
			logger.Fatalln(err)
			return err
		}
	} else {
		localCrc, _ := CalculateHash(localPath, "crc64")
		cosCrc, _ := ShowHash(c, cosPath, "crc64")
		if cosCrc == localCrc {
			logger.Infof("Skip cos://%s/%s\n", bucketName, cosPath)
			return nil
		}

		SingleDownload(c, bucketName, cosPath, localPath, op)
	}
	return nil
}

func SyncMultiDownload(c *cos.Client, bucketName, cosDir, localDir, include, exclude string, op *DownloadOptions) {
	if localDir != "" && localDir[len(localDir)-1] != '/' {
		localDir += "/"
	}
	if cosDir != "" && cosDir[len(cosDir)-1] != '/' {
		cosDir += "/"
	}
	objects := GetObjectsListRecursive(c, cosDir, 0, include, exclude)
	if len(objects) == 0 {
		logger.Warningf("cosDir: cos://%s is empty\n", cosDir)
		return
	}
	for _, o := range objects {
		objName := o.Key[len(cosDir):]
		localPath := localDir + objName
		SyncSingleDownload(c, bucketName, o.Key, localPath, op)

	}
}
