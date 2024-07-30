package util

import (
	"context"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func CreateSymlink(c *cos.Client, cosUrl StorageUrl, linkKey string) error {
	opt := &cos.ObjectPutSymlinkOptions{
		SymlinkTarget: cosUrl.(*CosUrl).Object,
	}
	_, err := c.Object.PutSymlink(context.Background(), linkKey, opt)
	return err
}

func GetSymlink(c *cos.Client, linkKey string) (res string, err error) {
	res, _, err = c.Object.GetSymlink(context.Background(), linkKey, nil)
	return res, err
}
