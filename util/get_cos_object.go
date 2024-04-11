package util

import (
	"context"
	"fmt"
	logger "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/url"
	"os"
	"strings"
)

func GetCosKeys(c *cos.Client, cosUrl StorageUrl, keys map[string]string, fo *FileOperations) error {

	chFiles := make(chan objectInfoType, ChannelSize)
	chFinish := make(chan error, 2)
	go ReadCosKeys(keys, cosUrl, chFiles, chFinish)
	go getCosObjectList(c, cosUrl, chFiles, chFinish, fo, false, false)
	select {
	case err := <-chFinish:
		if err != nil {
			return err
		}
	}

	return nil
}

func ReadCosKeys(keys map[string]string, cosUrl StorageUrl, chObjects <-chan objectInfoType, chFinish chan<- error) {
	totalCount := 0
	fmt.Printf("\n")
	for objectInfo := range chObjects {
		totalCount++
		keys[objectInfo.relativeKey] = objectInfo.prefix
		if len(keys) > MaxSyncNumbers {
			fmt.Printf("\n")
			chFinish <- fmt.Errorf("over max sync numbers %d", MaxSyncNumbers)
			break
		}
	}

	fmt.Printf("\r%s,total cos object count:%d", cosUrl.ToString(), totalCount)
	chFinish <- nil
}

func CheckCosPathType(c *cos.Client, prefix string, limit int, retryCount ...int) (isDir bool) {
	if prefix == "" {
		return true
	}

	// cos路径若不以路径分隔符结尾，则添加
	if !strings.HasSuffix(prefix, CosSeparator) && prefix != "" {
		prefix += CosSeparator
	}

	retries := 0
	if len(retryCount) > 0 {
		retries = retryCount[0]
	}

	opt := &cos.BucketGetOptions{
		Prefix:       prefix,
		Delimiter:    "",
		EncodingType: "url",
		Marker:       "",
		MaxKeys:      limit,
	}

	res, err := tryGetBucket(c, opt, retries)
	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}

	isDir = true
	if len(res.Contents) == 0 {
		isDir = false
	}
	return isDir
}

func CheckCosObjectExist(c *cos.Client, prefix string) (exist bool) {
	if prefix == "" {
		return false
	}

	exist, err := c.Object.IsExist(context.Background(), prefix)
	if err != nil {
		logger.Fatalln(err)
	}

	return exist
}

func getCosObjectList(c *cos.Client, cosUrl StorageUrl, chObjects chan<- objectInfoType, chError chan<- error, fo *FileOperations, scanSizeNum bool, withFinishSignal bool) {
	if chObjects != nil {
		defer close(chObjects)
	}

	prefix := cosUrl.(*CosUrl).Object
	marker := ""
	limit := 0
	retries := fo.Operation.RetryNum
	delimiter := ""
	if fo.Operation.OnlyCurrentDir {
		delimiter = "/"
	}

	isTruncated := true
	for isTruncated {
		// 实例化请求参数
		opt := &cos.BucketGetOptions{
			Prefix:       prefix,
			Delimiter:    delimiter,
			EncodingType: "url",
			Marker:       marker,
			MaxKeys:      limit,
		}
		res, err := tryGetBucket(c, opt, retries)
		if err != nil {
			if scanSizeNum {
				fo.Monitor.setScanError(err)
				return
			} else {
				chError <- err
				return
			}

		}
		for _, object := range res.Contents {
			object.Key, _ = url.QueryUnescape(object.Key)
			if cosObjectMatchPatterns(object.Key, fo.Operation.Filters) {
				if scanSizeNum {
					fo.Monitor.updateScanSizeNum(object.Size, 1)
				} else {
					objPrefix := ""
					objKey := object.Key
					index := strings.LastIndex(cosUrl.(*CosUrl).Object, "/")
					if index > 0 {
						objPrefix = object.Key[:index+1]
						objKey = object.Key[index+1:]
					}
					chObjects <- objectInfoType{objPrefix, objKey, int64(object.Size), object.LastModified}
				}
			}
		}

		isTruncated = res.IsTruncated
		marker, _ = url.QueryUnescape(res.NextMarker)
	}

	if scanSizeNum {
		fo.Monitor.setScanEnd()
		freshProgress()
	}

	if withFinishSignal {
		chError <- nil
	}
}

func getCosObjectListForLs(c *cos.Client, cosUrl StorageUrl, marker string, limit int, recursive bool) (err error, objects []cos.Object, isTruncated bool, nextMarker string) {

	prefix := cosUrl.(*CosUrl).Object
	retries := 0
	delimiter := ""
	if !recursive {
		delimiter = "/"
	}

	// 实例化请求参数
	opt := &cos.BucketGetOptions{
		Prefix:       prefix,
		Delimiter:    delimiter,
		EncodingType: "url",
		Marker:       marker,
		MaxKeys:      limit,
	}
	res, err := tryGetBucket(c, opt, retries)
	if err != nil {
		return
	}

	objects = res.Contents
	isTruncated = res.IsTruncated
	marker, _ = url.QueryUnescape(res.NextMarker)
	return
}
