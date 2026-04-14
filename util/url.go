package util

import (
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/url"
)

// GenBucketURL generates a bucket URL based on the provided parameters.
//
// Parameters:
// - bucketIDName: The name of the bucket.
// - protocol: The protocol to use (e.g., "https").
// - endpoint: The endpoint to use (e.g., "s3.amazonaws.com").
// - customized: A boolean indicating whether to use the customized URL or not.
//
// Returns:
// A string representing the bucket URL.
func GenBucketURL(bucketIDName string, protocol string, endpoint string, customized bool) string {
	b := ""
	if customized {
		b = fmt.Sprintf("%s://%s", protocol, endpoint)
	} else {
		b = fmt.Sprintf("%s://%s.%s", protocol, bucketIDName, endpoint)
	}

	return b
}

// GenServiceURL generates the service URL based on the provided protocol and endpoint.
// If the endpoint is empty, it uses the region from the bucket to construct the URL.
// Returns the constructed service URL.
func GenServiceURL(protocol string, endpoint string) string {
	s := fmt.Sprintf("%s://%s", protocol, endpoint)
	return s
}

// GenCiURL generates the CI URL for a given bucket ID, protocol, and endpoint.
func GenCiURL(bucketIDName string, protocol string, endpoint string) string {
	c := fmt.Sprintf("%s://%s.%s", protocol, bucketIDName, endpoint)
	return c
}

// CreateURL 根据函数参数生成URL
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

// GenBaseURL 根据配置文件生成ServiceURL
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

// CreateBaseURL 根据函数参数生成ServiceURL
func CreateBaseURL(protocol string, endpoint string) *cos.BaseURL {
	service := GenServiceURL(protocol, endpoint)
	serviceURL, _ := url.Parse(service)

	return &cos.BaseURL{
		ServiceURL: serviceURL,
	}
}

// GenURL 根据配置文件生成URL
func GenURL(config *Config, param *Param, bucketName string) (url *cos.BaseURL, err error) {
	bucket, _, err := FindBucket(config, bucketName)
	if err != nil {
		return url, err
	}

	idName := bucket.Name
	endpoint := bucket.Endpoint
	if param.Endpoint != "" {
		endpoint = param.Endpoint
	}
	if endpoint == "" && bucket.Region != "" {
		endpoint = fmt.Sprintf("cos.%s.myqcloud.com", bucket.Region)
	}

	if endpoint == "" {
		return nil, fmt.Errorf("endpoint is missing")
	}

	protocol := "https"
	if config.Base.Protocol != "" {
		protocol = config.Base.Protocol
	}
	if param.Protocol != "" {
		protocol = param.Protocol
	}

	// 优先使用命令行参数 --customized，若未指定则回退到配置文件中桶的 customized 字段
	customized := param.Customized
	if !customized {
		customized = bucket.Customized
	}

	return CreateURL(idName, protocol, endpoint, customized), nil
}
