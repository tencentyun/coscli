package util

import (
	"net/http"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cfg := &Config{
		Base: BaseCfg{
			SecretID:  "test-secret-id",
			SecretKey: "test-secret-key",
			Protocol:  "https",
		},
		Buckets: []Bucket{
			{Name: "test-bucket-1234567890", Alias: "test-alias", Region: "ap-guangzhou", Endpoint: "cos.ap-guangzhou.myqcloud.com"},
		},
	}

	t.Run("SecretID 为空时返回错误", func(t *testing.T) {
		emptyCfg := &Config{Base: BaseCfg{SecretKey: "key"}}
		p := &Param{}
		_, err := NewClient(emptyCfg, p, "")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SecretKey 为空时返回错误", func(t *testing.T) {
		emptyCfg := &Config{Base: BaseCfg{SecretID: "id"}}
		p := &Param{}
		_, err := NewClient(emptyCfg, p, "")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("bucketName 为空时创建 Service 客户端", func(t *testing.T) {
		p := &Param{Endpoint: "service.cos.myqcloud.com"}
		c, err := NewClient(cfg, p, "")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})

	t.Run("bucketName 不存在时返回错误", func(t *testing.T) {
		p := &Param{}
		_, err := NewClient(cfg, p, "nonexistent-bucket")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("bucketName 存在时成功创建客户端", func(t *testing.T) {
		p := &Param{}
		c, err := NewClient(cfg, p, "test-alias")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})

	t.Run("param.SecretID 覆盖 config.SecretID", func(t *testing.T) {
		p := &Param{SecretID: "param-secret-id"}
		c, err := NewClient(cfg, p, "test-alias")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})

	t.Run("param.SecretKey 覆盖 config.SecretKey", func(t *testing.T) {
		p := &Param{SecretKey: "param-secret-key"}
		c, err := NewClient(cfg, p, "test-alias")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})

	t.Run("param.SessionToken 被正确设置", func(t *testing.T) {
		p := &Param{SessionToken: "test-session-token"}
		c, err := NewClient(cfg, p, "test-alias")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})

	t.Run("CloseAutoSwitchHost=false 时开启备用域名切换", func(t *testing.T) {
		p := &Param{CloseAutoSwitchHost: "false"}
		c, err := NewClient(cfg, p, "test-alias")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !c.Conf.RetryOpt.AutoSwitchHost {
			t.Error("期望 AutoSwitchHost=true")
		}
	})

	t.Run("config.Base.CloseAutoSwitchHost=false 时开启备用域名切换", func(t *testing.T) {
		cfgSwitch := &Config{
			Base: BaseCfg{
				SecretID:            "test-id",
				SecretKey:           "test-key",
				CloseAutoSwitchHost: "false",
			},
			Buckets: []Bucket{
				{Name: "test-bucket-1234567890", Alias: "test-alias", Region: "ap-guangzhou", Endpoint: "cos.ap-guangzhou.myqcloud.com"},
			},
		}
		p := &Param{}
		c, err := NewClient(cfgSwitch, p, "test-alias")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !c.Conf.RetryOpt.AutoSwitchHost {
			t.Error("期望 AutoSwitchHost=true")
		}
	})

	t.Run("传入 FileOperations 时使用长连接池", func(t *testing.T) {
		p := &Param{}
		fo := &FileOperations{
			Operation: Operation{
				Routines: 5,
			},
		}
		c, err := NewClient(cfg, p, "test-alias", fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})

	t.Run("传入 FileOperations 且 LongLinksNums>0 时使用指定连接数", func(t *testing.T) {
		p := &Param{}
		fo := &FileOperations{
			Operation: Operation{
				Routines:      5,
				LongLinksNums: 10,
			},
		}
		c, err := NewClient(cfg, p, "test-alias", fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})

	t.Run("传入 FileOperations 且 ThreadNum>0 时连接池按 Routines*ThreadNum 估算", func(t *testing.T) {
		p := &Param{}
		fo := &FileOperations{
			Operation: Operation{
				Routines:  4,
				ThreadNum: 6,
			},
		}
		c, err := NewClient(cfg, p, "test-alias", fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})

	t.Run("传入 FileOperations 且 Routines<=0 时按 1*ThreadNum 兜底", func(t *testing.T) {
		p := &Param{}
		fo := &FileOperations{
			Operation: Operation{
				Routines:  0,
				ThreadNum: 0,
			},
		}
		c, err := NewClient(cfg, p, "test-alias", fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})

	t.Run("传入 FileOperations 且 MaxThreadNum>0 时连接池按 Routines*MaxThreadNum 估算", func(t *testing.T) {
		p := &Param{}
		fo := &FileOperations{
			Operation: Operation{
				Routines:     3,
				ThreadNum:    0,
				MaxThreadNum: 16,
			},
		}
		c, err := NewClient(cfg, p, "test-alias", fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})

	t.Run("传入 FileOperations 且 MaxThreadNum<=0 时使用默认 32 兜底", func(t *testing.T) {
		p := &Param{}
		fo := &FileOperations{
			Operation: Operation{
				Routines:     3,
				ThreadNum:    0,
				MaxThreadNum: 0,
			},
		}
		c, err := NewClient(cfg, p, "test-alias", fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})

	t.Run("传入 FileOperations 且 DisableLongLinks=true 时不使用长连接池", func(t *testing.T) {
		p := &Param{}
		fo := &FileOperations{
			Operation: Operation{
				Routines:         5,
				DisableLongLinks: true,
			},
		}
		c, err := NewClient(cfg, p, "test-alias", fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})

	t.Run("传入 FileOperations 且 ErrRetryNum>0 时使用自定义重试次数", func(t *testing.T) {
		p := &Param{}
		fo := &FileOperations{
			Operation: Operation{
				ErrRetryNum:      3,
				ErrRetryInterval: 2,
			},
		}
		c, err := NewClient(cfg, p, "test-alias", fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c.Conf.RetryOpt.Count != 3 {
			t.Errorf("期望 RetryOpt.Count=3，实际: %d", c.Conf.RetryOpt.Count)
		}
		// Interval 应为 2 秒而非 2 纳秒
		if c.Conf.RetryOpt.Interval != 2*time.Second {
			t.Errorf("期望 RetryOpt.Interval=2s，实际: %v", c.Conf.RetryOpt.Interval)
		}
	})

	t.Run("传入 FileOperations 且 ErrRetryNum=0 时不进行重试", func(t *testing.T) {
		p := &Param{}
		fo := &FileOperations{
			Operation: Operation{
				ErrRetryNum: 0,
			},
		}
		c, err := NewClient(cfg, p, "test-alias", fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c.Conf.RetryOpt.Count != 0 {
			t.Errorf("期望 RetryOpt.Count=0（不重试），实际: %d", c.Conf.RetryOpt.Count)
		}
	})

	t.Run("传入 FileOperations 且 ErrRetryInterval 未指定时使用默认 1 秒", func(t *testing.T) {
		p := &Param{}
		fo := &FileOperations{
			Operation: Operation{
				ErrRetryNum: 5,
				// ErrRetryInterval 使用零值
			},
		}
		c, err := NewClient(cfg, p, "test-alias", fo)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c.Conf.RetryOpt.Interval != 1*time.Second {
			t.Errorf("期望 RetryOpt.Interval=1s，实际: %v", c.Conf.RetryOpt.Interval)
		}
	})

	t.Run("未传入 FileOperations 时使用默认重试次数 10", func(t *testing.T) {
		p := &Param{}
		c, err := NewClient(cfg, p, "test-alias")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c.Conf.RetryOpt.Count != 10 {
			t.Errorf("期望 RetryOpt.Count=10（默认），实际: %d", c.Conf.RetryOpt.Count)
		}
		if c.Conf.RetryOpt.Interval != 1*time.Second {
			t.Errorf("期望 RetryOpt.Interval=1s（默认），实际: %v", c.Conf.RetryOpt.Interval)
		}
	})

	t.Run("UserAgent 被正确设置", func(t *testing.T) {
		p := &Param{}
		c, err := NewClient(cfg, p, "test-alias")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		expectedUA := Package + "-" + Version
		if c.UserAgent != expectedUA {
			t.Errorf("期望 UserAgent=%s，实际: %s", expectedUA, c.UserAgent)
		}
	})
}

func TestCreateClient(t *testing.T) {
	cfg := &Config{
		Base: BaseCfg{
			SecretID:  "test-secret-id",
			SecretKey: "test-secret-key",
			Protocol:  "https",
		},
	}

	t.Run("基本创建成功", func(t *testing.T) {
		p := &Param{Endpoint: "cos.ap-guangzhou.myqcloud.com"}
		c, err := CreateClient(cfg, p, "test-bucket-1234567890")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})

	t.Run("config.Base.Protocol 被正确使用", func(t *testing.T) {
		cfgHttp := &Config{
			Base: BaseCfg{
				SecretID:  "test-id",
				SecretKey: "test-key",
				Protocol:  "http",
			},
		}
		p := &Param{Endpoint: "cos.ap-guangzhou.myqcloud.com"}
		c, err := CreateClient(cfgHttp, p, "test-bucket-1234567890")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})

	t.Run("param.Protocol 覆盖 config.Base.Protocol", func(t *testing.T) {
		p := &Param{Endpoint: "cos.ap-guangzhou.myqcloud.com", Protocol: "http"}
		c, err := CreateClient(cfg, p, "test-bucket-1234567890")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})

	t.Run("CloseAutoSwitchHost=false 时开启备用域名切换", func(t *testing.T) {
		p := &Param{
			Endpoint:            "cos.ap-guangzhou.myqcloud.com",
			CloseAutoSwitchHost: "false",
		}
		c, err := CreateClient(cfg, p, "test-bucket-1234567890")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !c.Conf.RetryOpt.AutoSwitchHost {
			t.Error("期望 AutoSwitchHost=true")
		}
	})

	t.Run("UserAgent 被正确设置", func(t *testing.T) {
		p := &Param{Endpoint: "cos.ap-guangzhou.myqcloud.com"}
		c, err := CreateClient(cfg, p, "test-bucket-1234567890")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		expectedUA := Package + "-" + Version
		if c.UserAgent != expectedUA {
			t.Errorf("期望 UserAgent=%s，实际: %s", expectedUA, c.UserAgent)
		}
	})

	t.Run("param.SecretID 覆盖 config.SecretID", func(t *testing.T) {
		p := &Param{
			SecretID: "param-id",
			Endpoint: "cos.ap-guangzhou.myqcloud.com",
		}
		c, err := CreateClient(cfg, p, "test-bucket-1234567890")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if c == nil {
			t.Fatal("期望 client 不为 nil")
		}
	})
}

// TestGetProxyFunc 测试 getProxyFunc 所有分支：
// 1. param.Proxy 为空 + config.Base.Proxy 为空 → 返回 nil
// 2. config.Base.Proxy 非空 → 返回有效 ProxyFunc
// 3. param.Proxy 非空 → 覆盖 config.Base.Proxy
// 4. 非法 URL → 返回 nil（注：url.Parse 非常宽松，这里用包含控制字符的字符串触发失败）
func TestGetProxyFunc(t *testing.T) {
	t.Run("both empty returns nil", func(t *testing.T) {
		cfg := &Config{}
		p := &Param{}
		fn := getProxyFunc(cfg, p)
		if fn != nil {
			t.Errorf("expected nil proxy func when both empty, got non-nil")
		}
	})

	t.Run("use config base proxy", func(t *testing.T) {
		cfg := &Config{}
		cfg.Base.Proxy = "http://127.0.0.1:8080"
		p := &Param{}
		fn := getProxyFunc(cfg, p)
		if fn == nil {
			t.Fatalf("expected non-nil proxy func from config")
		}
		req, _ := http.NewRequest("GET", "https://examplebucket-1234567890.cos.ap-guangzhou.myqcloud.com/", nil)
		u, err := fn(req)
		if err != nil {
			t.Fatalf("proxy func returned error: %v", err)
		}
		if u == nil || u.Host != "127.0.0.1:8080" {
			t.Errorf("expected proxy host 127.0.0.1:8080, got %v", u)
		}
	})

	t.Run("param proxy overrides config", func(t *testing.T) {
		cfg := &Config{}
		cfg.Base.Proxy = "http://127.0.0.1:8080"
		p := &Param{Proxy: "socks5://10.0.0.1:1080"}
		fn := getProxyFunc(cfg, p)
		if fn == nil {
			t.Fatalf("expected non-nil proxy func from param")
		}
		req, _ := http.NewRequest("GET", "https://examplebucket-1234567890.cos.ap-guangzhou.myqcloud.com/", nil)
		u, err := fn(req)
		if err != nil {
			t.Fatalf("proxy func returned error: %v", err)
		}
		if u == nil || u.Scheme != "socks5" || u.Host != "10.0.0.1:1080" {
			t.Errorf("expected param proxy socks5://10.0.0.1:1080, got %v", u)
		}
	})

	t.Run("invalid url returns nil", func(t *testing.T) {
		cfg := &Config{}
		// 使用包含 ASCII 控制字符的 URL，使 url.Parse 返回 error
		p := &Param{Proxy: "http://\x7f:8080"}
		fn := getProxyFunc(cfg, p)
		if fn != nil {
			t.Errorf("expected nil proxy func for invalid url, got non-nil")
		}
	})
}
