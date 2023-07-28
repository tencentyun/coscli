package util

import (
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/http"
)

var secretID, secretKey, secretToken string

// 根据桶别名，从配置文件中加载信息，创建客户端
func NewClient(config *Config, param *Param, bucketName string) *cos.Client {
	if config.Base.Mode == "CvmRole" {
		// 若使用 CvmRole 方式，则需请求请求CAM的服务，获取临时密钥
		data := CamAuth(config.Base.CvmRoleName)
		secretID = data.TmpSecretId
		secretKey = data.TmpSecretKey
		secretToken = data.Token
	} else {
		// SecretKey 方式则直接获取用户配置文件中设置的密钥
		secretID = config.Base.SecretID
		secretKey = config.Base.SecretKey
		secretToken = config.Base.SessionToken
	}

	if param.SecretID != "" {
		secretID = param.SecretID
	}
	if param.SecretKey != "" {
		secretKey = param.SecretKey
	}
	if param.SessionToken != "" {
		secretToken = param.SessionToken
	}
	if bucketName == "" { // 不指定 bucket，则创建用于发送 Service 请求的客户端
		return cos.NewClient(nil, &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:     secretID,
				SecretKey:    secretKey,
				SessionToken: secretToken,
			},
		})
	} else {
		return cos.NewClient(GenURL(config, param, bucketName), &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:     secretID,
				SecretKey:    secretKey,
				SessionToken: secretToken,
			},
		})
	}
}

// 根据函数参数创建客户端
func CreateClient(config *Config, param *Param, bucketIDName string) *cos.Client {
	if config.Base.Mode == "CvmRole" {
		// 若使用 CvmRole 方式，则需请求请求CAM的服务，获取临时密钥
		data := CamAuth(config.Base.CvmRoleName)
		secretID = data.TmpSecretId
		secretKey = data.TmpSecretKey
		secretToken = data.Token
	} else {
		// SecretKey 方式则直接获取用户配置文件中设置的密钥
		secretID = config.Base.SecretID
		secretKey = config.Base.SecretKey
		secretToken = config.Base.SessionToken
	}

	if param.SecretID != "" {
		secretID = param.SecretID
	}
	if param.SecretKey != "" {
		secretKey = param.SecretKey
	}
	if param.SessionToken != "" {
		secretToken = param.SessionToken
	}
	return cos.NewClient(CreateURL(bucketIDName, config.Base.Protocol, param.Endpoint), &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:     secretID,
			SecretKey:    secretKey,
			SessionToken: secretToken,
		},
	})
}
