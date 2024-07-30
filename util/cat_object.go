package util

import (
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"io"
	"os"
)

func CatObject(c *cos.Client, cosUrl StorageUrl) error {
	opt := &cos.ObjectGetOptions{
		ResponseContentType: "text/html",
	}
	res, err := c.Object.Get(context.Background(), cosUrl.(*CosUrl).Object, opt)
	if err != nil {
		return err
	}

	io.Copy(os.Stdout, res.Body)
	fmt.Printf("\n")
	return nil
}
