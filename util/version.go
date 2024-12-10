package util

import (
	"context"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func GetBucketVersioning(c *cos.Client) (res *cos.BucketGetVersionResult, err error) {
	res, _, err = c.Bucket.GetVersioning(context.Background())
	if err != nil {
		return nil, err
	}
	return res, err
}
