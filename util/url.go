package util

import (
	"fmt"
	"net/url"
	"os"

	logger "github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func GenBucketURL(bucketIDName string, protocol string, endpoint string, customized bool) string {
	b := ""
	if customized {
		b = fmt.Sprintf("%s://%s", protocol, endpoint)
	} else {
		b = fmt.Sprintf("%s://%s.%s", protocol, bucketIDName, endpoint)
	}

	return b
}

func GenServiceURL(protocol string, endpoint string) string {
	s := fmt.Sprintf("%s://%s", protocol, endpoint)
	return s
}

func GenCiURL(bucketIDName string, protocol string, endpoint string) string {
	c := fmt.Sprintf("%s://%s.%s", protocol, bucketIDName, endpoint)
	return c
}

// 根据函数参数生成URL
func CreateURL(idName string, protocol string, endpoint string, customized bool) *cos.BaseURL {
	b := GenBucketURL(idName, protocol, endpoint, customized)
	s := GenServiceURL(protocol, endpoint)
	c := GenCiURL(idName, protocol, endpoint)

	bucketURL, _ := url.Parse(b)
	serviceURL, _ := url.Parse(s)
	ciURL, _ := url.Parse(c)

	return &cos.BaseURL{
		BucketURL:  bucketURL,
		ServiceURL: serviceURL,
		CIURL:      ciURL,
	}
}

// 根据配置文件生成ServiceURL
func GenBaseURL(config *Config, param *Param) *cos.BaseURL {
	if param.Endpoint == "" {
		return nil
	}
	endpoint := param.Endpoint

	protocol := "https"
	if config.Base.Protocol != "" {
		protocol = config.Base.Protocol
	}
	if param.Protocol != "" {
		protocol = param.Protocol
	}

	return CreateBaseURL(protocol, endpoint)
}

// 根据函数参数生成ServiceURL
func CreateBaseURL(protocol string, endpoint string) *cos.BaseURL {
	service := GenServiceURL(protocol, endpoint)
	serviceURL, _ := url.Parse(service)

	return &cos.BaseURL{
		ServiceURL: serviceURL,
	}
}

// 根据配置文件生成URL
func GenURL(config *Config, param *Param, bucketName string) *cos.BaseURL {
	bucket, _, err := FindBucket(config, bucketName)
	if err != nil {
		logger.Fatalln(err)
		os.Exit(1)
	}

	idName := bucket.Name
	endpoint := bucket.Endpoint
	if param.Endpoint != "" {
		endpoint = param.Endpoint
	}
	if endpoint == "" && bucket.Region != "" {
		endpoint = fmt.Sprintf("cos.%s.myqcloud.com", bucket.Region)
	}

	protocol := "https"
	if config.Base.Protocol != "" {
		protocol = config.Base.Protocol
	}
	if param.Protocol != "" {
		protocol = param.Protocol
	}

	customized := param.Customized

	return CreateURL(idName, protocol, endpoint, customized)
}
