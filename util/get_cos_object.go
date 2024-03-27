package util

import (
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
	go GetCosKeyList(c, cosUrl, chFiles, chFinish, fo)
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
		//fmt.Printf("\r%s,total cos object count:%d", cosUrl.ToString(), totalCount)
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

func GetCosKeyList(c *cos.Client, cosUrl StorageUrl, chObjects chan<- objectInfoType, chFinish chan<- error, fo *FileOperations) {
	cosPath := cosUrl.(*CosUrl)
	err := getObjectList(c, cosPath, chObjects, fo)
	if err != nil {
		chFinish <- err
	}
}

func getObjectList(c *cos.Client, cosUrl StorageUrl, chObjects chan<- objectInfoType, fo *FileOperations) error {
	defer close(chObjects)
	// 列表参数
	prefix := cosUrl.(*CosUrl).Object
	marker := ""
	limit := 1000
	retries := fo.Operation.RetryNum
	delimiter := ""
	if fo.Operation.OnlyCurrentDir {
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

	isTruncated := true
	for isTruncated {
		res, err := tryGetBucket(c, opt, retries)
		if err != nil {
			logger.Fatalln(err)
			os.Exit(1)
		}

		for _, object := range res.Contents {
			objPrefix := ""
			objKey := object.Key
			index := strings.LastIndex(cosUrl.(*CosUrl).Object, "/")
			if index > 0 {
				objPrefix = object.Key[:index+1]
				objKey = object.Key[index+1:]
			}

			if cosObjectMatchPatterns(object.Key, fo.Operation.Filters) {
				chObjects <- objectInfoType{objPrefix, objKey, int64(object.Size), object.LastModified}
			}
		}

		isTruncated = res.IsTruncated
		marker, _ = url.QueryUnescape(res.NextMarker)
	}

	return nil
}
