package util

import (
	"testing"
)

func TestGenBucketURL(t *testing.T) {
	tests := []struct {
		name         string
		bucketIDName string
		protocol     string
		endpoint     string
		customized   bool
		expected     string
	}{
		{"标准域名", "mybucket-1234567890", "https", "cos.ap-guangzhou.myqcloud.com", false, "https://mybucket-1234567890.cos.ap-guangzhou.myqcloud.com"},
		{"自定义域名", "mybucket-1234567890", "https", "my-custom-domain.com", true, "https://my-custom-domain.com"},
		{"http 协议", "mybucket-1234567890", "http", "cos.ap-guangzhou.myqcloud.com", false, "http://mybucket-1234567890.cos.ap-guangzhou.myqcloud.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenBucketURL(tt.bucketIDName, tt.protocol, tt.endpoint, tt.customized)
			if result != tt.expected {
				t.Errorf("期望 %s，实际: %s", tt.expected, result)
			}
		})
	}
}

func TestGenServiceURL(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		endpoint string
		expected string
	}{
		{"https 协议", "https", "service.cos.myqcloud.com", "https://service.cos.myqcloud.com"},
		{"http 协议", "http", "service.cos.myqcloud.com", "http://service.cos.myqcloud.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenServiceURL(tt.protocol, tt.endpoint)
			if result != tt.expected {
				t.Errorf("期望 %s，实际: %s", tt.expected, result)
			}
		})
	}
}

func TestGenCiURL(t *testing.T) {
	result := GenCiURL("mybucket-1234567890", "https", "ci.ap-guangzhou.myqcloud.com")
	expected := "https://mybucket-1234567890.ci.ap-guangzhou.myqcloud.com"
	if result != expected {
		t.Errorf("期望 %s，实际: %s", expected, result)
	}
}

func TestCreateURL(t *testing.T) {
	t.Run("标准域名生成完整 BaseURL", func(t *testing.T) {
		result := CreateURL("mybucket-1234567890", "https", "cos.ap-guangzhou.myqcloud.com", false)
		if result == nil {
			t.Fatal("期望 result 不为 nil")
		}
		if result.BucketURL == nil {
			t.Error("期望 BucketURL 不为 nil")
		}
		if result.ServiceURL == nil {
			t.Error("期望 ServiceURL 不为 nil")
		}
		if result.CIURL == nil {
			t.Error("期望 CIURL 不为 nil")
		}
		expected := "https://mybucket-1234567890.cos.ap-guangzhou.myqcloud.com"
		if result.BucketURL.String() != expected {
			t.Errorf("期望 BucketURL=%s，实际: %s", expected, result.BucketURL.String())
		}
	})

	t.Run("自定义域名 BucketURL 不含桶名", func(t *testing.T) {
		result := CreateURL("mybucket-1234567890", "https", "my-custom-domain.com", true)
		if result == nil {
			t.Fatal("期望 result 不为 nil")
		}
		expected := "https://my-custom-domain.com"
		if result.BucketURL.String() != expected {
			t.Errorf("期望 BucketURL=%s，实际: %s", expected, result.BucketURL.String())
		}
	})
}

func TestCreateBaseURL(t *testing.T) {
	t.Run("生成 ServiceURL", func(t *testing.T) {
		result := CreateBaseURL("https", "service.cos.myqcloud.com")
		if result == nil {
			t.Fatal("期望 result 不为 nil")
		}
		if result.ServiceURL == nil {
			t.Error("期望 ServiceURL 不为 nil")
		}
		expected := "https://service.cos.myqcloud.com"
		if result.ServiceURL.String() != expected {
			t.Errorf("期望 ServiceURL=%s，实际: %s", expected, result.ServiceURL.String())
		}
	})
}

func TestGenBaseURL(t *testing.T) {
	t.Run("endpoint 为空时返回默认 CosServiceDomain 的 ServiceURL", func(t *testing.T) {
		cfg := &Config{}
		p := &Param{Endpoint: ""}
		result := GenBaseURL(cfg, p)
		if result == nil {
			t.Fatal("期望 result 不为 nil")
		}
		if result.ServiceURL == nil {
			t.Fatal("期望 ServiceURL 不为 nil")
		}
		// endpoint 为空时使用 CosServiceDomain 作为默认值
		if result.ServiceURL.Host != CosServiceDomain {
			t.Errorf("期望 ServiceURL.Host=%s，实际: %s", CosServiceDomain, result.ServiceURL.Host)
		}
		// endpoint 为空（ls 列桶最典型场景）时也必须使用安全的 https，
		// 不能返回 nil 让 SDK 回退到 http://service.cos.myqcloud.com 默认域名。
		if result.ServiceURL.Scheme != "https" {
			t.Errorf("期望默认 scheme=https，实际: %s", result.ServiceURL.Scheme)
		}
	})

	t.Run("endpoint 为空时 config.Base.Protocol=https 生效", func(t *testing.T) {
		// 回归用例：修复前 endpoint 为空时 GenBaseURL 返回 nil，SDK 回退到
		// 写死的 http://service.cos.myqcloud.com，导致 ls 列桶忽略用户配置的 HTTPS。
		cfg := &Config{Base: BaseCfg{Protocol: "https"}}
		p := &Param{Endpoint: ""}
		result := GenBaseURL(cfg, p)
		if result == nil || result.ServiceURL == nil {
			t.Fatal("期望 ServiceURL 不为 nil")
		}
		if result.ServiceURL.Scheme != "https" {
			t.Errorf("期望 scheme=https，实际: %s", result.ServiceURL.Scheme)
		}
		if result.ServiceURL.Host != CosServiceDomain {
			t.Errorf("期望 Host=%s，实际: %s", CosServiceDomain, result.ServiceURL.Host)
		}
	})

	t.Run("endpoint 为空时 config.Base.Protocol=http 生效", func(t *testing.T) {
		cfg := &Config{Base: BaseCfg{Protocol: "http"}}
		p := &Param{Endpoint: ""}
		result := GenBaseURL(cfg, p)
		if result == nil || result.ServiceURL == nil {
			t.Fatal("期望 ServiceURL 不为 nil")
		}
		if result.ServiceURL.Scheme != "http" {
			t.Errorf("期望 scheme=http，实际: %s", result.ServiceURL.Scheme)
		}
	})

	t.Run("endpoint 为空时 param.Protocol 优先于 config.Base.Protocol", func(t *testing.T) {
		cfg := &Config{Base: BaseCfg{Protocol: "http"}}
		p := &Param{Endpoint: "", Protocol: "https"}
		result := GenBaseURL(cfg, p)
		if result == nil || result.ServiceURL == nil {
			t.Fatal("期望 ServiceURL 不为 nil")
		}
		if result.ServiceURL.Scheme != "https" {
			t.Errorf("期望 scheme=https，实际: %s", result.ServiceURL.Scheme)
		}
	})

	t.Run("使用 param.Endpoint 生成 ServiceURL", func(t *testing.T) {
		cfg := &Config{}
		p := &Param{Endpoint: "service.cos.myqcloud.com"}
		result := GenBaseURL(cfg, p)
		if result == nil {
			t.Fatal("期望 result 不为 nil")
		}
		if result.ServiceURL == nil {
			t.Error("期望 ServiceURL 不为 nil")
		}
	})

	t.Run("config.Base.Protocol 优先于默认 https", func(t *testing.T) {
		cfg := &Config{Base: BaseCfg{Protocol: "http"}}
		p := &Param{Endpoint: "service.cos.myqcloud.com"}
		result := GenBaseURL(cfg, p)
		if result == nil {
			t.Fatal("期望 result 不为 nil")
		}
		if result.ServiceURL.Scheme != "http" {
			t.Errorf("期望 scheme=http，实际: %s", result.ServiceURL.Scheme)
		}
	})

	t.Run("param.Protocol 优先于 config.Base.Protocol", func(t *testing.T) {
		cfg := &Config{Base: BaseCfg{Protocol: "http"}}
		p := &Param{Endpoint: "service.cos.myqcloud.com", Protocol: "https"}
		result := GenBaseURL(cfg, p)
		if result == nil {
			t.Fatal("期望 result 不为 nil")
		}
		if result.ServiceURL.Scheme != "https" {
			t.Errorf("期望 scheme=https，实际: %s", result.ServiceURL.Scheme)
		}
	})
}

func TestGenURL(t *testing.T) {
	cfg := &Config{
		Buckets: []Bucket{
			{Name: "test-bucket-1234567890", Alias: "test-alias", Region: "ap-guangzhou", Endpoint: "cos.ap-guangzhou.myqcloud.com"},
		},
	}

	t.Run("桶不存在时返回错误", func(t *testing.T) {
		p := &Param{}
		_, err := GenURL(cfg, p, "nonexistent-bucket")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("使用桶配置的 endpoint 生成 URL", func(t *testing.T) {
		p := &Param{}
		result, err := GenURL(cfg, p, "test-alias")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result == nil {
			t.Fatal("期望 result 不为 nil")
		}
	})

	t.Run("param.Endpoint 覆盖桶配置 endpoint", func(t *testing.T) {
		p := &Param{Endpoint: "custom.endpoint.com"}
		result, err := GenURL(cfg, p, "test-alias")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result.BucketURL.Host != "test-bucket-1234567890.custom.endpoint.com" {
			t.Errorf("期望 host 包含 custom.endpoint.com，实际: %s", result.BucketURL.Host)
		}
	})

	t.Run("endpoint 为空且无 region 时返回错误", func(t *testing.T) {
		cfgNoEndpoint := &Config{
			Buckets: []Bucket{
				{Name: "test-bucket-1234567890", Alias: "no-endpoint"},
			},
		}
		p := &Param{}
		_, err := GenURL(cfgNoEndpoint, p, "no-endpoint")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("无 endpoint 但有 region 时自动推导 endpoint", func(t *testing.T) {
		cfgRegionOnly := &Config{
			Buckets: []Bucket{
				{Name: "test-bucket-1234567890", Alias: "region-only", Region: "ap-beijing"},
			},
		}
		p := &Param{}
		result, err := GenURL(cfgRegionOnly, p, "region-only")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result.BucketURL.Host != "test-bucket-1234567890.cos.ap-beijing.myqcloud.com" {
			t.Errorf("期望 host 包含 cos.ap-beijing.myqcloud.com，实际: %s", result.BucketURL.Host)
		}
	})

	t.Run("customized=true 时使用自定义域名", func(t *testing.T) {
		cfgCustomized := &Config{
			Buckets: []Bucket{
				{Name: "test-bucket-1234567890", Alias: "custom-alias", Endpoint: "my-custom-domain.com", Customized: true},
			},
		}
		p := &Param{}
		result, err := GenURL(cfgCustomized, p, "custom-alias")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result.BucketURL.Host != "my-custom-domain.com" {
			t.Errorf("期望 host=my-custom-domain.com，实际: %s", result.BucketURL.Host)
		}
	})
}
