package util

import (
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/http"
	"net/url"
	"time"
)

var secretID, secretKey, secretToken string

// getProxyFunc 根据配置和参数返回 Transport 所需的 Proxy 函数，优先级：
// 命令行参数 > 配置文件 base 级别。若均为空或解析失败则返回 nil（不使用代理）。
// 支持 http/https/socks5 等 URL 格式，如：http://user:pass@127.0.0.1:8080 、 socks5://127.0.0.1:1080。
func getProxyFunc(config *Config, param *Param) func(*http.Request) (*url.URL, error) {
	proxyStr := param.Proxy
	if proxyStr == "" {
		proxyStr = config.Base.Proxy
	}
	if proxyStr == "" {
		return nil
	}
	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		return nil
	}
	return http.ProxyURL(proxyURL)
}

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
		authTransport := &cos.AuthorizationTransport{
			SecretID:     secretID,
			SecretKey:    secretKey,
			SessionToken: secretToken,
		}
		if proxyFn := getProxyFunc(config, param); proxyFn != nil {
			authTransport.Transport = &http.Transport{Proxy: proxyFn}
		}
		client = cos.NewClient(GenBaseURL(config, param), &http.Client{
			Transport: authTransport,
		})
	} else {
		url, err := GenURL(config, param, bucketName)
		if err != nil {
			return client, err
		}

		proxyFn := getProxyFunc(config, param)
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
					// 上限由 --max-thread-num 控制（默认 32），这里按该上限预留连接池，
					// 避免运行期连接不足导致长连接退化为短连接。
					threadNum = options[0].Operation.MaxThreadNum
					if threadNum <= 0 {
						threadNum = defaultMaxThreadNum
					}
				}
				longLinksNums = routines * threadNum
			}
			innerTransport := &http.Transport{
				MaxIdleConnsPerHost: longLinksNums,
				MaxIdleConns:        longLinksNums,
			}
			if proxyFn != nil {
				innerTransport.Proxy = proxyFn
			}
			httpClient = &http.Client{
				Transport: &cos.AuthorizationTransport{
					SecretID:     secretID,
					SecretKey:    secretKey,
					SessionToken: secretToken,
					Transport:    innerTransport,
				},
			}
		} else {
			// 若没有传递 options 或者没有设置 DisableLongLinks
			authTransport := &cos.AuthorizationTransport{
				SecretID:     secretID,
				SecretKey:    secretKey,
				SessionToken: secretToken,
			}
			if proxyFn != nil {
				authTransport.Transport = &http.Transport{Proxy: proxyFn}
			}
			httpClient = &http.Client{
				Transport: authTransport,
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

	// 服务端错误重试
	// - 未传入 FileOperations（简单操作，如 ls 等）：使用默认 10 次，间隔 1s
	// - 传入 FileOperations：完全尊重用户配置
	//   · ErrRetryNum=0 表示不重试，>0 表示按配置次数重试 5xx 错误
	//   · ErrRetryInterval 单位为秒，未指定（<=0）时默认 1s
	// 注意：time.Duration(n) 本身是纳秒，必须显式乘以 time.Second。
	if len(options) > 0 && options[0] != nil {
		client.Conf.RetryOpt.Count = options[0].Operation.ErrRetryNum
		if options[0].Operation.ErrRetryInterval > 0 {
			client.Conf.RetryOpt.Interval = time.Duration(options[0].Operation.ErrRetryInterval) * time.Second
		} else {
			client.Conf.RetryOpt.Interval = 1 * time.Second
		}
	} else {
		client.Conf.RetryOpt.Count = 10
		client.Conf.RetryOpt.Interval = 1 * time.Second
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

	authTransport := &cos.AuthorizationTransport{
		SecretID:     secretID,
		SecretKey:    secretKey,
		SessionToken: secretToken,
	}
	if proxyFn := getProxyFunc(config, param); proxyFn != nil {
		authTransport.Transport = &http.Transport{Proxy: proxyFn}
	}
	client = cos.NewClient(CreateURL(bucketIDName, protocol, param.Endpoint, false), &http.Client{
		Transport: authTransport,
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

	// 错误重试（默认 10 次，每次间隔 2 秒）
	client.Conf.RetryOpt.Count = 10
	client.Conf.RetryOpt.Interval = 2 * time.Second

	// 修改 UserAgent
	client.UserAgent = Package + "-" + Version

	return client, nil
}
