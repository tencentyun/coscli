package util

import (
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/url"
	"os"
)

func GenBucketURL(bucketIDName string, region string) string {
	b := fmt.Sprintf("https://%s.cos.%s.cos.yun.unionpay.com", bucketIDName, region)
	return b
}

func GenServiceURL(region string) string {
	s := fmt.Sprintf("https://cos.%s.cos.yun.unionpay.com", region)
	return s
}

func GenCiURL(bucketIDName string, region string) string {
	c := fmt.Sprintf("https://%s.ci.%s.cos.yun.unionpay.com", bucketIDName, region)
	return c
}

// 根据函数参数生成URL
func CreateURL(idName string, region string) *cos.BaseURL {
	b := GenBucketURL(idName, region)
	s := GenServiceURL(region)
	c := GenCiURL(idName, region)

	bucketURL, _ := url.Parse(b)
	serviceURL, _ := url.Parse(s)
	ciURL, _ := url.Parse(c)

	return &cos.BaseURL{
		BucketURL:  bucketURL,
		ServiceURL: serviceURL,
		CIURL:      ciURL,
	}
}

// 根据配置文件生成URL
func GenURL(config *Config, bucketName string) *cos.BaseURL {
	bucket, _, err := FindBucket(config, bucketName)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	idName := bucket.Name
	region := bucket.Region

	return CreateURL(idName, region)
}
