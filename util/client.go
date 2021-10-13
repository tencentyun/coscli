package util

import (
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/http"
)

// 根据桶别名，从配置文件中加载信息，创建客户端
func NewClient(config *Config, bucketName string) *cos.Client {
	if bucketName == "" {  // 不指定 bucket，则创建用于发送 Service 请求的客户端
		return cos.NewClient(nil, &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  config.Base.SecretID,
				SecretKey: config.Base.SecretKey,
			},
		})
	} else {
		return cos.NewClient(GenURL(config, bucketName), &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  config.Base.SecretID,
				SecretKey: config.Base.SecretKey,
			},
		})
	}
}

// 根据函数参数创建客户端
func CreateClient(config *Config, bucketIDName string, region string) *cos.Client {
	return cos.NewClient(CreateURL(bucketIDName, region), &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  config.Base.SecretID,
			SecretKey: config.Base.SecretKey,
		},
	})
}
