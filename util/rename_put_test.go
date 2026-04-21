package util

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	cosgo "github.com/tencentyun/cos-go-sdk-v5"
)

// mustParseURL 解析 URL 辅助函数
func mustParseURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

// mockHttpClientDoFunc 已在 testmain_test.go 中声明

func TestPutRename(t *testing.T) {
	// http.Client.Do 已在 TestMain 中全局打桩，通过 mockHttpClientDoFunc 控制每个子测试的行为

	config := &Config{
		Base: BaseCfg{
			SecretID:  "test-id",
			SecretKey: "test-key",
		},
	}
	param := &Param{}
	ctx := context.Background()
	c := cosgo.NewClient(&cosgo.BaseURL{
		BucketURL: mustParseURL("https://test-bucket.cos.ap-guangzhou.myqcloud.com"),
	}, &http.Client{})

	t.Run("dstURL 格式错误（不包含 /）时返回错误", func(t *testing.T) {
		// dstURL 不含 /，SplitN 结果长度 < 2
		_, err := PutRename(ctx, config, param, c, "src-name", "invalid-dst-url-without-slash", false)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("http.Client.Do 失败时返回错误", func(t *testing.T) {
		mockHttpClientDoFunc = func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("mock http error")
		}
		_, err := PutRename(ctx, config, param, c, "src-name", "bucket/dst-path", false)
		if err == nil {
			t.Error("期望返回错误（HTTP 调用失败），但得到 nil")
		}
		mockHttpClientDoFunc = nil
	})

	t.Run("HTTP 返回 200 时成功", func(t *testing.T) {
		mockHttpClientDoFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("")),
				Header:     http.Header{},
			}, nil
		}
		resp, err := PutRename(ctx, config, param, c, "src-name", "bucket/dst-path", false)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		if resp == nil || resp.StatusCode != 200 {
			t.Errorf("期望 StatusCode=200，实际 %v", resp)
		}
		mockHttpClientDoFunc = nil
	})

	t.Run("HTTP 返回 404 时返回错误", func(t *testing.T) {
		mockHttpClientDoFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Body:       ioutil.NopCloser(strings.NewReader(`<?xml version="1.0"?><Error><Code>NoSuchKey</Code></Error>`)),
				Header:     http.Header{"Content-Type": []string{"application/xml"}},
			}, nil
		}
		_, err := PutRename(ctx, config, param, c, "src-name", "bucket/dst-path", false)
		if err == nil {
			t.Error("期望返回错误（404），但得到 nil")
		}
		mockHttpClientDoFunc = nil
	})

	t.Run("param 覆盖 config.Base 的 secret", func(t *testing.T) {
		mockHttpClientDoFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("")),
				Header:     http.Header{},
			}, nil
		}
		paramOverride := &Param{
			SecretID:  "param-id",
			SecretKey: "param-key",
		}
		_, err := PutRename(ctx, config, paramOverride, c, "src-name", "bucket/dst-path", true)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
		mockHttpClientDoFunc = nil
	})
}
