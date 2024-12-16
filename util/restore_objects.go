package util

import (
	"context"
	"encoding/xml"
	"fmt"
	logger "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/url"
)

var succeedNum, failedNum, errTypeNum int

func RestoreObject(c *cos.Client, bucketName, objectKey string, days int, mode string) error {
	opt := &cos.ObjectRestoreOptions{
		XMLName:       xml.Name{},
		Days:          days,
		Tier:          &cos.CASJobParameters{Tier: mode},
		XOptionHeader: nil,
	}

	logger.Infof("Restore cos://%s/%s\n", bucketName, objectKey)
	_, err := c.Object.PostRestore(context.Background(), objectKey, opt)
	if err != nil {
		logger.Errorln(err)
		return err
	}
	return nil
}

func RestoreObjects(c *cos.Client, cosUrl StorageUrl, days int, mode string, filters []FilterOptionType) error {
	// 根据s.Header判断是否是融合桶或者普通桶
	s, err := c.Bucket.Head(context.Background())
	if err != nil {
		return err
	}
	logger.Infof("Start Restore %s", cosUrl.(*CosUrl).Bucket+cosUrl.(*CosUrl).Object)
	if s.Header.Get("X-Cos-Bucket-Arch") == "OFS" {
		bucketName := cosUrl.(*CosUrl).Bucket
		prefix := cosUrl.(*CosUrl).Object
		err = restoreOfsObjects(c, bucketName, prefix, filters, days, mode, "")
	} else {
		err = restoreCosObjects(c, cosUrl, filters, days, mode)
	}
	logger.Infof("Restore %s completed,total num: %d,success num: %d,restore error num: %d,error type num: %d", cosUrl.(*CosUrl).Bucket+cosUrl.(*CosUrl).Object, succeedNum+failedNum+errTypeNum, succeedNum, failedNum, errTypeNum)
	return nil
}

func restoreCosObjects(c *cos.Client, cosUrl StorageUrl, filters []FilterOptionType, days int, mode string) error {
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
				if cosObjectMatchPatterns(object.Key, filters) {
					err := RestoreObject(c, cosUrl.(*CosUrl).Bucket, object.Key, days, mode)
					if err != nil {
						failedNum += 1
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

func restoreOfsObjects(c *cos.Client, bucketName, prefix string, filters []FilterOptionType, days int, mode string, marker string) error {
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
				if cosObjectMatchPatterns(object.Key, filters) {
					err := RestoreObject(c, bucketName, object.Key, days, mode)
					if err != nil {
						failedNum += 1
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
				err = restoreOfsObjects(c, bucketName, commonPrefix, filters, days, mode, "")
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
