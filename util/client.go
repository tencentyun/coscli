package util

import (
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/http"
	"time"
)

var secretID, secretKey, secretToken string

// NewClient 创建一个新的客户端实例，根据配置文件加载信息。
// 参数:
// - config *Config: 配置信息
// - param *Param: 参数信息
// - bucketName string: 桶名称
// - options ...*FileOperations: 文件操作选项
// 返回:
// - client *cos.Client: 创建的客户端实例
// - err error: 错误信息
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
				// 用户显式指定，完全尊重用户配置
				longLinksNums = options[0].Operation.LongLinksNums
			} else {
				// 真实并发度 ≈ Routines（文件级并发） × ThreadNum（单文件分块并发）
				// 仅按 Routines 设置会导致分块上传时连接频繁重建
				routines := options[0].Operation.Routines
				if routines <= 0 {
					routines = 1
				}
				threadNum := options[0].Operation.ThreadNum
				if threadNum <= 0 {
					// ThreadNum=0 时由 getThreadNumByPartSize 按文件大小自动推导，
					// 最大可达 12，这里按上限预留，避免运行期连接不足
					threadNum = 12
				}
				longLinksNums = routines * threadNum
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

	// 切换域名开关，优先使用参数中的开关，若为空再使用配置文件中的开关
	CloseAutoSwitchHost := param.CloseAutoSwitchHost
	if CloseAutoSwitchHost == "" {
		CloseAutoSwitchHost = config.Base.CloseAutoSwitchHost
	}

	// 切换备用域名开关
	if CloseAutoSwitchHost == "false" {
		client.Conf.RetryOpt.AutoSwitchHost = true
	}

	// 服务端错误重试（默认10次，每次间隔1s）
	if len(options) > 0 && options[0] != nil && options[0].Operation.ErrRetryNum > 0 {
		client.Conf.RetryOpt.Count = options[0].Operation.ErrRetryNum
		if options[0].Operation.ErrRetryInterval > 0 {
			client.Conf.RetryOpt.Interval = time.Duration(options[0].Operation.ErrRetryInterval)
		} else {
			client.Conf.RetryOpt.Interval = time.Duration(1)
		}
	} else {
		client.Conf.RetryOpt.Count = 10
		client.Conf.RetryOpt.Interval = time.Duration(1)
	}

	// 修改 UserAgent
	client.UserAgent = Package + "-" + Version

	return client, nil
}

// CreateClient 根据函数参数创建客户端
// config: *Config, 配置信息
// param: *Param, 参数信息
// bucketIDName: string, 存储桶ID或名称
// 返回值: (*cos.Client, error), 创建的客户端对象和可能发生的错误
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

	// 切换域名开关，优先使用参数中的开关，若为空再使用配置文件中的开关
	CloseAutoSwitchHost := param.CloseAutoSwitchHost
	if CloseAutoSwitchHost == "" {
		CloseAutoSwitchHost = config.Base.CloseAutoSwitchHost
	}

	// 切换备用域名开关
	if CloseAutoSwitchHost == "false" {
		client.Conf.RetryOpt.AutoSwitchHost = true
	}

	// 错误重试
	client.Conf.RetryOpt.Count = 10
	client.Conf.RetryOpt.Interval = 2

	// 修改 UserAgent
	client.UserAgent = Package + "-" + Version

	return client, nil
}
