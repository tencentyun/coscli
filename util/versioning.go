package util

import (
	"context"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func GetBucketVersioning(c *cos.Client) (res *cos.BucketGetVersionResult, resp *cos.Response, err error) {
	res, resp, err = c.Bucket.GetVersioning(context.Background())
	if err != nil {
		return nil, nil, err
	}
	return res, resp, err
}

func PutBucketVersioning(c *cos.Client, status string) (resp *cos.Response, err error) {
	opt := &cos.BucketPutVersionOptions{
		Status: status,
	}
	resp, err = c.Bucket.PutVersioning(context.Background(), opt)
	return resp, err
}
