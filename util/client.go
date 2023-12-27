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
	// 若参数中有传 SecretID 或 SecretKey ，需将之前赋值的SessionToken置为空，否则会出现使用参数的 SecretID 和 SecretKey ，却使用了CvmRole方式返回的token，导致鉴权失败
	if param.SecretID != "" {
		secretID = param.SecretID
		secretToken = ""
	}
	if param.SecretKey != "" {
		secretKey = param.SecretKey
		secretToken = ""
	}
	if param.SessionToken != "" {
		secretToken = param.SessionToken
	}

	var client *cos.Client
	if bucketName == "" { // 不指定 bucket，则创建用于发送 Service 请求的客户端
		client = cos.NewClient(GenBaseURL(config, param), &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:     secretID,
				SecretKey:    secretKey,
				SessionToken: secretToken,
			},
		})

	} else {
		client = cos.NewClient(GenURL(config, param, bucketName), &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:     secretID,
				SecretKey:    secretKey,
				SessionToken: secretToken,
			},
		})
	}

	// 切换备用域名开关
	if config.Base.CloseAutoSwitchHost == "true" {
		client.Conf.RetryOpt.AutoSwitchHost = false
	}

	return client
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

	// 若参数中有传 SecretID 或 SecretKey ，需将之前赋值的SessionToken置为空，否则会出现使用参数的 SecretID 和 SecretKey ，却使用了CvmRole方式返回的token，导致鉴权失败
	if param.SecretID != "" {
		secretID = param.SecretID
		secretToken = ""
	}
	if param.SecretKey != "" {
		secretKey = param.SecretKey
		secretToken = ""
	}
	if param.SessionToken != "" {
		secretToken = param.SessionToken
	}

	protocol := "https"
	if config.Base.Protocol != "" {
		protocol = config.Base.Protocol
	}
	if param.Protocol != "" {
		protocol = param.Protocol
	}
	return cos.NewClient(CreateURL(bucketIDName, protocol, param.Endpoint), &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:     secretID,
			SecretKey:    secretKey,
			SessionToken: secretToken,
		},
	})
}
