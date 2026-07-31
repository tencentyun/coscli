package util

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	"github.com/tencentyun/cos-go-sdk-v5"
)

var mockObjectHeadForHashFunc func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error)

func TestShowHash(t *testing.T) {
	// 复用 stat_test.go 中已有的 mockHeadFunc 会冲突，这里用独立变量
	// 注意：Object.Head 已在 stat_test.go 中打桩，这里需要在同一个 patches 对象上叠加
	// 但由于是不同 Test 函数，patches 已经 Reset，可以重新打桩
	var o *cos.ObjectService
	patches := ApplyMethodFunc(reflect.TypeOf(o), "Head",
		func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return mockObjectHeadForHashFunc(ctx, name, opt, id...)
		})
	defer patches.Reset()

	t.Run("hashType=crc64 成功获取", func(t *testing.T) {
		mockObjectHeadForHashFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			h := http.Header{}
			h.Set("x-cos-hash-crc64ecma", "12345678901234")
			h.Set("etag", `"abc123def456"`)
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
		}
		hash, base64Hash, resp, err := ShowHash(newTestClient(), "test.txt", "crc64")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if hash != "12345678901234" {
			t.Errorf("期望 hash=12345678901234，实际: %s", hash)
		}
		if base64Hash != "" {
			t.Errorf("期望 base64Hash 为空（crc64 不返回 base64），实际: %s", base64Hash)
		}
		if resp == nil {
			t.Error("期望 resp 不为 nil")
		}
	})

	t.Run("hashType=md5 成功获取", func(t *testing.T) {
		mockObjectHeadForHashFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			h := http.Header{}
			h.Set("etag", `"d41d8cd98f00b204e9800998ecf8427e"`)
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
		}
		hash, base64Hash, resp, err := ShowHash(newTestClient(), "test.txt", "md5")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if hash != "d41d8cd98f00b204e9800998ecf8427e" {
			t.Errorf("期望 hash=d41d8cd98f00b204e9800998ecf8427e，实际: %s", hash)
		}
		if base64Hash == "" {
			t.Error("期望 base64Hash 不为空")
		}
		if resp == nil {
			t.Error("期望 resp 不为 nil")
		}
	})

	t.Run("hashType 不支持时返回错误", func(t *testing.T) {
		mockObjectHeadForHashFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			h := http.Header{}
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
		}
		_, _, _, err := ShowHash(newTestClient(), "test.txt", "sha256")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK Head 调用失败时返回错误", func(t *testing.T) {
		mockObjectHeadForHashFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return nil, fmt.Errorf("mock head error")
		}
		_, _, _, err := ShowHash(newTestClient(), "test.txt", "crc64")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})
}

func TestGetHead(t *testing.T) {
	var o *cos.ObjectService
	patches := ApplyMethodFunc(reflect.TypeOf(o), "Head",
		func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return mockObjectHeadForHashFunc(ctx, name, opt, id...)
		})
	defer patches.Reset()

	t.Run("SDK Head 调用失败时返回错误", func(t *testing.T) {
		mockObjectHeadForHashFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return nil, fmt.Errorf("mock head error")
		}
		_, err := GetHead(newTestClient(), "test.txt")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("SDK Head 调用成功", func(t *testing.T) {
		mockObjectHeadForHashFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			h := http.Header{}
			h.Set("Content-Type", "text/plain")
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
		}
		resp, err := GetHead(newTestClient(), "test.txt")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if resp == nil {
			t.Error("期望 resp 不为 nil")
		}
	})

	t.Run("SDK Head 带 versionId 调用成功", func(t *testing.T) {
		mockObjectHeadForHashFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			if len(id) == 0 || id[0] != "v-001" {
				return nil, fmt.Errorf("期望 versionId=v-001")
			}
			h := http.Header{}
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
		}
		resp, err := GetHead(newTestClient(), "test.txt", "v-001")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if resp == nil {
			t.Error("期望 resp 不为 nil")
		}
	})
}

func TestCalculateHash(t *testing.T) {
	// 辅助函数：创建临时文件
	makeTmp := func(content string) string {
		f, err := os.CreateTemp("", "coscli-hash-test-*")
		if err != nil {
			t.Fatalf("创建临时文件失败: %v", err)
		}
		if _, err := f.WriteString(content); err != nil {
			t.Fatalf("写入临时文件失败: %v", err)
		}
		f.Close()
		return f.Name()
	}

	t.Run("计算本地文件 md5", func(t *testing.T) {
		tmpFile := makeTmp("hello world")
		defer os.Remove(tmpFile)

		hash, base64Hash, err := CalculateHash(tmpFile, "md5")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		// "hello world" 的 MD5 是 5eb63bbbe01eeed093cb22bb8f5acdc3
		if hash != "5eb63bbbe01eeed093cb22bb8f5acdc3" {
			t.Errorf("期望 hash=5eb63bbbe01eeed093cb22bb8f5acdc3，实际: %s", hash)
		}
		if base64Hash == "" {
			t.Error("期望 base64Hash 不为空")
		}
	})

	t.Run("计算本地文件 crc64", func(t *testing.T) {
		tmpFile := makeTmp("hello world")
		defer os.Remove(tmpFile)

		hash, base64Hash, err := CalculateHash(tmpFile, "crc64")
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if hash == "" {
			t.Error("期望 hash 不为空")
		}
		if base64Hash != "" {
			t.Errorf("期望 base64Hash 为空（crc64 不返回 base64），实际: %s", base64Hash)
		}
	})

	t.Run("不支持的 hashType 返回错误", func(t *testing.T) {
		tmpFile := makeTmp("test")
		defer os.Remove(tmpFile)

		_, _, err := CalculateHash(tmpFile, "sha256")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("文件不存在时返回错误", func(t *testing.T) {
		_, _, err := CalculateHash("/nonexistent/path/file.txt", "md5")
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})
}
