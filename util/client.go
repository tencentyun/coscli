package util

import (
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/http"
)

var secretID, secretKey, secretToken string

// 根据桶别名，从配置文件中加载信息，创建客户端
func NewClient(config *Config, param *Param, bucketName string, options ...*FileOperations) (client *cos.Client, err error) {
	if config.Base.Mode == "CvmRole" {
		// 若使用 CvmRole 方式，则需请求请求CAM的服务，获取临时密钥
		data, err = CamAuth(config.Base.CvmRoleName)
		if err != nil {
			return client, err
		}
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

	if secretID == "" {
		return client, fmt.Errorf("secretID is missing ")
	}

	if secretKey == "" {
		return client, fmt.Errorf("secretKey is missing")
	}

	if bucketName == "" { // 不指定 bucket，则创建用于发送 Service 请求的客户端
		if param.Endpoint == "" {
			return client, fmt.Errorf("endpoint is missing")
		}
		client = cos.NewClient(GenBaseURL(config, param), &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:     secretID,
				SecretKey:    secretKey,
				SessionToken: secretToken,
			},
		})

	} else {
		url, err := GenURL(config, param, bucketName)
		if err != nil {
			return client, err
		}

		var httpClient *http.Client
		// 如果使用长链接则调整连接池大小至并发数
		if len(options) > 0 && options[0] != nil && !options[0].Operation.DisableLongLinks {
			longLinksNums := 0
			if options[0].Operation.LongLinksNums > 0 {
				longLinksNums = options[0].Operation.LongLinksNums
			} else {
				longLinksNums = options[0].Operation.Routines
			}
			httpClient = &http.Client{
				Transport: &cos.AuthorizationTransport{
					SecretID:     secretID,
					SecretKey:    secretKey,
					SessionToken: secretToken,
					Transport: &http.Transport{
						MaxIdleConnsPerHost: longLinksNums,
						MaxIdleConns:        longLinksNums,
					},
				},
			}
		} else {
			// 若没有传递 options 或者没有设置 DisableLongLinks
			httpClient = &http.Client{
				Transport: &cos.AuthorizationTransport{
					SecretID:     secretID,
					SecretKey:    secretKey,
					SessionToken: secretToken,
				},
			}
		}

		client = cos.NewClient(url, httpClient)
	}

	// 切换备用域名开关
	if config.Base.CloseAutoSwitchHost == "true" {
		client.Conf.RetryOpt.AutoSwitchHost = false
	}

	// 错误重试
	client.Conf.RetryOpt.Count = 10
	client.Conf.RetryOpt.Interval = 2

	// 修改 UserAgent
	client.UserAgent = Package + "-" + Version

	return client, nil
}

// 根据函数参数创建客户端
func CreateClient(config *Config, param *Param, bucketIDName string) (client *cos.Client, err error) {
	if config.Base.Mode == "CvmRole" {
		// 若使用 CvmRole 方式，则需请求请求CAM的服务，获取临时密钥
		data, err = CamAuth(config.Base.CvmRoleName)
		if err != nil {
			return client, err
		}

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

	client = cos.NewClient(CreateURL(bucketIDName, protocol, param.Endpoint, false), &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:     secretID,
			SecretKey:    secretKey,
			SessionToken: secretToken,
		},
	})

	// 切换备用域名开关
	if config.Base.CloseAutoSwitchHost == "true" {
		client.Conf.RetryOpt.AutoSwitchHost = false
	}

	// 错误重试
	client.Conf.RetryOpt.Count = 10
	client.Conf.RetryOpt.Interval = 2

	// 修改 UserAgent
	client.UserAgent = Package + "-" + Version

	return client, nil
}
