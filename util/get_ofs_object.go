package util

import (
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/url"
	"strings"
)

func getOfsObjectList(c *cos.Client, cosUrl StorageUrl, chObjects chan<- objectInfoType, chError chan<- error, fo *FileOperations, scanSizeNum bool, withFinishSignal bool) {
	if chObjects != nil {
		defer close(chObjects)
	}

	prefix := cosUrl.(*CosUrl).Object
	marker := ""
	limit := 0

	delimiter := ""
	if fo.Operation.OnlyCurrentDir {
		delimiter = "/"
	}

	err := getOfsObjectListRecursion(c, cosUrl, chObjects, chError, fo, scanSizeNum, prefix, marker, limit, delimiter)

	if err != nil && scanSizeNum {
		fo.Monitor.setScanError(err)
	}

	if scanSizeNum {
		fo.Monitor.setScanEnd()
		freshProgress()
	}

	if withFinishSignal {
		chError <- err
	}
}

func getOfsObjectListRecursion(c *cos.Client, cosUrl StorageUrl, chObjects chan<- objectInfoType, chError chan<- error, fo *FileOperations, scanSizeNum bool, prefix string, marker string, limit int, delimiter string) error {
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
		res, err := tryGetObjects(c, opt)
		if err != nil {
			return err
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

		if len(res.CommonPrefixes) > 0 {
			for _, commonPrefix := range res.CommonPrefixes {
				commonPrefix, _ = url.QueryUnescape(commonPrefix)

				if cosObjectMatchPatterns(commonPrefix, fo.Operation.Filters) {
					if scanSizeNum {
						fo.Monitor.updateScanSizeNum(0, 1)
					} else {
						objPrefix := ""
						objKey := commonPrefix
						index := strings.LastIndex(cosUrl.(*CosUrl).Object, "/")
						if index > 0 {
							objPrefix = commonPrefix[:index+1]
							objKey = commonPrefix[index+1:]
						}
						chObjects <- objectInfoType{objPrefix, objKey, int64(0), ""}
					}
				}

				if delimiter == "" {
					err = getOfsObjectListRecursion(c, cosUrl, chObjects, chError, fo, scanSizeNum, commonPrefix, marker, limit, delimiter)
					if err != nil {
						return err
					}
				}
			}
		}

		isTruncated = res.IsTruncated
		marker, _ = url.QueryUnescape(res.NextMarker)
	}
	return nil
}

func getOfsObjectListForLs(c *cos.Client, prefix string, marker string, limit int, recursive bool) (err error, objects []cos.Object, commonPrefixes []string, isTruncated bool, nextMarker string) {

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
	res, err := tryGetObjects(c, opt)
	if err != nil {
		return
	}

	objects = res.Contents
	commonPrefixes = res.CommonPrefixes
	isTruncated = res.IsTruncated
	nextMarker, _ = url.QueryUnescape(res.NextMarker)
	return
}

func GetOfsKeys(c *cos.Client, cosUrl StorageUrl, keys map[string]string, fo *FileOperations) error {

	chFiles := make(chan objectInfoType, ChannelSize)
	chFinish := make(chan error, 2)
	go ReadCosKeys(keys, cosUrl, chFiles, chFinish)
	go getOfsObjectList(c, cosUrl, chFiles, chFinish, fo, false, false)
	select {
	case err := <-chFinish:
		if err != nil {
			return err
		}
	}

	return nil
}
