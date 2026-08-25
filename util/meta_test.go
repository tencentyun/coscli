package util

import (
	"net/http"
	"testing"
)

func TestMetaStringToHeader(t *testing.T) {
	t.Run("空字符串返回空 Meta", func(t *testing.T) {
		result, err := MetaStringToHeader("")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result.ContentType != "" {
			t.Errorf("期望 ContentType 为空，实际: %s", result.ContentType)
		}
		if result.MetaChange {
			t.Error("期望 MetaChange=false")
		}
	})

	t.Run("解析标准 header 字段", func(t *testing.T) {
		meta := "Content-Type:text/plain#Cache-Control:no-cache#Content-Encoding:gzip"
		result, err := MetaStringToHeader(meta)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result.ContentType != "text/plain" {
			t.Errorf("期望 ContentType=text/plain，实际: %s", result.ContentType)
		}
		if result.CacheControl != "no-cache" {
			t.Errorf("期望 CacheControl=no-cache，实际: %s", result.CacheControl)
		}
		if result.ContentEncoding != "gzip" {
			t.Errorf("期望 ContentEncoding=gzip，实际: %s", result.ContentEncoding)
		}
		if result.MetaChange {
			t.Error("期望 MetaChange=false（无 x-cos-meta-* 字段）")
		}
	})

	t.Run("解析自定义 x-cos-meta-* 字段", func(t *testing.T) {
		meta := "x-cos-meta-author:test-user#x-cos-meta-project:coscli"
		result, err := MetaStringToHeader(meta)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if !result.MetaChange {
			t.Error("期望 MetaChange=true")
		}
		if result.XCosMetaXXX == nil {
			t.Fatal("期望 XCosMetaXXX 不为 nil")
		}
		if result.XCosMetaXXX.Get("x-cos-meta-author") != "test-user" {
			t.Errorf("期望 x-cos-meta-author=test-user，实际: %s", result.XCosMetaXXX.Get("x-cos-meta-author"))
		}
		if result.XCosMetaXXX.Get("x-cos-meta-project") != "coscli" {
			t.Errorf("期望 x-cos-meta-project=coscli，实际: %s", result.XCosMetaXXX.Get("x-cos-meta-project"))
		}
	})

	t.Run("解析 Content-Disposition 和 Content-Language", func(t *testing.T) {
		meta := "Content-Disposition:attachment; filename=test.txt#Content-Language:zh-CN"
		result, err := MetaStringToHeader(meta)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result.ContentDisposition != "attachment; filename=test.txt" {
			t.Errorf("期望 ContentDisposition=attachment; filename=test.txt，实际: %s", result.ContentDisposition)
		}
		if result.ContentLanguage != "zh-CN" {
			t.Errorf("期望 ContentLanguage=zh-CN，实际: %s", result.ContentLanguage)
		}
	})

	t.Run("解析有效的 Expires 字段（RFC3339 格式）", func(t *testing.T) {
		meta := "Expires:2025-12-31T23:59:59Z"
		result, err := MetaStringToHeader(meta)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result.Expires == "" {
			t.Error("期望 Expires 不为空")
		}
	})

	t.Run("Expires 格式错误时返回错误", func(t *testing.T) {
		meta := "Expires:invalid-date-format"
		_, err := MetaStringToHeader(meta)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("解析 Content-Length 字段", func(t *testing.T) {
		meta := "Content-Length:1024"
		result, err := MetaStringToHeader(meta)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result.ContentLength != 1024 {
			t.Errorf("期望 ContentLength=1024，实际: %d", result.ContentLength)
		}
	})

	t.Run("Content-Length 非数字时返回错误", func(t *testing.T) {
		meta := "Content-Length:not-a-number"
		_, err := MetaStringToHeader(meta)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("kv 格式错误时返回错误（缺少冒号）", func(t *testing.T) {
		meta := "InvalidKeyWithoutColon"
		_, err := MetaStringToHeader(meta)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("value 中包含冒号时正确解析", func(t *testing.T) {
		meta := "Content-Type:application/json; charset=utf-8"
		result, err := MetaStringToHeader(meta)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result.ContentType != "application/json; charset=utf-8" {
			t.Errorf("期望 ContentType=application/json; charset=utf-8，实际: %s", result.ContentType)
		}
	})

	t.Run("混合标准字段和自定义字段", func(t *testing.T) {
		meta := "Content-Type:image/png#x-cos-meta-owner:alice"
		result, err := MetaStringToHeader(meta)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if result.ContentType != "image/png" {
			t.Errorf("期望 ContentType=image/png，实际: %s", result.ContentType)
		}
		if !result.MetaChange {
			t.Error("期望 MetaChange=true")
		}
		if result.XCosMetaXXX == nil {
			t.Fatal("期望 XCosMetaXXX 不为 nil")
		}
		if result.XCosMetaXXX.Get("x-cos-meta-owner") != "alice" {
			t.Errorf("期望 x-cos-meta-owner=alice，实际: %s", result.XCosMetaXXX.Get("x-cos-meta-owner"))
		}
	})

	t.Run("XCosMetaXXX 类型为 *http.Header", func(t *testing.T) {
		meta := "x-cos-meta-key:value"
		result, err := MetaStringToHeader(meta)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if _, ok := interface{}(result.XCosMetaXXX).(*http.Header); !ok {
			t.Error("期望 XCosMetaXXX 类型为 *http.Header")
		}
	})
}
