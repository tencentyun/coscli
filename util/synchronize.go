package util

import (
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"os"
)

func SyncSingleUpload(c *cos.Client, localPath string, bucketName string, cosPath string, storageClass string, rateLimiting float32, partSize int64, threadNum int) {
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
			SingleUpload(c, localPath, bucketName, cosPath, storageClass, rateLimiting, partSize, threadNum)
		} else {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		if resp.StatusCode != 404 {
			//isWindowsAbsolute, err := regexp.MatchString(WindowsAbsolutePattern, localPath)
			//if err != nil {
			//	_, _ = fmt.Fprintln(os.Stderr, err)
			//	os.Exit(1)
			//}
			//// 补全 localPath
			//if localPath[0] != '/' && !isWindowsAbsolute {
			//	dirPath, err := os.Getwd()
			//	if err != nil {
			//		_, _ = fmt.Fprintln(os.Stderr, err)
			//		os.Exit(1)
			//	}
			//	localPath = dirPath + "/" + localPath
			//}
			cosCrc := resp.Header.Get("x-cos-hash-crc64ecma")
			localCrc, _ := CalculateHash(localPath, "crc64")
			if cosCrc == localCrc {
				fmt.Println("Skip", localPath)
				return
			}
		}

		SingleUpload(c, localPath, bucketName, cosPath, storageClass, rateLimiting, partSize, threadNum)
	}
}

func SyncMultiUpload(c *cos.Client, localDir string, bucketName string, cosDir string, include string, exclude string, storageClass string, rateLimiting float32, partSize int64, threadNum int) {
	if localDir != "" && (localDir[len(localDir)-1] != '/' || localDir[len(localDir)-1] !='\\') {
		localDir += "/"
	}
	if cosDir != "" && cosDir[len(cosDir)-1] != '/' {
		cosDir += "/"
	}

	files := GetLocalFilesListRecursive(localDir, include, exclude)

	for _, f := range files {
		localPath := localDir + f
		cosPath := cosDir + f

		SyncSingleUpload(c, localPath, bucketName, cosPath, storageClass, rateLimiting, partSize, threadNum)
	}
}

func SyncSingleDownload(c *cos.Client, bucketName string, cosPath string, localPath string, rateLimiting float32, partSize int64, threadNum int) {
	_, err := os.Stat(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不在本地，下载
			SingleDownload(c, bucketName, cosPath, localPath, rateLimiting, partSize, threadNum)
		} else {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		localCrc, _ := CalculateHash(localPath, "crc64")
		cosCrc, _ := ShowHash(c, cosPath, "crc64")
		if cosCrc == localCrc {
			fmt.Printf("Skip cos://%s/%s\n", bucketName, localPath)
			return
		}

		SingleDownload(c, bucketName, cosPath, localPath, rateLimiting, partSize, threadNum)
	}
}

func SyncMultiDownload(c *cos.Client, bucketName string, cosDir string, localDir string, include string, exclude string, rateLimiting float32, partSize int64, threadNum int) {
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
		SyncSingleDownload(c, bucketName, o.Key, localPath, rateLimiting, partSize, threadNum)
	}
}
