package util

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestCheckResponse(t *testing.T) {
	t.Run("2xx 状态码返回 nil", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader("")),
		}
		err := checkResponse(resp)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("201 状态码返回 nil", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 201,
			Body:       ioutil.NopCloser(strings.NewReader("")),
		}
		err := checkResponse(resp)
		if err != nil {
			t.Errorf("期望无错误，但得到: %v", err)
		}
	})

	t.Run("404 状态码返回错误", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 404,
			Body:       ioutil.NopCloser(strings.NewReader(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchKey</Code><Message>The specified key does not exist.</Message></Error>`)),
			Header:     http.Header{"Content-Type": []string{"application/xml"}},
		}
		err := checkResponse(resp)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("500 状态码返回错误", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 500,
			Body:       ioutil.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
		}
		err := checkResponse(resp)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("JSON 格式错误响应", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 400,
			Body:       ioutil.NopCloser(strings.NewReader(`{"code":400,"message":"bad request","request_id":"req-001"}`)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}
		err := checkResponse(resp)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})
}

func TestAddHeaderOptions(t *testing.T) {
	t.Run("nil 指针时返回原 header", func(t *testing.T) {
		header := http.Header{}
		result, err := addHeaderOptions(header, (*ObjectMoveOptions)(nil))
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("期望 header 为空，实际 %v", result)
		}
	})

	t.Run("有效 opt 时添加 header 字段", func(t *testing.T) {
		header := http.Header{}
		opt := &ObjectMoveOptions{
			XCosRenameSource: "/source-object",
		}
		result, err := addHeaderOptions(header, opt)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result.Get("x-cos-rename-source") != "/source-object" {
			t.Errorf("期望 x-cos-rename-source=/source-object，实际 %s", result.Get("x-cos-rename-source"))
		}
	})
}
