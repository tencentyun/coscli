package util

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// newTestClient 构造一个最小可用的 cos.Client（SDK 方法已被打桩，不会真实发起请求）
func newTestClient() *cos.Client {
	return cos.NewClient(nil, nil)
}

// mockHeadFunc 是可替换的 Head mock 函数，通过变量控制行为，避免多次打桩同一方法
var mockHeadFunc func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error)

func TestStatObject(t *testing.T) {
	// Object.Head 已在 TestMain 中全局打桩，通过 mockHeadFunc 变量控制每个子测试的行为

	t.Run("SDK Head 调用失败（无 versionId）", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return nil, fmt.Errorf("mock head error")
		}
		info, err := StatObject(newTestClient(), "test.txt", "", BucketTypeCos)
		if err == nil {
			t.Errorf("期望返回错误，但得到 nil")
		}
		if info != nil {
			t.Errorf("期望 info 为 nil，但得到 %v", info)
		}
	})

	t.Run("SDK Head 调用失败（有 versionId）", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			return nil, fmt.Errorf("mock head with version error")
		}
		info, err := StatObject(newTestClient(), "test.txt", "v-001", BucketTypeCos)
		if err == nil {
			t.Errorf("期望返回错误，但得到 nil")
		}
		if info != nil {
			t.Errorf("期望 info 为 nil，但得到 %v", info)
		}
	})

	t.Run("成功获取对象元数据（无 versionId）", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			h := http.Header{}
			h.Set("ETag", `"abc123"`)
			h.Set("Content-Type", "text/plain")
			h.Set("Content-Length", "1024")
			h.Set("Last-Modified", "Wed, 16 Apr 2025 10:00:00 GMT")
			h.Set("Cache-Control", "no-cache")
			h.Set("Content-Disposition", "attachment; filename=test.txt")
			h.Set("Content-Encoding", "gzip")
			h.Set("Content-Language", "zh-CN")
			h.Set("Expires", "Thu, 17 Apr 2025 10:00:00 GMT")
			h.Set("x-cos-storage-class", "STANDARD")
			h.Set("x-cos-version-id", "")
			h.Set("x-cos-object-type", "normal")
			h.Set("x-cos-hash-crc64ecma", "12345678901234")
			h.Set("x-cos-meta-author", "test-user")
			h.Set("x-cos-meta-project", "coscli")
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
		}
		info, err := StatObject(newTestClient(), "test.txt", "", BucketTypeCos)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if info == nil {
			t.Fatal("期望 info 不为 nil")
		}
		assertEqual(t, `"abc123"`, info.ETag, "ETag")
		assertEqual(t, "text/plain", info.ContentType, "ContentType")
		assertEqual(t, "1024", info.ContentLength, "ContentLength")
		assertEqual(t, "Wed, 16 Apr 2025 10:00:00 GMT", info.LastModified, "LastModified")
		assertEqual(t, "no-cache", info.CacheControl, "CacheControl")
		assertEqual(t, "attachment; filename=test.txt", info.ContentDisposition, "ContentDisposition")
		assertEqual(t, "gzip", info.ContentEncoding, "ContentEncoding")
		assertEqual(t, "zh-CN", info.ContentLanguage, "ContentLanguage")
		assertEqual(t, "Thu, 17 Apr 2025 10:00:00 GMT", info.Expires, "Expires")
		assertEqual(t, "STANDARD", info.StorageClass, "StorageClass")
		assertEqual(t, "normal", info.ObjectType, "ObjectType")
		assertEqual(t, "12345678901234", info.CRC64, "CRC64")
		// 自定义元数据
		assertEqual(t, "test-user", info.CustomMeta["x-cos-meta-author"], "CustomMeta[author]")
		assertEqual(t, "coscli", info.CustomMeta["x-cos-meta-project"], "CustomMeta[project]")
	})

	t.Run("成功获取对象元数据（有 versionId）", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			h := http.Header{}
			h.Set("ETag", `"def456"`)
			h.Set("Content-Type", "application/json")
			h.Set("Content-Length", "512")
			h.Set("Last-Modified", "Wed, 16 Apr 2025 12:00:00 GMT")
			h.Set("x-cos-version-id", "v-001")
			h.Set("x-cos-storage-class", "STANDARD_IA")
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
		}
		info, err := StatObject(newTestClient(), "test.json", "v-001", BucketTypeCos)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if info == nil {
			t.Fatal("期望 info 不为 nil")
		}
		assertEqual(t, `"def456"`, info.ETag, "ETag")
		assertEqual(t, "application/json", info.ContentType, "ContentType")
		assertEqual(t, "v-001", info.VersionId, "VersionId")
		assertEqual(t, "STANDARD_IA", info.StorageClass, "StorageClass")
	})

	t.Run("无自定义元数据时 CustomMeta 为空 map", func(t *testing.T) {
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			h := http.Header{}
			h.Set("ETag", `"xyz789"`)
			h.Set("Content-Type", "image/png")
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: h}}, nil
		}
		info, err := StatObject(newTestClient(), "image.png", "", BucketTypeCos)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if info == nil {
			t.Fatal("期望 info 不为 nil")
		}
		if info.CustomMeta == nil {
			t.Error("期望 CustomMeta 不为 nil")
		}
		if len(info.CustomMeta) != 0 {
			t.Errorf("期望 CustomMeta 为空，但得到 %v", info.CustomMeta)
		}
	})

	t.Run("OFS 桶指定 versionId 时不携带 versionId", func(t *testing.T) {
		var capturedIds []string
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			capturedIds = id
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
		}
		_, err := StatObject(newTestClient(), "file.txt", "v-001", BucketTypeOfs)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(capturedIds) != 0 {
			t.Errorf("OFS 桶不应携带 versionId，但捕获到: %v", capturedIds)
		}
	})

	t.Run("COS 桶指定 versionId 时携带 versionId", func(t *testing.T) {
		var capturedIds []string
		mockHeadFunc = func(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
			capturedIds = id
			return &cos.Response{Response: &http.Response{StatusCode: 200, Header: http.Header{}}}, nil
		}
		_, err := StatObject(newTestClient(), "file.txt", "v-001", BucketTypeCos)
		if err != nil {
			t.Fatalf("期望无错误，但得到: %v", err)
		}
		if len(capturedIds) != 1 || capturedIds[0] != "v-001" {
			t.Errorf("COS 桶应携带 versionId [v-001]，但捕获到: %v", capturedIds)
		}
	})
}

// assertEqual 是一个简单的断言辅助函数
func assertEqual(t *testing.T, expected, actual, field string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: 期望 %q，实际 %q", field, expected, actual)
	}
}
