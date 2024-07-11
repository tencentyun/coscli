package util

import (
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/url"
)

func getOfsObjectListForLs(c *cos.Client, prefix string, marker string, limit int, recursive bool) (err error, objects []cos.Object, commonPrefixes []string, isTruncated bool, nextMarker string) {

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
	commonPrefixes = res.CommonPrefixes
	isTruncated = res.IsTruncated
	nextMarker, _ = url.QueryUnescape(res.NextMarker)
	return
}
