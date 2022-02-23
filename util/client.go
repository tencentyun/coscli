package util

import (
	"net/http"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// 根据桶别名，从配置文件中加载信息，创建客户端
func NewClient(config *Config, param *Param, bucketName string) *cos.Client {
	secretID := config.Base.SecretID
	secretKey := config.Base.SecretKey
	if param.SecretID != "" {
		secretID = param.SecretID
	}
	if param.SecretKey != "" {
		secretKey = param.SecretKey
	}
	if bucketName == "" { // 不指定 bucket，则创建用于发送 Service 请求的客户端
		return cos.NewClient(nil, &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  secretID,
				SecretKey: secretKey,
			},
		})
	} else {
		return cos.NewClient(GenURL(config, param, bucketName), &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  secretID,
				SecretKey: secretKey,
			},
		})
	}
}

// 根据函数参数创建客户端
func CreateClient(config *Config, param *Param, bucketIDName string) *cos.Client {
	secretID := config.Base.SecretID
	secretKey := config.Base.SecretKey
	if param.SecretID != "" {
		secretID = param.SecretID
	}
	if param.SecretKey != "" {
		secretKey = param.SecretKey
	}
	return cos.NewClient(CreateURL(bucketIDName, config.Base.Protocol, param.Endpoint), &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
		},
	})
}
