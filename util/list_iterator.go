package util

import (
	"context"
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/url"
)

func GetObjectsListIterator(c *cos.Client, prefix, marker string, include, exclude string) (objects []cos.Object,
	isTruncated bool, nextMarker string, commonPrefixes []string, err error) {
	opt := &cos.BucketGetOptions{
		Prefix:       prefix,
		Delimiter:    "",
		EncodingType: "",
		Marker:       marker,
		MaxKeys:      0,
	}

	res, _, err := c.Bucket.Get(context.Background(), opt)
	if err != nil {
		return objects, isTruncated, nextMarker, commonPrefixes, err
	}

	objects = append(objects, res.Contents...)
	commonPrefixes = res.CommonPrefixes

	isTruncated = res.IsTruncated
	nextMarker, _ = url.QueryUnescape(res.NextMarker)

	if len(include) > 0 {
		objects = MatchCosPattern(objects, include, true)
	}
	if len(exclude) > 0 {
		objects = MatchCosPattern(objects, exclude, false)
	}

	return objects, isTruncated, nextMarker, commonPrefixes, err
}
