package util

import (
	"context"
	"encoding/xml"
	"fmt"
	logger "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
	"math/rand"
	"net/url"
	"path/filepath"
	"time"
)

var succeedNum, failedNum, errTypeNum int

func RestoreObjects(c *cos.Client, cosUrl StorageUrl, fo *FileOperations) error {
	// 根据s.Header判断是否是融合桶或者普通桶
	s, err := c.Bucket.Head(context.Background())
	if err != nil {
		return err
	}
	logger.Infof("Start Restore %s", cosUrl.(*CosUrl).Bucket+cosUrl.(*CosUrl).Object)
	if s.Header.Get("X-Cos-Bucket-Arch") == "OFS" {
		bucketName := cosUrl.(*CosUrl).Bucket
		prefix := cosUrl.(*CosUrl).Object
		err = restoreOfsObjects(c, bucketName, prefix, fo, "")
	} else {
		err = restoreCosObjects(c, cosUrl, fo)
	}

	absErrOutputPath, _ := filepath.Abs(fo.ErrOutput.Path)

	if failedNum > 0 {
		logger.Warningf("Restore %s completed, total num: %d,success num: %d,restore error num: %d,error type num: %d,Some objects restore failed, please check the detailed information in dir %s.\n", cosUrl.(*CosUrl).Bucket+cosUrl.(*CosUrl).Object, succeedNum+failedNum+errTypeNum, succeedNum, failedNum, errTypeNum, absErrOutputPath)
	} else {
		logger.Infof("Restore %s completed,total num: %d,success num: %d,restore error num: %d,error type num: %d", cosUrl.(*CosUrl).Bucket+cosUrl.(*CosUrl).Object, succeedNum+failedNum+errTypeNum, succeedNum, failedNum, errTypeNum)
	}

	return nil
}

func restoreCosObjects(c *cos.Client, cosUrl StorageUrl, fo *FileOperations) error {
	var err error
	var objects []cos.Object
	marker := ""
	isTruncated := true

	for isTruncated {
		err, objects, _, isTruncated, marker = getCosObjectListForLs(c, cosUrl, marker, 0, true)
		if err != nil {
			return fmt.Errorf("list objects error : %v", err)
		}

		for _, object := range objects {
			if object.StorageClass == Archive || object.StorageClass == MAZArchive || object.StorageClass == DeepArchive {
				object.Key, _ = url.QueryUnescape(object.Key)
				if cosObjectMatchPatterns(object.Key, fo.Operation.Filters) {
					resp, err := TryRestoreObject(c, cosUrl.(*CosUrl).Bucket, object.Key, fo.Operation.Days, fo.Operation.RestoreMode)
					if err != nil {
						if resp != nil && resp.StatusCode == 409 {
							succeedNum += 1
						} else {
							failedNum += 1
							writeError(fmt.Sprintf("restore %s failed , errMsg:%v\n", object.Key, err), fo)
						}
					} else {
						succeedNum += 1
					}
				}
			} else {
				errTypeNum += 1
			}

		}
	}

	return nil
}

func TryRestoreObject(c *cos.Client, bucketName, objectKey string, days int, mode string) (resp *cos.Response, err error) {

	logger.Infof("Restore cos://%s/%s\n", bucketName, objectKey)
	opt := &cos.ObjectRestoreOptions{
		XMLName:       xml.Name{},
		Days:          days,
		Tier:          &cos.CASJobParameters{Tier: mode},
		XOptionHeader: nil,
	}

	for i := 0; i <= 10; i++ {
		resp, err = c.Object.PostRestore(context.Background(), objectKey, opt)
		if err != nil {
			if resp != nil && resp.StatusCode == 503 {
				if i == 10 {
					return resp, err
				} else {
					fmt.Println("Error 503: Service rate limiting. Retrying...")
					waitTime := time.Duration(rand.Intn(10)+1) * time.Second
					time.Sleep(waitTime)
					continue
				}
			} else {
				return resp, err
			}
		} else {
			return resp, err
		}
	}
	return resp, err
}

func restoreOfsObjects(c *cos.Client, bucketName, prefix string, fo *FileOperations, marker string) error {
	var err error
	var objects []cos.Object
	var commonPrefixes []string
	isTruncated := true

	for isTruncated {
		err, objects, commonPrefixes, isTruncated, marker = getOfsObjectListForLs(c, prefix, marker, 0, true)
		if err != nil {
			return fmt.Errorf("list objects error : %v", err)
		}

		for _, object := range objects {
			if object.StorageClass == Archive || object.StorageClass == MAZArchive || object.StorageClass == DeepArchive {
				object.Key, _ = url.QueryUnescape(object.Key)
				if cosObjectMatchPatterns(object.Key, fo.Operation.Filters) {
					resp, err := TryRestoreObject(c, bucketName, object.Key, fo.Operation.Days, fo.Operation.RestoreMode)
					if err != nil {
						if resp != nil && resp.StatusCode == 409 {
							succeedNum += 1
						} else {
							failedNum += 1
							writeError(fmt.Sprintf("restore %s failed , errMsg:%v\n", object.Key, err), fo)
						}
					} else {
						succeedNum += 1
					}
				}
			} else {
				errTypeNum += 1
			}
		}

		if len(commonPrefixes) > 0 {
			for _, commonPrefix := range commonPrefixes {
				commonPrefix, _ = url.QueryUnescape(commonPrefix)
				// 递归目录
				err = restoreOfsObjects(c, bucketName, commonPrefix, fo, "")
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
