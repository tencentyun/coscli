package util

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

// mockHttpClientDoFunc 已在 testmain_test.go 中声明
// http.Client.Do 已在 TestMain 中全局打桩

func TestCamAuth(t *testing.T) {
	t.Run("roleName 为空时返回错误", func(t *testing.T) {
		_, err := CamAuth("")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("HTTP 请求失败时返回错误", func(t *testing.T) {
		mockHttpClientDoFunc = func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("mock http error")
		}
		_, err := CamAuth("test-role")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("响应体 JSON 解析失败时返回错误", func(t *testing.T) {
		mockHttpClientDoFunc = func(req *http.Request) (*http.Response, error) {
			body := `not-valid-json`
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
			}, nil
		}
		_, err := CamAuth("test-role")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("Code 非 Success 时返回错误", func(t *testing.T) {
		mockHttpClientDoFunc = func(req *http.Request) (*http.Response, error) {
			body := `{"Code":"Failed","TmpSecretId":"","TmpSecretKey":"","Token":""}`
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
			}, nil
		}
		_, err := CamAuth("test-role")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("成功获取 CAM 授权", func(t *testing.T) {
		mockHttpClientDoFunc = func(req *http.Request) (*http.Response, error) {
			// 验证请求 URL 包含 roleName
			if req.URL.String() != CamUrl+"test-role" {
				return nil, fmt.Errorf("期望 URL=%s，实际: %s", CamUrl+"test-role", req.URL.String())
			}
			body := `{"Code":"Success","TmpSecretId":"tmp-id-001","TmpSecretKey":"tmp-key-001","Token":"tmp-token-001","ExpiredTime":1700000000,"Expiration":"2023-11-15T00:00:00Z"}`
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
			}, nil
		}
		result, err := CamAuth("test-role")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result.TmpSecretId != "tmp-id-001" {
			t.Errorf("期望 TmpSecretId=tmp-id-001，实际: %s", result.TmpSecretId)
		}
		if result.TmpSecretKey != "tmp-key-001" {
			t.Errorf("期望 TmpSecretKey=tmp-key-001，实际: %s", result.TmpSecretKey)
		}
		if result.Token != "tmp-token-001" {
			t.Errorf("期望 Token=tmp-token-001，实际: %s", result.Token)
		}
		if result.Code != "Success" {
			t.Errorf("期望 Code=Success，实际: %s", result.Code)
		}
	})
}
